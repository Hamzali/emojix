package service_test

import (
	"context"
	"testing"
	"time"

	"emojix/service"
)

func TestGameLoop_Timeout(t *testing.T) {
	fc := service.NewFakeClock()
	calls := make(chan string, 1)

	gl := service.NewGameLoop(fc)
	gl.SetOnTurnEndHandler(func(ctx context.Context, gameID string) {
		calls <- gameID
	})

	gl.Start(context.Background(), "g1", 60*time.Second)
	fc.Advance(61 * time.Second)

	select {
	case id := <-calls:
		if id != "g1" {
			t.Fatalf("expected g1, got %s", id)
		}
	case <-time.After(time.Second):
		t.Fatal("OnTurnEnd not called after timeout")
	}
}

func TestGameLoop_EndGameTurn(t *testing.T) {
	fc := service.NewFakeClock()
	calls := make(chan string, 1)

	gl := service.NewGameLoop(fc)
	gl.SetOnTurnEndHandler(func(ctx context.Context, gameID string) {
		calls <- gameID
	})

	gl.Start(context.Background(), "g1", 60*time.Second)
	gl.EndGameTurn("g1") // signal early

	select {
	case id := <-calls:
		if id != "g1" {
			t.Fatalf("expected g1, got %s", id)
		}
	case <-time.After(time.Second):
		t.Fatal("OnTurnEnd not called after EndGameTurn")
	}
}

func TestGameLoop_TimeoutBeforeAllGuessed(t *testing.T) {
	fc := service.NewFakeClock()
	calls := make(chan string, 1)

	gl := service.NewGameLoop(fc)
	gl.SetOnTurnEndHandler(func(ctx context.Context, gameID string) {
		calls <- gameID
	})

	gl.Start(context.Background(), "g1", 60*time.Second)
	fc.Advance(30 * time.Second) // advance partway
	gl.EndGameTurn("g1")         // all guessed
	fc.Advance(31 * time.Second) // past original timeout

	select {
	case id := <-calls:
		if id != "g1" {
			t.Fatalf("expected g1, got %s", id)
		}
	case <-time.After(time.Second):
		t.Fatal("OnTurnEnd not called")
	}

	// Should be called exactly once
	select {
	case <-calls:
		t.Fatal("OnTurnEnd called twice")
	default:
	}
}

func TestGameLoop_DoubleEndGameTurn(t *testing.T) {
	fc := service.NewFakeClock()
	callCount := 0

	gl := service.NewGameLoop(fc)
	gl.SetOnTurnEndHandler(func(ctx context.Context, gameID string) {
		callCount++
	})

	gl.Start(context.Background(), "g1", 60*time.Second)
	gl.EndGameTurn("g1")
	gl.EndGameTurn("g1") // second send should be dropped (buffered ch + default)

	// Wait for goroutine to process
	time.Sleep(10 * time.Millisecond)

	if callCount != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}
}

func TestGameLoop_StartDuplicateNoOp(t *testing.T) {
	fc := service.NewFakeClock()
	calls := make(chan string, 1)

	gl := service.NewGameLoop(fc)
	gl.SetOnTurnEndHandler(func(ctx context.Context, gameID string) {
		calls <- gameID
	})

	gl.Start(context.Background(), "g1", 60*time.Second)
	gl.Start(context.Background(), "g1", 30*time.Second) // should be no-op, first Start still active

	fc.Advance(31 * time.Second) // past 30s, but NOT past 60s
	// OnTurnEnd should NOT fire yet

	select {
	case <-calls:
		t.Fatal("OnTurnEnd called at 30s, but Start was a no-op")
	default:
	}

	fc.Advance(30 * time.Second) // now past 60s total

	select {
	case id := <-calls:
		if id != "g1" {
			t.Fatalf("expected g1, got %s", id)
		}
	case <-time.After(time.Second):
		t.Fatal("OnTurnEnd not called after 60s")
	}
}

func TestGameLoop_AllGuessedAfterStop(t *testing.T) {
	fc := service.NewFakeClock()
	calls := make(chan string, 1)

	gl := service.NewGameLoop(fc)
	gl.SetOnTurnEndHandler(func(ctx context.Context, gameID string) {
		calls <- gameID
	})

	gl.Start(context.Background(), "g1", 60*time.Second)
	gl.StopGame("g1")
	gl.EndGameTurn("g1") // no-op, should not trigger callback

	select {
	case <-calls:
		t.Fatal("OnTurnEnd called after StopGame")
	case <-time.After(10 * time.Millisecond):
	}
}

func TestGameLoop_AllGuessedWrongGameID(t *testing.T) {
	fc := service.NewFakeClock()
	calls := make(chan string, 1)

	gl := service.NewGameLoop(fc)
	gl.SetOnTurnEndHandler(func(ctx context.Context, gameID string) {
		calls <- gameID
	})

	gl.Start(context.Background(), "g1", 60*time.Second)
	gl.EndGameTurn("nonexistent") // wrong gameID, no-op

	fc.Advance(61 * time.Second) // should still fire from timeout

	select {
	case id := <-calls:
		if id != "g1" {
			t.Fatalf("expected g1, got %s", id)
		}
	case <-time.After(time.Second):
		t.Fatal("OnTurnEnd not called")
	}
}

func TestGameLoop_StopAll(t *testing.T) {
	fc := service.NewFakeClock()
	calls := make(chan string, 2)

	gl := service.NewGameLoop(fc)
	gl.SetOnTurnEndHandler(func(ctx context.Context, gameID string) {
		calls <- gameID
	})

	gl.Start(context.Background(), "g1", 60*time.Second)
	gl.Start(context.Background(), "g2", 60*time.Second)
	gl.Stop()

	// No callbacks should fire after Stop
	select {
	case <-calls:
		t.Fatal("OnTurnEnd called after Stop")
	case <-time.After(10 * time.Millisecond):
	}
}

func TestGameLoop_MultipleTurns(t *testing.T) {
	// One Start call, multiple turn cycles via EndGameTurn and timeout.
	fc := service.NewFakeClock()
	calls := make(chan string, 3)

	gl := service.NewGameLoop(fc)
	gl.SetOnTurnEndHandler(func(ctx context.Context, gameID string) {
		calls <- gameID
	})

	gl.Start(context.Background(), "g1", 60*time.Second)

	// Turn 1: end early
	gl.EndGameTurn("g1")
	<-calls

	// Turn 2: let it timeout
	fc.Advance(61 * time.Second)
	<-calls

	// Turn 3: end early again
	gl.EndGameTurn("g1")
	<-calls

	// No extra calls
	select {
	case <-calls:
		t.Fatal("too many OnTurnEnd calls")
	default:
	}
}
