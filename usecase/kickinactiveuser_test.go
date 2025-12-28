package usecase_test

import (
	"context"
	"emojix/model"
	"emojix/repository"
	"emojix/service"
	"emojix/usecase"
	"testing"
	"time"
)

func TestKickInactiveUser(t *testing.T) {

	subsMock := func(gameID string) []string {
		return []string{
			"user-1",
			"user-2",
			"user-3",
		}
	}

	t.Run("kicks inactive user", func(t *testing.T) {
		pubCh := make(chan int)
		mgn := &service.MockGameNotifier{
			SubsMock: subsMock,
			PubMock: func(gameID, userID string, notif service.GameNotification) {
				err := assertCalledWithError("GameID", "game-id", gameID)
				if err != nil {
					t.Error(err)
				}
				err = assertCalledWithError("UserID", "user-4", userID)
				if err != nil {
					t.Error(err)
				}

				notifType := notif.GetType()
				if notifType != "left" {
					t.Error(err)
				}

				notifData := notif.GetData()
				if notifData != "user-4" {
					t.Error(err)
				}

				pubCh <- 1
				close(pubCh)
			},
		}

		mgr := &repository.MockGameRepository{
			SetPlayerStateMock: func(ctx context.Context, gameID, userID, state model.PlayerState) error {
				err := assertCalledWithError("GameID", "game-id", gameID)
				if err != nil {
					t.Error(err)
				}
				err = assertCalledWithError("UserID", "user-4", userID)
				if err != nil {
					t.Error(err)
				}
				err = assertCalledWithError("State", "inactive", state)
				if err != nil {
					t.Error(err)
				}
				return nil
			},
		}
		emojixUsecase := usecase.NewEmojixUsecase(
			nil,
			mgr,
			nil,
			nil,
			mgn,
		)

		err := emojixUsecase.KickInactiveUser(context.Background(), "game-id", "user-4")

		if err != nil {
			t.Errorf("expected to not error but got %v", err)
		}

		if mgr.SetPlayerStateCalled != true {
			t.Error("expected GameRepository.SetPlayerState to be called")
		}

		select {
		case <-pubCh:
		case <-time.After(time.Second * 1):
		}

		if mgn.PubCalled != true {
			t.Error("expected NotifierService.Pub to be called")
		}
	})

	t.Run("keeps user if its active", func(t *testing.T) {
		mgn := &service.MockGameNotifier{
			SubsMock: subsMock,
			PubMock: func(gameID, userID string, notif service.GameNotification) {
			},
		}

		mgr := &repository.MockGameRepository{
			SetPlayerStateMock: func(ctx context.Context, gameID, userID, state model.PlayerState) error {
				return nil
			},
		}
		emojixUsecase := usecase.NewEmojixUsecase(
			nil,
			mgr,
			nil,
			nil,
			mgn,
		)

		err := emojixUsecase.KickInactiveUser(context.Background(), "game-id", "user-1")

		if err != nil {
			t.Errorf("expected to not error but got %v", err)
		}

		if mgr.SetPlayerStateCalled != false {
			t.Error("expected GameRepository.SetPlayerState not to be called")
		}

		if mgn.PubCalled != false {
			t.Error("expected NotifierService.Pub not to be called")
		}
	})

}
