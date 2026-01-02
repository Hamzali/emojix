package emojix

import (
	"context"
	"emojix/model"
	"emojix/usecase"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type EmojixServer interface {
	Start()
}

type webServer struct {
	templates     template.Template
	emojixUsecase usecase.EmojixUsecase
}

func NewWebServer(emojixUsecase usecase.EmojixUsecase) (EmojixServer, error) {
	templates := *template.Must(template.ParseGlob("templates/*.gohtml"))
	return &webServer{
		templates,
		emojixUsecase,
	}, nil
}

func (e *webServer) Start() {
	http.Handle("GET /static/", http.FileServer(http.Dir("./")))
	http.HandleFunc("POST /game/new", e.NewGame)
	http.HandleFunc("GET /game/join", e.JoinGame)
	http.HandleFunc("GET /game/{id}/join", e.JoinGame)
	http.HandleFunc("GET /game/{id}/loading", e.LoadingGame)
	http.HandleFunc("GET /game/{id}", e.Game)
	http.HandleFunc("POST /game/{id}/message", e.Message)
	http.HandleFunc("POST /game/{id}/guess", e.Guess)
	http.HandleFunc("GET /game/{id}/sse", e.Sse)
	http.HandleFunc("GET /init", e.InitSession)
	http.HandleFunc("GET /", e.Index)
	log.Fatal(http.ListenAndServe(":9000", nil))
}

func (e *webServer) renderTemplate(w http.ResponseWriter, name string, p any) error {
	err := e.templates.ExecuteTemplate(w, name, p)
	if err != nil {
		return err
	}

	return nil
}

func (e *webServer) handleError(w http.ResponseWriter, err error, msg string) {
	log.Printf("%s: %v\n", msg, err)
	w.WriteHeader(http.StatusInternalServerError)
	_ = e.renderTemplate(w, "error.gohtml", nil)
}

type IndexPageData struct {
	Title    string
	Nickname string
}

const userIdCookieKey = "userid"

const nicknameCookieKey = "nickname"

type Session struct {
	UserID   string
	Nickname string
}

func (e *webServer) getSession(w http.ResponseWriter, r *http.Request) (Session, error) {
	redirectToInit := func() {
		toUrl := fmt.Sprintf("/init?from=%s", r.URL.RawPath)
		http.Redirect(w, r, toUrl, http.StatusFound)
	}

	userIdCookie, err := r.Cookie(userIdCookieKey)
	if err != nil {
		redirectToInit()
		return Session{}, err
	}

	nicknameCookie, err := r.Cookie(nicknameCookieKey)
	if err != nil {
		redirectToInit()
		return Session{}, err
	}

	return Session{
		UserID:   userIdCookie.Value,
		Nickname: nicknameCookie.Value,
	}, nil
}

func setCookie(key string, value string) string {
	return fmt.Sprintf("%s=%s; Path=/; Secure; HttpOnly", key, value)
}

func (e *webServer) InitSession(w http.ResponseWriter, r *http.Request) {
	// TODO: check if there is already a user
	user, err := e.emojixUsecase.InitUser(r.Context())

	if err != nil {
		e.handleError(w, err, "failed to init user")
		return
	}

	fromUrl := r.URL.Query().Get("from")

	if fromUrl == "" {
		fromUrl = "/"
	}

	w.Header().Add("Set-Cookie", setCookie(userIdCookieKey, user.ID))
	w.Header().Add("Set-Cookie", setCookie(nicknameCookieKey, user.Nickname))

	http.Redirect(w, r, fromUrl, http.StatusFound)
}

func (e *webServer) Index(w http.ResponseWriter, r *http.Request) {
	// TODO: check why the fuck this page is called 3 times!
	session, err := e.getSession(w, r)
	if err != nil {
		log.Println("no session redirecting to /init")
		return
	}

	err = e.renderTemplate(w, "index.gohtml", IndexPageData{Title: "Emojix!", Nickname: session.Nickname})
	if err != nil {
		e.handleError(w, err, "failed to render template")
		return
	}
}

func (e *webServer) JoinGame(w http.ResponseWriter, r *http.Request) {
	session, err := e.getSession(w, r)
	if err != nil {
		return
	}
	gameID := r.PathValue("id")
	if gameID == "" {
		gameID = r.URL.Query().Get("game-id")
	}

	ctx := r.Context()

	err = e.emojixUsecase.JoinGame(ctx, gameID, session.UserID)
	if err != nil {
		e.handleError(w, err, "failed to join")
		return
	}

	gameUrl := fmt.Sprintf("/game/%s", gameID)
	http.Redirect(w, r, gameUrl, http.StatusFound)
}

func (e *webServer) NewGame(w http.ResponseWriter, r *http.Request) {
	session, err := e.getSession(w, r)
	if err != nil {
		return
	}
	ctx := r.Context()

	game, err := e.emojixUsecase.InitGame(ctx, session.UserID)
	if err != nil {
		e.handleError(w, err, "failed to create game")
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/game/%s", game.ID), http.StatusSeeOther)
}

type GamePageData struct {
	GameID      string
	Leaderboard []model.LeaderboardEntry
	Messages    []model.GameStateMessage

	MaskedWord []string
	EmojiHint  string
}

func (e *webServer) Game(w http.ResponseWriter, r *http.Request) {
	session, err := e.getSession(w, r)
	if err != nil {
		return
	}
	gameID := r.PathValue("id")

	ctx := r.Context()

	gameState, err := e.emojixUsecase.GameState(ctx, gameID, session.UserID)
	if gameState.TurnEnded {
		log.Println("All guessed waiting for new turn!")
		http.Redirect(w, r, fmt.Sprintf("/game/%s/loading", gameID), http.StatusSeeOther)

		return
	}

	if err != nil {
		e.handleError(w, err, "failed to load game")
		return
	}

	pageData := GamePageData{
		GameID:      gameState.GameID,
		Leaderboard: gameState.Leaderboard,
		Messages:    gameState.Messages,
		MaskedWord:  strings.Split(gameState.Word, ""),
		EmojiHint:   gameState.Hint,
	}
	err = e.renderTemplate(w, "game.gohtml", &pageData)
	if err != nil {
		e.handleError(w, err, "failed to render page")
		return
	}
}

type GameLoadingPageData struct {
	GameID string
}

func (e *webServer) LoadingGame(w http.ResponseWriter, r *http.Request) {
	// TODO: check user if it belogs to the game and game state to avoid serving this page to unwanted users
	// using the session id and game id information.
	gameID := r.PathValue("id")

	pageData := GameLoadingPageData{GameID: gameID}
	err := e.renderTemplate(w, "loading-game.gohtml", &pageData)
	if err != nil {
		e.handleError(w, err, "failed to render page")
		return
	}

}

func (e *webServer) Message(w http.ResponseWriter, r *http.Request) {
	session, err := e.getSession(w, r)
	if err != nil {
		return
	}

	gameID := r.PathValue("id")
	ctx := r.Context()

	// get message content from form body content field
	err = r.ParseForm()
	if err != nil {
		e.handleError(w, err, "failed to parse form")
		return
	}

	content := r.PostForm.Get("content")

	err = e.emojixUsecase.Message(ctx, gameID, session.UserID, content)
	if err != nil {
		e.handleError(w, err, "failed to send message")
		return
	}

	msg := model.GameStateMessage{Me: true, Content: content, Nickname: session.Nickname}
	err = e.renderTemplate(w, "game-msg.gohtml", &msg)
	if err != nil {
		e.handleError(w, err, "failed to render")
		return
	}
}

func (e *webServer) Guess(w http.ResponseWriter, r *http.Request) {
	session, err := e.getSession(w, r)
	if err != nil {
		return
	}
	gameID := r.PathValue("id")
	ctx := r.Context()

	// get message content from form body content field
	err = r.ParseForm()
	if err != nil {
		e.handleError(w, err, "failed to parse form")
		return
	}

	content := r.PostForm.Get("content")

	// process message
	err = e.emojixUsecase.Guess(ctx, gameID, session.UserID, content)
	if err != nil {
		e.handleError(w, err, "failed to process guess")
		return
	}

	msg := model.GameStateMessage{Me: true, Content: content, Nickname: session.Nickname}
	err = e.renderTemplate(w, "game-msg.gohtml", &msg)
	if err != nil {
		e.handleError(w, err, "failed to render")
		return
	}
}

func (e *webServer) Sse(w http.ResponseWriter, r *http.Request) {
	userIdCookie, err := r.Cookie(userIdCookieKey)
	if err != nil {
		e.handleError(w, err, "no user id")
		return
	}
	userID := userIdCookie.Value

	gameID := r.PathValue("id")

	// TODO: validate session id and game id

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	log.Println("sse connected", userID, gameID)

	rc := http.NewResponseController(w)
	if rc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("failed to intialize the response controller")
		return
	}

	sendSseMsg := func(msgType string, content string) error {
		safeContent := strings.ReplaceAll(content, "\n", "")
		sseContent := fmt.Sprintf("event: %s\ndata: %s\n\n", msgType, safeContent)
		io.WriteString(w, sseContent)
		err := rc.Flush()
		return err

	}

	err = sendSseMsg("init", "")
	if err != nil {
		log.Printf("failed to flush with err: %v", err)
		return
	}

	err = e.emojixUsecase.GameUpdates(r.Context(), gameID, userID, func(notifType string, data string) error {
		if notifType == "msg" {
			msgNotif := usecase.GameMsgNotification{}

			err := msgNotif.ParseData(data)
			if err != nil {
				return err
			}

			gameMsg := model.GameStateMessage{
				Me:       userID == msgNotif.UserID,
				Nickname: msgNotif.Nickname,
				Content:  msgNotif.Content,
			}
			var sseContent strings.Builder
			err = e.templates.ExecuteTemplate(&sseContent, "game-msg.gohtml", &gameMsg)
			if err != nil {
				return err
			}

			sseMsg := sseContent.String()

			err = sendSseMsg(notifType, sseMsg)

			return err
		}

		err := sendSseMsg(notifType, data)

		return err
	})

	if err != nil {
		log.Printf("failed to send message: %v", err)
	}

	go func() {
		// TODO: experiment and find a better wait amount
		time.Sleep(time.Second * 5)

		ctx := context.Background()
		err := e.emojixUsecase.KickInactiveUser(ctx, gameID, userID)
		if err != nil {
			log.Println("failed to kick inactive user", err)
		}
	}()
}
