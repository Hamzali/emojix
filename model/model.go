package model

import "time"

type Game struct {
	ID string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type Word struct {
	ID   string
	Word string
	Hint string
}

type GameTurn struct {
	ID     string
	GameID string
	WordID string

	CreatedAt time.Time
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
	Nickname    string
	Me          bool
	GuessedWord bool
	Score       int
}

type GameStateMessage struct {
	Me       bool
	Content  string
	Nickname string
}

type GameState struct {
	GameID        string
	CurrentUserID string
	TurnID        string
	TurnEnded     bool
	Word          string
	Hint          string
	Messages      []GameStateMessage
	Leaderboard   []LeaderboardEntry
}
