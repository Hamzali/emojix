package usecase

import (
	"context"
	"crypto/rand"
	"emojix/model"
	"emojix/repository"
	"emojix/service"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"maps"
	mathRand "math/rand"
	"regexp"
	"slices"
	"strings"
	"time"
)

type GameUpdateHandler = func(notifType string, data string) error
type EmojixUsecase interface {
	InitUser(ctx context.Context) (model.User, error)
	InitGame(ctx context.Context, userID string) (model.Game, error)
	JoinGame(ctx context.Context, gameID string, userID string) error
	Guess(ctx context.Context, gameID string, userID string, word string) error
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
) EmojixUsecase {
	return &emojixUsecase{
		userRepo,
		gameRepo,
		wordRepo,
		unitOfWorkFactory,
		gameNotifier,
	}
}

type emojixUsecase struct {
	userRepo          repository.UserRepository
	gameRepo          repository.GameRepository
	wordRepo          repository.WordRepository
	unitOfWorkFactory repository.UnitOfWorkFactory
	gameNotifier      service.GameNotifier
}

func (e *emojixUsecase) GameUpdates(ctx context.Context, gameID string, userID string, handler GameUpdateHandler) error {
	gameSubCh := e.gameNotifier.Sub(gameID, userID)
	for {

		select {
		case notif := <-gameSubCh:
			err := handler(notif.GetType(), notif.GetData())
			if err != nil {
				e.gameNotifier.Unsub(userID)
				return err
			}
		case <-ctx.Done():
			e.gameNotifier.Unsub(userID)
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

func (gmn *UserLeftNotification) ParseData(data string) error {
	return nil
}

func maskContent(content string, word string, currUserID string, senderUserID string, guessedWord bool) string {
	notSelf := currUserID != senderUserID
	if strings.EqualFold(word, content) && (notSelf || !guessedWord) {
		return "***"
	}

	return content
}

func (e *emojixUsecase) GameState(ctx context.Context, gameID string, currentUserID string) (model.GameState, error) {
	gameState := model.GameState{}
	players, err := e.gameRepo.GetPlayers(ctx, gameID)
	if err != nil {
		return gameState, err
	}

	activePlayers := []model.Player{}
	for _, p := range players {
		if p.State == model.InactivePlayerState {
			continue
		}

		activePlayers = append(activePlayers, p)
	}

	hasPlayer := slices.ContainsFunc(activePlayers, func(p model.Player) bool {
		return p.ID == currentUserID
	})

	if !hasPlayer {
		return gameState, errors.New("user not in the game")
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

	word, err := e.wordRepo.FindByID(ctx, latestTurn.WordID)
	if err != nil {
		return gameState, err
	}

	leaderboardMap := map[string]model.LeaderboardEntry{}

	isGuessedWord := func(playerID string) bool {
		for _, score := range scores {
			if score.PlayerID == playerID && score.TurnID == latestTurn.ID {
				return true
			}
		}
		return false
	}

	scoreMap := map[string]int{}
	for _, score := range scores {
		scoreMap[score.PlayerID] += score.Score
	}

	turnEnded := true
	for _, player := range activePlayers {
		entry := model.LeaderboardEntry{
			Nickname:    player.Nickname,
			Me:          player.ID == currentUserID,
			GuessedWord: isGuessedWord(player.ID),
			Score:       scoreMap[player.ID],
		}

		// if at least one person not guessed yet then turn is still not ended
		if !entry.GuessedWord {
			turnEnded = false
		}

		leaderboardMap[player.ID] = entry
	}

	gameMessages := []model.GameStateMessage{}

	for _, msg := range messages {
		le := leaderboardMap[msg.PlayerID]

		gm := model.GameStateMessage{
			Me:       msg.PlayerID == currentUserID,
			Content:  msg.Content,
			Nickname: le.Nickname,
		}

		gm.Content = maskContent(gm.Content, word.Word, currentUserID, msg.PlayerID, le.GuessedWord)

		gameMessages = append(gameMessages, gm)
	}

	// newest message at top
	slices.Reverse(gameMessages)

	currentPlayer := leaderboardMap[currentUserID]

	wordMaskRegex := regexp.MustCompile(`\w`)
	gameWord := word.Word

	if !currentPlayer.GuessedWord {
		gameWord = wordMaskRegex.ReplaceAllString(gameWord, "*")
	}

	gameState.Leaderboard = slices.Collect(maps.Values(leaderboardMap))
	gameState.Messages = gameMessages
	gameState.Word = gameWord
	gameState.Hint = word.Hint
	gameState.TurnID = latestTurn.ID
	gameState.TurnEnded = turnEnded
	gameState.GameID = gameID
	gameState.CurrentUserID = currentUserID

	return gameState, nil

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

func pickGameWord(allWords []model.Word) model.Word {
	wordsLength := len(allWords)
	randWordIndex := mathRand.Intn(wordsLength)
	pickedWord := allWords[randWordIndex]
	return pickedWord
}

func (e *emojixUsecase) InitGame(ctx context.Context, userID string) (model.Game, error) {
	uow, err := e.unitOfWorkFactory.New(ctx)
	if err != nil {
		return model.Game{}, err
	}
	defer uow.Rollback()

	gameRepo := uow.GameRepository()

	game, err := gameRepo.Create(ctx)
	if err != nil {
		return model.Game{}, err
	}

	err = gameRepo.AddPlayer(ctx, game.ID, userID)
	if err != nil {
		return model.Game{}, err
	}

	allWords, err := e.wordRepo.GetAll(ctx)
	if err != nil {
		return model.Game{}, err
	}

	pickedWord := pickGameWord(allWords)

	err = gameRepo.AddTurn(ctx, game.ID, pickedWord.ID)
	if err != nil {
		return model.Game{}, err
	}

	if err = uow.Commit(); err != nil {
		return model.Game{}, err
	}

	return game, nil
}

type GameMsgNotification struct {
	UserID   string
	Nickname string
	Content  string
}

func (gmn *GameMsgNotification) GetType() string {
	return "msg"
}

func (gmn *GameMsgNotification) GetData() string {
	return fmt.Sprintf("%s,%s,%s", gmn.UserID, gmn.Nickname, gmn.Content)
}

func (gmn *GameMsgNotification) ParseData(data string) error {

	items := strings.Split(data, ",")

	if len(items) != 3 {
		return errors.New("invalid msg content")
	}

	gmn.UserID = items[0]
	gmn.Nickname = items[1]
	gmn.Content = items[2]

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

func (gmn *GameCorrectGuessNotification) ParseData(data string) error {
	return nil
}

type GameTurnEndNotification struct {
}

func (gmn *GameTurnEndNotification) GetType() string {
	return "turnended"
}

func (gmn *GameTurnEndNotification) GetData() string {
	return ""
}

func (gmn *GameTurnEndNotification) ParseData(data string) error {
	return nil
}

func (e *emojixUsecase) Guess(ctx context.Context, gameID string, userID string, content string) error {
	currPlayer, err := e.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	turn, err := e.gameRepo.GetLatestTurn(ctx, gameID)
	if err != nil {
		return err
	}
	turnID := turn.ID

	word, err := e.wordRepo.FindByID(ctx, turn.WordID)
	if err != nil {
		return err
	}
	gameWord := word.Word

	uow, err := e.unitOfWorkFactory.New(ctx)
	if err != nil {
		return err
	}
	defer uow.Rollback()

	gameRepo := uow.GameRepository()

	msg, err := gameRepo.SendMessage(ctx, gameID, turnID, userID, content)
	if err != nil {
		return err
	}

	// TODO: make fancier word comparison
	guessedWord := strings.EqualFold(content, gameWord)
	if !guessedWord {
		err = uow.Commit()
		go e.gameNotifier.Pub(gameID, userID, &GameMsgNotification{userID, currPlayer.Nickname, content})
		return err
	}

	// check if the turn is ended
	players, err := gameRepo.GetPlayers(ctx, gameID)
	if err != nil {
		return err
	}

	scores, err := gameRepo.GetScores(ctx, gameID)
	if err != nil {
		return err
	}

	guessedCount := 1
	for _, p := range players {
		for _, s := range scores {
			if s.PlayerID == p.ID && s.TurnID == turnID {
				guessedCount += 1
			}
		}
	}

	pointCoeff := len(players) / guessedCount
	basePoint := 10
	point := basePoint * pointCoeff

	err = gameRepo.AddScore(ctx, gameID, userID, msg.ID, turnID, point)
	if err != nil {
		return err
	}

	err = uow.Commit()
	if err != nil {
		return err
	}

	go e.gameNotifier.Pub(gameID, userID, &GameMsgNotification{userID, currPlayer.Nickname, "***"})
	go e.gameNotifier.Pub(gameID, userID, &GameCorrectGuessNotification{userID, currPlayer.Nickname})

	if guessedCount == len(players) {
		go e.gameNotifier.PubAll(gameID, &GameTurnEndNotification{})
		go func() {
			time.Sleep(5 * time.Second)
			err := e.newGameTurn(context.Background(), gameID)
			if err != nil {
				log.Printf("failed to create new turn err: %v\n", err)
			}
		}()
	}

	return nil

}

func (e *emojixUsecase) newGameTurn(ctx context.Context, gameID string) error {
	allWords, err := e.wordRepo.GetAll(ctx)
	if err != nil {
		return err
	}

	pickedWord := pickGameWord(allWords)
	err = e.gameRepo.AddTurn(ctx, gameID, pickedWord.ID)

	return nil
}

func (e *emojixUsecase) Message(ctx context.Context, gameID string, userID string, content string) error {
	turn, err := e.gameRepo.GetLatestTurn(ctx, gameID)
	if err != nil {
		return err
	}

	currPlayer, err := e.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	_, err = e.gameRepo.SendMessage(ctx, gameID, turn.ID, userID, content)
	if err != nil {
		return err
	}

	go e.gameNotifier.Pub(gameID, userID, &GameMsgNotification{userID, currPlayer.Nickname, content})

	return nil
}

func (e *emojixUsecase) Leaderboard(ctx context.Context, gameID, currentUserID string) ([]model.LeaderboardEntry, error) {
	leaderboardEntries := []model.LeaderboardEntry{}
	players, err := e.gameRepo.GetPlayers(ctx, gameID)
	if err != nil {
		return leaderboardEntries, err
	}

	activePlayers := []model.Player{}
	for _, p := range players {
		if p.State == model.InactivePlayerState {
			continue
		}

		activePlayers = append(activePlayers, p)
	}

	hasPlayer := slices.ContainsFunc(activePlayers, func(p model.Player) bool {
		return p.ID == currentUserID
	})

	if !hasPlayer {
		return leaderboardEntries, errors.New("user not in the game")
	}

	scores, err := e.gameRepo.GetScores(ctx, gameID)
	if err != nil {
		return leaderboardEntries, err
	}

	latestTurn, err := e.gameRepo.GetLatestTurn(ctx, gameID)
	if err != nil {
		return leaderboardEntries, err
	}

	isGuessedWord := func(playerID string) bool {
		for _, score := range scores {
			if score.PlayerID == playerID && score.TurnID == latestTurn.ID {
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
		entry := model.LeaderboardEntry{
			Nickname:    player.Nickname,
			Me:          player.ID == currentUserID,
			GuessedWord: isGuessedWord(player.ID),
			Score:       scoreMap[player.ID],
		}

		leaderboardEntries = append(leaderboardEntries, entry)
	}

	return leaderboardEntries, nil
}

func (e *emojixUsecase) GameWord(ctx context.Context, gameID, currentUserID string) (string, error) {
	latestTurn, err := e.gameRepo.GetLatestTurn(ctx, gameID)
	if err != nil {
		return "", err
	}

	word, err := e.wordRepo.FindByID(ctx, latestTurn.WordID)
	if err != nil {
		return "", err
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
