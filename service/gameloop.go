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
	Now() time.Time
}

// RealClock is the default clock implementation using real time.
type RealClock struct{}

func (RealClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

func (RealClock) Now() time.Time {
	return time.Now()
}

type GameLoop interface {
	// Start begins the game loop for a game. Called once per game.
	// The first turn timer does not start until BeginTurn is called.
	// Logs a warning and returns early if gameID already has an active loop.
	Start(ctx context.Context, gameID string, duration time.Duration)

	// BeginTurn starts the turn timer after the teller has picked a word.
	// Blocks until the timer is armed so EndGameTurn/clock.Advance are safe after return.
	BeginTurn(gameID string)

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
	beginChs  map[string]chan struct{} // gameID -> begin-turn signal
	endChs    map[string]chan struct{} // gameID -> end-turn signal
	armedChs  map[string]chan struct{} // gameID -> closed when turn timer is armed
	cancels   map[string]context.CancelFunc
	clock     Clock
	onTurnEnd OnTurnEndHandler
}

// NewRealClock creates a new RealClock that uses real time.
func NewRealClock() Clock {
	return RealClock{}
}

func NewGameLoop(clock Clock) GameLoop {
	return &gameLoop{
		beginChs: make(map[string]chan struct{}),
		endChs:   make(map[string]chan struct{}),
		armedChs: make(map[string]chan struct{}),
		cancels:  make(map[string]context.CancelFunc),
		clock:    clock,
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

	// Wait for BeginTurn before the first timer. End channel is registered only
	// while a turn is active so EndGameTurn during pick phase is a no-op.
	beginCh := make(chan struct{}, 1)
	l.beginChs[gameID] = beginCh
	l.mu.Unlock()

	go l.run(ctx, gameID, duration, beginCh)
}

func (l *gameLoop) BeginTurn(gameID string) {
	l.mu.Lock()
	beginCh, ok := l.beginChs[gameID]
	if !ok {
		l.mu.Unlock()
		return
	}
	armed := make(chan struct{})
	l.armedChs[gameID] = armed
	l.mu.Unlock()

	select {
	case beginCh <- struct{}{}:
	default:
		// Already signaled; still wait for arm in case a prior begin is in flight.
	}

	// Block until run() has registered endCh + timer so callers can Advance/End safely.
	<-armed
}

func (l *gameLoop) EndGameTurn(gameID string) {
	l.mu.Lock()
	ch, ok := l.endChs[gameID]
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
	// Unblock any BeginTurn waiter.
	if armed, ok := l.armedChs[gameID]; ok {
		select {
		case <-armed:
		default:
			close(armed)
		}
		delete(l.armedChs, gameID)
	}
	delete(l.beginChs, gameID)
	delete(l.endChs, gameID)
}

func (l *gameLoop) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, cancel := range l.cancels {
		cancel()
	}
	for _, armed := range l.armedChs {
		select {
		case <-armed:
		default:
			close(armed)
		}
	}
	l.beginChs = make(map[string]chan struct{})
	l.endChs = make(map[string]chan struct{})
	l.armedChs = make(map[string]chan struct{})
	l.cancels = make(map[string]context.CancelFunc)
}

func (l *gameLoop) run(ctx context.Context, gameID string, duration time.Duration, beginCh chan struct{}) {
	for {
		// Wait for teller pick before starting the turn timer.
		select {
		case <-ctx.Done():
			return
		case <-beginCh:
		}

		endCh := make(chan struct{}, 1)
		// Register timer before signaling armed so Advance after BeginTurn is reliable.
		timerCh := l.clock.After(duration)

		l.mu.Lock()
		l.endChs[gameID] = endCh
		// Refresh begin channel for the next pick phase after this turn ends.
		beginCh = make(chan struct{}, 1)
		l.beginChs[gameID] = beginCh
		armed := l.armedChs[gameID]
		delete(l.armedChs, gameID)
		l.mu.Unlock()

		if armed != nil {
			close(armed)
		}

		select {
		case <-ctx.Done():
			return
		case <-endCh:
		case <-timerCh:
		}

		l.mu.Lock()
		delete(l.endChs, gameID)
		l.mu.Unlock()

		if l.onTurnEnd != nil {
			l.onTurnEnd(context.Background(), gameID)
		}
	}
}
