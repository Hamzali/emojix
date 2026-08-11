package usecase

import (
	"context"
	"crypto/rand"
	"database/sql"
	"emojix/model"
	"emojix/repository"
	"emojix/service"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	mathRand "math/rand"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
)

type GameUpdateHandler = func(notifType string, data string) error

// ErrUserNotFound is returned when a user id is not present in the store
// (e.g. stale cookie after a DB reset).
var ErrUserNotFound = errors.New("user not found")

// ErrUserNotInGame is returned when the caller is not an active player in the game.
var ErrUserNotInGame = errors.New("user not in the game")

type EmojixUsecase interface {
	InitUser(ctx context.Context) (model.User, error)
	GetUser(ctx context.Context, userID string) (model.User, error)
	ListWordLists(ctx context.Context) ([]model.WordList, error)
	InitGame(ctx context.Context, userID string, listID string) (model.Game, error)
	JoinGame(ctx context.Context, gameID string, userID string) error
	PickWord(ctx context.Context, gameID string, userID string, wordID string) error
	// Guess records a guess. correct is true when the guess matches the word.
	Guess(ctx context.Context, gameID string, userID string, word string) (correct bool, err error)
	Message(ctx context.Context, gameID string, userID string, word string) error
	GameState(ctx context.Context, gameID string, userID string) (model.GameState, error)
	GameUpdates(ctx context.Context, gameID string, userID string, handler GameUpdateHandler) error
	KickInactiveUser(ctx context.Context, gameID, userID string) error
	Leaderboard(ctx context.Context, gameID, userID string) ([]model.LeaderboardEntry, error)
	GameWord(ctx context.Context, gameID, userID string) (string, error)
}

func NewEmojixUsecase(
	userRepo repository.UserRepository,
	gameRepo repository.GameRepository,
	wordRepo repository.WordRepository,
	unitOfWorkFactory repository.UnitOfWorkFactory,
	gameNotifier service.GameNotifier,
	gameLoop service.GameLoop,
	clock service.Clock,
) EmojixUsecase {
	uc := &emojixUsecase{
		userRepo,
		gameRepo,
		wordRepo,
		unitOfWorkFactory,
		gameNotifier,
		gameLoop,
		clock,
	}

	gameLoop.SetOnTurnEndHandler(func(ctx context.Context, gameID string) {
		uc.onTurnEnd(ctx, gameID)
	})

	return uc
}

type emojixUsecase struct {
	userRepo          repository.UserRepository
	gameRepo          repository.GameRepository
	wordRepo          repository.WordRepository
	unitOfWorkFactory repository.UnitOfWorkFactory
	gameNotifier      service.GameNotifier
	gameLoop          service.GameLoop
	clock             service.Clock
}

func (e *emojixUsecase) GameUpdates(ctx context.Context, gameID string, userID string, handler GameUpdateHandler) error {
	gameSubCh, cleanup := e.gameNotifier.Sub(gameID, userID)
	defer cleanup()
	for {

		select {
		case notif := <-gameSubCh:
			err := handler(notif.GetType(), notif.GetData())
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}

type UserLeftNotification struct {
	UserID string
}

func (gmn *UserLeftNotification) GetType() string {
	return "left"
}

func (gmn *UserLeftNotification) GetData() string {
	return fmt.Sprintf("%s", gmn.UserID)
}

func (e *emojixUsecase) KickInactiveUser(ctx context.Context, gameID, userID string) error {
	activePlayers := e.gameNotifier.Subs(gameID)
	if slices.Contains(activePlayers, userID) {
		return nil
	}
	err := e.gameRepo.SetPlayerState(ctx, gameID, userID, model.InactivePlayerState)

	go e.gameNotifier.Pub(gameID, userID, &UserLeftNotification{userID})

	return err
}

// GotItMessage is the public system line shown for a correct guess so the raw
// word is never leaked to unsolved players (or anyone via chat history).
func GotItMessage(nickname string) string {
	return nickname + " got it!"
}

// MaskMessage rewrites a stored chat/guess line for display. Correct guesses
// become a system announcement for everyone.
func MaskMessage(content, word, nickname string) (display string, isSystem bool) {
	if strings.EqualFold(content, word) {
		return GotItMessage(nickname), true
	}
	return content, false
}

const turnDuration = time.Second * 60

// ErrNoWords is returned when a new turn cannot be created because the word
// repository has no words to pick from.
var ErrNoWords = errors.New("no words available to pick for a new turn")

func (e *emojixUsecase) ListWordLists(ctx context.Context) ([]model.WordList, error) {
	return e.wordRepo.GetLists(ctx)
}

func (e *emojixUsecase) GameState(ctx context.Context, gameID string, currentUserID string) (model.GameState, error) {
	gameState := model.GameState{}
	players, err := e.gameRepo.GetPlayers(ctx, gameID)
	if err != nil {
		return gameState, err
	}

	activePlayers := e.filterActivePlayers(players)
	err = e.isPlayerInGame(currentUserID, activePlayers)
	if err != nil {
		return gameState, err
	}

	messages, err := e.gameRepo.GetMessages(ctx, gameID)
	if err != nil {
		return gameState, err
	}

	scores, err := e.gameRepo.GetScores(ctx, gameID)
	if err != nil {
		return gameState, err
	}

	latestTurn, err := e.gameRepo.GetLatestTurn(ctx, gameID)
	if err != nil {
		return gameState, err
	}

	gameState.TurnID = latestTurn.ID
	gameState.GameID = gameID
	gameState.CurrentUserID = currentUserID
	gameState.IsTeller = latestTurn.TellerID == currentUserID
	gameState.AwaitingPick = latestTurn.WordID == ""
	gameState.TellerNickname = tellerNickname(activePlayers, latestTurn.TellerID)

	leaderboard := e.buildLeaderboard(currentUserID, latestTurn.ID, latestTurn.TellerID, scores, activePlayers)
	gameState.Leaderboard = leaderboard

	if gameState.AwaitingPick {
		if gameState.IsTeller {
			opts, err := e.loadWordOptions(ctx, latestTurn)
			if err != nil {
				return gameState, err
			}
			gameState.WordOptions = opts
		}
		return gameState, nil
	}

	word, err := e.wordRepo.FindByID(ctx, latestTurn.WordID)
	if err != nil {
		return gameState, err
	}

	// Prefer the live per-turn board; fall back to word.Hint for pre-migration rows.
	if latestTurn.EmojiHint != "" {
		gameState.Hint = latestTurn.EmojiHint
	} else {
		gameState.Hint = word.Hint
	}
	gameState.TurnStartedAt = latestTurn.StartedAt
	gameState.LetterCount, gameState.WordCount = wordShape(word.Word)

	// check if turn ended and decide to mask word or not
	leaderboardEntryMap := map[string]model.LeaderboardEntry{}
	var currPlayerEntry model.LeaderboardEntry
	allGuessed := true
	for _, entry := range leaderboard {
		if entry.Me {
			currPlayerEntry = entry
		}

		leaderboardEntryMap[entry.PlayerID] = entry

		// Teller already knows the word; they don't need to guess.
		if entry.PlayerID == latestTurn.TellerID {
			continue
		}
		if !entry.GuessedWord {
			allGuessed = false
		}
	}
	// Solo teller (no guessers): turn is not "all guessed".
	if e.countGuessers(activePlayers, latestTurn.TellerID) == 0 {
		allGuessed = false
	}

	wordMaskRegex := regexp.MustCompile(`\w`)
	gameWord := word.Word
	// Teller always sees the real word; others only after guessing.
	if !gameState.IsTeller && !currPlayerEntry.GuessedWord {
		gameWord = wordMaskRegex.ReplaceAllString(gameWord, "*")
	}

	turnEndTime := gameState.TurnStartedAt.Add(turnDuration)
	now := e.clock.Now()
	turnTimedOut := !latestTurn.StartedAt.IsZero() && now.After(turnEndTime)

	gameState.TurnEnded = allGuessed || turnTimedOut
	gameState.Word = gameWord

	// prepare messages (repo order is oldest→newest; keep newest at bottom)
	gameMessages := []model.GameStateMessage{}
	for _, msg := range messages {
		le := leaderboardEntryMap[msg.PlayerID]
		display, isSystem := MaskMessage(msg.Content, word.Word, le.Nickname)
		gm := model.GameStateMessage{
			Me:       le.Me,
			Content:  display,
			Nickname: le.Nickname,
			IsSystem: isSystem,
		}
		gameMessages = append(gameMessages, gm)
	}
	gameState.Messages = gameMessages

	return gameState, nil

}

func (e *emojixUsecase) loadWordOptions(ctx context.Context, turn model.GameTurn) ([]model.Word, error) {
	ids := []string{turn.OptionA, turn.OptionB, turn.OptionC}
	opts := make([]model.Word, 0, 3)
	for _, id := range ids {
		if id == "" {
			continue
		}
		w, err := e.wordRepo.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
		opts = append(opts, w)
	}
	return opts, nil
}

func (e *emojixUsecase) countGuessers(players []model.Player, tellerID string) int {
	n := 0
	for _, p := range players {
		if p.ID != tellerID {
			n++
		}
	}
	return n
}

// generateRandomID generates a secure random session ID
func generateRandomID() (string, error) {
	// Create a byte slice of size 16 (128 bits)
	bytes := make([]byte, 16)

	// Fill the byte slice with random values
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Encode the bytes to a hexadecimal string
	return hex.EncodeToString(bytes), nil
}

// NICKNAME Generation
var animals = []string{
	"cat",
	"dog",
	"mouse",
}

var adjectives = []string{
	"silly",
	"handsome",
	"angry",
}

func pickRandItem(items []string) string {
	return items[mathRand.Intn(len(items))]
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}

	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func generateNickname() string {
	animal := pickRandItem(animals)
	adj := pickRandItem(adjectives)
	return fmt.Sprintf("%s%s", capitalize(adj), capitalize(animal))
}

func (e *emojixUsecase) InitUser(ctx context.Context) (model.User, error) {
	userID, err := generateRandomID()
	if err != nil {
		return model.User{}, err
	}

	nickname := generateNickname()

	err = e.userRepo.CreateOrUpdate(ctx, userID, repository.UserCreateOrUpdateParams{Nickname: nickname})

	if err != nil {
		return model.User{}, err
	}

	return model.User{ID: userID, Nickname: nickname}, nil
}

func (e *emojixUsecase) GetUser(ctx context.Context, userID string) (model.User, error) {
	user, err := e.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, ErrUserNotFound
		}
		return model.User{}, err
	}
	return user, nil
}

func pickWordOptions(unused []model.Word, n int) []model.Word {
	if n > len(unused) {
		n = len(unused)
	}
	// Fisher-Yates partial shuffle for n picks.
	shuffled := slices.Clone(unused)
	for i := 0; i < n; i++ {
		j := i + mathRand.Intn(len(shuffled)-i)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	return shuffled[:n]
}

func (e *emojixUsecase) InitGame(ctx context.Context, userID string, listID string) (model.Game, error) {
	uow, err := e.unitOfWorkFactory.New(ctx)
	if err != nil {
		return model.Game{}, err
	}
	defer uow.Rollback()

	gameRepo := uow.GameRepository()

	game, err := gameRepo.Create(ctx, listID)
	if err != nil {
		return model.Game{}, err
	}

	err = gameRepo.AddPlayer(ctx, game.ID, userID)
	if err != nil {
		return model.Game{}, err
	}

	err = e.newGameTurn(ctx, gameRepo, game.ID, listID)
	if err != nil {
		return model.Game{}, err
	}

	if err = uow.Commit(); err != nil {
		return model.Game{}, err
	}

	// Start the game loop AFTER commit. Timer waits for BeginTurn (teller pick).
	e.gameLoop.Start(context.Background(), game.ID, turnDuration)

	return game, nil
}

var (
	ErrNotTeller       = errors.New("only the teller can pick the word")
	ErrAlreadyPicked   = errors.New("word already picked for this turn")
	ErrInvalidOption   = errors.New("word is not one of the turn options")
	ErrTellerEmojiOnly = errors.New("teller may only send emoji")
	ErrEmptyMessage    = errors.New("message is empty")
)

// tellerMessagePenalty is taken from the teller's current-turn points only
// (not banked totals). Floor is 0 — messages are free once turn points are gone.
const tellerMessagePenalty = 2

type WordPickedNotification struct{}

func (n *WordPickedNotification) GetType() string { return "wordpicked" }
func (n *WordPickedNotification) GetData() string { return "" }

type NewTurnNotification struct{}

func (n *NewTurnNotification) GetType() string { return "newturn" }
func (n *NewTurnNotification) GetData() string { return "" }

func (e *emojixUsecase) PickWord(ctx context.Context, gameID, userID, wordID string) error {
	turn, err := e.gameRepo.GetLatestTurn(ctx, gameID)
	if err != nil {
		return err
	}
	if turn.WordID != "" {
		return ErrAlreadyPicked
	}
	if turn.TellerID != userID {
		return ErrNotTeller
	}
	if wordID != turn.OptionA && wordID != turn.OptionB && wordID != turn.OptionC {
		return ErrInvalidOption
	}

	word, err := e.wordRepo.FindByID(ctx, wordID)
	if err != nil {
		return err
	}

	if err := e.gameRepo.SetTurnWord(ctx, turn.ID, wordID, word.Hint); err != nil {
		return err
	}

	e.gameLoop.BeginTurn(gameID)
	go e.gameNotifier.PubAll(gameID, &WordPickedNotification{})
	return nil
}

type GameMsgNotification struct {
	UserID   string
	Nickname string
	Content  string
	IsSystem bool
}

func (gmn *GameMsgNotification) GetType() string {
	return "msg"
}

func (gmn *GameMsgNotification) GetData() string {
	if gmn.IsSystem {
		return fmt.Sprintf("%s,%s,%s,1", gmn.UserID, gmn.Nickname, gmn.Content)
	}
	return fmt.Sprintf("%s,%s,%s", gmn.UserID, gmn.Nickname, gmn.Content)
}

func (gmn *GameMsgNotification) ParseData(data string) error {
	items := strings.Split(data, ",")
	if len(items) < 3 || len(items) > 4 {
		return errors.New("invalid msg content")
	}

	gmn.UserID = items[0]
	gmn.Nickname = items[1]
	gmn.Content = items[2]
	gmn.IsSystem = len(items) == 4 && items[3] == "1"

	return nil
}

type GameCorrectGuessNotification struct {
	UserID   string
	Nickname string
}

func (gmn *GameCorrectGuessNotification) GetType() string {
	return "guessed"
}

func (gmn *GameCorrectGuessNotification) GetData() string {
	return fmt.Sprintf("%s,%s", gmn.UserID, gmn.Nickname)
}

type GameTurnEndNotification struct {
}

func (gmn *GameTurnEndNotification) GetType() string {
	return "turnended"
}

func (gmn *GameTurnEndNotification) GetData() string {
	return ""
}

func (e *emojixUsecase) Guess(ctx context.Context, gameID string, userID string, content string) (bool, error) {
	currPlayer, err := e.userRepo.FindByID(ctx, userID)
	if err != nil {
		return false, err
	}

	turn, err := e.gameRepo.GetLatestTurn(ctx, gameID)
	if err != nil {
		return false, err
	}
	if turn.WordID == "" {
		return false, errors.New("turn has not started yet")
	}
	// Teller already knows the word; ignore their guesses.
	if turn.TellerID == userID {
		return false, errors.New("teller cannot guess")
	}
	turnID := turn.ID

	word, err := e.wordRepo.FindByID(ctx, turn.WordID)
	if err != nil {
		return false, err
	}
	gameWord := word.Word

	uow, err := e.unitOfWorkFactory.New(ctx)
	if err != nil {
		return false, err
	}
	defer uow.Rollback()

	gameRepo := uow.GameRepository()

	msg, err := gameRepo.SendMessage(ctx, gameID, turnID, userID, content)
	if err != nil {
		return false, err
	}

	// TODO: make fancier word comparison
	guessedWord := strings.EqualFold(content, gameWord)
	if !guessedWord {
		// Only publish after a successful commit so a failed commit does not
		// broadcast a chat message that was never persisted.
		if err = uow.Commit(); err != nil {
			return false, err
		}
		go e.gameNotifier.Pub(gameID, userID, &GameMsgNotification{UserID: userID, Nickname: currPlayer.Nickname, Content: content})
		return false, nil
	}

	// check if the turn is ended
	players, err := gameRepo.GetPlayers(ctx, gameID)
	if err != nil {
		return false, err
	}

	scores, err := gameRepo.GetScores(ctx, gameID)
	if err != nil {
		return false, err
	}

	// Duplicate-correct-guess: this user already scored on this turn. Idempotent
	// no-op — no second AddScore, no guessed notif, no EndGameTurn. The
	// SendMessage above is still committed so the chat record stays consistent.
	for _, s := range scores {
		if s.PlayerID == userID && s.TurnID == turnID {
			return true, uow.Commit()
		}
	}

	// Count distinct non-teller players who already guessed this turn. Teller
	// rows (bonus / message penalty) must not count — they are not guesses.
	// Current user is about to be scored, so total after this AddScore is len+1.
	// TODO: the scoring formula below drifts from the README (+5 base, +1/sec
	// left, -1 wrong guess). Alignment is a separate backlog decision.
	guessedPlayers := map[string]struct{}{}
	for _, s := range scores {
		if s.TurnID == turnID && s.PlayerID != turn.TellerID {
			guessedPlayers[s.PlayerID] = struct{}{}
		}
	}
	totalGuessers := len(guessedPlayers) + 1

	// Use active guessers only (exclude teller and inactive).
	activePlayers := e.filterActivePlayers(players)
	guesserCount := e.countGuessers(activePlayers, turn.TellerID)
	if guesserCount == 0 {
		guesserCount = 1 // avoid div by zero; solo edge case
	}
	pointCoeff := guesserCount / totalGuessers
	if pointCoeff < 1 {
		pointCoeff = 1
	}
	basePoint := 10
	point := basePoint * pointCoeff

	err = gameRepo.AddScore(ctx, gameID, userID, msg.ID, turnID, point)
	if err != nil {
		return false, err
	}

	// Teller scores per correct guess (more solvers → more teller points over the turn).
	const tellerPointsPerCorrectGuess = 5
	if turn.TellerID != "" {
		err = gameRepo.AddScore(ctx, gameID, turn.TellerID, msg.ID, turnID, tellerPointsPerCorrectGuess)
		if err != nil {
			return false, err
		}
	}

	err = uow.Commit()
	if err != nil {
		return false, err
	}

	systemLine := GotItMessage(currPlayer.Nickname)
	go e.gameNotifier.Pub(gameID, userID, &GameMsgNotification{
		UserID: userID, Nickname: currPlayer.Nickname, Content: systemLine, IsSystem: true,
	})
	go e.gameNotifier.Pub(gameID, userID, &GameCorrectGuessNotification{userID, currPlayer.Nickname})

	if totalGuessers == e.countGuessers(activePlayers, turn.TellerID) {
		e.gameLoop.EndGameTurn(gameID)
	}

	return true, nil
}

func (e *emojixUsecase) onTurnEnd(ctx context.Context, gameID string) {
	e.gameNotifier.PubAll(gameID, &GameTurnEndNotification{})
	<-e.clock.After(5 * time.Second)

	game, err := e.gameRepo.FindByID(ctx, gameID)
	if err != nil {
		log.Printf("failed to load game for new turn: %v", err)
		e.gameLoop.StopGame(gameID)
		return
	}

	err = e.newGameTurn(ctx, e.gameRepo, gameID, game.ListID)
	if err != nil {
		log.Printf("failed to create new turn, retrying: %v", err)
		<-e.clock.After(time.Second)
		err = e.newGameTurn(ctx, e.gameRepo, gameID, game.ListID)
	}
	if err != nil {
		log.Printf("failed to create new turn after retry, stopping game: %v", err)
		e.gameLoop.StopGame(gameID)
		return
	}
	e.gameNotifier.PubAll(gameID, &NewTurnNotification{})
}

func (e *emojixUsecase) newGameTurn(ctx context.Context, gr repository.GameRepository, gameID, listID string) error {
	unused, err := e.wordRepo.GetUnusedByList(ctx, listID, gameID)
	if err != nil {
		return err
	}
	if len(unused) == 0 {
		return ErrNoWords
	}

	options := pickWordOptions(unused, 3)
	// Pad to 3 slots by repeating the last option when the list is nearly empty.
	for len(options) < 3 {
		options = append(options, options[len(options)-1])
	}

	players, err := gr.GetPlayers(ctx, gameID)
	if err != nil {
		return err
	}
	active := e.filterActivePlayers(players)
	if len(active) == 0 {
		return errors.New("no active players")
	}
	// Stable teller rotation by join order.
	slices.SortFunc(active, func(a, b model.Player) int {
		return a.JoinedAt.Compare(b.JoinedAt)
	})
	turnCount, err := gr.CountTurns(ctx, gameID)
	if err != nil {
		return err
	}
	teller := active[turnCount%len(active)]

	_, err = gr.AddTurn(ctx, repository.AddTurnParams{
		GameID:   gameID,
		TellerID: teller.ID,
		OptionA:  options[0].ID,
		OptionB:  options[1].ID,
		OptionC:  options[2].ID,
	})
	return err
}

// IsEmojiOnly reports whether s is non-empty and contains no letters or digits
// (emoji / symbols / spaces only). Used to keep teller chat emoji-only.
func IsEmojiOnly(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	has := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return false
		}
		has = true
	}
	return has
}

func (e *emojixUsecase) Message(ctx context.Context, gameID string, userID string, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return ErrEmptyMessage
	}

	turn, err := e.gameRepo.GetLatestTurn(ctx, gameID)
	if err != nil {
		return err
	}

	currPlayer, err := e.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	isTeller := turn.TellerID == userID
	if isTeller && !IsEmojiOnly(content) {
		return ErrTellerEmojiOnly
	}

	msg, err := e.gameRepo.SendMessage(ctx, gameID, turn.ID, userID, content)
	if err != nil {
		return err
	}

	if isTeller {
		scores, err := e.gameRepo.GetScores(ctx, gameID)
		if err != nil {
			return err
		}
		turnPts := 0
		for _, s := range scores {
			if s.PlayerID == userID && s.TurnID == turn.ID {
				turnPts += s.Score
			}
		}
		if turnPts > 0 {
			penalty := tellerMessagePenalty
			if penalty > turnPts {
				penalty = turnPts
			}
			// ponytail: non-atomic message+penalty; UOW if we ever see partial fails
			if err := e.gameRepo.AddScore(ctx, gameID, userID, msg.ID, turn.ID, -penalty); err != nil {
				return err
			}
		}
	}

	go e.gameNotifier.Pub(gameID, userID, &GameMsgNotification{UserID: userID, Nickname: currPlayer.Nickname, Content: content})

	return nil
}

func (e *emojixUsecase) buildLeaderboard(currentUserID string, latestTurnID string, tellerID string, scores []model.Score, activePlayers []model.Player) []model.LeaderboardEntry {
	leaderboardEntries := []model.LeaderboardEntry{}
	isGuessedWord := func(playerID string) bool {
		if playerID == tellerID {
			return true // teller counts as "done" for display
		}
		for _, score := range scores {
			if score.PlayerID == playerID && score.TurnID == latestTurnID {
				return true
			}
		}
		return false
	}

	scoreMap := map[string]int{}
	for _, score := range scores {
		scoreMap[score.PlayerID] += score.Score
	}

	for _, player := range activePlayers {
		score := scoreMap[player.ID]
		if score < 0 {
			score = 0
		}
		entry := model.LeaderboardEntry{
			PlayerID:    player.ID,
			Nickname:    player.Nickname,
			Me:          player.ID == currentUserID,
			GuessedWord: isGuessedWord(player.ID),
			IsTeller:    player.ID == tellerID,
			Score:       score,
		}

		leaderboardEntries = append(leaderboardEntries, entry)
	}

	return leaderboardEntries

}

// wordShape returns letter count (spaces excluded) and whitespace-separated
// word count for the secret phrase shown under the mask blanks.
func wordShape(word string) (letters, words int) {
	fields := strings.Fields(word)
	words = len(fields)
	for _, f := range fields {
		letters += len([]rune(f))
	}
	return letters, words
}

func tellerNickname(players []model.Player, tellerID string) string {
	for _, p := range players {
		if p.ID == tellerID {
			return p.Nickname
		}
	}
	return ""
}

func (e *emojixUsecase) filterActivePlayers(players []model.Player) []model.Player {
	activePlayers := []model.Player{}
	for _, p := range players {
		if p.State == model.InactivePlayerState {
			continue
		}

		activePlayers = append(activePlayers, p)
	}

	return activePlayers
}

func (e *emojixUsecase) isPlayerInGame(currentUserID string, activePlayers []model.Player) error {
	if len(activePlayers) == 0 {
		return ErrUserNotInGame
	}
	hasPlayer := slices.ContainsFunc(activePlayers, func(p model.Player) bool {
		return p.ID == currentUserID
	})
	if !hasPlayer {
		return ErrUserNotInGame
	}

	return nil
}

func (e *emojixUsecase) Leaderboard(ctx context.Context, gameID, currentUserID string) ([]model.LeaderboardEntry, error) {
	leaderboardEntries := []model.LeaderboardEntry{}
	players, err := e.gameRepo.GetPlayers(ctx, gameID)
	if err != nil {
		return leaderboardEntries, err
	}

	activePlayers := e.filterActivePlayers(players)
	err = e.isPlayerInGame(currentUserID, activePlayers)
	if err != nil {
		return leaderboardEntries, err
	}

	scores, err := e.gameRepo.GetScores(ctx, gameID)
	if err != nil {
		return leaderboardEntries, err
	}

	latestTurn, err := e.gameRepo.GetLatestTurn(ctx, gameID)
	if err != nil {
		return leaderboardEntries, err
	}

	leaderboardEntries = e.buildLeaderboard(currentUserID, latestTurn.ID, latestTurn.TellerID, scores, activePlayers)

	return leaderboardEntries, nil
}

func (e *emojixUsecase) GameWord(ctx context.Context, gameID, currentUserID string) (string, error) {
	latestTurn, err := e.gameRepo.GetLatestTurn(ctx, gameID)
	if err != nil {
		return "", err
	}
	if latestTurn.WordID == "" {
		return "", nil
	}

	word, err := e.wordRepo.FindByID(ctx, latestTurn.WordID)
	if err != nil {
		return "", err
	}

	if latestTurn.TellerID == currentUserID {
		return word.Word, nil
	}

	scores, err := e.gameRepo.GetScores(ctx, gameID)
	if err != nil {
		return "", err
	}

	guessedWord := false
	for _, score := range scores {
		if score.PlayerID == currentUserID && score.TurnID == latestTurn.ID {
			guessedWord = true
			break
		}
	}

	wordMaskRegex := regexp.MustCompile(`\w`)
	gameWord := word.Word

	if !guessedWord {
		gameWord = wordMaskRegex.ReplaceAllString(gameWord, "*")
	}

	return gameWord, nil
}
