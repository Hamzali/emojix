package service

type MockGameNotifier struct {
	GameNotifier

	PubMock   func(gameID string, userID string, notif GameNotification)
	PubCalled bool
	SubMock   func(gameID string, userID string) chan GameNotification
	UnsubMock func(userID string)
}

func (m *MockGameNotifier) Pub(gameID string, userID string, notif GameNotification) {
	m.PubCalled = true
	m.PubMock(gameID, userID, notif)
}
func (m *MockGameNotifier) Sub(gameID string, userID string) chan GameNotification {
	return m.SubMock(gameID, userID)
}
func (m *MockGameNotifier) Unsub(gameID string) {
	m.UnsubMock(gameID)
}
