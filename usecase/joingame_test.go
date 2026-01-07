package usecase_test

import (
	"context"
	"emojix/model"
	"emojix/repository"
	"emojix/service"
	"emojix/usecase"
	"errors"
	"fmt"
	"testing"
)

func TestJoinGame(t *testing.T) {

	t.Run("adds player", func(t *testing.T) {
		mur := &repository.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{
					ID:       "new-player-id",
					Nickname: "NewPlayer",
				}, nil

			},
		}
		mgr := &repository.MockGameRepository{
			GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
				err := assertCalledWithError("GameID", "some-game-id", id)
				if err != nil {
					t.Error(err)
				}

				return []model.Player{{ID: "other-player", Nickname: "OtherPlayer"}}, nil
			},
			AddPlayerMock: func(ctx context.Context, id, playerID string) error {
				err := assertCalledWithError("GameID", "some-game-id", id)
				if err != nil {
					t.Error(err)
				}

				err = assertCalledWithError("PlayerID", "new-player-id", playerID)
				if err != nil {
					t.Error(err)
				}

				return nil
			},
		}
		pubCh := make(chan int)
		mgns := &service.MockGameNotifier{
			PubMock: func(gameID, userID string, notif service.GameNotification) {
				pubCh <- 0
				err := assertCalledWithError("GameID", "some-game-id", gameID)
				if err != nil {
					t.Error(err)
				}

				err = assertCalledWithError("PlayerID", "new-player-id", userID)
				if err != nil {
					t.Error(err)
				}

				err = assertCalledWithError("NotifType", "join", notif.GetType())
				if err != nil {
					t.Error(err)
				}
				err = assertCalledWithError("NotifData", "new-player-id,NewPlayer", notif.GetData())
				if err != nil {
					t.Error(err)
				}
			},
		}

		emojiUsecase := usecase.NewEmojixUsecase(mur, mgr, nil, nil, mgns)

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
		mur := &repository.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{
					ID:       "other-player-id",
					Nickname: "OtherPlayer",
				}, nil

			},
		}

		mgr := &repository.MockGameRepository{
			GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
				return []model.Player{{ID: "other-player-id", Nickname: "OtherPlayer", State: model.ActivePlayerState}}, nil
			},
			AddPlayerMock: func(ctx context.Context, id, playerID string) error {
				return nil
			},
		}
		mgns := &service.MockGameNotifier{
			PubMock: func(gameID, userID string, notif service.GameNotification) {},
		}

		emojiUsecase := usecase.NewEmojixUsecase(mur, mgr, nil, nil, mgns)

		ctx := context.Background()
		err := emojiUsecase.JoinGame(ctx, "some-game-id", "other-player-id")
		if !errors.Is(usecase.ErrJoinGameUserAlreadyJoined, err) {
			t.Errorf("expected already joined error but got %v", err)
		}

		if mgr.AddPlayerCalled == true {
			t.Error("expected GameRepository.AddPlayer not to be called")
		}

		if mgns.PubCalled == true {
			t.Error("expected GameNotifier.Pub not to be called")
		}
	})

	t.Run("reactivates user joined and kicked before", func(t *testing.T) {
		mur := &repository.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{
					ID:       "kicked-player-id",
					Nickname: "KickedPlayer",
				}, nil

			},
		}

		mgr := &repository.MockGameRepository{
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

		mgns := &service.MockGameNotifier{
			PubMock: func(gameID, userID string, notif service.GameNotification) {},
		}
		emojiUsecase := usecase.NewEmojixUsecase(mur, mgr, nil, nil, mgns)

		ctx := context.Background()
		err := emojiUsecase.JoinGame(ctx, "some-game-id", "kicked-player-id")
		if err != nil {
			t.Errorf("expected no error but got %v", err)
		}

		if mgr.AddPlayerCalled == true {
			t.Error("expected GameRepository.AddPlayer not to be called")
		}

		if mgns.PubCalled == true {
			t.Error("expected GameNotifier.Pub to be called")
		}

		if mgr.SetPlayerStateCalled == false {
			t.Error("expected GameRepository.SetPlayerState to be called")
		}
	})

	t.Run("fails to add if room is full", func(t *testing.T) {
		mur := &repository.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{
					ID:       "new-player-id",
					Nickname: "NewPlayer",
				}, nil

			},
		}

		addPlayerCalled := false
		mgr := &repository.MockGameRepository{
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
		mgns := &service.MockGameNotifier{
			PubMock: func(gameID, userID string, notif service.GameNotification) {},
		}

		emojiUsecase := usecase.NewEmojixUsecase(mur, mgr, nil, nil, mgns)

		ctx := context.Background()
		err := emojiUsecase.JoinGame(ctx, "some-game-id", "new-player-id")
		if !errors.Is(usecase.ErrJoinGameRoomFull, err) {
			t.Errorf("expected room full error but got %v", err)
		}

		if addPlayerCalled == true {
			t.Error("expected GameRepository.AddPlayer not to be called")
		}

		if mgns.PubCalled == true {
			t.Error("expected GameNotifier.Pub not to be called")
		}
	})
}
