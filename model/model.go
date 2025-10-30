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

type Player struct {
	ID       string
	Nickname string

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
	PlayerID string
	Score    int
}
