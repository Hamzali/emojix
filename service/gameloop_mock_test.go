package service

import (
	"context"
	"testing"
)

// TestMockGameLoop_CapturesOnTurnEndHandler ensures NewEmojixUsecase (or any
// caller) does not have its OnTurnEndHandler swallowed by the mock: the handler
// passed to SetOnTurnEndHandler is retained and invocable via FireOnTurnEnd.
func TestMockGameLoop_CapturesOnTurnEndHandler(t *testing.T) {
	mock := &MockGameLoop{}

	done := make(chan string, 1)
	handler := OnTurnEndHandler(func(_ context.Context, gameID string) {
		done <- gameID
	})

	mock.SetOnTurnEndHandler(handler)

	if !mock.SetOnTurnEndHandlerCalled {
		t.Fatal("SetOnTurnEndHandlerCalled not set")
	}
	if mock.OnTurnEndHandler == nil {
		t.Fatal("OnTurnEndHandler was not captured")
	}

	ctx := context.Background()
	mock.FireOnTurnEnd(ctx, "g1")

	select {
	case got := <-done:
		if got != "g1" {
			t.Fatalf("got gameID %q, want %q", got, "g1")
		}
	default:
		t.Fatal("FireOnTurnEnd did not invoke the captured handler")
	}
}