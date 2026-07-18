package service

import "sync"

// MockGameNotifier is a test double for GameNotifier.
//
// Concurrency: PubCalled and PubAllCalled are guarded by mu so production
// code that spawns `go notifier.Pub(...)` does not race with assertions.
// Tests should additionally wait for the spawned goroutine to finish via
// channels (T02 pattern) before asserting on the *Called flags.
//
// Default behavior when a Mock field is nil:
//   - PubMock / PubAllMock: no-op (publishes legitimately "do nothing" in
//     negative test cases), the *Called flag is still set.
//   - SubMock / SubsMock: panic — these must return values, so an unset
//     mock panicking is the correct "you forgot to wire it" signal.
//
// The embedded GameNotifier interface is deliberately left nil so that adding
// a new interface method without a mock panics rather than silently no-ops
// (the panic comes from the nil embedded interface, not from the unset Mock
// func — kept distinct from Sub/Subs on purpose).
type MockGameNotifier struct {
	GameNotifier

	mu sync.Mutex

	PubMock      func(gameID string, userID string, notif GameNotification)
	PubCalled    bool
	PubAllMock   func(gameID string, notif GameNotification)
	PubAllCalled bool

	SubMock  func(gameID string, userID string) (chan GameNotification, func())
	SubsMock func(gameID string) []string
}

func (m *MockGameNotifier) Pub(gameID string, userID string, notif GameNotification) {
	m.mu.Lock()
	m.PubCalled = true
	m.mu.Unlock()
	if m.PubMock != nil {
		m.PubMock(gameID, userID, notif)
	}
}

func (m *MockGameNotifier) PubAll(gameID string, notif GameNotification) {
	m.mu.Lock()
	m.PubAllCalled = true
	m.mu.Unlock()
	if m.PubAllMock != nil {
		m.PubAllMock(gameID, notif)
	}
}

func (m *MockGameNotifier) Sub(gameID string, userID string) (chan GameNotification, func()) {
	return m.SubMock(gameID, userID)
}

func (m *MockGameNotifier) Subs(gameID string) []string {
	return m.SubsMock(gameID)
}
