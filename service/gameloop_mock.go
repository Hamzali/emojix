package service

import (
	"context"
	"time"
)

// MockGameLoop is a mock implementation of GameLoop for testing.
// Embeds the GameLoop interface for default no-op on unimplemented methods.
type MockGameLoop struct {
	GameLoop

	StartMock                 func(ctx context.Context, gameID string, duration time.Duration)
	StartCalled               bool
	EndGameTurnMock           func(gameID string)
	EndGameTurnCalled         bool
	SetOnTurnEndHandlerMock   func(handler OnTurnEndHandler)
	SetOnTurnEndHandlerCalled bool
	OnTurnEndHandler          OnTurnEndHandler
	StopGameMock              func(gameID string)
	StopGameCalled            bool
	StopMock                  func()
	StopCalled                bool
}

func (m *MockGameLoop) Start(ctx context.Context, gameID string, duration time.Duration) {
	m.StartCalled = true
	if m.StartMock != nil {
		m.StartMock(ctx, gameID, duration)
	}
}

func (m *MockGameLoop) EndGameTurn(gameID string) {
	m.EndGameTurnCalled = true
	if m.EndGameTurnMock != nil {
		m.EndGameTurnMock(gameID)
	}
}

func (m *MockGameLoop) SetOnTurnEndHandler(handler OnTurnEndHandler) {
	m.SetOnTurnEndHandlerCalled = true
	m.OnTurnEndHandler = handler
	if m.SetOnTurnEndHandlerMock != nil {
		m.SetOnTurnEndHandlerMock(handler)
	}
}

// FireOnTurnEnd invokes the handler captured by SetOnTurnEndHandler, if any.
// It allows tests to drive onTurnEnd deterministically without real timers.
func (m *MockGameLoop) FireOnTurnEnd(ctx context.Context, gameID string) {
	if m.OnTurnEndHandler != nil {
		m.OnTurnEndHandler(ctx, gameID)
	}
}

func (m *MockGameLoop) StopGame(gameID string) {
	m.StopGameCalled = true
	if m.StopGameMock != nil {
		m.StopGameMock(gameID)
	}
}

func (m *MockGameLoop) Stop() {
	m.StopCalled = true
	if m.StopMock != nil {
		m.StopMock()
	}
}
