package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

// TestUnitOfWork exercises the real sqlite transaction wrapper: Commit must
// persist writes made through the UoW's GameRepository, Rollback must discard
// them. Uses the same single-connection in-memory DB as the other repository
// tests.
func TestUnitOfWork(t *testing.T) {
	t.Run("Commit persists the created game", func(t *testing.T) {
		db := newTestDB(t)
		factory := NewUnitOfWorkFactory(db)

		uow, err := factory.New(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		game, err := uow.GameRepository().Create(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if err := uow.Commit(); err != nil {
			t.Fatal(err)
		}

		// deferred Rollback after Commit must be a harmless no-op.
		if err := uow.Rollback(); !errors.Is(err, sql.ErrTxDone) {
			t.Errorf("expected sql.ErrTxDone rolling back after commit, got %v", err)
		}

		found, err := NewGameRepository(db).FindByID(context.Background(), game.ID)
		if err != nil {
			t.Fatalf("expected committed game to be visible, got %v", err)
		}
		if found.ID != game.ID {
			t.Errorf("expected game id %s but got %s", game.ID, found.ID)
		}
	})

	t.Run("Rollback discards the created game", func(t *testing.T) {
		db := newTestDB(t)
		factory := NewUnitOfWorkFactory(db)

		uow, err := factory.New(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		game, err := uow.GameRepository().Create(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if err := uow.Rollback(); err != nil {
			t.Fatal(err)
		}

		_, err = NewGameRepository(db).FindByID(context.Background(), game.ID)
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("expected sql.ErrNoRows after rollback, got %v", err)
		}
	})
}

// NOTE: "uncommitted writes are invisible outside the transaction" cannot be
// tested with the single-connection in-memory DB used here: the open
// transaction holds the only connection, so any outside read would block
// forever waiting for it.
