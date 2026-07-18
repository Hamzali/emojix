package emojix

import (
	"context"
	"emojix/model"
	"emojix/usecase"
	"io"
	"sync"
)

// MockEmojixUsecase is a per-method func-field mock of usecase.EmojixUsecase.
//
// Every method has a func field that defaults to a defensive no-op (or
// zero-returning) implementation so that a server test only has to set the
// methods it cares about. Each method also records how often it was called
// and (where useful) the args of the last call.
//
// It is intentionally not part of a reusable mock package: it is only used by
// server tests in this package.
type MockEmojixUsecase struct {
	mu sync.Mutex

	InitUserFn      func(ctx context.Context) (model.User, error)
	InitUserCalls   int
	InitUserLastCtx context.Context

	InitGameFn         func(ctx context.Context, userID string) (model.Game, error)
	InitGameCalls      int
	InitGameLastUserID string

	JoinGameFn         func(ctx context.Context, gameID string, userID string) error
	JoinGameCalls      int
	JoinGameLastGameID string
	JoinGameLastUserID string

	GuessFn         func(ctx context.Context, gameID string, userID string, word string) error
	GuessCalls      int
	GuessLastGameID string
	GuessLastUserID string
	GuessLastWord   string

	MessageFn         func(ctx context.Context, gameID string, userID string, word string) error
	MessageCalls      int
	MessageLastGameID string
	MessageLastUserID string
	MessageLastWord   string

	GameStateFn         func(ctx context.Context, gameID string, userID string) (model.GameState, error)
	GameStateCalls      int
	GameStateLastGameID string
	GameStateLastUserID string

	GameUpdatesFn         func(ctx context.Context, gameID string, userID string, handler usecase.GameUpdateHandler) error
	GameUpdatesCalls      int
	GameUpdatesLastGameID string
	GameUpdatesLastUserID string

	KickInactiveUserFn         func(ctx context.Context, gameID, userID string) error
	KickInactiveUserCalls      int
	KickInactiveUserLastGameID string
	KickInactiveUserLastUserID string

	LeaderboardFn         func(ctx context.Context, gameID, userID string) ([]model.LeaderboardEntry, error)
	LeaderboardCalls      int
	LeaderboardLastGameID string
	LeaderboardLastUserID string

	GameWordFn         func(ctx context.Context, gameID, userID string) (string, error)
	GameWordCalls      int
	GameWordLastGameID string
	GameWordLastUserID string
}

func newMockUsecase() *MockEmojixUsecase {
	m := &MockEmojixUsecase{}
	// defensive no-op / zero defaults
	m.InitUserFn = func(ctx context.Context) (model.User, error) {
		return model.User{}, nil
	}
	m.InitGameFn = func(ctx context.Context, userID string) (model.Game, error) {
		return model.Game{}, nil
	}
	m.JoinGameFn = func(ctx context.Context, gameID, userID string) error {
		return nil
	}
	m.GuessFn = func(ctx context.Context, gameID, userID, word string) error {
		return nil
	}
	m.MessageFn = func(ctx context.Context, gameID, userID, word string) error {
		return nil
	}
	m.GameStateFn = func(ctx context.Context, gameID, userID string) (model.GameState, error) {
		return model.GameState{}, nil
	}
	m.GameUpdatesFn = func(ctx context.Context, gameID, userID string, handler usecase.GameUpdateHandler) error {
		return nil
	}
	m.KickInactiveUserFn = func(ctx context.Context, gameID, userID string) error {
		return nil
	}
	m.LeaderboardFn = func(ctx context.Context, gameID, userID string) ([]model.LeaderboardEntry, error) {
		return nil, nil
	}
	m.GameWordFn = func(ctx context.Context, gameID, userID string) (string, error) {
		return "", nil
	}
	return m
}

func (m *MockEmojixUsecase) InitUser(ctx context.Context) (model.User, error) {
	m.mu.Lock()
	m.InitUserCalls++
	m.InitUserLastCtx = ctx
	m.mu.Unlock()
	return m.InitUserFn(ctx)
}

func (m *MockEmojixUsecase) InitGame(ctx context.Context, userID string) (model.Game, error) {
	m.mu.Lock()
	m.InitGameCalls++
	m.InitGameLastUserID = userID
	m.mu.Unlock()
	return m.InitGameFn(ctx, userID)
}

func (m *MockEmojixUsecase) JoinGame(ctx context.Context, gameID string, userID string) error {
	m.mu.Lock()
	m.JoinGameCalls++
	m.JoinGameLastGameID = gameID
	m.JoinGameLastUserID = userID
	m.mu.Unlock()
	return m.JoinGameFn(ctx, gameID, userID)
}

func (m *MockEmojixUsecase) Guess(ctx context.Context, gameID string, userID string, word string) error {
	m.mu.Lock()
	m.GuessCalls++
	m.GuessLastGameID = gameID
	m.GuessLastUserID = userID
	m.GuessLastWord = word
	m.mu.Unlock()
	return m.GuessFn(ctx, gameID, userID, word)
}

func (m *MockEmojixUsecase) Message(ctx context.Context, gameID string, userID string, word string) error {
	m.mu.Lock()
	m.MessageCalls++
	m.MessageLastGameID = gameID
	m.MessageLastUserID = userID
	m.MessageLastWord = word
	m.mu.Unlock()
	return m.MessageFn(ctx, gameID, userID, word)
}

func (m *MockEmojixUsecase) GameState(ctx context.Context, gameID string, userID string) (model.GameState, error) {
	m.mu.Lock()
	m.GameStateCalls++
	m.GameStateLastGameID = gameID
	m.GameStateLastUserID = userID
	m.mu.Unlock()
	return m.GameStateFn(ctx, gameID, userID)
}

func (m *MockEmojixUsecase) GameUpdates(ctx context.Context, gameID string, userID string, handler usecase.GameUpdateHandler) error {
	m.mu.Lock()
	m.GameUpdatesCalls++
	m.GameUpdatesLastGameID = gameID
	m.GameUpdatesLastUserID = userID
	m.mu.Unlock()
	return m.GameUpdatesFn(ctx, gameID, userID, handler)
}

func (m *MockEmojixUsecase) KickInactiveUser(ctx context.Context, gameID, userID string) error {
	m.mu.Lock()
	m.KickInactiveUserCalls++
	m.KickInactiveUserLastGameID = gameID
	m.KickInactiveUserLastUserID = userID
	m.mu.Unlock()
	return m.KickInactiveUserFn(ctx, gameID, userID)
}

func (m *MockEmojixUsecase) Leaderboard(ctx context.Context, gameID, userID string) ([]model.LeaderboardEntry, error) {
	m.mu.Lock()
	m.LeaderboardCalls++
	m.LeaderboardLastGameID = gameID
	m.LeaderboardLastUserID = userID
	m.mu.Unlock()
	return m.LeaderboardFn(ctx, gameID, userID)
}

func (m *MockEmojixUsecase) GameWord(ctx context.Context, gameID, userID string) (string, error) {
	m.mu.Lock()
	m.GameWordCalls++
	m.GameWordLastGameID = gameID
	m.GameWordLastUserID = userID
	m.mu.Unlock()
	return m.GameWordFn(ctx, gameID, userID)
}

// Compile-time guard.
var _ usecase.EmojixUsecase = (*MockEmojixUsecase)(nil)

// MockView is a per-method func-field mock of the (unexported) View interface.
//
// Each render method records how often it was called, the last args it saw,
// and (if the corresponding func field is set) delegates to it — which lets a
// test opt into writing canned bytes to the io.Writer.
type MockView struct {
	mu sync.Mutex

	renderErrorPageFn     func(wr io.Writer) error
	renderErrorPageCalls  int
	renderErrorPageWriter io.Writer

	renderIndexPageFn        func(wr io.Writer, params IndexPageViewParam) error
	renderIndexPageCalls     int
	renderIndexPageLastParam IndexPageViewParam
	renderIndexPageWriter    io.Writer

	renderGamePageFn        func(wr io.Writer, params GamePageViewParam) error
	renderGamePageCalls     int
	renderGamePageLastParam GamePageViewParam
	renderGamePageWriter    io.Writer

	renderGameWordFn        func(wr io.Writer, params GameWordViewParam) error
	renderGameWordCalls     int
	renderGameWordLastParam GameWordViewParam
	renderGameWordWriter    io.Writer

	renderGameMsgFn        func(wr io.Writer, params GameMsgViewParam) error
	renderGameMsgCalls     int
	renderGameMsgLastParam GameMsgViewParam
	renderGameMsgWriter    io.Writer

	renderGameLeaderboardFn        func(wr io.Writer, params GameLeaderboardViewParam) error
	renderGameLeaderboardCalls     int
	renderGameLeaderboardLastParam GameLeaderboardViewParam
	renderGameLeaderboardWriter    io.Writer

	renderGameLoadingPageFn        func(wr io.Writer, params GameLoadingPageViewParam) error
	renderGameLoadingPageCalls     int
	renderGameLoadingPageLastParam GameLoadingPageViewParam
	renderGameLoadingPageWriter    io.Writer
}

func (m *MockView) renderErrorPage(wr io.Writer) error {
	m.mu.Lock()
	m.renderErrorPageCalls++
	m.renderErrorPageWriter = wr
	m.mu.Unlock()
	if m.renderErrorPageFn != nil {
		return m.renderErrorPageFn(wr)
	}
	return nil
}

func (m *MockView) renderIndexPage(wr io.Writer, params IndexPageViewParam) error {
	m.mu.Lock()
	m.renderIndexPageCalls++
	m.renderIndexPageLastParam = params
	m.renderIndexPageWriter = wr
	m.mu.Unlock()
	if m.renderIndexPageFn != nil {
		return m.renderIndexPageFn(wr, params)
	}
	return nil
}

func (m *MockView) renderGamePage(wr io.Writer, params GamePageViewParam) error {
	m.mu.Lock()
	m.renderGamePageCalls++
	m.renderGamePageLastParam = params
	m.renderGamePageWriter = wr
	m.mu.Unlock()
	if m.renderGamePageFn != nil {
		return m.renderGamePageFn(wr, params)
	}
	return nil
}

func (m *MockView) renderGameWord(wr io.Writer, params GameWordViewParam) error {
	m.mu.Lock()
	m.renderGameWordCalls++
	m.renderGameWordLastParam = params
	m.renderGameWordWriter = wr
	m.mu.Unlock()
	if m.renderGameWordFn != nil {
		return m.renderGameWordFn(wr, params)
	}
	return nil
}

func (m *MockView) renderGameMsg(wr io.Writer, params GameMsgViewParam) error {
	m.mu.Lock()
	m.renderGameMsgCalls++
	m.renderGameMsgLastParam = params
	m.renderGameMsgWriter = wr
	m.mu.Unlock()
	if m.renderGameMsgFn != nil {
		return m.renderGameMsgFn(wr, params)
	}
	return nil
}

func (m *MockView) renderGameLeaderboard(wr io.Writer, params GameLeaderboardViewParam) error {
	m.mu.Lock()
	m.renderGameLeaderboardCalls++
	m.renderGameLeaderboardLastParam = params
	m.renderGameLeaderboardWriter = wr
	m.mu.Unlock()
	if m.renderGameLeaderboardFn != nil {
		return m.renderGameLeaderboardFn(wr, params)
	}
	return nil
}

func (m *MockView) renderGameLoadingPage(wr io.Writer, params GameLoadingPageViewParam) error {
	m.mu.Lock()
	m.renderGameLoadingPageCalls++
	m.renderGameLoadingPageLastParam = params
	m.renderGameLoadingPageWriter = wr
	m.mu.Unlock()
	if m.renderGameLoadingPageFn != nil {
		return m.renderGameLoadingPageFn(wr, params)
	}
	return nil
}

// Compile-time guard.
var _ View = (*MockView)(nil)
