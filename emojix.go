package emojix

import (
	"crypto/rand"
	"emojix/model"
	"emojix/repository"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	mathRand "math/rand"
	"net/http"
	"regexp"
	"strings"
)

type emojix struct {
	templates template.Template
	userRepo  repository.UserRepository
	gameRepo  repository.GameRepository
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
		templates: *template.Must(template.ParseGlob("templates/*.gohtml")),
		userRepo:  repository.NewUserRepository(db),
		gameRepo:  repository.NewGameRepository(db),
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

func (e *emojix) renderTemplate(w http.ResponseWriter, name string, p interface{}) error {
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

	game, err := e.gameRepo.Create(ctx, "Harry Potter and the Philosopher’s Stone", "🪄💎🏰")
	if err != nil {
		e.handleError(w, err, "failed to create game")
		return
	}

	err = e.gameRepo.AddPlayer(ctx, game.ID, sessionID)
	if err != nil {
		e.handleError(w, err, "failed to add player to game")
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

	game, err := e.gameRepo.FindByID(r.Context(), gameID)
	if err != nil {
		e.handleError(w, err, "failed to find game")
		return
	}

	players, err := e.gameRepo.GetPlayers(r.Context(), gameID)
	if err != nil {
		e.handleError(w, err, "failed to get players")
		return
	}

	messages, err := e.gameRepo.GetMessages(r.Context(), gameID)
	if err != nil {
		e.handleError(w, err, "failed to get messages")
		return
	}

	leaderboard := []LeaderboardEntry{}
	currentPlayerIndex := 0

	for index, player := range players {
		entry := LeaderboardEntry{
			Nickname:    player.Nickname,
			Me:          player.ID == sessionID,
			GuessedWord: false,
			Score:       0,
		}

		if entry.Me {
			currentPlayerIndex = index
		}

		leaderboard = append(leaderboard, entry)
	}

	gameMessages := []GameMessage{}

	for _, message := range messages {

		player := model.Player{}
		playerIndex := 0
		for index, p := range players {
			if p.ID == message.PlayerID {
				player = p
				playerIndex = index
				break
			}
		}

		gm := GameMessage{
			Me:       message.PlayerID == sessionID,
			Content:  message.Content,
			Nickname: player.Nickname,
		}

		guessedWord := message.Content == game.Word

		if guessedWord {
			leaderboard[playerIndex].GuessedWord = true
		}

		if guessedWord && message.PlayerID != sessionID {
			gm.Content = "***"
		}

		gameMessages = append(gameMessages, gm)
	}

	currentPlayer := leaderboard[currentPlayerIndex]

	wordMaskRegex := regexp.MustCompile(`\w`)
	gameWord := game.Word

	if !currentPlayer.GuessedWord {
		gameWord = wordMaskRegex.ReplaceAllString(game.Word, "*")
	}

	pageData := GamePageData{
		Game:        game,
		Leaderboard: leaderboard,
		Messages:    gameMessages,
		MaskedWord:  strings.Split(gameWord, ""),
	}
	err = e.renderTemplate(w, "game.gohtml", &pageData)
	if err != nil {
		log.Printf("failed with err %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (e *emojix) Message(w http.ResponseWriter, r *http.Request) {
	sessionID := e.getSessionID(w, r)
	gameID := r.PathValue("id")

	// get message content from form body content field
	err := r.ParseForm()
	if err != nil {
		log.Printf("failed with err %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	content := r.PostForm.Get("content")

	// save message
	err = e.gameRepo.SendMessage(r.Context(), gameID, sessionID, content)
	if err != nil {
		log.Printf("failed with err %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// refresh the page
	http.Redirect(w, r, fmt.Sprintf("/game/%s", gameID), http.StatusSeeOther)

}
