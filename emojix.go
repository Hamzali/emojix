package emojix

import (
	"context"
	"crypto/rand"
	"database/sql"
	"emojix/model"
	"emojix/repository"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"maps"
	mathRand "math/rand"
	"net/http"
	"regexp"
	"slices"
	"strings"
)

type emojix struct {
	db        *sql.DB
	templates template.Template
	userRepo  repository.UserRepository
	gameRepo  repository.GameRepository
	wordRepo  repository.WordRepository
}

type Emojix interface {
	StartServer()
}

func NewEmojix() (Emojix, error) {
	db, err := repository.InitDB("emojix.db")
	if err != nil {
		return nil, err
	}
	return &emojix{
		db:        db,
		templates: *template.Must(template.ParseGlob("templates/*.gohtml")),
		userRepo:  repository.NewUserRepository(db),
		gameRepo:  repository.NewGameRepository(db),
		wordRepo:  repository.NewWordRepository(db),
	}, nil
}

func (e *emojix) StartServer() {
	http.HandleFunc("POST /game/new", e.NewGame)
	http.HandleFunc("GET /game/join", e.JoinGame)
	http.HandleFunc("GET /game/{id}/join", e.JoinGame)
	http.HandleFunc("GET /game/{id}", e.Game)
	http.HandleFunc("POST /game/{id}/message", e.Message)
	http.HandleFunc("GET /", e.Index)
	log.Fatal(http.ListenAndServe(":9000", nil))
}

func (e *emojix) CreateGame(ctx context.Context, userID string) (model.Game, error) {
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("failed to begin transaction err %v\n", err)
		return model.Game{}, err
	}
	defer tx.Rollback()

	gameRepo := repository.NewGameRepository(tx)
	wordsRepo := repository.NewWordRepository(tx)

	game, err := gameRepo.Create(ctx)
	if err != nil {
		return model.Game{}, err
	}

	err = gameRepo.AddPlayer(ctx, game.ID, userID)
	if err != nil {
		return model.Game{}, err
	}

	allWords, err := wordsRepo.GetAll(ctx)
	if err != nil {
		return model.Game{}, err
	}

	wordsLength := len(allWords)
	randWordIndex := mathRand.Intn(wordsLength)
	pickedWord := allWords[randWordIndex]

	err = gameRepo.AddTurn(ctx, game.ID, pickedWord.ID)

	if err = tx.Commit(); err != nil {
		return model.Game{}, err
	}

	return game, nil
}

func (e *emojix) renderTemplate(w http.ResponseWriter, name string, p any) error {
	err := e.templates.ExecuteTemplate(w, name, p)
	if err != nil {
		return err
	}

	return nil
}

func (e *emojix) handleError(w http.ResponseWriter, err error, msg string) {
	log.Printf("%s: %v\n", msg, err)
	w.WriteHeader(http.StatusInternalServerError)
	_ = e.renderTemplate(w, "error.gohtml", nil)
}

type IndexPageData struct {
	Title    string
	Nickname string
}

// GenerateRandomID generates a secure random session ID
func GenerateRandomID() (string, error) {
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

func GenerateNickname() string {
	animal := pickRandItem(animals)
	adj := pickRandItem(adjectives)
	return fmt.Sprintf("%s%s", capitalize(adj), capitalize(animal))
}

const sessionCookieKey = "session-id"

func (e *emojix) getSessionID(w http.ResponseWriter, r *http.Request) string {
	sessionID, err := r.Cookie(sessionCookieKey)

	if err != nil {
		err = nil
		newSessionID, err := GenerateRandomID()
		if err != nil {
			log.Printf("failed to generate session id err %v\n", err)
			return ""
		}
		w.Header().Add("Set-Cookie", sessionCookieKey+"="+newSessionID)
		return newSessionID
	}

	return sessionID.Value
}

const nicknameCookieKey = "nickname"

func (e *emojix) getNickname(w http.ResponseWriter, r *http.Request) string {
	nicknameCookie, err := r.Cookie(nicknameCookieKey)

	if err != nil {
		nickname := GenerateNickname()
		w.Header().Add("Set-Cookie", nicknameCookieKey+"="+nickname)
		return nickname
	}

	return nicknameCookie.Value
}

func (e *emojix) Index(w http.ResponseWriter, r *http.Request) {
	log.SetPrefix("GET / ")
	sessionID := e.getSessionID(w, r)
	nickname := e.getNickname(w, r)

	err := e.userRepo.CreateOrUpdate(r.Context(), sessionID, repository.UserCreateOrUpdateParams{Nickname: nickname})
	if err != nil {
		e.handleError(w, err, "failed to create or update user")
		return
	}

	err = e.renderTemplate(w, "index.gohtml", IndexPageData{Title: "Hey, mom!", Nickname: nickname})
	if err != nil {
		e.handleError(w, err, "failed to render template")
		return
	}
}

func (e *emojix) JoinGame(w http.ResponseWriter, r *http.Request) {
	gameID := r.PathValue("id")
	if gameID == "" {
		gameID = r.URL.Query().Get("game-id")
	}

	logPrefix := fmt.Sprintf("GET /game/%s/join ", gameID)
	log.SetPrefix(logPrefix)

	log.Println("game ID", gameID)
	sessionID := e.getSessionID(w, r)
	err := e.gameRepo.AddPlayer(r.Context(), gameID, sessionID)
	if err != nil {
		e.handleError(w, err, "failed to add player to game")
		return
	}

	gameUrl := fmt.Sprintf("/game/%s", gameID)
	http.Redirect(w, r, gameUrl, http.StatusMovedPermanently)
}

func (e *emojix) NewGame(w http.ResponseWriter, r *http.Request) {
	log.SetPrefix("GET /game/new")
	sessionID := e.getSessionID(w, r)
	ctx := r.Context()

	game, err := e.CreateGame(ctx, sessionID)
	if err != nil {
		e.handleError(w, err, "failed to create game")
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/game/%s", game.ID), http.StatusSeeOther)
}

type LeaderboardEntry struct {
	Nickname    string
	Me          bool
	GuessedWord bool
	Score       int
}

type GameMessage struct {
	Me       bool
	Content  string
	Nickname string
}

type GamePageData struct {
	Game        model.Game
	Leaderboard []LeaderboardEntry
	Messages    []GameMessage

	MaskedWord []string
	EmojiHint  string
}

func (e *emojix) Game(w http.ResponseWriter, r *http.Request) {
	log.SetPrefix("GET /game ")
	sessionID := e.getSessionID(w, r)
	gameID := r.PathValue("id")

	ctx := r.Context()
	game, err := e.gameRepo.FindByID(ctx, gameID)
	if err != nil {
		e.handleError(w, err, "failed to find game")
		return
	}

	players, err := e.gameRepo.GetPlayers(ctx, gameID)
	if err != nil {
		e.handleError(w, err, "failed to get players")
		return
	}

	messages, err := e.gameRepo.GetMessages(ctx, gameID)
	if err != nil {
		e.handleError(w, err, "failed to get messages")
		return
	}

	scores, err := e.gameRepo.GetScores(ctx, gameID)
	if err != nil {
		e.handleError(w, err, "failed to get scores")
		return
	}

	latestTurn, err := e.gameRepo.GetLatestTurn(ctx, gameID)
	if err != nil {
		e.handleError(w, err, "failed to get turn")
		return
	}

	word, err := e.wordRepo.FindByID(ctx, latestTurn.WordID)
	if err != nil {
		e.handleError(w, err, "failed to get turn")
		return
	}

	leaderboardMap := map[string]LeaderboardEntry{}

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

	for _, player := range players {
		entry := LeaderboardEntry{
			Nickname:    player.Nickname,
			Me:          player.ID == sessionID,
			GuessedWord: isGuessedWord(player.ID),
			Score:       scoreMap[player.ID],
		}
		leaderboardMap[player.ID] = entry
	}

	gameMessages := []GameMessage{}

	for _, msg := range messages {
		le := leaderboardMap[msg.PlayerID]

		gm := GameMessage{
			Me:       msg.PlayerID == sessionID,
			Content:  msg.Content,
			Nickname: le.Nickname,
		}

		if strings.EqualFold(word.Word, gm.Content) && !gm.Me && le.GuessedWord {
			gm.Content = "***"
		}

		gameMessages = append(gameMessages, gm)
	}

	currentPlayer := leaderboardMap[sessionID]

	wordMaskRegex := regexp.MustCompile(`\w`)
	gameWord := word.Word

	if !currentPlayer.GuessedWord {
		gameWord = wordMaskRegex.ReplaceAllString(gameWord, "*")
	}

	leaderboard := slices.Collect(maps.Values(leaderboardMap))

	pageData := GamePageData{
		Game:        game,
		Leaderboard: leaderboard,
		Messages:    gameMessages,
		MaskedWord:  strings.Split(gameWord, ""),
		EmojiHint:   word.Hint,
	}
	err = e.renderTemplate(w, "game.gohtml", &pageData)
	if err != nil {
		log.Printf("failed with err %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (e *emojix) SaveMessage(
	ctx context.Context,
	gameID string,
	turnID string,
	userID string,
	gameWord string,
	content string,
) error {
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	gameRepo := repository.NewGameRepository(tx)

	msg, err := gameRepo.SendMessage(ctx, gameID, turnID, userID, content)
	if err != nil {
		return err
	}

	// TODO: make fancier word comparison
	guessedWord := strings.EqualFold(content, gameWord)
	if !guessedWord {
		err = tx.Commit()
		return err
	}

	// TODO: make fancier score calculation
	err = gameRepo.AddScore(ctx, gameID, userID, msg.ID, turnID, 10)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (e *emojix) Message(w http.ResponseWriter, r *http.Request) {
	sessionID := e.getSessionID(w, r)
	gameID := r.PathValue("id")
	ctx := r.Context()

	// get message content from form body content field
	err := r.ParseForm()
	if err != nil {
		e.handleError(w, err, "failed to parse form")
		return
	}

	content := r.PostForm.Get("content")

	turn, err := e.gameRepo.GetLatestTurn(ctx, gameID)
	if err != nil {
		e.handleError(w, err, "failed to get turn")
		return
	}

	word, err := e.wordRepo.FindByID(ctx, turn.WordID)
	if err != nil {
		e.handleError(w, err, "failed to get turn")
		return
	}

	// save message
	err = e.SaveMessage(ctx, gameID, turn.ID, sessionID, word.Word, content)
	if err != nil {
		e.handleError(w, err, "failed to save message")
		return
	}

	// refresh the page
	http.Redirect(w, r, fmt.Sprintf("/game/%s", gameID), http.StatusSeeOther)
}
