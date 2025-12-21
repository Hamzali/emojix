package service

type MockGameNotifier struct {
	GameNotifier

	PubMock         func(gameID string, userID string, notif GameNotification)
	PubCalled       bool
	SubMock         func(gameID string, userID string) chan GameNotification
	SubsMock        func(gameID string) []string
	UnsubMock       func(userID string)
	UnsubMockCalled bool
}

func (m *MockGameNotifier) Pub(gameID string, userID string, notif GameNotification) {
	m.PubCalled = true
	m.PubMock(gameID, userID, notif)
}
func (m *MockGameNotifier) Sub(gameID string, userID string) chan GameNotification {
	return m.SubMock(gameID, userID)
}
func (m *MockGameNotifier) Subs(gameID string) []string {
	return m.SubsMock(gameID)
}
func (m *MockGameNotifier) Unsub(gameID string) {
	m.UnsubMockCalled = true
	m.UnsubMock(gameID)
}
