package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
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

	for _, p := range players {
		if p.ID == player.ID {
			return ErrJoinGameUserAlreadyJoined
		}
	}

	if len(players) >= maxRoomCapacity {
		return ErrJoinGameRoomFull
	}

	err = e.gameRepo.AddPlayer(ctx, gameID, player.ID)
	if err != nil {
		return err
	}

	log.Println("added player")

	e.gameNotifier.Pub(gameID, player.ID, &GameJoinNotification{
		Nickname: player.Nickname,
		PlayerID: player.ID,
	})

	log.Println("why are we waiting")

	return nil
}
