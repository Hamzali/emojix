package emojix

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"emojix/repository"
	"emojix/service"
	"emojix/usecase"
)

// newE2EServer wires the REAL layers (sqlite DB, repositories, usecase, HTML
// view) exactly like cmd/emojix serve, behind an httptest server.
//
// A temp FILE database is used (not :memory:) because InitGame reads words
// outside its transaction; with a single-connection in-memory DB that read
// would deadlock waiting for the connection held by the open transaction.
func newE2EServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "e2e.db")
	db, err := repository.InitSqliteDB(dbPath)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	migrator, err := repository.NewSQLiteMigrator(db, dbPath, "database/migrations")
	if err != nil {
		t.Fatalf("new migrator: %v", err)
	}
	if err := migrator.UpCmd(); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	// Dense enough for 3 options; one list, three words.
	_, err = db.Exec(`
		INSERT INTO word_lists (id, title) VALUES ('l1', 'Test List');
		INSERT INTO words (id, list_id, word, hint) VALUES
			('w1', 'l1', 'Apple', 'fruit hint 🍎'),
			('w2', 'l1', 'Banana', 'yellow 🍌'),
			('w3', 'l1', 'Cherry', 'red 🍒');
	`)
	if err != nil {
		t.Fatalf("seed words: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	gameRepo := repository.NewGameRepository(db)
	wordRepo := repository.NewWordRepository(db)
	unitOfWorkFactory := repository.NewUnitOfWorkFactory(db)
	gameNotifier := service.NewGameNotifier()
	gameLoop := service.NewGameLoop(service.NewRealClock())
	t.Cleanup(gameLoop.Stop)

	uc := usecase.NewEmojixUsecase(
		userRepo,
		gameRepo,
		wordRepo,
		unitOfWorkFactory,
		gameNotifier,
		gameLoop,
		service.NewRealClock(),
	)

	srv := &webServer{view: NewHTMLView(), emojixUsecase: uc, kickDelay: defaultKickDelay}

	ts := httptest.NewServer(srv.mux())
	t.Cleanup(ts.Close)

	// Don't follow redirects: each step asserts status + Location explicitly.
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return ts, client
}

func doWithCookies(t *testing.T, client *http.Client, method, urlStr string, body io.Reader, cookies []*http.Cookie) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, urlStr, err)
	}
	return resp
}

func initSession(t *testing.T, ts *httptest.Server, client *http.Client) (cookies []*http.Cookie, nickname string) {
	t.Helper()
	resp, err := client.Get(ts.URL + "/init")
	if err != nil {
		t.Fatalf("GET /init: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("GET /init status = %d, want 302", resp.StatusCode)
	}
	cookies = resp.Cookies()
	for _, c := range cookies {
		if c.Name == nicknameCookieKey {
			nickname = c.Value
		}
	}
	if nickname == "" {
		t.Fatal("nickname cookie not set")
	}
	return cookies, nickname
}

// TestE2EInitNewGameGuessFlow drives a full session through the real stack:
// init host → create game → pick word → second player joins → guesses → turn ends.
func TestE2EInitNewGameGuessFlow(t *testing.T) {
	ts, client := newE2EServer(t)

	// 1. Host session.
	hostCookies, hostNick := initSession(t, ts, client)

	// 2. Create a game with list l1.
	form := url.Values{"list-id": {"l1"}}
	resp := doWithCookies(t, client, "POST", ts.URL+"/game/new", strings.NewReader(form.Encode()), hostCookies)
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST /game/new status = %d, want 303", resp.StatusCode)
	}
	gamePath := resp.Header.Get("Location")
	if !strings.HasPrefix(gamePath, "/game/") {
		t.Fatalf("POST /game/new Location = %q, want /game/{id}", gamePath)
	}

	// 3. Host (teller) sees pick UI with 3 options.
	resp = doWithCookies(t, client, "GET", ts.URL+gamePath, nil, hostCookies)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200", gamePath, resp.StatusCode)
	}
	page := string(body)
	if !strings.Contains(page, "Pick a word") {
		t.Errorf("teller page missing pick prompt")
	}
	if !strings.Contains(page, "Apple") || !strings.Contains(page, "Banana") || !strings.Contains(page, "Cherry") {
		t.Errorf("teller page missing word options")
	}

	// 4. Host picks Apple.
	form = url.Values{"word-id": {"w1"}}
	resp = doWithCookies(t, client, "POST", ts.URL+gamePath+"/pick", strings.NewReader(form.Encode()), hostCookies)
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST pick status = %d, want 303", resp.StatusCode)
	}

	// 5. Guesser session joins.
	guesserCookies, guesserNick := initSession(t, ts, client)
	resp = doWithCookies(t, client, "GET", ts.URL+gamePath+"/join", nil, guesserCookies)
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("join status = %d, want 302", resp.StatusCode)
	}

	// 6. Guesser sees masked word + hint, not the plain word.
	resp = doWithCookies(t, client, "GET", ts.URL+gamePath, nil, guesserCookies)
	body, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s as guesser status = %d, want 200", gamePath, resp.StatusCode)
	}
	page = string(body)
	if !strings.Contains(page, "fruit hint 🍎") {
		t.Errorf("game page missing emoji hint")
	}
	if strings.Contains(page, "Apple") {
		t.Errorf("game page leaks the unmasked word")
	}
	if got := strings.Count(page, "<p>*</p>"); got != 5 {
		t.Errorf("masked word: got %d mask chars, want 5 (Apple)", got)
	}
	if !strings.Contains(page, guesserNick) {
		t.Errorf("game page missing guesser nickname %q", guesserNick)
	}

	// 7. Guesser guesses correctly.
	form = url.Values{"content": {"apple"}}
	resp = doWithCookies(t, client, "POST", ts.URL+gamePath+"/guess", strings.NewReader(form.Encode()), guesserCookies)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST guess status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Hx-Trigger"); got != "guessed" {
		t.Errorf("Hx-Trigger = %q, want guessed", got)
	}

	// 8. Only guesser needed to finish (teller excluded) → loading redirect.
	resp = doWithCookies(t, client, "GET", ts.URL+gamePath, nil, guesserCookies)
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("GET %s after guess status = %d, want 303", gamePath, resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != gamePath+"/loading" {
		t.Errorf("Location = %q, want %s", loc, gamePath+"/loading")
	}

	// 9. Loading page renders.
	resp = doWithCookies(t, client, "GET", ts.URL+gamePath+"/loading", nil, guesserCookies)
	body, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET loading status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(string(body), "Loading Game Turn") {
		t.Errorf("loading page missing expected content")
	}

	// 10. Leaderboard reflects the score.
	resp = doWithCookies(t, client, "GET", ts.URL+gamePath+"/leaderboard", nil, guesserCookies)
	body, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET leaderboard status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(string(body), guesserNick) {
		t.Errorf("leaderboard missing nickname %q", guesserNick)
	}
	_ = hostNick
}
