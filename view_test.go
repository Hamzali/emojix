package emojix

import (
	"bytes"
	"emojix/model"
	"strings"
	"testing"
	"time"
)

// TestNewHTMLViewParsesAllTemplates asserts that NewHTMLView parses every
// embedded template without panicking. A panic (from template.Must on a
// malformed or missing file) surfaces as a test failure here.
func TestNewHTMLViewParsesAllTemplates(t *testing.T) {
	_ = NewHTMLView()
}

// TestRenderEveryTemplate renders every View method with representative
// sample params and asserts that each render succeeds, produces a non-empty
// buffer, and contains a representative expected substring. The goal is to
// catch template shape/typo regressions in a single test, not to lock in
// exact HTML (which would couple the test to design changes).
func TestRenderEveryTemplate(t *testing.T) {
	view := NewHTMLView()

	type tc struct {
		name     string
		contains string
		render   func(buf *bytes.Buffer) error
	}
	cases := []tc{
		{
			name:     "renderErrorPage",
			contains: "Something broke",
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
					LetterCount:   4,
					WordCount:     1,
				})
			},
		},
		{
			name:     "renderGamePage status chrome",
			contains: "teller",
			render: func(buf *bytes.Buffer) error {
				return view.renderGamePage(buf, GamePageViewParam{
					GameID: "game-1",
					Leaderboard: []model.LeaderboardEntry{
						{PlayerID: "p1", Nickname: "Me-nickname", Me: true, Score: 1},
						{PlayerID: "p2", Nickname: "Announcer", Me: false, IsTeller: true, GuessedWord: true, Score: 2},
					},
					MaskedWord:    []string{"*", "*", "*", " ", "*", "*", "*", "*", "*"},
					EmojiHint:     "🍨",
					TurnStartedAt: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
					LetterCount:   8,
					WordCount:     2,
				})
			},
		},
		{
			name:     "renderGamePage waiting names teller",
			contains: "Waiting for Announcer",
			render: func(buf *bytes.Buffer) error {
				return view.renderGamePage(buf, GamePageViewParam{
					GameID:         "game-1",
					AwaitingPick:   true,
					IsTeller:       false,
					TellerNickname: "Announcer",
					Leaderboard: []model.LeaderboardEntry{
						{PlayerID: "p2", Nickname: "Announcer", IsTeller: true},
					},
				})
			},
		},
		{
			name:     "renderGamePageInPlaceTurnRefresh",
			contains: `hx-trigger="sse:turnended,sse:wordpicked,sse:newturn"`,
			render: func(buf *bytes.Buffer) error {
				return view.renderGamePage(buf, GamePageViewParam{GameID: "game-1"})
			},
		},
		{
			name:     "renderGamePageTurnEnded",
			contains: "Next turn",
			render: func(buf *bytes.Buffer) error {
				return view.renderGamePage(buf, GamePageViewParam{
					GameID:    "game-1",
					TurnEnded: true,
				})
			},
		},
		{
			name:     "renderGamePageTellerEmojiKeyboard",
			contains: "emoji-keyboard",
			render: func(buf *bytes.Buffer) error {
				return view.renderGamePage(buf, GamePageViewParam{
					GameID:        "game-1",
					IsTeller:      true,
					EmojiKeyboard: TellerEmojiKeyboard,
				})
			},
		},
		{
			name:     "renderGameWord",
			contains: "letter is-blank",
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
			name:     "renderGameMsgSystem",
			contains: "is-system",
			render: func(buf *bytes.Buffer) error {
				return view.renderGameMsg(buf, GameMsgViewParam{
					Content: "Ada got it!", Nickname: "Ada", IsSystem: true,
				})
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
			name:     "renderGameLeaderboard teller badge",
			contains: "teller-badge",
			render: func(buf *bytes.Buffer) error {
				return view.renderGameLeaderboard(buf, GameLeaderboardViewParam{
					Leaderboard: []model.LeaderboardEntry{
						{PlayerID: "p1", Nickname: "n1", IsTeller: true, GuessedWord: true, Score: 1},
					},
				})
			},
		},
		{
			name:     "renderGameLoadingPage",
			contains: "Next turn",
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

func TestRenderGamePageStatusChromeDetails(t *testing.T) {
	view := NewHTMLView()
	var buf bytes.Buffer
	err := view.renderGamePage(&buf, GamePageViewParam{
		GameID: "game-1",
		Leaderboard: []model.LeaderboardEntry{
			{PlayerID: "p2", Nickname: "Announcer", IsTeller: true, GuessedWord: true, Score: 2},
		},
		MaskedWord:    []string{"*", "*", "*"},
		TurnStartedAt: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
		LetterCount:   8,
		WordCount:     2,
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"teller-badge",
		"8 letters · 2 words",
		"turn-timer-text",
		"word-meta",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}
