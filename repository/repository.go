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
	Create(ctx context.Context, word string, hint string) (model.Game, error)

	AddPlayer(ctx context.Context, gameID string, userID string) error
	GetPlayers(ctx context.Context, gameID string) ([]model.Player, error)

	GetMessages(ctx context.Context, gameID string) ([]model.Message, error)
	SendMessage(ctx context.Context, gameID string, userID string, content string) error
}
