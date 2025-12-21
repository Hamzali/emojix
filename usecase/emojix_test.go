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

func assertValueErrorMsg(fieldName string, expectedValue any, testValue any) string {
	return fmt.Sprintf("expected to have %s '%v' but got '%v'", fieldName, expectedValue, testValue)
}
func assertValueError(field string, expectedValue any, testValue any) error {
	if expectedValue == testValue {
		return nil
	}
	msg := assertValueErrorMsg(field, expectedValue, testValue)
	return errors.New(msg)
}

func assertCalledWithMsg(paramName string, expectedParam any, testParam any) string {
	return fmt.Sprintf("expected to have called param %s with value '%v' but got '%v'", paramName, expectedParam, testParam)
}
func assertCalledWithError(paramName string, expectedParam any, testParam any) error {
	if expectedParam == testParam {
		return nil
	}
	msg := assertCalledWithMsg(paramName, expectedParam, testParam)
	return errors.New(msg)
}

func assertGameState(expectedGameState model.GameState, gameState model.GameState) error {
	// basic call assertions
	err := assertValueError("GameID", expectedGameState.GameID, gameState.GameID)
	if err != nil {
		return err
	}
	err = assertValueError("CurrentUserID", expectedGameState.CurrentUserID, gameState.CurrentUserID)
	if err != nil {
		return err
	}

	// turn assertions
	err = assertValueError("TurnID", expectedGameState.TurnID, gameState.TurnID)
	if err != nil {
		return err
	}
	err = assertValueError("TurnEnded", expectedGameState.TurnEnded, gameState.TurnEnded)
	if err != nil {
		return err
	}
	err = assertValueError("Word", expectedGameState.Word, gameState.Word)
	if err != nil {
		return err
	}
	err = assertValueError("Hint", expectedGameState.Hint, gameState.Hint)
	if err != nil {
		return err
	}

	// message assertions
	err = assertValueError("Message Length", len(expectedGameState.Messages), len(gameState.Messages))
	if err != nil {
		return err
	}
	for i, m := range gameState.Messages {
		expectedMsg := expectedGameState.Messages[i]
		err = assertValueError(fmt.Sprintf("Message[%d].Nickname", i), expectedMsg.Nickname, m.Nickname)
		if err != nil {
			return err
		}
		err = assertValueError(fmt.Sprintf("Message[%d].Me", i), expectedMsg.Me, m.Me)
		if err != nil {
			return err
		}
		err = assertValueError(fmt.Sprintf("Message[%d].Content", i), expectedMsg.Content, m.Content)
		if err != nil {
			return err
		}
	}

	// leaderboard assertions
	err = assertValueError("Leaderboard Length", len(expectedGameState.Leaderboard), len(gameState.Leaderboard))
	if err != nil {
		return err
	}
	for i, l := range gameState.Leaderboard {
		expectedLeaderboard := expectedGameState.Leaderboard[i]
		err = assertValueError(fmt.Sprintf("Leaderboard[%d].Nickname", i), expectedLeaderboard.Nickname, l.Nickname)
		if err != nil {
			return err
		}
		err = assertValueError(fmt.Sprintf("Leaderboard[%d].Me", i), expectedLeaderboard.Me, l.Me)
		if err != nil {
			return err
		}
		err = assertValueError(fmt.Sprintf("Leaderboard[%d].Score", i), expectedLeaderboard.Score, l.Score)
		if err != nil {
			return err
		}
		err = assertValueError(fmt.Sprintf("Leaderboard[%d].GuessedWord", i), expectedLeaderboard.GuessedWord, l.GuessedWord)
		if err != nil {
			return err
		}
	}

	return err
}

func TestGameState(t *testing.T) {

	t.Run("initial empty state", func(t *testing.T) {
		expectedGameID := "some-game-id"
		mgr := &repository.MockGameRepository{
			GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
				errMsg := assertCalledWithError("GameID", expectedGameID, id)
				if errMsg != nil {
					t.Error(errMsg)
				}
				return []model.Player{
					{ID: "some-user-id", Nickname: "SomeNick"},
				}, nil
			},
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				errMsg := assertCalledWithError("GameID", expectedGameID, id)
				if errMsg != nil {
					t.Error(errMsg)
				}
				return model.GameTurn{
					ID:     "some-turn-id",
					WordID: "some-word-id",
				}, nil
			},
			GetMessagesMock: func(ctx context.Context, id string) ([]model.Message, error) {
				errMsg := assertCalledWithError("GameID", expectedGameID, id)
				if errMsg != nil {
					t.Error(errMsg)
				}
				return []model.Message{}, nil
			},
			GetScoresMock: func(ctx context.Context, id string) ([]model.Score, error) {
				errMsg := assertCalledWithError("GameID", expectedGameID, id)
				if errMsg != nil {
					t.Error(errMsg)
				}
				return []model.Score{}, nil
			},
		}

		expectedWordID := "some-word-id"
		mwr := &repository.MockWordRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.Word, error) {
				errMsg := assertCalledWithError("WordID", expectedWordID, id)
				if errMsg != nil {
					t.Error(errMsg)
				}

				return model.Word{ID: "some-word-id", Word: "Some Word", Hint: "Some Hint"}, nil
			},
		}

		emojixUsecase := usecase.NewEmojixUsecase(
			nil,
			mgr,
			mwr,
			nil,
			nil,
		)

		ctx := context.Background()
		gameState, err := emojixUsecase.GameState(ctx, "some-game-id", "some-user-id")
		if err != nil {
			t.Fatal(err)
		}

		err = assertGameState(model.GameState{
			GameID:        "some-game-id",
			CurrentUserID: "some-user-id",
			TurnID:        "some-turn-id",
			TurnEnded:     false,
			Word:          "**** ****",
			Hint:          "Some Hint",
			Messages:      []model.GameStateMessage{},
			Leaderboard: []model.LeaderboardEntry{
				{Nickname: "SomeNick", Me: true, GuessedWord: false, Score: 0},
			},
		}, gameState)
		if err != nil {
			t.Error(err)
			return
		}

	})

	t.Run("should not mask word to user guessed", func(t *testing.T) {
		mgr := &repository.MockGameRepository{}
		mgr.GetPlayersMock = func(ctx context.Context, id string) ([]model.Player, error) {
			return []model.Player{
				{ID: "p-1", Nickname: "Player1"},
				{ID: "p-2", Nickname: "Player2"},
				{ID: "p-3", Nickname: "Player3"},
			}, nil
		}
		mgr.GetLatestTurnMock = func(ctx context.Context, id string) (model.GameTurn, error) {
			return model.GameTurn{
				ID: "last-turn-id", WordID: "some-word-id",
			}, nil
		}
		mgr.GetScoresMock = func(ctx context.Context, id string) ([]model.Score, error) {
			return []model.Score{
				{PlayerID: "p-1", Score: 10, TurnID: "last-turn-id", GameID: "some-game-id", MessageID: "guess-msg-id"},
			}, nil

		}
		mgr.GetMessagesMock = func(ctx context.Context, id string) ([]model.Message, error) {
			return []model.Message{
				{ID: "guess-msg-id", PlayerID: "p-1", Content: "Some Word"},
			}, nil
		}

		mwr := &repository.MockWordRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.Word, error) {
				return model.Word{ID: "some-word-id", Word: "Some Word", Hint: "Some Hint"}, nil
			},
		}

		emojixUsecase := usecase.NewEmojixUsecase(
			nil,
			mgr,
			mwr,
			nil,
			nil,
		)

		ctx := context.Background()
		gameState, err := emojixUsecase.GameState(ctx, "some-game-id", "p-1")
		if err != nil {
			t.Fatal(err)
		}

		err = assertGameState(model.GameState{
			GameID:        "some-game-id",
			CurrentUserID: "p-1",
			TurnID:        "last-turn-id",
			TurnEnded:     false,
			Word:          "Some Word",
			Hint:          "Some Hint",
			Messages: []model.GameStateMessage{
				{Nickname: "Player1", Me: true, Content: "Some Word"},
			},
			Leaderboard: []model.LeaderboardEntry{
				{Nickname: "Player1", Me: true, GuessedWord: true, Score: 10},
				{Nickname: "Player2", Me: false, GuessedWord: false, Score: 0},
				{Nickname: "Player3", Me: false, GuessedWord: false, Score: 0},
			},
		}, gameState)
		if err != nil {
			t.Error(err)
			return
		}

	})

	// TODO: complete the tests even more to discover more bugs
	t.Run("should order messages from oldest to newest", func(t *testing.T) {

	})

	t.Run("should sum up all scores in leaderboard", func(t *testing.T) {

	})

	t.Run("turn should end when all players guessed the word", func(t *testing.T) {

	})
	t.Run("should ", func(t *testing.T) {

	})
	t.Run("GameRepository.GetPlayer Failure", func(t *testing.T) {
		mgr := &repository.MockGameRepository{}
		mockErr := errors.New("players failed")
		mgr.GetPlayersMock = func(ctx context.Context, id string) ([]model.Player, error) {
			return nil, mockErr
		}
		emojixUsecase := usecase.NewEmojixUsecase(
			nil,
			mgr,
			nil,
			nil,
			nil,
		)

		ctx := context.Background()
		_, err := emojixUsecase.GameState(ctx, "some-game-id", "some-user-id")
		if !errors.Is(mockErr, err) {
			t.Errorf("expected to have error %v but got %v", mockErr, err)
		}
	})
}

func TestGameUpdates(t *testing.T) {
	ch := make(chan service.GameNotification)
	mgn := &service.MockGameNotifier{
		SubMock: func(gameID, userID string) chan service.GameNotification {
			err := assertCalledWithError("GameID", "some-game-id", gameID)
			if err != nil {
				t.Error(err)
			}

			err = assertCalledWithError("UserID", "some-user-id", userID)
			if err != nil {
				t.Error(err)
			}

			return ch
		},
		UnsubMock: func(userID string) {
			err := assertCalledWithError("UserID", "some-user-id", userID)
			if err != nil {
				t.Error(err)
			}

		},
	}
	emojixUsecase := usecase.NewEmojixUsecase(
		nil,
		nil,
		nil,
		nil,
		mgn,
	)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ch <- &usecase.GameJoinNotification{Nickname: "nick-1", PlayerID: "player-1"}
		cancel()
	}()

	msgCount := 0
	err := emojixUsecase.GameUpdates(ctx, "some-game-id", "some-user-id", func(notifType string, content string) error {

		msgCount += 1

		expectedType := "join"
		if expectedType != notifType {
			t.Errorf("expected to have notif type '%s' but got '%s'", expectedType, notifType)
		}

		expectedContent := "player-1,nick-1"
		if expectedContent != content {

			t.Errorf("expected to have notif content '%s' but got '%s'", expectedContent, content)
		}

		return nil
	})

	if err != nil {
		t.Errorf("expected to not error but got %v", err)
	}

	if mgn.UnsubMockCalled != true {
		t.Error("expected to unsubscribe")
	}
}
