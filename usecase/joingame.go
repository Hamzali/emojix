package usecase

import (
	"context"
	"emojix/model"
	"errors"
	"fmt"
)

var maxRoomCapacity = 10

var ErrJoinGameUserAlreadyJoined = errors.New("already joined")
var ErrJoinGameRoomFull = errors.New("room is full")

type GameJoinNotification struct {
	Nickname string
	PlayerID string
}

func (gmn *GameJoinNotification) GetType() string {
	return "join"
}

func (gmn *GameJoinNotification) GetData() string {
	return fmt.Sprintf("%s,%s", gmn.PlayerID, gmn.Nickname)
}

func (gmn *GameJoinNotification) ParseData(data string) error {
	return nil
}

func (e *emojixUsecase) JoinGame(ctx context.Context, gameID string, userID string) error {
	// TODO: in the future there can be multiple users joined the game but only 10 of them can be active at the same time
	// this repository call only get full list of players who joined the game, after addign activity logic with realtime features
	// update this call as well
	player, err := e.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	players, err := e.gameRepo.GetPlayers(ctx, gameID)
	if err != nil {
		return err
	}

	activePlayers := []model.Player{}
	prevInactiveUser := false
	for _, p := range players {
		if p.State == model.ActivePlayerState {
			activePlayers = append(activePlayers, p)
		}

		if p.ID == player.ID && p.State == model.InactivePlayerState {
			prevInactiveUser = true
		}
		if p.ID == player.ID && p.State == model.ActivePlayerState {
			return ErrJoinGameUserAlreadyJoined
		}
	}

	if len(activePlayers) >= maxRoomCapacity {
		return ErrJoinGameRoomFull
	}

	if prevInactiveUser {
		err = e.gameRepo.SetPlayerState(ctx, gameID, player.ID, model.ActivePlayerState)
	} else {
		err = e.gameRepo.AddPlayer(ctx, gameID, player.ID)
	}

	if err != nil {
		return err
	}

	go e.gameNotifier.Pub(gameID, player.ID, &GameJoinNotification{
		Nickname: player.Nickname,
		PlayerID: player.ID,
	})

	return nil
}
