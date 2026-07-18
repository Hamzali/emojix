package emojix

import (
	"bytes"
	"emojix/model"
	"os"
	"strings"
	"testing"
	"time"
)

// Templates are parsed from relative paths like "template/index.gohtml", so
// these tests must run from the repo root. Fail fast with a clear message if a
// contributor runs them from the wrong directory instead of getting a cryptic
// template.ParseFiles error deep in NewHTMLView.
func requireTemplateDir(t *testing.T) {
	t.Helper()
	if _, err := os.Stat("template/index.gohtml"); err != nil {
		t.Fatalf("template/index.gohtml not found — run tests from the repo root: %v", err)
	}
}

// TestNewHTMLViewParsesAllTemplates asserts that NewHTMLView parses every
// template/*.gohtml without panicking. A panic (from template.Must on a
// malformed or missing file) surfaces as a test failure here.
func TestNewHTMLViewParsesAllTemplates(t *testing.T) {
	requireTemplateDir(t)
	_ = NewHTMLView()
}

// TestRenderEveryTemplate renders every View method with representative
// sample params and asserts that each render succeeds, produces a non-empty
// buffer, and contains a representative expected substring. The goal is to
// catch template shape/typo regressions in a single test, not to lock in
// exact HTML (which would couple the test to design changes).
func TestRenderEveryTemplate(t *testing.T) {
	requireTemplateDir(t)
	view := NewHTMLView()

	type tc struct {
		name     string
		contains string
		render   func(buf *bytes.Buffer) error
	}
	cases := []tc{
		{
			name:     "renderErrorPage",
			contains: "Unexpected Error",
			render: func(buf *bytes.Buffer) error {
				return view.renderErrorPage(buf)
			},
		},
		{
			name:     "renderIndexPage",
			contains: "Welcome, <em>y</em>",
			render: func(buf *bytes.Buffer) error {
				return view.renderIndexPage(buf, IndexPageViewParam{Title: "x", Nickname: "y"})
			},
		},
		{
			name:     "renderGamePage",
			contains: "Me-nickname",
			render: func(buf *bytes.Buffer) error {
				return view.renderGamePage(buf, GamePageViewParam{
					GameID: "game-1",
					Leaderboard: []model.LeaderboardEntry{
						{PlayerID: "p1", Nickname: "Me-nickname", Me: true, GuessedWord: true, Score: 10},
						{PlayerID: "p2", Nickname: "Other-nickname", Me: false, GuessedWord: false, Score: 3},
					},
					Messages: []model.GameStateMessage{
						{Me: true, Content: "guess-1", Nickname: "Me-nickname"},
						{Me: false, Content: "guess-2", Nickname: "Other-nickname"},
					},
					MaskedWord:    []string{"_", "_", "_", "_"},
					EmojiHint:     "🐝",
					TurnStartedAt: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
				})
			},
		},
		{
			name:     "renderGameWord",
			contains: "_",
			render: func(buf *bytes.Buffer) error {
				return view.renderGameWord(buf, GameWordViewParam{MaskedWord: []string{"_", "_", "_", "_"}})
			},
		},
		{
			name:     "renderGameMsg",
			contains: "hello",
			render: func(buf *bytes.Buffer) error {
				return view.renderGameMsg(buf, GameMsgViewParam{Me: true, Content: "hello", Nickname: "y"})
			},
		},
		{
			name:     "renderGameLeaderboard",
			contains: "n1",
			render: func(buf *bytes.Buffer) error {
				return view.renderGameLeaderboard(buf, GameLeaderboardViewParam{
					Leaderboard: []model.LeaderboardEntry{
						{PlayerID: "p1", Nickname: "n1", Score: 1},
						{PlayerID: "p2", Nickname: "n2", Score: 2},
					},
				})
			},
		},
		{
			name:     "renderGameLoadingPage",
			contains: "Loading Game Turn",
			render: func(buf *bytes.Buffer) error {
				return view.renderGameLoadingPage(buf, GameLoadingPageViewParam{GameID: "game-1"})
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := c.render(&buf); err != nil {
				t.Fatalf("%s: render returned error: %v", c.name, err)
			}
			if buf.Len() == 0 {
				t.Fatalf("%s: rendered buffer is empty", c.name)
			}
			if !strings.Contains(buf.String(), c.contains) {
				t.Fatalf("%s: rendered output missing expected substring %q\noutput:\n%s", c.name, c.contains, buf.String())
			}
		})
	}
}
