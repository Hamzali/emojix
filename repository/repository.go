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

type AddTurnParams struct {
	GameID   string
	TellerID string
	OptionA  string
	OptionB  string
	OptionC  string
}

type GameRepository interface {
	FindByID(ctx context.Context, id string) (model.Game, error)
	Create(ctx context.Context, listID string) (model.Game, error)

	// Players/Users
	AddPlayer(ctx context.Context, gameID string, userID string) error
	SetPlayerState(ctx context.Context, gameID string, userID string, state model.PlayerState) error
	GetPlayers(ctx context.Context, gameID string) ([]model.Player, error)

	GetLatestTurn(ctx context.Context, gameID string) (model.GameTurn, error)
	AddTurn(ctx context.Context, params AddTurnParams) (model.GameTurn, error)
	// SetTurnWord assigns the picked word and seeds emoji_hint (typically word.Hint).
	SetTurnWord(ctx context.Context, turnID string, wordID string, emojiHint string) error
	// SetTurnEmojiHint replaces the full live emoji board for the turn.
	SetTurnEmojiHint(ctx context.Context, turnID string, emojiHint string) error
	CountTurns(ctx context.Context, gameID string) (int, error)

	// Message/Content
	GetMessages(ctx context.Context, gameID string) ([]model.Message, error)
	SendMessage(ctx context.Context, gameID string, turnID string, userID string, content string) (model.Message, error)

	GetScores(ctx context.Context, gameID string) ([]model.Score, error)
	AddScore(ctx context.Context, gameID string, userID string, messageID string, turnID string, score int) error
}

type WordRepository interface {
	GetLists(ctx context.Context) ([]model.WordList, error)
	// GetUnusedByList returns words in listID not yet played (word_id set) in gameID.
	GetUnusedByList(ctx context.Context, listID, gameID string) ([]model.Word, error)
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
