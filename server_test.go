package emojix

import (
	"context"
	"emojix/model"
	"emojix/usecase"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- test helpers -------------------------------------------------------

func newServer(uc *MockEmojixUsecase, view *MockView) *webServer {
	return &webServer{view: view, emojixUsecase: uc}
}

// withSession returns r with both session cookies set.
func withSession(r *http.Request, userID, nickname string) *http.Request {
	r.AddCookie(&http.Cookie{Name: userIdCookieKey, Value: userID})
	r.AddCookie(&http.Cookie{Name: nicknameCookieKey, Value: nickname})
	return r
}

func newReq(method, target string, body io.Reader) *http.Request {
	return httptest.NewRequest(method, target, body)
}

// setGameID attaches a path value to the request (so r.PathValue("id") works
// for direct-handler tests instead of going through a pattern-registered mux).
func setGameID(r *http.Request, id string) *http.Request {
	r.SetPathValue("id", id)
	return r
}

// errSentinel is a stable error for failure cases.
var errSentinel = errors.New("boom")

// --- Index --------------------------------------------------------------

func TestIndex_HasSession_Renders(t *testing.T) {
	uc := newMockUsecase()
	view := &MockView{}
	srv := newServer(uc, view)

	r := withSession(newReq("GET", "/", nil), "u1", "sillyCat")
	w := httptest.NewRecorder()

	srv.Index(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if view.renderIndexPageCalls != 1 {
		t.Fatalf("renderIndexPageCalls = %d, want 1", view.renderIndexPageCalls)
	}
	if got := view.renderIndexPageLastParam; got.Title != "Emojix!" || got.Nickname != "sillyCat" {
		t.Errorf("renderIndexPage params = %+v", got)
	}
}

func TestIndex_NoSession_RedirectsToInit(t *testing.T) {
	uc := newMockUsecase()
	view := &MockView{}
	srv := newServer(uc, view)

	r := newReq("GET", "/", nil)
	w := httptest.NewRecorder()

	srv.Index(w, r)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/init?from=/" {
		t.Errorf("Location = %q, want /init?from=/", loc)
	}
	if view.renderIndexPageCalls != 0 {
		t.Errorf("renderIndexPageCalls = %d, want 0", view.renderIndexPageCalls)
	}
}

func TestIndex_RenderError_500(t *testing.T) {
	uc := newMockUsecase()
	view := &MockView{}
	view.renderIndexPageFn = func(io.Writer, IndexPageViewParam) error { return errSentinel }
	srv := newServer(uc, view)

	r := withSession(newReq("GET", "/", nil), "u1", "nick")
	w := httptest.NewRecorder()

	srv.Index(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if view.renderErrorPageCalls != 1 {
		t.Errorf("renderErrorPageCalls = %d, want 1", view.renderErrorPageCalls)
	}
}

// --- InitSession -------------------------------------------------------

func TestInitSession_SetsCookiesAndRedirects(t *testing.T) {
	uc := newMockUsecase()
	uc.InitUserFn = func(ctx context.Context) (model.User, error) {
		return model.User{ID: "u1", Nickname: "sillyCat"}, nil
	}
	view := &MockView{}
	srv := newServer(uc, view)

	r := newReq("GET", "/init?from=/game/x", nil)
	w := httptest.NewRecorder()

	srv.InitSession(w, r)

	if uc.InitUserCalls != 1 {
		t.Fatalf("InitUserCalls = %d, want 1", uc.InitUserCalls)
	}
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/game/x" {
		t.Errorf("Location = %q, want /game/x", loc)
	}
	cookies := w.Header()["Set-Cookie"]
	if len(cookies) != 2 {
		t.Fatalf("Set-Cookie entries = %d, want 2: %v", len(cookies), cookies)
	}
	wantUserID := "userid=u1; Path=/; HttpOnly"
	wantNick := "nickname=sillyCat; Path=/; HttpOnly"
	if cookies[0] != wantUserID {
		t.Errorf("Set-Cookie[0] = %q, want %q", cookies[0], wantUserID)
	}
	if cookies[1] != wantNick {
		t.Errorf("Set-Cookie[1] = %q, want %q", cookies[1], wantNick)
	}
}

func TestInitSession_DefaultFromRedirect(t *testing.T) {
	uc := newMockUsecase()
	uc.InitUserFn = func(ctx context.Context) (model.User, error) {
		return model.User{ID: "u1", Nickname: "nick"}, nil
	}
	srv := newServer(uc, &MockView{})

	r := newReq("GET", "/init", nil)
	w := httptest.NewRecorder()

	srv.InitSession(w, r)

	if loc := w.Header().Get("Location"); loc != "/" {
		t.Errorf("Location = %q, want /", loc)
	}
}

func TestInitSession_ProdSetsSecureCookie(t *testing.T) {
	t.Setenv("ENV", "prod")
	uc := newMockUsecase()
	uc.InitUserFn = func(ctx context.Context) (model.User, error) {
		return model.User{ID: "u1", Nickname: "nick"}, nil
	}
	srv := newServer(uc, &MockView{})

	r := newReq("GET", "/init", nil)
	w := httptest.NewRecorder()

	srv.InitSession(w, r)

	cookies := w.Header()["Set-Cookie"]
	if len(cookies) != 2 {
		t.Fatalf("Set-Cookie entries = %d, want 2", len(cookies))
	}
	for _, c := range cookies {
		if !strings.HasSuffix(c, "; Secure") {
			t.Errorf("cookie %q missing Secure suffix", c)
		}
	}
}

func TestInitSession_InitUserError_500(t *testing.T) {
	uc := newMockUsecase()
	uc.InitUserFn = func(ctx context.Context) (model.User, error) {
		return model.User{}, errSentinel
	}
	view := &MockView{}
	srv := newServer(uc, view)

	r := newReq("GET", "/init", nil)
	w := httptest.NewRecorder()

	srv.InitSession(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if view.renderErrorPageCalls != 1 {
		t.Errorf("renderErrorPageCalls = %d, want 1", view.renderErrorPageCalls)
	}
}

// --- JoinGame ----------------------------------------------------------

func TestJoinGame_PathID_Redirects(t *testing.T) {
	uc := newMockUsecase()
	view := &MockView{}
	srv := newServer(uc, view)

	r := setGameID(withSession(newReq("GET", "/game/g1/join", nil), "u1", "nick"), "g1")
	w := httptest.NewRecorder()

	srv.JoinGame(w, r)

	if uc.JoinGameCalls != 1 || uc.JoinGameLastGameID != "g1" {
		t.Fatalf("JoinGame call = %+v", uc.JoinGameCalls)
	}
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/game/g1" {
		t.Errorf("Location = %q, want /game/g1", loc)
	}
}

func TestJoinGame_QueryID_Redirects(t *testing.T) {
	uc := newMockUsecase()
	srv := newServer(uc, &MockView{})

	r := withSession(newReq("GET", "/game/join?game-id=g2", nil), "u1", "nick")
	w := httptest.NewRecorder()

	srv.JoinGame(w, r)

	if uc.JoinGameLastGameID != "g2" {
		t.Fatalf("JoinGameLastGameID = %q, want g2", uc.JoinGameLastGameID)
	}
	if loc := w.Header().Get("Location"); loc != "/game/g2" {
		t.Errorf("Location = %q, want /game/g2", loc)
	}
}

func TestJoinGame_AlreadyJoined_500(t *testing.T) {
	uc := newMockUsecase()
	uc.JoinGameFn = func(ctx context.Context, gameID, userID string) error {
		return usecase.ErrJoinGameUserAlreadyJoined
	}
	view := &MockView{}
	srv := newServer(uc, view)

	r := setGameID(withSession(newReq("GET", "/game/g1/join", nil), "u1", "nick"), "g1")
	w := httptest.NewRecorder()

	srv.JoinGame(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (current behavior pins the UX gap as backlog)", w.Code)
	}
	if view.renderErrorPageCalls != 1 {
		t.Errorf("renderErrorPageCalls = %d, want 1", view.renderErrorPageCalls)
	}
}

func TestJoinGame_RoomFull_500(t *testing.T) {
	uc := newMockUsecase()
	uc.JoinGameFn = func(ctx context.Context, gameID, userID string) error {
		return usecase.ErrJoinGameRoomFull
	}
	view := &MockView{}
	srv := newServer(uc, view)

	r := setGameID(withSession(newReq("GET", "/game/g1/join", nil), "u1", "nick"), "g1")
	w := httptest.NewRecorder()

	srv.JoinGame(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestJoinGame_NoSession_Redirects(t *testing.T) {
	uc := newMockUsecase()
	srv := newServer(uc, &MockView{})

	r := setGameID(newReq("GET", "/game/g1/join", nil), "g1")
	w := httptest.NewRecorder()

	srv.JoinGame(w, r)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/init?from=/game/g1/join" {
		t.Errorf("Location = %q, want /init?from=/game/g1/join", loc)
	}
	if uc.JoinGameCalls != 0 {
		t.Errorf("JoinGameCalls = %d, want 0", uc.JoinGameCalls)
	}
}

// --- NewGame -----------------------------------------------------------

func TestNewGame_Redirects303(t *testing.T) {
	uc := newMockUsecase()
	uc.InitGameFn = func(ctx context.Context, userID string) (model.Game, error) {
		return model.Game{ID: "g9"}, nil
	}
	view := &MockView{}
	srv := newServer(uc, view)

	r := withSession(newReq("POST", "/game/new", nil), "u1", "nick")
	w := httptest.NewRecorder()

	srv.NewGame(w, r)

	if uc.InitGameCalls != 1 || uc.InitGameLastUserID != "u1" {
		t.Fatalf("InitGame call = %+v", uc.InitGameCalls)
	}
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/game/g9" {
		t.Errorf("Location = %q, want /game/g9", loc)
	}
}

func TestNewGame_NoSession_Redirects(t *testing.T) {
	uc := newMockUsecase()
	srv := newServer(uc, &MockView{})

	r := newReq("POST", "/game/new", nil)
	w := httptest.NewRecorder()

	srv.NewGame(w, r)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", w.Code)
	}
	if uc.InitGameCalls != 0 {
		t.Errorf("InitGameCalls = %d, want 0", uc.InitGameCalls)
	}
}

func TestNewGame_InitGameError_500(t *testing.T) {
	uc := newMockUsecase()
	uc.InitGameFn = func(ctx context.Context, userID string) (model.Game, error) {
		return model.Game{}, errSentinel
	}
	view := &MockView{}
	srv := newServer(uc, view)

	r := withSession(newReq("POST", "/game/new", nil), "u1", "nick")
	w := httptest.NewRecorder()

	srv.NewGame(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if view.renderErrorPageCalls != 1 {
		t.Errorf("renderErrorPageCalls = %d, want 1", view.renderErrorPageCalls)
	}
}

// --- Game --------------------------------------------------------------

func TestGame_RendersGamePageWithMaskedWord(t *testing.T) {
	uc := newMockUsecase()
	uc.GameStateFn = func(ctx context.Context, gameID, userID string) (model.GameState, error) {
		return model.GameState{
			GameID:    "g1",
			Word:      "apple",
			Hint:      "fruit",
			TurnEnded: false,
			Leaderboard: []model.LeaderboardEntry{
				{PlayerID: "u1", Nickname: "nick", Me: true, Score: 5},
			},
		}, nil
	}
	view := &MockView{}
	srv := newServer(uc, view)

	r := setGameID(withSession(newReq("GET", "/game/g1", nil), "u1", "nick"), "g1")
	w := httptest.NewRecorder()

	srv.Game(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if view.renderGamePageCalls != 1 {
		t.Fatalf("renderGamePageCalls = %d, want 1", view.renderGamePageCalls)
	}
	got := view.renderGamePageLastParam
	if got.GameID != "g1" {
		t.Errorf("GameID = %q, want g1", got.GameID)
	}
	if want := strings.Split("apple", ""); !equalSlices(got.MaskedWord, want) {
		t.Errorf("MaskedWord = %v, want %v", got.MaskedWord, want)
	}
	if got.EmojiHint != "fruit" {
		t.Errorf("EmojiHint = %q, want fruit", got.EmojiHint)
	}
	if len(got.Leaderboard) != 1 || got.Leaderboard[0].Me != true {
		t.Errorf("Leaderboard = %+v", got.Leaderboard)
	}
}

func TestGame_TurnEnded_Redirects303(t *testing.T) {
	uc := newMockUsecase()
	uc.GameStateFn = func(ctx context.Context, gameID, userID string) (model.GameState, error) {
		return model.GameState{GameID: "g1", TurnEnded: true}, nil
	}
	view := &MockView{}
	srv := newServer(uc, view)

	r := setGameID(withSession(newReq("GET", "/game/g1", nil), "u1", "nick"), "g1")
	w := httptest.NewRecorder()

	srv.Game(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/game/g1/loading" {
		t.Errorf("Location = %q, want /game/g1/loading", loc)
	}
	if view.renderGamePageCalls != 0 {
		t.Errorf("renderGamePageCalls = %d, want 0", view.renderGamePageCalls)
	}
}

func TestGame_GameStateError_500(t *testing.T) {
	uc := newMockUsecase()
	uc.GameStateFn = func(ctx context.Context, gameID, userID string) (model.GameState, error) {
		return model.GameState{}, errSentinel
	}
	view := &MockView{}
	srv := newServer(uc, view)

	r := setGameID(withSession(newReq("GET", "/game/g1", nil), "u1", "nick"), "g1")
	w := httptest.NewRecorder()

	srv.Game(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if view.renderErrorPageCalls != 1 {
		t.Errorf("renderErrorPageCalls = %d, want 1", view.renderErrorPageCalls)
	}
}

// --- LoadingGame -------------------------------------------------------

func TestLoadingGame_Renders(t *testing.T) {
	uc := newMockUsecase()
	view := &MockView{}
	srv := newServer(uc, view)

	r := setGameID(newReq("GET", "/game/g1/loading", nil), "g1")
	w := httptest.NewRecorder()

	srv.LoadingGame(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if view.renderGameLoadingPageCalls != 1 {
		t.Fatalf("renderGameLoadingPageCalls = %d, want 1", view.renderGameLoadingPageCalls)
	}
	// NOTE: the TODO about owner/belonging check is intentionally untested here (backlog).
	if got := view.renderGameLoadingPageLastParam; got.GameID != "g1" {
		t.Errorf("GameID = %q, want g1", got.GameID)
	}
}

func TestLoadingGame_RenderError_500(t *testing.T) {
	uc := newMockUsecase()
	view := &MockView{}
	view.renderGameLoadingPageFn = func(io.Writer, GameLoadingPageViewParam) error { return errSentinel }
	srv := newServer(uc, view)

	r := setGameID(newReq("GET", "/game/g1/loading", nil), "g1")
	w := httptest.NewRecorder()

	srv.LoadingGame(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if view.renderErrorPageCalls != 1 {
		t.Errorf("renderErrorPageCalls = %d, want 1", view.renderErrorPageCalls)
	}
}

// --- Message -----------------------------------------------------------

func TestMessage_RendersGameMsgForCurrentUser(t *testing.T) {
	uc := newMockUsecase()
	view := &MockView{}
	srv := newServer(uc, view)

	form := "content=hi+there"
	r := setGameID(
		withSession(newReq("POST", "/game/g1/message", strings.NewReader(form)), "u1", "sillyCat"),
		"g1",
	)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.Message(w, r)

	if uc.MessageCalls != 1 {
		t.Fatalf("MessageCalls = %d, want 1", uc.MessageCalls)
	}
	if uc.MessageLastWord != "hi there" {
		t.Errorf("MessageLastWord = %q, want 'hi there'", uc.MessageLastWord)
	}
	if view.renderGameMsgCalls != 1 {
		t.Fatalf("renderGameMsgCalls = %d, want 1", view.renderGameMsgCalls)
	}
	got := view.renderGameMsgLastParam
	if !got.Me {
		t.Errorf("Me = false, want true (current user)")
	}
	if got.Content != "hi there" {
		t.Errorf("Content = %q, want 'hi there'", got.Content)
	}
	if got.Nickname != "sillyCat" {
		t.Errorf("Nickname = %q, want sillyCat", got.Nickname)
	}
}

func TestMessage_UsecaseError_500(t *testing.T) {
	uc := newMockUsecase()
	uc.MessageFn = func(ctx context.Context, gameID, userID, content string) error {
		return errSentinel
	}
	view := &MockView{}
	srv := newServer(uc, view)

	r := setGameID(
		withSession(newReq("POST", "/game/g1/message", strings.NewReader("content=x")), "u1", "nick"),
		"g1",
	)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.Message(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

// --- Guess -------------------------------------------------------------

func TestGuess_SetsHxTriggerAndRendersGameMsg(t *testing.T) {
	uc := newMockUsecase()
	view := &MockView{}
	srv := newServer(uc, view)

	r := setGameID(
		withSession(newReq("POST", "/game/g1/guess", strings.NewReader("content=apple")), "u1", "sillyCat"),
		"g1",
	)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.Guess(w, r)

	if uc.GuessCalls != 1 || uc.GuessLastWord != "apple" {
		t.Fatalf("Guess = %+v", uc.GuessCalls)
	}
	if got := w.Header().Get("Hx-Trigger"); got != "guessed" {
		t.Errorf("Hx-Trigger = %q, want guessed", got)
	}
	if view.renderGameMsgCalls != 1 {
		t.Fatalf("renderGameMsgCalls = %d, want 1", view.renderGameMsgCalls)
	}
	got := view.renderGameMsgLastParam
	if !got.Me {
		t.Errorf("Me = false, want true")
	}
	if got.Content != "apple" {
		t.Errorf("Content = %q, want apple", got.Content)
	}
	if got.Nickname != "sillyCat" {
		t.Errorf("Nickname = %q, want sillyCat", got.Nickname)
	}
}

func TestGuess_UsecaseError_500(t *testing.T) {
	uc := newMockUsecase()
	uc.GuessFn = func(ctx context.Context, gameID, userID, content string) error {
		return errSentinel
	}
	view := &MockView{}
	srv := newServer(uc, view)

	r := setGameID(
		withSession(newReq("POST", "/game/g1/guess", strings.NewReader("content=x")), "u1", "nick"),
		"g1",
	)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.Guess(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	// no Hx-Trigger on failure path
	if got := w.Header().Get("Hx-Trigger"); got != "" {
		t.Errorf("Hx-Trigger = %q, want empty", got)
	}
}

// --- Leaderboard -------------------------------------------------------

func TestLeaderboard_Renders(t *testing.T) {
	uc := newMockUsecase()
	uc.LeaderboardFn = func(ctx context.Context, gameID, userID string) ([]model.LeaderboardEntry, error) {
		return []model.LeaderboardEntry{{PlayerID: "u1", Nickname: "nick", Me: true, Score: 7}}, nil
	}
	view := &MockView{}
	srv := newServer(uc, view)

	r := setGameID(withSession(newReq("GET", "/game/g1/leaderboard", nil), "u1", "nick"), "g1")
	w := httptest.NewRecorder()

	srv.Leaderboard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if view.renderGameLeaderboardCalls != 1 {
		t.Fatalf("renderGameLeaderboardCalls = %d, want 1", view.renderGameLeaderboardCalls)
	}
	if len(view.renderGameLeaderboardLastParam.Leaderboard) != 1 {
		t.Errorf("Leaderboard = %+v", view.renderGameLeaderboardLastParam.Leaderboard)
	}
}

func TestLeaderboard_UsecaseError_500(t *testing.T) {
	uc := newMockUsecase()
	uc.LeaderboardFn = func(ctx context.Context, gameID, userID string) ([]model.LeaderboardEntry, error) {
		return nil, errSentinel
	}
	view := &MockView{}
	srv := newServer(uc, view)

	r := setGameID(withSession(newReq("GET", "/game/g1/leaderboard", nil), "u1", "nick"), "g1")
	w := httptest.NewRecorder()

	srv.Leaderboard(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if view.renderErrorPageCalls != 1 {
		t.Errorf("renderErrorPageCalls = %d, want 1", view.renderErrorPageCalls)
	}
}

// --- GameWord ----------------------------------------------------------

func TestGameWord_RendersMasked(t *testing.T) {
	uc := newMockUsecase()
	uc.GameWordFn = func(ctx context.Context, gameID, userID string) (string, error) {
		return "apple", nil
	}
	view := &MockView{}
	srv := newServer(uc, view)

	r := setGameID(withSession(newReq("GET", "/game/g1/word", nil), "u1", "nick"), "g1")
	w := httptest.NewRecorder()

	srv.GameWord(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if view.renderGameWordCalls != 1 {
		t.Fatalf("renderGameWordCalls = %d, want 1", view.renderGameWordCalls)
	}
	if want := strings.Split("apple", ""); !equalSlices(view.renderGameWordLastParam.MaskedWord, want) {
		t.Errorf("MaskedWord = %v, want %v", view.renderGameWordLastParam.MaskedWord, want)
	}
}

func TestGameWord_UsecaseError_500(t *testing.T) {
	uc := newMockUsecase()
	uc.GameWordFn = func(ctx context.Context, gameID, userID string) (string, error) {
		return "", errSentinel
	}
	view := &MockView{}
	srv := newServer(uc, view)

	r := setGameID(withSession(newReq("GET", "/game/g1/word", nil), "u1", "nick"), "g1")
	w := httptest.NewRecorder()

	srv.GameWord(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if view.renderErrorPageCalls != 1 {
		t.Errorf("renderErrorPageCalls = %d, want 1", view.renderErrorPageCalls)
	}
}

// --- Sse ---------------------------------------------------------------

func TestSse_HeadersInitEventAndKickOnContextCancel(t *testing.T) {
	uc := newMockUsecase()
	kicked := make(chan struct{})
	uc.GameUpdatesFn = func(ctx context.Context, gameID, userID string, h usecase.GameUpdateHandler) error {
		<-ctx.Done()
		return nil
	}
	uc.KickInactiveUserFn = func(ctx context.Context, gameID, userID string) error {
		close(kicked)
		return nil
	}
	view := &MockView{}
	srv := newServer(uc, view)

	// Inject a near-zero kick delay so the test does not wait 30s.
	old := kickInactiveDelay
	kickInactiveDelay = 10 * time.Millisecond
	t.Cleanup(func() { kickInactiveDelay = old })

	ctx, cancel := context.WithCancel(context.Background())
	r := httptest.NewRequestWithContext(ctx, "GET", "/game/g1/sse", nil)
	r.AddCookie(&http.Cookie{Name: userIdCookieKey, Value: "u1"})
	r.SetPathValue("id", "g1")
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		srv.Sse(w, r)
		close(done)
	}()

	cancel() // unblocks GameUpdates

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Sse handler did not return after context cancel")
	}

	// headers
	if got := w.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", got)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", got)
	}
	if got := w.Header().Get("X-Accel-Buffering"); got != "no" {
		t.Errorf("X-Accel-Buffering = %q, want no", got)
	}

	// init event flushed before GameUpdates is entered
	if !strings.Contains(w.Body.String(), "event: init\n") {
		t.Errorf("body %q missing init event", w.Body.String())
	}

	// kick goroutine fires after the (near-zero) delay
	select {
	case <-kicked:
	case <-time.After(2 * time.Second):
		t.Fatal("KickInactiveUser was not called")
	}
	if uc.KickInactiveUserCalls != 1 {
		t.Errorf("KickInactiveUserCalls = %d, want 1", uc.KickInactiveUserCalls)
	}
	if uc.KickInactiveUserLastGameID != "g1" || uc.KickInactiveUserLastUserID != "u1" {
		t.Errorf("KickInactiveUser args = g1=%q u1=%q", uc.KickInactiveUserLastGameID, uc.KickInactiveUserLastUserID)
	}
}

func TestSse_MissingCookie_500(t *testing.T) {
	uc := newMockUsecase()
	view := &MockView{}
	srv := newServer(uc, view)

	r := httptest.NewRequest("GET", "http://example.com/game/g1/sse", nil)
	r.SetPathValue("id", "g1")
	w := httptest.NewRecorder()

	srv.Sse(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if view.renderErrorPageCalls != 1 {
		t.Errorf("renderErrorPageCalls = %d, want 1", view.renderErrorPageCalls)
	}
	if uc.GameUpdatesCalls != 0 {
		t.Errorf("GameUpdatesCalls = %d, want 0", uc.GameUpdatesCalls)
	}
}

// --- mux smoke test ----------------------------------------------------

// TestRouting_SmokeTest is a single smoke test that mounts the registered
// default mux through httptest.NewServer to verify the pattern-based routes
// wire up correctly (without coupling the per-handler behavior tests to the
// mux). It only checks that a couple of routes resolve without panicking.
func TestRouting_SmokeTest(t *testing.T) {
	uc := newMockUsecase()
	uc.InitUserFn = func(ctx context.Context) (model.User, error) {
		return model.User{ID: "u1", Nickname: "nick"}, nil
	}
	uc.GameStateFn = func(ctx context.Context, gameID, userID string) (model.GameState, error) {
		return model.GameState{GameID: "g1", Word: "apple"}, nil
	}
	view := &MockView{}
	srv := newServer(uc, view)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", srv.Index)
	mux.HandleFunc("GET /init", srv.InitSession)
	mux.HandleFunc("POST /game/new", srv.NewGame)
	mux.HandleFunc("GET /game/join", srv.JoinGame)
	mux.HandleFunc("GET /game/{id}/join", srv.JoinGame)
	mux.HandleFunc("GET /game/{id}/loading", srv.LoadingGame)
	mux.HandleFunc("GET /game/{id}", srv.Game)
	mux.HandleFunc("GET /game/{id}/leaderboard", srv.Leaderboard)
	mux.HandleFunc("GET /game/{id}/word", srv.GameWord)
	mux.HandleFunc("POST /game/{id}/message", srv.Message)
	mux.HandleFunc("POST /game/{id}/guess", srv.Guess)
	mux.HandleFunc("GET /game/{id}/sse", srv.Sse)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Don't follow redirects: we only want to verify a route resolves and is
	// dispatched to our handler without panicking.
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// /init creates a user and sets cookies; 302 is success here.
	resp, err := client.Get(ts.URL + "/init")
	if err != nil {
		t.Fatalf("GET /init: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("GET /init status = %d, want 302", resp.StatusCode)
	}

	// /game/join?game-id=g1 (query-id variant) with session cookies — expect 302.
	req, _ := http.NewRequest("GET", ts.URL+"/game/join?game-id=g1", nil)
	req.AddCookie(&http.Cookie{Name: userIdCookieKey, Value: "u1"})
	req.AddCookie(&http.Cookie{Name: nicknameCookieKey, Value: "nick"})
	resp2, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /game/join: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusFound {
		t.Errorf("GET /game/join status = %d, want 302", resp2.StatusCode)
	}
}

// --- misc helpers ------------------------------------------------------

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
