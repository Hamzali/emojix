package repository

import (
	"context"
	"emojix/model"
)

type MockGameRepository struct {
	GameRepository
	GetPlayersMock       func(ctx context.Context, id string) ([]model.Player, error)
	GetMessagesMock      func(ctx context.Context, id string) ([]model.Message, error)
	GetScoresMock        func(ctx context.Context, id string) ([]model.Score, error)
	GetLatestTurnMock    func(ctx context.Context, id string) (model.GameTurn, error)
	AddPlayerMock        func(ctx context.Context, id string, playerID string) error
	AddPlayerCalled      bool
	SetPlayerStateMock   func(ctx context.Context, gameID, userID string, state model.PlayerState) error
	SetPlayerStateCalled bool
}

func (m *MockGameRepository) GetPlayers(ctx context.Context, id string) ([]model.Player, error) {
	return m.GetPlayersMock(ctx, id)
}
func (m *MockGameRepository) GetMessages(ctx context.Context, id string) ([]model.Message, error) {
	return m.GetMessagesMock(ctx, id)
}
func (m *MockGameRepository) GetScores(ctx context.Context, id string) ([]model.Score, error) {
	return m.GetScoresMock(ctx, id)
}
func (m *MockGameRepository) GetLatestTurn(ctx context.Context, id string) (model.GameTurn, error) {
	return m.GetLatestTurnMock(ctx, id)
}
func (m *MockGameRepository) AddPlayer(ctx context.Context, id string, playerID string) error {
	m.AddPlayerCalled = true
	return m.AddPlayerMock(ctx, id, playerID)
}
func (m *MockGameRepository) SetPlayerState(ctx context.Context, gameID, userID string, state model.PlayerState) error {
	m.SetPlayerStateCalled = true
	return m.SetPlayerStateMock(ctx, gameID, userID, state)
}

type MockWordRepository struct {
	WordRepository
	FindByIDMock func(ctx context.Context, id string) (model.Word, error)
}

func (m *MockWordRepository) FindByID(ctx context.Context, id string) (model.Word, error) {
	return m.FindByIDMock(ctx, id)
}

type MockUserRepository struct {
	UserRepository
	FindByIDMock func(ctx context.Context, id string) (model.User, error)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id string) (model.User, error) {
	return m.FindByIDMock(ctx, id)
}
