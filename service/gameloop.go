package service

import (
	"context"
	"log"
	"sync"
	"time"
)

// OnTurnEndHandler is the callback type for when a turn ends.
// It runs synchronously in the GameLoop's goroutine.
type OnTurnEndHandler func(ctx context.Context, gameID string)

// Clock interface for time-based operations. Allows deterministic testing.
type Clock interface {
	After(d time.Duration) <-chan time.Time
}



// RealClock is the default clock implementation using real time.
type RealClock struct{}

func (RealClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

type GameLoop interface {
	// Start begins the game loop for a game. Called once per game.
	// Logs a warning and returns early if gameID already has an active loop.
	Start(ctx context.Context, gameID string, duration time.Duration)

	// EndGameTurn signals that all players have guessed the current turn.
	// Thread-safe, non-blocking.
	EndGameTurn(gameID string)

	// SetOnTurnEndHandler sets the handler called when a turn ends.
	// Must be called before Start.
	SetOnTurnEndHandler(handler OnTurnEndHandler)

	// StopGame cancels a specific game's loop (e.g., game ended, all players left).
	StopGame(gameID string)

	// Stop cancels ALL game loops (e.g., server shutdown).
	Stop()
}

type gameLoop struct {
	mu        sync.Mutex
	chs       map[string]chan struct{}       // gameID -> signal channel
	cancels   map[string]context.CancelFunc  // gameID -> cancel func
	clock     Clock
	onTurnEnd OnTurnEndHandler
}

// NewRealClock creates a new RealClock that uses real time.
func NewRealClock() Clock {
	return RealClock{}
}

func NewGameLoop(clock Clock) GameLoop {
	return &gameLoop{
		chs:     make(map[string]chan struct{}),
		cancels: make(map[string]context.CancelFunc),
		clock:   clock,
	}
}

func (l *gameLoop) SetOnTurnEndHandler(handler OnTurnEndHandler) {
	l.onTurnEnd = handler
}

func (l *gameLoop) Start(ctx context.Context, gameID string, duration time.Duration) {
	l.mu.Lock()
	if _, ok := l.cancels[gameID]; ok {
		l.mu.Unlock()
		log.Printf("WARNING: GameLoop.Start called twice for game %s", gameID)
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	l.cancels[gameID] = cancel

	// Pre-create channel and timer for the first iteration.
	// This ensures the signal channel is immediately available for EndGameTurn
	// and the timer is created before any clock.Advance() call.
	firstCh := make(chan struct{}, 1)
	l.chs[gameID] = firstCh
	firstTimerCh := l.clock.After(duration)
	l.mu.Unlock()

	go l.run(ctx, gameID, duration, firstCh, firstTimerCh)
}

func (l *gameLoop) EndGameTurn(gameID string) {
	l.mu.Lock()
	ch, ok := l.chs[gameID]
	l.mu.Unlock()

	if !ok {
		return // no active turn, or loop not started
	}

	// Non-blocking send: if channel is full (race between multiple callers), drop
	select {
	case ch <- struct{}{}:
	default:
	}
}

func (l *gameLoop) StopGame(gameID string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if cancel, ok := l.cancels[gameID]; ok {
		cancel()
		delete(l.cancels, gameID)
	}
	delete(l.chs, gameID)
}

func (l *gameLoop) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, cancel := range l.cancels {
		cancel()
	}
	l.chs = make(map[string]chan struct{})
	l.cancels = make(map[string]context.CancelFunc)
}

func (l *gameLoop) run(ctx context.Context, gameID string, duration time.Duration, ch chan struct{}, timerCh <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ch:
		case <-timerCh:
		}

		l.mu.Lock()
		delete(l.chs, gameID)
		l.mu.Unlock()

		if l.onTurnEnd != nil {
			l.onTurnEnd(context.Background(), gameID)
		}

		// Prepare for next iteration: create new channel and timer
		ch = make(chan struct{}, 1)

		l.mu.Lock()
		l.chs[gameID] = ch
		l.mu.Unlock()

		timerCh = l.clock.After(duration)
	}
}
