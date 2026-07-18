package emojix

import (
	"context"
	"emojix/model"
	"emojix/usecase"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type EmojixServer interface {
	Start()
}

type webServer struct {
	view          View
	emojixUsecase usecase.EmojixUsecase
	// kickDelay is how long the Sse handler waits before kicking an
	// inactive user. It is a field (rather than a package var) so server
	// tests can inject a near-zero duration without touching global state.
	kickDelay time.Duration
}

func NewWebServer(emojixUsecase usecase.EmojixUsecase, view View) (EmojixServer, error) {
	return &webServer{
		view:          view,
		emojixUsecase: emojixUsecase,
		kickDelay:     defaultKickDelay,
	}, nil
}

// mux returns the router with every route registered. It is shared by Start
// and by routing tests so the test exercises the real route table.
func (e *webServer) mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.FileServer(http.Dir("./")))
	mux.HandleFunc("POST /game/new", e.NewGame)
	mux.HandleFunc("GET /game/join", e.JoinGame)
	mux.HandleFunc("GET /game/{id}/join", e.JoinGame)
	mux.HandleFunc("GET /game/{id}/loading", e.LoadingGame)
	mux.HandleFunc("GET /game/{id}", e.Game)
	mux.HandleFunc("GET /game/{id}/leaderboard", e.Leaderboard)
	mux.HandleFunc("GET /game/{id}/word", e.GameWord)
	mux.HandleFunc("POST /game/{id}/message", e.Message)
	mux.HandleFunc("POST /game/{id}/guess", e.Guess)
	mux.HandleFunc("GET /game/{id}/sse", e.Sse)
	mux.HandleFunc("GET /init", e.InitSession)
	mux.HandleFunc("GET /", e.Index)
	return mux
}

func (e *webServer) Start() {
	log.Fatal(http.ListenAndServe("0.0.0.0:9000", e.mux()))
}

func (e *webServer) handleError(w http.ResponseWriter, err error, msg string) {
	log.Printf("%s: %v\n", msg, err)
	w.WriteHeader(http.StatusInternalServerError)
	_ = e.view.renderErrorPage(w)
}

const userIdCookieKey = "userid"

const nicknameCookieKey = "nickname"

// defaultKickDelay is how long the Sse handler waits before kicking an
// inactive user in production.
const defaultKickDelay = time.Second * 30

type Session struct {
	UserID   string
	Nickname string
}

func (e *webServer) getSession(w http.ResponseWriter, r *http.Request) (Session, error) {
	redirectToInit := func() {
		toUrl := fmt.Sprintf("/init?from=%s", r.URL.Path)
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
	cookieOptions := []string{"Path=/", "HttpOnly"}

	// TODO: setup a general config and consume here
	if os.Getenv("ENV") == "prod" {
		cookieOptions = append(cookieOptions, "Secure")
	}

	return fmt.Sprintf("%s=%s; %s", key, value, strings.Join(cookieOptions, "; "))
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
	session, err := e.getSession(w, r)
	if err != nil {
		log.Println("no session redirecting to /init")
		return
	}

	err = e.view.renderIndexPage(w, IndexPageViewParam{Title: "Emojix!", Nickname: session.Nickname})
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

func (e *webServer) Game(w http.ResponseWriter, r *http.Request) {
	session, err := e.getSession(w, r)
	if err != nil {
		return
	}
	gameID := r.PathValue("id")

	ctx := r.Context()

	gameState, err := e.emojixUsecase.GameState(ctx, gameID, session.UserID)
	if err != nil {
		e.handleError(w, err, "failed to load game")
		return
	}

	if gameState.TurnEnded {
		log.Println("All guessed waiting for new turn!")
		http.Redirect(w, r, fmt.Sprintf("/game/%s/loading", gameID), http.StatusSeeOther)

		return
	}

	pageData := GamePageViewParam{
		GameID:        gameState.GameID,
		Leaderboard:   gameState.Leaderboard,
		Messages:      gameState.Messages,
		MaskedWord:    strings.Split(gameState.Word, ""),
		EmojiHint:     gameState.Hint,
		TurnStartedAt: gameState.TurnStartedAt,
	}
	err = e.view.renderGamePage(w, pageData)
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

	err := e.view.renderGameLoadingPage(w, GameLoadingPageViewParam{GameID: gameID})
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
	err = e.view.renderGameMsg(w, msg)
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

	w.Header().Set("Hx-Trigger", "guessed")
	msg := model.GameStateMessage{Me: true, Content: content, Nickname: session.Nickname}
	err = e.view.renderGameMsg(w, msg)
	if err != nil {
		e.handleError(w, err, "failed to render")
		return
	}
}

func (e *webServer) Leaderboard(w http.ResponseWriter, r *http.Request) {
	session, err := e.getSession(w, r)
	if err != nil {
		return
	}
	gameID := r.PathValue("id")
	ctx := r.Context()

	leaderboardEntries, err := e.emojixUsecase.Leaderboard(ctx, gameID, session.UserID)
	if err != nil {
		e.handleError(w, err, "failed to fetch leaderboard")
		return
	}

	vieaParam := GameLeaderboardViewParam{leaderboardEntries}
	err = e.view.renderGameLeaderboard(w, vieaParam)
	if err != nil {
		e.handleError(w, err, "failed to render leaderboard")
		return
	}
}

func (e *webServer) GameWord(w http.ResponseWriter, r *http.Request) {
	session, err := e.getSession(w, r)
	if err != nil {
		return
	}
	gameID := r.PathValue("id")
	ctx := r.Context()

	gameWord, err := e.emojixUsecase.GameWord(ctx, gameID, session.UserID)
	if err != nil {
		e.handleError(w, err, "failed to fetch word")
		return
	}

	pageParam := GameWordViewParam{strings.Split(gameWord, "")}
	err = e.view.renderGameWord(w, pageParam)
	if err != nil {
		e.handleError(w, err, "failed to render word")
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
			err = e.view.renderGameMsg(&sseContent, gameMsg)
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
		// NOTE: should be higher than the turn start wait time
		time.Sleep(e.kickDelay)

		ctx := context.Background()
		err := e.emojixUsecase.KickInactiveUser(ctx, gameID, userID)
		if err != nil {
			log.Println("failed to kick inactive user", err)
		}
	}()
}
