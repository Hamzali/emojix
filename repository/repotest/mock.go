// Package repotest holds test doubles for the repository package. It is a
// separate package (like net/http/httptest) so the doubles are importable by
// other packages' tests without shipping them in the production binary.
//
// Concurrency: the *Called flags are plain bools with no mutex. This is safe
// because production code only ever calls repository methods synchronously
// (only GameNotifier.Pub is spawned in a goroutine). If that ever changes,
// guard the flags the way servicetest.MockGameNotifier does.
package repotest

import (
	"context"
	"emojix/model"
	"emojix/repository"
)

type MockGameRepository struct {
	repository.GameRepository
	FindByIDMock         func(ctx context.Context, id string) (model.Game, error)
	CreateMock           func(ctx context.Context, listID string) (model.Game, error)
	CreateCalled         bool
	GetPlayersMock       func(ctx context.Context, id string) ([]model.Player, error)
	GetMessagesMock      func(ctx context.Context, id string) ([]model.Message, error)
	GetScoresMock        func(ctx context.Context, id string) ([]model.Score, error)
	GetLatestTurnMock    func(ctx context.Context, id string) (model.GameTurn, error)
	AddTurnMock          func(ctx context.Context, params repository.AddTurnParams) (model.GameTurn, error)
	AddTurnCalled        bool
	SetTurnWordMock      func(ctx context.Context, turnID, wordID, emojiHint string) error
	SetTurnWordCalled    bool
	SetTurnWordLastHint  string
	CountTurnsMock       func(ctx context.Context, gameID string) (int, error)
	SendMessageMock      func(ctx context.Context, gameID string, turnID string, userID string, content string) (model.Message, error)
	SendMessageCalled    bool
	AddPlayerMock        func(ctx context.Context, id string, playerID string) error
	AddPlayerCalled      bool
	SetPlayerStateMock   func(ctx context.Context, gameID, userID string, state model.PlayerState) error
	SetPlayerStateCalled bool
	AddScoreMock         func(ctx context.Context, gameID string, userID string, messageID string, turnID string, score int) error
	AddScoreCalled       bool
}

func (m *MockGameRepository) FindByID(ctx context.Context, id string) (model.Game, error) {
	return m.FindByIDMock(ctx, id)
}

func (m *MockGameRepository) Create(ctx context.Context, listID string) (model.Game, error) {
	m.CreateCalled = true
	return m.CreateMock(ctx, listID)
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
func (m *MockGameRepository) AddTurn(ctx context.Context, params repository.AddTurnParams) (model.GameTurn, error) {
	m.AddTurnCalled = true
	return m.AddTurnMock(ctx, params)
}
func (m *MockGameRepository) SetTurnWord(ctx context.Context, turnID, wordID, emojiHint string) error {
	m.SetTurnWordCalled = true
	m.SetTurnWordLastHint = emojiHint
	if m.SetTurnWordMock != nil {
		return m.SetTurnWordMock(ctx, turnID, wordID, emojiHint)
	}
	return nil
}
func (m *MockGameRepository) CountTurns(ctx context.Context, gameID string) (int, error) {
	if m.CountTurnsMock != nil {
		return m.CountTurnsMock(ctx, gameID)
	}
	return 0, nil
}
func (m *MockGameRepository) SendMessage(ctx context.Context, gameID string, turnID string, userID string, content string) (model.Message, error) {
	m.SendMessageCalled = true
	return m.SendMessageMock(ctx, gameID, turnID, userID, content)
}
func (m *MockGameRepository) AddPlayer(ctx context.Context, id string, playerID string) error {
	m.AddPlayerCalled = true
	return m.AddPlayerMock(ctx, id, playerID)
}
func (m *MockGameRepository) SetPlayerState(ctx context.Context, gameID, userID string, state model.PlayerState) error {
	m.SetPlayerStateCalled = true
	return m.SetPlayerStateMock(ctx, gameID, userID, state)
}
func (m *MockGameRepository) AddScore(ctx context.Context, gameID string, userID string, messageID string, turnID string, score int) error {
	m.AddScoreCalled = true
	return m.AddScoreMock(ctx, gameID, userID, messageID, turnID, score)
}

type MockWordRepository struct {
	repository.WordRepository
	FindByIDMock         func(ctx context.Context, id string) (model.Word, error)
	GetListsMock         func(ctx context.Context) ([]model.WordList, error)
	GetUnusedByListMock  func(ctx context.Context, listID, gameID string) ([]model.Word, error)
	GetUnusedByListCount int
}

func (m *MockWordRepository) FindByID(ctx context.Context, id string) (model.Word, error) {
	return m.FindByIDMock(ctx, id)
}

func (m *MockWordRepository) GetLists(ctx context.Context) ([]model.WordList, error) {
	if m.GetListsMock != nil {
		return m.GetListsMock(ctx)
	}
	return nil, nil
}

func (m *MockWordRepository) GetUnusedByList(ctx context.Context, listID, gameID string) ([]model.Word, error) {
	m.GetUnusedByListCount++
	return m.GetUnusedByListMock(ctx, listID, gameID)
}

type MockUserRepository struct {
	repository.UserRepository
	FindByIDMock         func(ctx context.Context, id string) (model.User, error)
	CreateOrUpdateMock   func(ctx context.Context, id string, params repository.UserCreateOrUpdateParams) error
	CreateOrUpdateCalled bool
}

func (m *MockUserRepository) FindByID(ctx context.Context, id string) (model.User, error) {
	return m.FindByIDMock(ctx, id)
}

func (m *MockUserRepository) CreateOrUpdate(ctx context.Context, id string, params repository.UserCreateOrUpdateParams) error {
	m.CreateOrUpdateCalled = true
	return m.CreateOrUpdateMock(ctx, id, params)
}

type MockUnitOfWork struct {
	repository.UnitOfWork
	GameRepositoryMock *MockGameRepository
	RollbackMock       func() error
	CommitMock         func() error
	RollbackCalled     bool
	CommitCalled       bool
}

// GameRepository implements repository.UnitOfWork.
func (uow *MockUnitOfWork) GameRepository() repository.GameRepository {
	return uow.GameRepositoryMock
}

// Commit implements repository.UnitOfWork.
func (uow *MockUnitOfWork) Commit() error {
	uow.CommitCalled = true
	return uow.CommitMock()
}

// Rollback implements repository.UnitOfWork.
func (uow *MockUnitOfWork) Rollback() error {
	uow.RollbackCalled = true
	return uow.RollbackMock()
}

type MockUnitOfWorkFactory struct {
	repository.UnitOfWorkFactory
	NewMock func(ctx context.Context) (repository.UnitOfWork, error)
}

// New implements repository.UnitOfWorkFactory.
func (f *MockUnitOfWorkFactory) New(ctx context.Context) (repository.UnitOfWork, error) {
	if f.NewMock != nil {
		return f.NewMock(ctx)
	}
	return &MockUnitOfWork{}, nil
}
