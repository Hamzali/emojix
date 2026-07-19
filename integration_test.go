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
// view) exactly like cmd/server/main.go, behind an httptest server.
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

	// A single word makes the random word pick deterministic.
	_, err = db.Exec("INSERT INTO words (id, word, hint) VALUES ('w1', 'Apple', 'fruit hint 🍎');")
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

func doWithCookies(t *testing.T, client *http.Client, method, url string, body io.Reader, cookies []*http.Cookie) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
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
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

// TestE2EInitNewGameGuessFlow drives a full session through the real stack:
// init session → create game → see masked word → guess correctly → turn ends.
func TestE2EInitNewGameGuessFlow(t *testing.T) {
	ts, client := newE2EServer(t)

	// 1. Init session: creates the user and sets both cookies.
	resp, err := client.Get(ts.URL + "/init")
	if err != nil {
		t.Fatalf("GET /init: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("GET /init status = %d, want 302", resp.StatusCode)
	}
	cookies := resp.Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected 2 session cookies, got %d", len(cookies))
	}
	var nickname string
	for _, c := range cookies {
		if c.Name == nicknameCookieKey {
			nickname = c.Value
		}
	}
	if nickname == "" {
		t.Fatal("nickname cookie not set")
	}

	// 2. Create a game: redirects to the new game page.
	resp = doWithCookies(t, client, "POST", ts.URL+"/game/new", nil, cookies)
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST /game/new status = %d, want 303", resp.StatusCode)
	}
	gamePath := resp.Header.Get("Location")
	if !strings.HasPrefix(gamePath, "/game/") {
		t.Fatalf("POST /game/new Location = %q, want /game/{id}", gamePath)
	}

	// 3. Game page: masked word, hint and own nickname visible; the word
	// itself must NOT leak.
	resp = doWithCookies(t, client, "GET", ts.URL+gamePath, nil, cookies)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200", gamePath, resp.StatusCode)
	}
	page := string(body)
	if !strings.Contains(page, "fruit hint 🍎") {
		t.Errorf("game page missing emoji hint")
	}
	if strings.Contains(page, "Apple") {
		t.Errorf("game page leaks the unmasked word")
	}
	if got := strings.Count(page, "<p>*</p>"); got != 5 {
		t.Errorf("masked word: got %d mask chars, want 5 (Apple)", got)
	}
	if !strings.Contains(page, nickname) {
		t.Errorf("game page missing own nickname %q", nickname)
	}

	// 4. Guess correctly (lowercase: comparison is case-insensitive).
	form := url.Values{"content": {"apple"}}
	resp = doWithCookies(t, client, "POST", ts.URL+gamePath+"/guess", strings.NewReader(form.Encode()), cookies)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST guess status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Hx-Trigger"); got != "guessed" {
		t.Errorf("Hx-Trigger = %q, want guessed", got)
	}

	// 5. Game page again: the only player guessed, so the turn ended and the
	// page redirects to the loading screen.
	resp = doWithCookies(t, client, "GET", ts.URL+gamePath, nil, cookies)
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("GET %s after guess status = %d, want 303", gamePath, resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != gamePath+"/loading" {
		t.Errorf("Location = %q, want %s", loc, gamePath+"/loading")
	}

	// 6. Loading page renders.
	resp = doWithCookies(t, client, "GET", ts.URL+gamePath+"/loading", nil, cookies)
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

	// 7. Leaderboard reflects the score earned by the guess.
	resp = doWithCookies(t, client, "GET", ts.URL+gamePath+"/leaderboard", nil, cookies)
	body, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET leaderboard status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(string(body), nickname) {
		t.Errorf("leaderboard missing nickname %q", nickname)
	}
}
