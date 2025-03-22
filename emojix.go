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
		templates: *template.Must(template.ParseGlob("templates/*.html")),
		userRepo:  repository.NewUserRepository(db),
		gameRepo:  repository.NewGameRepository(db),
	}, nil
}

func (e *emojix) StartServer() {
	http.HandleFunc("GET /", e.Index)
	http.HandleFunc("POST /game/new", e.NewGame)
	http.HandleFunc("GET /game/{id}", e.Game)
	http.HandleFunc("GET /game/{id}/join", e.JoinGame)
	http.HandleFunc("POST /game/{id}/message", e.Message)
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
	_ = e.renderTemplate(w, "error.html", nil)
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
		err = nil
		nickname := GenerateNickname()
		if err != nil {
			log.Printf("failed to generate session id err %v\n", err)
			return ""
		}
		w.Header().Add("Set-Cookie", nicknameCookieKey+"="+nickname)
		return nickname
	}

	return nicknameCookie.Value
}

func (e *emojix) Index(w http.ResponseWriter, r *http.Request) {
	log.SetPrefix("GET /")
	sessionID := e.getSessionID(w, r)
	nickname := e.getNickname(w, r)

	err := e.userRepo.CreateOrUpdate(r.Context(), sessionID, repository.UserCreateOrUpdateParams{Nickname: nickname})
	if err != nil {
		e.handleError(w, err, "failed to create or update user")
		return
	}

	err = e.renderTemplate(w, "index.html", IndexPageData{Title: "Hey, mom!", Nickname: nickname})
	if err != nil {
		e.handleError(w, err, "failed to render template")
		return
	}
}

func (e *emojix) JoinGame(w http.ResponseWriter, r *http.Request) {
	gameID := r.PathValue("id")
	logPrefix := fmt.Sprintf("GET /game/%s/join ", gameID)
	log.SetPrefix(logPrefix)

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

	game, err := e.gameRepo.Create(ctx)
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

type GamePageData struct {
	Game            model.Game
	CurrentPlayerID string
	Players         []model.Player
	Messages        []model.Message
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

	pageData := GamePageData{
		Game:            game,
		Players:         players,
		CurrentPlayerID: sessionID,
		Messages:        messages,
	}
	err = e.renderTemplate(w, "game.html", &pageData)
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

/*
 */
