package service

import (
	"sync"
	"time"
)

// FakeClock is a deterministic clock for testing GameLoop.
// Use Advance() to move time forward and fire timers.
type FakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
}

type fakeTimer struct {
	clock    *FakeClock
	firingAt time.Time
	ch       chan time.Time
}

func NewFakeClock() *FakeClock {
	return &FakeClock{now: time.Now()}
}

func (fc *FakeClock) After(d time.Duration) <-chan time.Time {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	t := &fakeTimer{
		clock:    fc,
		firingAt: fc.now.Add(d),
		ch:       make(chan time.Time, 1),
	}
	fc.timers = append(fc.timers, t)
	return t.ch
}

// Advance moves the clock forward, firing any timers that elapse.
func (fc *FakeClock) Advance(d time.Duration) {
	fc.mu.Lock()
	fc.now = fc.now.Add(d)

	var fired []*fakeTimer
	remaining := make([]*fakeTimer, 0, len(fc.timers))
	for _, t := range fc.timers {
		if !fc.now.Before(t.firingAt) {
			fired = append(fired, t)
		} else {
			remaining = append(remaining, t)
		}
	}
	fc.timers = remaining
	fc.mu.Unlock()

	// Send outside lock to avoid deadlock
	for _, t := range fired {
		select {
		case t.ch <- t.firingAt:
		default:
		}
	}
}

// Now returns the current fake time.
func (fc *FakeClock) Now() time.Time {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.now
}
