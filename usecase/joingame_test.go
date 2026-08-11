package usecase_test

import (
	"context"
	"emojix/model"
	"emojix/repository"
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

	t.Run("second player starts the game loop", func(t *testing.T) {
		mur := &repotest.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{ID: "new-player-id", Nickname: "NewPlayer"}, nil
			},
		}
		getPlayersCalls := 0
		mgr := &repotest.MockGameRepository{
			GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
				getPlayersCalls++
				if getPlayersCalls == 1 {
					// Pre-join roster (capacity check).
					return []model.Player{
						{ID: "host-id", Nickname: "Host", State: model.ActivePlayerState},
					}, nil
				}
				// Post-join roster (tryStartGame).
				return []model.Player{
					{ID: "host-id", Nickname: "Host", State: model.ActivePlayerState},
					{ID: "new-player-id", Nickname: "NewPlayer", State: model.ActivePlayerState},
				}, nil
			},
			AddPlayerMock: func(ctx context.Context, id, playerID string) error { return nil },
			FindByIDMock: func(ctx context.Context, id string) (model.Game, error) {
				return model.Game{ID: id, ListID: "list-1"}, nil
			},
			CountTurnsMock: func(ctx context.Context, gameID string) (int, error) { return 0, nil },
			AddTurnMock: func(ctx context.Context, params repository.AddTurnParams) (model.GameTurn, error) {
				return model.GameTurn{ID: "turn-1", GameID: params.GameID, TellerID: params.TellerID}, nil
			},
		}
		mwr := &repotest.MockWordRepository{
			GetUnusedByListMock: func(ctx context.Context, listID, gameID string) ([]model.Word, error) {
				return []model.Word{{ID: "w1", Word: "Alpha"}, {ID: "w2", Word: "Beta"}, {ID: "w3", Word: "Gamma"}}, nil
			},
		}
		startCh := make(chan struct{}, 1)
		gl := &servicetest.MockGameLoop{
			StartMock: func(ctx context.Context, gameID string, turnDuration, pickDuration time.Duration) {
				startCh <- struct{}{}
			},
		}
		pubAllCh := make(chan struct{}, 1)
		pubCh := make(chan struct{}, 1)
		mgns := &servicetest.MockGameNotifier{
			PubMock: func(gameID, userID string, notif service.GameNotification) {
				pubCh <- struct{}{}
			},
			PubAllMock: func(gameID string, notif service.GameNotification) {
				if notif.GetType() != "newturn" {
					t.Errorf("PubAll type: got %q, want newturn", notif.GetType())
				}
				pubAllCh <- struct{}{}
			},
		}
		emojiUsecase := usecase.NewEmojixUsecase(mur, mgr, mwr, nil, mgns, gl, service.NewRealClock())
		err := emojiUsecase.JoinGame(context.Background(), "some-game-id", "new-player-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !mgr.AddTurnCalled {
			t.Error("expected AddTurn when second player joins")
		}
		select {
		case <-startCh:
		case <-time.After(time.Second):
			t.Fatal("expected gameLoop.Start")
		}
		select {
		case <-pubAllCh:
		case <-time.After(time.Second):
			t.Fatal("expected PubAll newturn")
		}
		select {
		case <-pubCh:
		case <-time.After(time.Second):
			t.Fatal("expected join Pub")
		}
	})

}
