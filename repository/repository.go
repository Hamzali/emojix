package repository

import (
	"context"
	"emojix/model"
)

type UserCreateOrUpdateParams struct {
	Nickname string
}

type UserRepository interface {
	FindByID(ctx context.Context, id string) (model.User, error)
	CreateOrUpdate(ctx context.Context, id string, params UserCreateOrUpdateParams) error
}

type GameRepository interface {
	FindByID(ctx context.Context, id string) (model.Game, error)
	Create(ctx context.Context) (model.Game, error)

	// Players/Users
	AddPlayer(ctx context.Context, gameID string, userID string) error
	SetPlayerState(ctx context.Context, gameID string, userID string, state model.PlayerState) error
	GetPlayers(ctx context.Context, gameID string) ([]model.Player, error)

	GetLatestTurn(ctx context.Context, gameID string) (model.GameTurn, error)
	AddTurn(ctx context.Context, gameID string, wordID string) error

	// Message/Content
	GetMessages(ctx context.Context, gameID string) ([]model.Message, error)
	SendMessage(ctx context.Context, gameID string, turnID string, userID string, content string) (model.Message, error)

	GetScores(ctx context.Context, gameID string) ([]model.Score, error)
	AddScore(ctx context.Context, gameID string, userID string, messageID string, turnID string, score int) error
}

type WordRepository interface {
	GetAll(ctx context.Context) ([]model.Word, error)
	FindByID(ctx context.Context, id string) (model.Word, error)
}

type UnitOfWorkFactory interface {
	New(ctx context.Context) (UnitOfWork, error)
}

type UnitOfWork interface {
	GameRepository() GameRepository

	Rollback() error
	Commit() error
}
