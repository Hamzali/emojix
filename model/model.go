package model

import "time"

type Game struct {
	ID string

	CreatedAt time.Time
	UpdatedAt time.Time
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
	CreatedAt time.Time
}

type User struct {
	ID       string
	Nickname string

	CreatedAt time.Time
	UpdatedAt time.Time
}
