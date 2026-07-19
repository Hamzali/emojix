package usecase_test

import (
	"context"
	"emojix/model"
	"emojix/repository/repotest"
	"emojix/service"
	"emojix/service/servicetest"
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
		mgn := &servicetest.MockGameNotifier{
			SubsMock: subsMock,
			PubMock: func(gameID, userID string, notif service.GameNotification) {
				assertCalledWith(t, "GameID", "game-id", gameID)
				assertCalledWith(t, "UserID", "user-4", userID)
				assertCalledWith(t, "NotifType", "left", notif.GetType())
				assertCalledWith(t, "NotifData", "user-4", notif.GetData())

				pubCh <- 1
				close(pubCh)
			},
		}

		mgr := &repotest.MockGameRepository{
			SetPlayerStateMock: func(ctx context.Context, gameID, userID, state model.PlayerState) error {
				assertCalledWith(t, "GameID", "game-id", gameID)
				assertCalledWith(t, "UserID", "user-4", userID)
				assertCalledWith(t, "State", "inactive", state)
				return nil
			},
		}
		emojixUsecase := usecase.NewEmojixUsecase(
			nil,
			mgr,
			nil,
			nil,
			mgn,
			&servicetest.MockGameLoop{},
			service.NewRealClock(),
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
		pubCh := make(chan struct{}, 1)
		mgn := &servicetest.MockGameNotifier{
			SubsMock: subsMock,
			PubMock: func(gameID, userID string, notif service.GameNotification) {
				pubCh <- struct{}{}
			},
		}

		mgr := &repotest.MockGameRepository{
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
			&servicetest.MockGameLoop{},
			service.NewRealClock(),
		)

		err := emojixUsecase.KickInactiveUser(context.Background(), "game-id", "user-1")

		if err != nil {
			t.Errorf("expected to not error but got %v", err)
		}

		if mgr.SetPlayerStateCalled != false {
			t.Error("expected GameRepository.SetPlayerState not to be called")
		}

		assertPubNotCalled(t, pubCh)
	})

}
