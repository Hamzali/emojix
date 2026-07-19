package usecase_test

import (
	"context"
	"emojix/model"
	"emojix/repository/repotest"
	"emojix/service"
	"emojix/service/servicetest"
	"emojix/usecase"
	"errors"
	"fmt"
	"testing"
	"time"
)

// assertPubNotCalled provides a race-free negative assertion for the
// production `go gameNotifier.Pub(...)` goroutine: it fails the test if a Pub
// invocation is observed on pubCh within a short bounded timeout, instead of
// sleeping and then reading a bool flag written from another goroutine.
func assertPubNotCalled(t *testing.T, pubCh <-chan struct{}) {
	t.Helper()
	select {
	case <-pubCh:
		t.Fatal("expected GameNotifier.Pub not to be called")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestJoinGame(t *testing.T) {

	t.Run("adds player", func(t *testing.T) {
		mur := &repotest.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{
					ID:       "new-player-id",
					Nickname: "NewPlayer",
				}, nil

			},
		}
		mgr := &repotest.MockGameRepository{
			GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
				assertCalledWith(t, "GameID", "some-game-id", id)

				return []model.Player{{ID: "other-player", Nickname: "OtherPlayer"}}, nil
			},
			AddPlayerMock: func(ctx context.Context, id, playerID string) error {
				assertCalledWith(t, "GameID", "some-game-id", id)
				assertCalledWith(t, "PlayerID", "new-player-id", playerID)

				return nil
			},
		}
		pubCh := make(chan int)
		mgns := &servicetest.MockGameNotifier{
			PubMock: func(gameID, userID string, notif service.GameNotification) {
				// Assert before signalling: the signal unblocks the test, so
				// asserting after it could race test completion (t.Error in a
				// finished test panics).
				assertCalledWith(t, "GameID", "some-game-id", gameID)
				assertCalledWith(t, "PlayerID", "new-player-id", userID)
				assertCalledWith(t, "NotifType", "join", notif.GetType())
				assertCalledWith(t, "NotifData", "new-player-id,NewPlayer", notif.GetData())

				pubCh <- 0
			},
		}

		emojiUsecase := usecase.NewEmojixUsecase(mur, mgr, nil, nil, mgns, &servicetest.MockGameLoop{}, service.NewRealClock())

		ctx := context.Background()
		err := emojiUsecase.JoinGame(ctx, "some-game-id", "new-player-id")
		if err != nil {
			t.Errorf("expected no error but got %v", err)
		}

		if mgr.AddPlayerCalled == false {
			t.Error("expected GameRepository.AddPlayer to be called")
		}

		<-pubCh
		if mgns.PubCalled == false {
			t.Error("expected GameNotifier.Pub to be called")
		}

	})

	t.Run("fails to add if player is already in game and active", func(t *testing.T) {
		mur := &repotest.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{
					ID:       "other-player-id",
					Nickname: "OtherPlayer",
				}, nil

			},
		}

		mgr := &repotest.MockGameRepository{
			GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
				return []model.Player{{ID: "other-player-id", Nickname: "OtherPlayer", State: model.ActivePlayerState}}, nil
			},
			AddPlayerMock: func(ctx context.Context, id, playerID string) error {
				return nil
			},
		}
		pubCh := make(chan struct{}, 1)
		mgns := &servicetest.MockGameNotifier{
			PubMock: func(gameID, userID string, notif service.GameNotification) {
				pubCh <- struct{}{}
			},
		}

		emojiUsecase := usecase.NewEmojixUsecase(mur, mgr, nil, nil, mgns, &servicetest.MockGameLoop{}, service.NewRealClock())

		ctx := context.Background()
		err := emojiUsecase.JoinGame(ctx, "some-game-id", "other-player-id")
		if !errors.Is(usecase.ErrJoinGameUserAlreadyJoined, err) {
			t.Errorf("expected already joined error but got %v", err)
		}

		if mgr.AddPlayerCalled == true {
			t.Error("expected GameRepository.AddPlayer not to be called")
		}

		assertPubNotCalled(t, pubCh)
	})

	t.Run("reactivates user joined and kicked before", func(t *testing.T) {
		mur := &repotest.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{
					ID:       "kicked-player-id",
					Nickname: "KickedPlayer",
				}, nil

			},
		}

		mgr := &repotest.MockGameRepository{
			GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
				return []model.Player{
					{ID: "kicked-player-id", Nickname: "KickedPlayer", State: model.InactivePlayerState},
					{ID: "other-player-id", Nickname: "OtherPlayer", State: model.ActivePlayerState},
				}, nil
			},
			AddPlayerMock: func(ctx context.Context, id, playerID string) error {
				return nil
			},
			SetPlayerStateMock: func(ctx context.Context, gameID, userID, state model.PlayerState) error {
				return nil
			},
		}

		pubCh := make(chan struct{}, 1)
		mgns := &servicetest.MockGameNotifier{
			PubMock: func(gameID, userID string, notif service.GameNotification) {
				pubCh <- struct{}{}
			},
		}
		emojiUsecase := usecase.NewEmojixUsecase(mur, mgr, nil, nil, mgns, &servicetest.MockGameLoop{}, service.NewRealClock())

		ctx := context.Background()
		err := emojiUsecase.JoinGame(ctx, "some-game-id", "kicked-player-id")
		if err != nil {
			t.Errorf("expected no error but got %v", err)
		}

		if mgr.AddPlayerCalled == true {
			t.Error("expected GameRepository.AddPlayer not to be called")
		}

		// Wait for the go Pub(...) goroutine to finish before reading PubCalled.
		select {
		case <-pubCh:
		case <-time.After(time.Second):
			t.Fatal("expected GameNotifier.Pub to be called")
		}

		if mgns.PubCalled == false {
			t.Error("expected GameNotifier.Pub to be called")
		}

		if mgr.SetPlayerStateCalled == false {
			t.Error("expected GameRepository.SetPlayerState to be called")
		}
	})

	t.Run("fails to add if room is full", func(t *testing.T) {
		mur := &repotest.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{
					ID:       "new-player-id",
					Nickname: "NewPlayer",
				}, nil

			},
		}

		addPlayerCalled := false
		mgr := &repotest.MockGameRepository{
			GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
				players := []model.Player{}
				for i := range 10 {
					players = append(players, model.Player{
						ID:       fmt.Sprintf("player-%d", i),
						Nickname: fmt.Sprintf("Player%d", i),
						State:    model.ActivePlayerState,
					})
				}
				return players, nil
			},
			AddPlayerMock: func(ctx context.Context, id, playerID string) error {
				addPlayerCalled = true
				return nil
			},
		}
		pubCh := make(chan struct{}, 1)
		mgns := &servicetest.MockGameNotifier{
			PubMock: func(gameID, userID string, notif service.GameNotification) {
				pubCh <- struct{}{}
			},
		}

		emojiUsecase := usecase.NewEmojixUsecase(mur, mgr, nil, nil, mgns, &servicetest.MockGameLoop{}, service.NewRealClock())

		ctx := context.Background()
		err := emojiUsecase.JoinGame(ctx, "some-game-id", "new-player-id")
		if !errors.Is(usecase.ErrJoinGameRoomFull, err) {
			t.Errorf("expected room full error but got %v", err)
		}

		if addPlayerCalled == true {
			t.Error("expected GameRepository.AddPlayer not to be called")
		}

		assertPubNotCalled(t, pubCh)
	})
}
