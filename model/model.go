package model

import "time"

type Game struct {
	ID     string
	ListID string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type WordList struct {
	ID    string
	Title string
}

type Word struct {
	ID     string
	ListID string
	Word   string
	Hint   string
}

type GameTurn struct {
	ID        string
	GameID    string
	WordID    string // empty until teller picks
	TellerID  string
	OptionA   string
	OptionB   string
	OptionC   string
	EmojiHint string // live emoji board; seeded from word.hint on pick
	CreatedAt time.Time
	StartedAt time.Time // zero until teller picks
}

type Score struct {
	GameID    string
	PlayerID  string
	MessageID string
	TurnID    string
	Score     int
	CreatedAt time.Time
}

type PlayerState = string

var ActivePlayerState PlayerState = "active"
var InactivePlayerState PlayerState = "inactive"

type Player struct {
	ID       string
	Nickname string

	State string

	JoinedAt time.Time
}

type Message struct {
	ID        string
	Content   string
	PlayerID  string
	TurnID    string
	CreatedAt time.Time
}

type User struct {
	ID       string
	Nickname string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type LeaderboardEntry struct {
	PlayerID    string
	Nickname    string
	Me          bool
	GuessedWord bool
	IsTeller    bool
	Score       int
}

type GameStateMessage struct {
	Me       bool
	Content  string
	Nickname string
	IsSystem bool // correct-guess announcement, not a chat line
	IsGuess  bool // wrong-guess line (live response; optional style)
}

type GameState struct {
	GameID         string
	CurrentUserID  string
	TurnID         string
	TurnStartedAt  time.Time
	TurnEnded      bool
	AwaitingPick   bool
	IsTeller       bool
	TellerNickname string
	WordOptions    []Word // teller-only, while AwaitingPick
	Word           string
	Hint           string
	LetterCount    int // letters in the secret word (spaces excluded)
	WordCount      int // whitespace-separated words in the secret
	Messages       []GameStateMessage
	Leaderboard    []LeaderboardEntry
}
