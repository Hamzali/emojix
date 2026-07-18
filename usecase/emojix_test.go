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
	"runtime"
	"testing"
	"time"
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
		mgr := &repotest.MockGameRepository{
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
					ID:        "some-turn-id",
					WordID:    "some-word-id",
					CreatedAt: time.Now(),
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
		mwr := &repotest.MockWordRepository{
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
			&servicetest.MockGameLoop{},
			service.NewRealClock(),
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
		mgr := &repotest.MockGameRepository{}
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
				CreatedAt: time.Now(),
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

		mwr := &repotest.MockWordRepository{
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
			&servicetest.MockGameLoop{},
			service.NewRealClock(),
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
				{PlayerID: "p-1", Nickname: "Player1", Me: true, GuessedWord: true, Score: 10},
				{PlayerID: "p-2", Nickname: "Player2", Me: false, GuessedWord: false, Score: 0},
				{PlayerID: "p-3", Nickname: "Player3", Me: false, GuessedWord: false, Score: 0},
			},
		}, gameState)
		if err != nil {
			t.Error(err)
			return
		}

	})

	// NOTE: the timeout branch of TurnEnded is covered separately by
	// TestGameState_TurnTimedOut (uses the T13 FakeClock seam).

	t.Run("should order messages from newest to oldest", func(t *testing.T) {
		expectedGameID := "some-game-id"
		mgr := &repotest.MockGameRepository{
			GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
				return []model.Player{{ID: "p-1", Nickname: "Player1"}}, nil
			},
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				return model.GameTurn{ID: "last-turn-id", WordID: "some-word-id", CreatedAt: time.Now()}, nil
			},
			GetMessagesMock: func(ctx context.Context, id string) ([]model.Message, error) {
				err := assertCalledWithError("GameID", expectedGameID, id)
				if err != nil {
					t.Error(err)
				}
				// Storage order: oldest first, newest last.
				return []model.Message{
					{ID: "m-old", PlayerID: "p-1", Content: "old"},
					{ID: "m-mid", PlayerID: "p-1", Content: "mid"},
					{ID: "m-new", PlayerID: "p-1", Content: "new"},
				}, nil
			},
			GetScoresMock: func(ctx context.Context, id string) ([]model.Score, error) {
				return []model.Score{}, nil
			},
		}
		mwr := &repotest.MockWordRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.Word, error) {
				return model.Word{ID: "some-word-id", Word: "Some Word", Hint: "Some Hint"}, nil
			},
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, mwr, nil, nil, &servicetest.MockGameLoop{}, service.NewRealClock())

		gameState, err := uc.GameState(context.Background(), expectedGameID, "p-1")
		if err != nil {
			t.Fatal(err)
		}

		// GameState reverses the repo message order so newest is first.
		got := []string{}
		for _, m := range gameState.Messages {
			got = append(got, m.Content)
		}
		want := []string{"new", "mid", "old"}
		if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
			t.Errorf("messages: got %v, want %v (newest first)", got, want)
		}
	})

	t.Run("should sum up all scores in leaderboard", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
				return []model.Player{
					{ID: "p-1", Nickname: "Player1"},
					{ID: "p-2", Nickname: "Player2"},
				}, nil
			},
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				return model.GameTurn{ID: "latest-turn", WordID: "some-word-id", CreatedAt: time.Now()}, nil
			},
			GetMessagesMock: func(ctx context.Context, id string) ([]model.Message, error) {
				return []model.Message{}, nil
			},
			GetScoresMock: func(ctx context.Context, id string) ([]model.Score, error) {
				// Scores span multiple turns; buildLeaderboard must sum them.
				return []model.Score{
					{PlayerID: "p-1", Score: 10, TurnID: "older-turn"},
					{PlayerID: "p-1", Score: 5, TurnID: "latest-turn"},
					{PlayerID: "p-2", Score: 3, TurnID: "older-turn"},
				}, nil
			},
		}
		mwr := &repotest.MockWordRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.Word, error) {
				return model.Word{ID: "some-word-id", Word: "Some Word", Hint: "Some Hint"}, nil
			},
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, mwr, nil, nil, &servicetest.MockGameLoop{}, service.NewRealClock())

		gameState, err := uc.GameState(context.Background(), "some-game-id", "p-1")
		if err != nil {
			t.Fatal(err)
		}

		scoreByPlayer := map[string]int{}
		for _, l := range gameState.Leaderboard {
			scoreByPlayer[l.PlayerID] = l.Score
		}
		if scoreByPlayer["p-1"] != 15 {
			t.Errorf("p-1 score: got %d, want 15 (sum across turns)", scoreByPlayer["p-1"])
		}
		if scoreByPlayer["p-2"] != 3 {
			t.Errorf("p-2 score: got %d, want 3", scoreByPlayer["p-2"])
		}
	})

	t.Run("turn should end when all players guessed the word", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
				return []model.Player{
					{ID: "p-1", Nickname: "Player1"},
					{ID: "p-2", Nickname: "Player2"},
					{ID: "p-3", Nickname: "Player3"},
				}, nil
			},
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				return model.GameTurn{ID: "last-turn-id", WordID: "some-word-id", CreatedAt: time.Now()}, nil
			},
			GetMessagesMock: func(ctx context.Context, id string) ([]model.Message, error) {
				return []model.Message{}, nil
			},
			GetScoresMock: func(ctx context.Context, id string) ([]model.Score, error) {
				// Every active player has a score on the latest turn → allGuessed.
				return []model.Score{
					{PlayerID: "p-1", Score: 10, TurnID: "last-turn-id"},
					{PlayerID: "p-2", Score: 10, TurnID: "last-turn-id"},
					{PlayerID: "p-3", Score: 10, TurnID: "last-turn-id"},
				}, nil
			},
		}
		mwr := &repotest.MockWordRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.Word, error) {
				return model.Word{ID: "some-word-id", Word: "Some Word", Hint: "Some Hint"}, nil
			},
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, mwr, nil, nil, &servicetest.MockGameLoop{}, service.NewRealClock())

		gameState, err := uc.GameState(context.Background(), "some-game-id", "p-1")
		if err != nil {
			t.Fatal(err)
		}
		if !gameState.TurnEnded {
			t.Errorf("expected TurnEnded to be true when all active players guessed, got false")
		}
	})

	t.Run("GameRepository.GetPlayer Failure", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{}
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
			&servicetest.MockGameLoop{},
			service.NewRealClock(),
		)

		ctx := context.Background()
		_, err := emojixUsecase.GameState(ctx, "some-game-id", "some-user-id")
		if !errors.Is(mockErr, err) {
			t.Errorf("expected to have error %v but got %v", mockErr, err)
		}
	})
}

func TestGameUpdates(t *testing.T) {
	cases := []struct {
		name       string
		notif      service.GameNotification
		preCancel  bool // cancel ctx before calling GameUpdates (cancellation case)
		handlerErr error
		wantType   string
		wantData   string
		wantErr    error
	}{
		{
			name:     "join",
			notif:    &usecase.GameJoinNotification{Nickname: "nick-1", PlayerID: "player-1"},
			wantType: "join",
			wantData: "player-1,nick-1",
		},
		{
			name:     "msg",
			notif:    &usecase.GameMsgNotification{UserID: "u1", Nickname: "n1", Content: "hi"},
			wantType: "msg",
			wantData: "u1,n1,hi",
		},
		{
			name:     "guessed",
			notif:    &usecase.GameCorrectGuessNotification{UserID: "u1", Nickname: "n1"},
			wantType: "guessed",
			wantData: "u1,n1",
		},
		{
			name:     "turnended",
			notif:    &usecase.GameTurnEndNotification{},
			wantType: "turnended",
			wantData: "",
		},
		{
			name:     "left",
			notif:    &usecase.UserLeftNotification{UserID: "u1"},
			wantType: "left",
			wantData: "u1",
		},
		{
			name:       "handler error aborts and returns error",
			notif:      &usecase.GameMsgNotification{UserID: "u1", Nickname: "n1", Content: "hi"},
			handlerErr: errors.New("boom"),
			wantErr:    errors.New("boom"),
		},
		{
			name:      "context cancellation returns nil with no notif",
			preCancel: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ch := make(chan service.GameNotification) // unbuffered: a send blocks until GameUpdates reads it
			cleanupCount := 0
			mgn := &servicetest.MockGameNotifier{
				SubMock: func(gameID, userID string) (chan service.GameNotification, func()) {
					if err := assertCalledWithError("GameID", "some-game-id", gameID); err != nil {
						t.Error(err)
					}
					if err := assertCalledWithError("UserID", "some-user-id", userID); err != nil {
						t.Error(err)
					}
					return ch, func() { cleanupCount++ }
				},
			}
			uc := usecase.NewEmojixUsecase(nil, nil, nil, nil, mgn, &servicetest.MockGameLoop{}, service.NewRealClock())

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var gotType, gotData string
			handler := func(notifType, content string) error {
				gotType = notifType
				gotData = content
				return tc.handlerErr
			}

			if tc.preCancel {
				cancel()
			} else {
				// Deliver the notif, then cancel so GameUpdates returns. The send
				// blocks until GameUpdates reads it, guaranteeing the notif is
				// processed before ctx is cancelled (no time.Sleep).
				go func() {
					ch <- tc.notif
					cancel()
				}()
			}

			err := uc.GameUpdates(ctx, "some-game-id", "some-user-id", handler)

			if tc.wantErr != nil {
				if err == nil || err.Error() != tc.wantErr.Error() {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if cleanupCount != 1 {
				t.Errorf("cleanup: got %d, want 1", cleanupCount)
			}

			if !tc.preCancel && tc.wantType != "" {
				if gotType != tc.wantType {
					t.Errorf("notif type: got %q, want %q", gotType, tc.wantType)
				}
				if gotData != tc.wantData {
					t.Errorf("notif data: got %q, want %q", gotData, tc.wantData)
				}
			}
		})
	}
}

func TestGameState_TurnTimedOut(t *testing.T) {
	clock := servicetest.NewFakeClock()
	turnStartedAt := clock.Now()

	mgr := &repotest.MockGameRepository{
		GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
			return []model.Player{{ID: "some-user-id", Nickname: "SomeNick"}}, nil
		},
		GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
			return model.GameTurn{
				ID:        "some-turn-id",
				WordID:    "some-word-id",
				CreatedAt: turnStartedAt,
			}, nil
		},
		GetMessagesMock: func(ctx context.Context, id string) ([]model.Message, error) {
			return []model.Message{}, nil
		},
		GetScoresMock: func(ctx context.Context, id string) ([]model.Score, error) {
			return []model.Score{}, nil
		},
	}

	mwr := &repotest.MockWordRepository{
		FindByIDMock: func(ctx context.Context, id string) (model.Word, error) {
			return model.Word{ID: "some-word-id", Word: "Some Word", Hint: "Some Hint"}, nil
		},
	}

	// Advance the fake clock past the turn duration. turnDuration is 60s.
	clock.Advance(time.Minute + time.Second)

	emojixUsecase := usecase.NewEmojixUsecase(
		nil,
		mgr,
		mwr,
		nil,
		nil,
		&servicetest.MockGameLoop{},
		clock,
	)

	gameState, err := emojixUsecase.GameState(context.Background(), "some-game-id", "some-user-id")
	if err != nil {
		t.Fatal(err)
	}

	if !gameState.TurnEnded {
		t.Errorf("expected TurnEnded to be true after the clock advanced past turnDuration, but got false")
	}
}

func newInitGameUsecase(t *testing.T, mur repository.UserRepository, mgr *repotest.MockGameRepository, mwr *repotest.MockWordRepository, gl *servicetest.MockGameLoop, commitErr error, newErr error) (usecase.EmojixUsecase, *repotest.MockUnitOfWork) {
	t.Helper()
	uow := &repotest.MockUnitOfWork{
		GameRepositoryMock: mgr,
		CommitMock:         func() error { return commitErr },
		RollbackMock:       func() error { return nil },
	}
	factory := &repotest.MockUnitOfWorkFactory{
		NewMock: func(ctx context.Context) (repository.UnitOfWork, error) {
			if newErr != nil {
				return nil, newErr
			}
			return uow, newErr
		},
	}
	uc := usecase.NewEmojixUsecase(mur, mgr, mwr, factory, nil, gl, service.NewRealClock())
	return uc, uow
}

func TestInitGame(t *testing.T) {
	const userID = "init-user-id"

	t.Run("happy path starts loop after commit", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			CreateMock: func(ctx context.Context) (model.Game, error) {
				return model.Game{ID: "game-1"}, nil
			},
			AddPlayerMock: func(ctx context.Context, gameID, playerID string) error {
				if err := assertCalledWithError("GameID", "game-1", gameID); err != nil {
					t.Error(err)
				}
				if err := assertCalledWithError("PlayerID", userID, playerID); err != nil {
					t.Error(err)
				}
				return nil
			},
			AddTurnMock: func(ctx context.Context, gameID, wordID string) (model.GameTurn, error) {
				if err := assertCalledWithError("GameID", "game-1", gameID); err != nil {
					t.Error(err)
				}
				if wordID != "w1" && wordID != "w2" {
					t.Errorf("AddTurn wordID %q not from GetAll list", wordID)
				}
				return model.GameTurn{ID: "turn-1"}, nil
			},
		}
		mwr := &repotest.MockWordRepository{
			GetAllMock: func(ctx context.Context) ([]model.Word, error) {
				return []model.Word{{ID: "w1", Word: "Alpha"}, {ID: "w2", Word: "Beta"}}, nil
			},
		}

		committed := false
		var startGameID string
		var startDur time.Duration
		startCalled := make(chan struct{}, 1)
		gl := &servicetest.MockGameLoop{
			StartMock: func(ctx context.Context, gameID string, duration time.Duration) {
				if !committed {
					t.Error("gameLoop.Start called before uow.Commit")
				}
				startGameID = gameID
				startDur = duration
				startCalled <- struct{}{}
			},
		}
		uc, uow := newInitGameUsecase(t, nil, mgr, mwr, gl, nil, nil)
		// Wrap Commit so we can observe ordering relative to Start.
		uow.CommitMock = func() error {
			committed = true
			return nil
		}

		game, err := uc.InitGame(context.Background(), userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if game.ID != "game-1" {
			t.Errorf("game ID: got %q, want game-1", game.ID)
		}
		if !mgr.CreateCalled || !mgr.AddPlayerCalled || !mgr.AddTurnCalled {
			t.Error("expected Create/AddPlayer/AddTurn to be called")
		}
		if !uow.CommitCalled {
			t.Error("expected Commit to be called")
		}
		if !gl.StartCalled {
			t.Error("expected gameLoop.Start to be called")
		}
		select {
		case <-startCalled:
		case <-time.After(time.Second):
			t.Fatal("StartMock was not invoked")
		}
		if startGameID != "game-1" {
			t.Errorf("Start gameID: got %q, want game-1", startGameID)
		}
		if startDur != time.Minute {
			t.Errorf("Start duration: got %v, want %v", startDur, time.Minute)
		}
	})

	t.Run("uow.New fails", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{}
		mwr := &repotest.MockWordRepository{}
		gl := &servicetest.MockGameLoop{}
		newErr := errors.New("uow new failed")
		uc, _ := newInitGameUsecase(t, nil, mgr, mwr, gl, nil, newErr)

		_, err := uc.InitGame(context.Background(), userID)
		if !errors.Is(err, newErr) {
			t.Fatalf("expected newErr, got %v", err)
		}
		if mgr.CreateCalled || mgr.AddPlayerCalled || mgr.AddTurnCalled {
			t.Error("no repo calls expected on uow.New failure")
		}
		if gl.StartCalled {
			t.Error("Start must not be called on uow.New failure")
		}
	})

	t.Run("gameRepo.Create fails rolls back and does not start", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			CreateMock: func(ctx context.Context) (model.Game, error) {
				return model.Game{}, errors.New("create failed")
			},
		}
		mwr := &repotest.MockWordRepository{}
		gl := &servicetest.MockGameLoop{}
		uc, uow := newInitGameUsecase(t, nil, mgr, mwr, gl, nil, nil)

		_, err := uc.InitGame(context.Background(), userID)
		if err == nil {
			t.Fatal("expected error from Create")
		}
		if mgr.AddPlayerCalled || mgr.AddTurnCalled {
			t.Error("AddPlayer/AddTurn must not be called on Create failure")
		}
		if uow.CommitCalled {
			t.Error("Commit must not be called on Create failure")
		}
		if !uow.RollbackCalled {
			t.Error("Rollback (deferred) must be called on Create failure")
		}
		if gl.StartCalled {
			t.Error("Start must not be called on Create failure")
		}
	})

	t.Run("AddTurn fails rolls back and does not start", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			CreateMock: func(ctx context.Context) (model.Game, error) {
				return model.Game{ID: "game-2"}, nil
			},
			AddPlayerMock: func(ctx context.Context, gameID, playerID string) error { return nil },
			AddTurnMock: func(ctx context.Context, gameID, wordID string) (model.GameTurn, error) {
				return model.GameTurn{}, errors.New("addturn failed")
			},
		}
		mwr := &repotest.MockWordRepository{
			GetAllMock: func(ctx context.Context) ([]model.Word, error) {
				return []model.Word{{ID: "w1", Word: "Alpha"}}, nil
			},
		}
		gl := &servicetest.MockGameLoop{}
		uc, uow := newInitGameUsecase(t, nil, mgr, mwr, gl, nil, nil)

		_, err := uc.InitGame(context.Background(), userID)
		if err == nil {
			t.Fatal("expected error from AddTurn")
		}
		if uow.CommitCalled {
			t.Error("Commit must not be called on AddTurn failure")
		}
		if !uow.RollbackCalled {
			t.Error("Rollback (deferred) must be called on AddTurn failure")
		}
		if gl.StartCalled {
			t.Error("Start must not be called on AddTurn failure")
		}
	})

	t.Run("empty word list returns ErrNoWords", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			CreateMock: func(ctx context.Context) (model.Game, error) {
				return model.Game{ID: "game-3"}, nil
			},
			AddPlayerMock: func(ctx context.Context, gameID, playerID string) error { return nil },
			AddTurnMock: func(ctx context.Context, gameID, wordID string) (model.GameTurn, error) {
				t.Error("AddTurn must not be called when there are no words")
				return model.GameTurn{}, nil
			},
		}
		mwr := &repotest.MockWordRepository{
			GetAllMock: func(ctx context.Context) ([]model.Word, error) {
				return []model.Word{}, nil
			},
		}
		gl := &servicetest.MockGameLoop{}
		uc, uow := newInitGameUsecase(t, nil, mgr, mwr, gl, nil, nil)

		_, err := uc.InitGame(context.Background(), userID)
		if !errors.Is(err, usecase.ErrNoWords) {
			t.Fatalf("expected ErrNoWords, got %v", err)
		}
		if uow.CommitCalled {
			t.Error("Commit must not be called on empty word list")
		}
		if !uow.RollbackCalled {
			t.Error("Rollback (deferred) must be called on empty word list")
		}
		if gl.StartCalled {
			t.Error("Start must not be called on empty word list")
		}
	})
}

func TestInitUser(t *testing.T) {
	t.Run("happy path generates id shape and nickname and persists", func(t *testing.T) {
		createCalled := make(chan struct{}, 1)
		var gotID string
		var gotNick string
		mur := &repotest.MockUserRepository{
			CreateOrUpdateMock: func(ctx context.Context, id string, params repository.UserCreateOrUpdateParams) error {
				gotID = id
				gotNick = params.Nickname
				createCalled <- struct{}{}
				return nil
			},
		}
		uc := usecase.NewEmojixUsecase(mur, nil, nil, nil, nil, &servicetest.MockGameLoop{}, service.NewRealClock())

		user, err := uc.InitUser(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(user.ID) != 32 {
			t.Errorf("user ID length: got %d, want 32 (16 hex-encoded bytes)", len(user.ID))
		}
		for _, c := range user.ID {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("user ID %q is not lowercase hex", user.ID)
				break
			}
		}
		// Nickname shape: capitalize(adjective) + capitalize(animal), e.g. "SillyCat".
		if len(user.Nickname) < 2 || len(user.Nickname) < 4 {
			t.Errorf("nickname too short: %q", user.Nickname)
		}
		if user.Nickname[0] < 'A' || user.Nickname[0] > 'Z' {
			t.Errorf("nickname %q must start with an uppercase letter", user.Nickname)
		}
		hasLower := false
		for _, c := range user.Nickname[1:] {
			if c >= 'a' && c <= 'z' {
				hasLower = true
				break
			}
		}
		if !hasLower {
			t.Errorf("nickname %q must contain lowercase letters", user.Nickname)
		}

		select {
		case <-createCalled:
		case <-time.After(time.Second):
			t.Fatal("CreateOrUpdate not called")
		}
		if gotID != user.ID {
			t.Errorf("CreateOrUpdate ID: got %q, want %q", gotID, user.ID)
		}
		if gotNick != user.Nickname {
			t.Errorf("CreateOrUpdate Nickname: got %q, want %q", gotNick, user.Nickname)
		}
	})

	t.Run("CreateOrUpdate fails propagates", func(t *testing.T) {
		mur := &repotest.MockUserRepository{
			CreateOrUpdateMock: func(ctx context.Context, id string, params repository.UserCreateOrUpdateParams) error {
				return errors.New("persist failed")
			},
		}
		uc := usecase.NewEmojixUsecase(mur, nil, nil, nil, nil, &servicetest.MockGameLoop{}, service.NewRealClock())

		_, err := uc.InitUser(context.Background())
		if err == nil {
			t.Fatal("expected error from CreateOrUpdate")
		}
	})
}

// --- T08: TestGuess helpers ---

func newGuessUsecase(mur repository.UserRepository, mgr *repotest.MockGameRepository, mwr *repotest.MockWordRepository, mgn *servicetest.MockGameNotifier, gl *servicetest.MockGameLoop, commitErr error) (usecase.EmojixUsecase, *repotest.MockUnitOfWork) {
	uow := &repotest.MockUnitOfWork{
		GameRepositoryMock: mgr,
		CommitMock:         func() error { return commitErr },
		RollbackMock:       func() error { return nil },
	}
	factory := &repotest.MockUnitOfWorkFactory{
		NewMock: func(ctx context.Context) (repository.UnitOfWork, error) {
			return uow, nil
		},
	}
	uc := usecase.NewEmojixUsecase(mur, mgr, mwr, factory, mgn, gl, service.NewRealClock())
	return uc, uow
}

func drainPub(t *testing.T, ch <-chan service.GameNotification, want int) []service.GameNotification {
	t.Helper()
	got := make([]service.GameNotification, 0, want)
	for i := 0; i < want; i++ {
		select {
		case n := <-ch:
			got = append(got, n)
		case <-time.After(time.Second):
			t.Fatalf("expected %d pub(s), got %d: %+v", want, len(got), got)
		}
	}
	return got
}

func assertNoPub(t *testing.T, ch <-chan service.GameNotification) {
	t.Helper()
	select {
	case n := <-ch:
		t.Fatalf("expected no pub, got type=%s data=%q", n.GetType(), n.GetData())
	case <-time.After(50 * time.Millisecond):
	}
}

func notifByType(ns []service.GameNotification, typ string) service.GameNotification {
	for _, n := range ns {
		if n.GetType() == typ {
			return n
		}
	}
	return nil
}

func TestGuess(t *testing.T) {
	const (
		gameID  = "game-1"
		userID  = "p-1"
		turnID  = "turn-1"
		wordID  = "w-1"
		theWord = "Secret"
	)

	// baseWordRepo returns a fixed word for the latest turn.
	baseWordRepo := func() *repotest.MockWordRepository {
		return &repotest.MockWordRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.Word, error) {
				return model.Word{ID: wordID, Word: theWord, Hint: "h"}, nil
			},
		}
	}
	// baseGameRepo wires the latest turn + a SendMessage that returns a message.
	baseGameRepo := func() *repotest.MockGameRepository {
		return &repotest.MockGameRepository{
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				if err := assertCalledWithError("GameID", gameID, id); err != nil {
					t.Error(err)
				}
				return model.GameTurn{ID: turnID, WordID: wordID}, nil
			},
			SendMessageMock: func(ctx context.Context, g, turn, u, content string) (model.Message, error) {
				return model.Message{ID: "msg-1", PlayerID: u, Content: content, TurnID: turn}, nil
			},
		}
	}

	t.Run("wrong guess publishes raw content after commit and scores nothing", func(t *testing.T) {
		mgr := baseGameRepo()
		mgr.GetPlayersMock = func(ctx context.Context, id string) ([]model.Player, error) { return nil, nil }
		mgr.GetScoresMock = func(ctx context.Context, id string) ([]model.Score, error) { return nil, nil }
		mur := &repotest.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{ID: userID, Nickname: "Nick1"}, nil
			},
		}
		pubCh := make(chan service.GameNotification, 2)
		mgn := &servicetest.MockGameNotifier{
			PubMock: func(g, u string, n service.GameNotification) { pubCh <- n },
		}
		gl := &servicetest.MockGameLoop{}
		uc, uow := newGuessUsecase(mur, mgr, baseWordRepo(), mgn, gl, nil)

		if err := uc.Guess(context.Background(), gameID, userID, "nope"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !uow.CommitCalled {
			t.Error("expected Commit to be called once on wrong guess")
		}
		if mgr.AddScoreCalled {
			t.Error("AddScore must not be called on a wrong guess")
		}
		if gl.EndGameTurnCalled {
			t.Error("EndGameTurn must not be called on a wrong guess")
		}
		pub := drainPub(t, pubCh, 1)[0]
		if pub.GetType() != "msg" {
			t.Errorf("pub type: got %q, want msg", pub.GetType())
		}
		if got := pub.GetData(); got != userID+",Nick1,nope" {
			t.Errorf("pub data: got %q, want %q", got, userID+",Nick1,nope")
		}
	})

	t.Run("correct first guess scores points and pubs guessed but does not end turn", func(t *testing.T) {
		mgr := baseGameRepo()
		mgr.GetPlayersMock = func(ctx context.Context, id string) ([]model.Player, error) {
			return []model.Player{
				{ID: userID, Nickname: "Nick1", State: model.ActivePlayerState},
				{ID: "p-2", Nickname: "Nick2", State: model.ActivePlayerState},
				{ID: "p-3", Nickname: "Nick3", State: model.ActivePlayerState},
			}, nil
		}
		mgr.GetScoresMock = func(ctx context.Context, id string) ([]model.Score, error) { return nil, nil }
		var scoredPoint int
		mgr.AddScoreMock = func(ctx context.Context, g, u, msg, turn string, point int) error {
			scoredPoint = point
			return nil
		}
		mur := &repotest.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{ID: userID, Nickname: "Nick1"}, nil
			},
		}
		pubCh := make(chan service.GameNotification, 2)
		mgn := &servicetest.MockGameNotifier{PubMock: func(g, u string, n service.GameNotification) { pubCh <- n }}
		gl := &servicetest.MockGameLoop{}
		uc, _ := newGuessUsecase(mur, mgr, baseWordRepo(), mgn, gl, nil)

		if err := uc.Guess(context.Background(), gameID, userID, theWord); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// 3 active players, no prior guesses → totalGuessers=1, coeff=3/1=3, point=10*3=30.
		// NOTE: formula kept as-is per backlog (README drift). See TODO in emojix.go.
		if scoredPoint != 30 {
			t.Errorf("AddScore point: got %d, want 30", scoredPoint)
		}
		if gl.EndGameTurnCalled {
			t.Error("EndGameTurn must not be called when not everyone has guessed")
		}
		pub := drainPub(t, pubCh, 2)
		if notifByType(pub, "msg") == nil {
			t.Error("expected a msg pub (masked ***)")
		}
		if notifByType(pub, "guessed") == nil {
			t.Error("expected a guessed pub")
		}
		if m := notifByType(pub, "msg"); m != nil && m.GetData() != userID+",Nick1,***" {
			t.Errorf("masked msg data: got %q, want %q", m.GetData(), userID+",Nick1,***")
		}
	})

	t.Run("last correct guess ends turn", func(t *testing.T) {
		mgr := baseGameRepo()
		mgr.GetPlayersMock = func(ctx context.Context, id string) ([]model.Player, error) {
			return []model.Player{
				{ID: userID, Nickname: "Nick1", State: model.ActivePlayerState},
				{ID: "p-2", Nickname: "Nick2", State: model.ActivePlayerState},
				{ID: "p-3", Nickname: "Nick3", State: model.ActivePlayerState},
			}, nil
		}
		mgr.GetScoresMock = func(ctx context.Context, id string) ([]model.Score, error) {
			return []model.Score{
				{PlayerID: "p-2", TurnID: turnID},
				{PlayerID: "p-3", TurnID: turnID},
			}, nil
		}
		endGameTurnCalled := make(chan struct{}, 1)
		mgr.AddScoreMock = func(ctx context.Context, g, u, msg, turn string, point int) error { return nil }
		mur := &repotest.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{ID: userID, Nickname: "Nick1"}, nil
			},
		}
		pubCh := make(chan service.GameNotification, 2)
		mgn := &servicetest.MockGameNotifier{PubMock: func(g, u string, n service.GameNotification) { pubCh <- n }}
		gl := &servicetest.MockGameLoop{
			EndGameTurnMock: func(g string) {
				if err := assertCalledWithError("GameID", gameID, g); err != nil {
					t.Error(err)
				}
				endGameTurnCalled <- struct{}{}
			},
		}
		uc, _ := newGuessUsecase(mur, mgr, baseWordRepo(), mgn, gl, nil)

		if err := uc.Guess(context.Background(), gameID, userID, theWord); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		select {
		case <-endGameTurnCalled:
		case <-time.After(time.Second):
			t.Fatal("expected EndGameTurn to be called on the last guess")
		}
		// 3 active, 2 other guessers → totalGuessers=3, coeff=3/3=1, point=10.
		drainPub(t, pubCh, 2)
	})

	t.Run("duplicate correct guess is idempotent (no second score / no guessed notif / no EndGameTurn)", func(t *testing.T) {
		mgr := baseGameRepo()
		mgr.GetPlayersMock = func(ctx context.Context, id string) ([]model.Player, error) {
			return []model.Player{
				{ID: userID, Nickname: "Nick1", State: model.ActivePlayerState},
				{ID: "p-2", Nickname: "Nick2", State: model.ActivePlayerState},
			}, nil
		}
		mgr.GetScoresMock = func(ctx context.Context, id string) ([]model.Score, error) {
			// current user already scored on this turn
			return []model.Score{{PlayerID: userID, TurnID: turnID, Score: 30}}, nil
		}
		mgr.AddScoreMock = func(ctx context.Context, g, u, msg, turn string, point int) error {
			t.Error("AddScore must not be called for a duplicate correct guess")
			return nil
		}
		mur := &repotest.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{ID: userID, Nickname: "Nick1"}, nil
			},
		}
		pubCh := make(chan service.GameNotification, 2)
		mgn := &servicetest.MockGameNotifier{PubMock: func(g, u string, n service.GameNotification) { pubCh <- n }}
		gl := &servicetest.MockGameLoop{
			EndGameTurnMock: func(g string) { t.Error("EndGameTurn must not be called on a duplicate guess") },
		}
		uc, uow := newGuessUsecase(mur, mgr, baseWordRepo(), mgn, gl, nil)

		if err := uc.Guess(context.Background(), gameID, userID, theWord); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !uow.CommitCalled {
			t.Error("expected Commit to be called (SendMessage write must be committed)")
		}
		if mgr.AddScoreCalled {
			t.Error("AddScoreCalled must be false on duplicate guess")
		}
		assertNoPub(t, pubCh)
	})

	t.Run("userRepo.FindByID fails propagates without writes or pub", func(t *testing.T) {
		mgr := baseGameRepo()
		mur := &repotest.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{}, errors.New("user not found")
			},
		}
		pubCh := make(chan service.GameNotification, 1)
		mgn := &servicetest.MockGameNotifier{PubMock: func(g, u string, n service.GameNotification) { pubCh <- n }}
		uc, uow := newGuessUsecase(mur, mgr, baseWordRepo(), mgn, &servicetest.MockGameLoop{}, nil)

		err := uc.Guess(context.Background(), gameID, userID, theWord)
		if err == nil {
			t.Fatal("expected error from FindByID")
		}
		if mgr.SendMessageCalled || uow.CommitCalled || mgr.AddScoreCalled {
			t.Error("no writes expected on FindByID failure")
		}
		assertNoPub(t, pubCh)
	})

	t.Run("GetLatestTurn fails propagates without writes or pub", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				return model.GameTurn{}, errors.New("turn fetch failed")
			},
		}
		mur := &repotest.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{ID: userID, Nickname: "Nick1"}, nil
			},
		}
		pubCh := make(chan service.GameNotification, 1)
		mgn := &servicetest.MockGameNotifier{PubMock: func(g, u string, n service.GameNotification) { pubCh <- n }}
		uc, uow := newGuessUsecase(mur, mgr, baseWordRepo(), mgn, &servicetest.MockGameLoop{}, nil)

		err := uc.Guess(context.Background(), gameID, userID, theWord)
		if err == nil {
			t.Fatal("expected error from GetLatestTurn")
		}
		if mgr.SendMessageCalled || uow.CommitCalled || mgr.AddScoreCalled {
			t.Error("no writes expected on GetLatestTurn failure")
		}
		assertNoPub(t, pubCh)
	})

	t.Run("SendMessage fails propagates without AddScore or pub", func(t *testing.T) {
		mgr := baseGameRepo()
		mgr.SendMessageMock = func(ctx context.Context, g, turn, u, content string) (model.Message, error) {
			return model.Message{}, errors.New("sendmessage failed")
		}
		mur := &repotest.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{ID: userID, Nickname: "Nick1"}, nil
			},
		}
		pubCh := make(chan service.GameNotification, 1)
		mgn := &servicetest.MockGameNotifier{PubMock: func(g, u string, n service.GameNotification) { pubCh <- n }}
		uc, _ := newGuessUsecase(mur, mgr, baseWordRepo(), mgn, &servicetest.MockGameLoop{}, nil)

		err := uc.Guess(context.Background(), gameID, userID, theWord)
		if err == nil {
			t.Fatal("expected error from SendMessage")
		}
		if mgr.AddScoreCalled {
			t.Error("AddScore must not be called when SendMessage fails")
		}
		assertNoPub(t, pubCh)
	})

	t.Run("wrong guess with commit failure does not pub", func(t *testing.T) {
		mgr := baseGameRepo()
		mur := &repotest.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{ID: userID, Nickname: "Nick1"}, nil
			},
		}
		pubCh := make(chan service.GameNotification, 1)
		mgn := &servicetest.MockGameNotifier{PubMock: func(g, u string, n service.GameNotification) { pubCh <- n }}
		commitErr := errors.New("commit failed")
		uc, uow := newGuessUsecase(mur, mgr, baseWordRepo(), mgn, &servicetest.MockGameLoop{}, commitErr)
		_ = uow

		err := uc.Guess(context.Background(), gameID, userID, "nope")
		if !errors.Is(err, commitErr) {
			t.Fatalf("expected commitErr, got %v", err)
		}
		assertNoPub(t, pubCh)
	})

	t.Run("EndGameTurn counts active players only (inactive players do not block turn end)", func(t *testing.T) {
		mgr := baseGameRepo()
		mgr.GetPlayersMock = func(ctx context.Context, id string) ([]model.Player, error) {
			return []model.Player{
				{ID: userID, Nickname: "Nick1", State: model.ActivePlayerState},
				{ID: "p-2", Nickname: "Nick2", State: model.ActivePlayerState},
				{ID: "p-3", Nickname: "Nick3", State: model.ActivePlayerState},
				{ID: "p-gone", Nickname: "Gone", State: model.InactivePlayerState},
			}, nil
		}
		mgr.GetScoresMock = func(ctx context.Context, id string) ([]model.Score, error) {
			return []model.Score{
				{PlayerID: "p-2", TurnID: turnID},
				{PlayerID: "p-3", TurnID: turnID},
			}, nil
		}
		endGameTurnCalled := make(chan struct{}, 1)
		mgr.AddScoreMock = func(ctx context.Context, g, u, msg, turn string, point int) error { return nil }
		mur := &repotest.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				return model.User{ID: userID, Nickname: "Nick1"}, nil
			},
		}
		pubCh := make(chan service.GameNotification, 2)
		mgn := &servicetest.MockGameNotifier{PubMock: func(g, u string, n service.GameNotification) { pubCh <- n }}
		gl := &servicetest.MockGameLoop{
			EndGameTurnMock: func(g string) { endGameTurnCalled <- struct{}{} },
		}
		uc, _ := newGuessUsecase(mur, mgr, baseWordRepo(), mgn, gl, nil)

		if err := uc.Guess(context.Background(), gameID, userID, theWord); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		select {
		case <-endGameTurnCalled:
		case <-time.After(time.Second):
			t.Fatal("expected EndGameTurn to fire even when an inactive player is present")
		}
		drainPub(t, pubCh, 2)
	})
}

// --- T09: TestMessage ---

func TestMessage(t *testing.T) {
	const (
		gameID = "game-1"
		userID = "p-1"
		turnID = "turn-1"
	)
	murFor := func(nick string, err error) *repotest.MockUserRepository {
		return &repotest.MockUserRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.User, error) {
				if err := assertCalledWithError("UserID", userID, id); err != nil {
					t.Error(err)
				}
				return model.User{ID: userID, Nickname: nick}, err
			},
		}
	}

	t.Run("happy path persists and pubs raw content; ParseData round-trips", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				if err := assertCalledWithError("GameID", gameID, id); err != nil {
					t.Error(err)
				}
				return model.GameTurn{ID: turnID}, nil
			},
			SendMessageMock: func(ctx context.Context, g, turn, u, content string) (model.Message, error) {
				if err := assertCalledWithError("GameID", gameID, g); err != nil {
					t.Error(err)
				}
				if err := assertCalledWithError("TurnID", turnID, turn); err != nil {
					t.Error(err)
				}
				if err := assertCalledWithError("UserID", userID, u); err != nil {
					t.Error(err)
				}
				if err := assertCalledWithError("Content", "hello", content); err != nil {
					t.Error(err)
				}
				return model.Message{ID: "m-1"}, nil
			},
		}
		pubCh := make(chan service.GameNotification, 1)
		mgn := &servicetest.MockGameNotifier{PubMock: func(g, u string, n service.GameNotification) {
			if err := assertCalledWithError("GameID", gameID, g); err != nil {
				t.Error(err)
			}
			if err := assertCalledWithError("UserID", userID, u); err != nil {
				t.Error(err)
			}
			pubCh <- n
		}}
		uc := usecase.NewEmojixUsecase(murFor("Nick1", nil), mgr, nil, nil, mgn, &servicetest.MockGameLoop{}, service.NewRealClock())

		if err := uc.Message(context.Background(), gameID, userID, "hello"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !mgr.SendMessageCalled {
			t.Error("expected SendMessage to be called")
		}
		n := drainPub(t, pubCh, 1)[0]
		if n.GetType() != "msg" {
			t.Errorf("pub type: got %q, want msg", n.GetType())
		}
		wantData := userID + ",Nick1,hello"
		if got := n.GetData(); got != wantData {
			t.Errorf("pub data: got %q, want %q", got, wantData)
		}
		// ParseData round-trip.
		parsed := &usecase.GameMsgNotification{}
		if err := parsed.ParseData(n.GetData()); err != nil {
			t.Fatalf("ParseData: %v", err)
		}
		if parsed.UserID != userID || parsed.Nickname != "Nick1" || parsed.Content != "hello" {
			t.Errorf("ParseData round-trip mismatch: %+v", parsed)
		}
	})

	t.Run("content equal to the secret word is published unmasked", func(t *testing.T) {
		// NOTE: Message deliberately does NOT mask chat content, even when it
		// equals the secret word. This is a game-integrity gap (a user can
		// type the literal word and it is broadcast unmasked). Pinning current
		// behavior here; masking is a behavior decision tracked as backlog.
		// TODO(backlog): mask chat content matching the secret word in Message.
		mgr := &repotest.MockGameRepository{
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				return model.GameTurn{ID: turnID, WordID: "w-1"}, nil
			},
			SendMessageMock: func(ctx context.Context, g, turn, u, content string) (model.Message, error) {
				return model.Message{ID: "m-1"}, nil
			},
		}
		// Provided for documentation; Message does not consult the word repo.
		mwr := &repotest.MockWordRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.Word, error) {
				return model.Word{ID: "w-1", Word: "Secret"}, nil
			},
		}
		pubCh := make(chan service.GameNotification, 1)
		mgn := &servicetest.MockGameNotifier{PubMock: func(g, u string, n service.GameNotification) { pubCh <- n }}
		uc := usecase.NewEmojixUsecase(murFor("Nick1", nil), mgr, mwr, nil, mgn, &servicetest.MockGameLoop{}, service.NewRealClock())

		if err := uc.Message(context.Background(), gameID, userID, "Secret"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		n := drainPub(t, pubCh, 1)[0]
		if got := n.GetData(); got != userID+",Nick1,Secret" {
			t.Errorf("expected unmasked secret word in pub data, got %q", got)
		}
	})

	t.Run("GetLatestTurn fails propagates without SendMessage or pub", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				return model.GameTurn{}, errors.New("turn failed")
			},
			SendMessageMock: func(ctx context.Context, g, turn, u, content string) (model.Message, error) {
				t.Error("SendMessage must not be called on GetLatestTurn failure")
				return model.Message{}, nil
			},
		}
		pubCh := make(chan service.GameNotification, 1)
		mgn := &servicetest.MockGameNotifier{PubMock: func(g, u string, n service.GameNotification) { pubCh <- n }}
		uc := usecase.NewEmojixUsecase(murFor("Nick1", nil), mgr, nil, nil, mgn, &servicetest.MockGameLoop{}, service.NewRealClock())

		err := uc.Message(context.Background(), gameID, userID, "hello")
		if err == nil {
			t.Fatal("expected error from GetLatestTurn")
		}
		if mgr.SendMessageCalled {
			t.Error("SendMessage must not be called")
		}
		assertNoPub(t, pubCh)
	})

	t.Run("userRepo.FindByID fails propagates without SendMessage or pub", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				return model.GameTurn{ID: turnID}, nil
			},
			SendMessageMock: func(ctx context.Context, g, turn, u, content string) (model.Message, error) {
				t.Error("SendMessage must not be called on FindByID failure")
				return model.Message{}, nil
			},
		}
		pubCh := make(chan service.GameNotification, 1)
		mgn := &servicetest.MockGameNotifier{PubMock: func(g, u string, n service.GameNotification) { pubCh <- n }}
		uc := usecase.NewEmojixUsecase(murFor("", errors.New("user not found")), mgr, nil, nil, mgn, &servicetest.MockGameLoop{}, service.NewRealClock())

		err := uc.Message(context.Background(), gameID, userID, "hello")
		if err == nil {
			t.Fatal("expected error from FindByID")
		}
		if mgr.SendMessageCalled {
			t.Error("SendMessage must not be called")
		}
		assertNoPub(t, pubCh)
	})

	t.Run("SendMessage fails propagates without pub", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				return model.GameTurn{ID: turnID}, nil
			},
			SendMessageMock: func(ctx context.Context, g, turn, u, content string) (model.Message, error) {
				return model.Message{}, errors.New("send failed")
			},
		}
		pubCh := make(chan service.GameNotification, 1)
		mgn := &servicetest.MockGameNotifier{PubMock: func(g, u string, n service.GameNotification) { pubCh <- n }}
		uc := usecase.NewEmojixUsecase(murFor("Nick1", nil), mgr, nil, nil, mgn, &servicetest.MockGameLoop{}, service.NewRealClock())

		err := uc.Message(context.Background(), gameID, userID, "hello")
		if err == nil {
			t.Fatal("expected error from SendMessage")
		}
		assertNoPub(t, pubCh)
	})
}

// --- T10: TestLeaderboard + TestGameWord ---

func TestLeaderboard(t *testing.T) {
	const gameID = "game-1"

	t.Run("happy path aggregates scores and marks Me/guessed on latest turn", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
				return []model.Player{
					{ID: "p-1", Nickname: "Nick1", State: model.ActivePlayerState},
					{ID: "p-2", Nickname: "Nick2", State: model.ActivePlayerState},
					{ID: "p-3", Nickname: "Nick3", State: model.ActivePlayerState},
				}, nil
			},
			GetScoresMock: func(ctx context.Context, id string) ([]model.Score, error) {
				return []model.Score{
					{PlayerID: "p-1", Score: 10, TurnID: "older"},
					{PlayerID: "p-1", Score: 5, TurnID: "latest"},
					{PlayerID: "p-2", Score: 5, TurnID: "latest"},
				}, nil
			},
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				return model.GameTurn{ID: "latest"}, nil
			},
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, nil, nil, nil, &servicetest.MockGameLoop{}, service.NewRealClock())

		entries, err := uc.Leaderboard(context.Background(), gameID, "p-1")
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 3 {
			t.Fatalf("entries: got %d, want 3", len(entries))
		}
		want := []model.LeaderboardEntry{
			{PlayerID: "p-1", Nickname: "Nick1", Me: true, GuessedWord: true, Score: 15},
			{PlayerID: "p-2", Nickname: "Nick2", Me: false, GuessedWord: true, Score: 5},
			{PlayerID: "p-3", Nickname: "Nick3", Me: false, GuessedWord: false, Score: 0},
		}
		for i, e := range entries {
			if e != want[i] {
				t.Errorf("entry[%d]: got %+v, want %+v", i, e, want[i])
			}
		}
	})

	t.Run("user not in game returns error and empty slice", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
				return []model.Player{
					{ID: "p-1", Nickname: "Nick1", State: model.ActivePlayerState},
					{ID: "p-2", Nickname: "Nick2", State: model.ActivePlayerState},
				}, nil
			},
			GetScoresMock:     func(ctx context.Context, id string) ([]model.Score, error) { return nil, nil },
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) { return model.GameTurn{ID: "latest"}, nil },
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, nil, nil, nil, &servicetest.MockGameLoop{}, service.NewRealClock())

		entries, err := uc.Leaderboard(context.Background(), gameID, "p-3")
		if err == nil {
			t.Fatal("expected error for a user not in the game")
		}
		if len(entries) != 0 {
			t.Errorf("expected empty slice on error, got %d entries", len(entries))
		}
	})

	t.Run("inactive players are excluded", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
				return []model.Player{
					{ID: "p-1", Nickname: "Nick1", State: model.ActivePlayerState},
					{ID: "p-2", Nickname: "Nick2", State: model.InactivePlayerState},
				}, nil
			},
			GetScoresMock:     func(ctx context.Context, id string) ([]model.Score, error) { return nil, nil },
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) { return model.GameTurn{ID: "latest"}, nil },
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, nil, nil, nil, &servicetest.MockGameLoop{}, service.NewRealClock())

		entries, err := uc.Leaderboard(context.Background(), gameID, "p-1")
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 {
			t.Fatalf("entries: got %d, want 1 (inactive excluded)", len(entries))
		}
		if entries[0].PlayerID != "p-1" {
			t.Errorf("entry: got %q, want p-1", entries[0].PlayerID)
		}
	})

	t.Run("GetPlayers fails propagates", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
				return nil, errors.New("players failed")
			},
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, nil, nil, nil, &servicetest.MockGameLoop{}, service.NewRealClock())
		_, err := uc.Leaderboard(context.Background(), gameID, "p-1")
		if err == nil {
			t.Fatal("expected error from GetPlayers")
		}
	})

	t.Run("GetScores fails propagates", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
				return []model.Player{{ID: "p-1", Nickname: "Nick1", State: model.ActivePlayerState}}, nil
			},
			GetScoresMock: func(ctx context.Context, id string) ([]model.Score, error) {
				return nil, errors.New("scores failed")
			},
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, nil, nil, nil, &servicetest.MockGameLoop{}, service.NewRealClock())
		_, err := uc.Leaderboard(context.Background(), gameID, "p-1")
		if err == nil {
			t.Fatal("expected error from GetScores")
		}
	})

	t.Run("GetLatestTurn fails propagates", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetPlayersMock: func(ctx context.Context, id string) ([]model.Player, error) {
				return []model.Player{{ID: "p-1", Nickname: "Nick1", State: model.ActivePlayerState}}, nil
			},
			GetScoresMock: func(ctx context.Context, id string) ([]model.Score, error) { return nil, nil },
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				return model.GameTurn{}, errors.New("turn failed")
			},
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, nil, nil, nil, &servicetest.MockGameLoop{}, service.NewRealClock())
		_, err := uc.Leaderboard(context.Background(), gameID, "p-1")
		if err == nil {
			t.Fatal("expected error from GetLatestTurn")
		}
	})
}

func TestGameWord(t *testing.T) {
	const (
		gameID = "game-1"
		userID = "p-1"
		wordID = "w-1"
		turnID = "t-1"
	)
	wordRepoFor := func(w model.Word) *repotest.MockWordRepository {
		return &repotest.MockWordRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.Word, error) {
				if err := assertCalledWithError("WordID", wordID, id); err != nil {
					t.Error(err)
				}
				return w, nil
			},
		}
	}

	t.Run("not guessed returns masked word", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				return model.GameTurn{ID: turnID, WordID: wordID}, nil
			},
			GetScoresMock: func(ctx context.Context, id string) ([]model.Score, error) {
				// a score for another player, but not the current user
				return []model.Score{{PlayerID: "p-2", TurnID: turnID}}, nil
			},
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, wordRepoFor(model.Word{ID: wordID, Word: "Secret"}), nil, nil, &servicetest.MockGameLoop{}, service.NewRealClock())

		got, err := uc.GameWord(context.Background(), gameID, userID)
		if err != nil {
			t.Fatal(err)
		}
		if want := "******"; got != want {
			t.Errorf("masked word: got %q, want %q", got, want)
		}
	})

	t.Run("guessed returns raw word", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				return model.GameTurn{ID: turnID, WordID: wordID}, nil
			},
			GetScoresMock: func(ctx context.Context, id string) ([]model.Score, error) {
				return []model.Score{{PlayerID: userID, TurnID: turnID}}, nil
			},
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, wordRepoFor(model.Word{ID: wordID, Word: "Secret"}), nil, nil, &servicetest.MockGameLoop{}, service.NewRealClock())

		got, err := uc.GameWord(context.Background(), gameID, userID)
		if err != nil {
			t.Fatal(err)
		}
		if got != "Secret" {
			t.Errorf("raw word: got %q, want %q", got, "Secret")
		}
	})

	t.Run("non-word chars leak unmasked through the regex", func(t *testing.T) {
		// The mask regex is `\w`, so spaces/punctuation/emoji are not replaced.
		// This documents the limitation; a better masking scheme is backlog.
		// TODO(backlog): replace the `\w`-based mask with a scheme that hides
		// the whole word regardless of character class.
		mgr := &repotest.MockGameRepository{
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				return model.GameTurn{ID: turnID, WordID: wordID}, nil
			},
			GetScoresMock: func(ctx context.Context, id string) ([]model.Score, error) { return nil, nil },
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, wordRepoFor(model.Word{ID: wordID, Word: "Hi 👋"}), nil, nil, &servicetest.MockGameLoop{}, service.NewRealClock())

		got, err := uc.GameWord(context.Background(), gameID, userID)
		if err != nil {
			t.Fatal(err)
		}
		if want := "** 👋"; got != want {
			t.Errorf("masked word: got %q, want %q", got, want)
		}
	})

	t.Run("GetLatestTurn fails returns empty string and error", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				return model.GameTurn{}, errors.New("turn failed")
			},
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, nil, nil, nil, &servicetest.MockGameLoop{}, service.NewRealClock())
		got, err := uc.GameWord(context.Background(), gameID, userID)
		if err == nil {
			t.Fatal("expected error from GetLatestTurn")
		}
		if got != "" {
			t.Errorf("expected empty string on error, got %q", got)
		}
	})

	t.Run("wordRepo.FindByID fails returns empty string and error", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				return model.GameTurn{ID: turnID, WordID: wordID}, nil
			},
		}
		mwr := &repotest.MockWordRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.Word, error) {
				return model.Word{}, errors.New("word failed")
			},
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, mwr, nil, nil, &servicetest.MockGameLoop{}, service.NewRealClock())
		got, err := uc.GameWord(context.Background(), gameID, userID)
		if err == nil {
			t.Fatal("expected error from FindByID")
		}
		if got != "" {
			t.Errorf("expected empty string on error, got %q", got)
		}
	})

	t.Run("GetScores fails returns empty string and error", func(t *testing.T) {
		mgr := &repotest.MockGameRepository{
			GetLatestTurnMock: func(ctx context.Context, id string) (model.GameTurn, error) {
				return model.GameTurn{ID: turnID, WordID: wordID}, nil
			},
			GetScoresMock: func(ctx context.Context, id string) ([]model.Score, error) {
				return nil, errors.New("scores failed")
			},
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, wordRepoFor(model.Word{ID: wordID, Word: "Secret"}), nil, nil, &servicetest.MockGameLoop{}, service.NewRealClock())
		got, err := uc.GameWord(context.Background(), gameID, userID)
		if err == nil {
			t.Fatal("expected error from GetScores")
		}
		if got != "" {
			t.Errorf("expected empty string on error, got %q", got)
		}
	})
}

// --- T12: TestOnTurnEnd ---

// driveClock repeatedly advances the fake clock so any clock.After timer the
// handler registers (before or after an Advance call) is fired on a subsequent
// iteration. This sidesteps the registration race (handler calls clock.After
// inside its own goroutine) without time.Sleep. done closes when the handler
// returns.
func driveClock(t *testing.T, fc *servicetest.FakeClock, done <-chan struct{}) {
	t.Helper()
	for i := 0; i < 10000; i++ {
		select {
		case <-done:
			return
		default:
		}
		fc.Advance(time.Hour) // fire any registered timer regardless of its duration
		runtime.Gosched()
	}
	t.Fatal("onTurnEnd did not finish in time")
}

func TestOnTurnEnd(t *testing.T) {
	const gameID = "game-1"

	// buildOnTurnEndUsecase wires the common mocks for onTurnEnd tests. The
	// pubAll/stop signals and counters are returned for assertions.
	type onTurnEndMocks struct {
		pubAllCount  int
		getAllCount  int
		addTurnCount int
		stopCount    int
		stopGameIDs  []string
		pubAllCh     chan struct{}
		stopCh       chan string
	}

	newUsecase := func(t *testing.T, addTurnFn func(call int) (model.GameTurn, error), getAllFn func(call int) ([]model.Word, error), stopAtAll bool) (*servicetest.MockGameLoop, *servicetest.FakeClock, *onTurnEndMocks) {
		t.Helper()
		m := &onTurnEndMocks{
			pubAllCh: make(chan struct{}, 4),
			stopCh:   make(chan string, 4),
		}
		mgr := &repotest.MockGameRepository{
			AddTurnMock: func(ctx context.Context, g, w string) (model.GameTurn, error) {
				m.addTurnCount++
				return addTurnFn(m.addTurnCount)
			},
		}
		mwr := &repotest.MockWordRepository{
			GetAllMock: func(ctx context.Context) ([]model.Word, error) {
				m.getAllCount++
				return getAllFn(m.getAllCount)
			},
		}
		mgn := &servicetest.MockGameNotifier{
			PubAllMock: func(g string, n service.GameNotification) {
				if err := assertCalledWithError("GameID", gameID, g); err != nil {
					t.Error(err)
				}
				if n.GetType() != "turnended" {
					t.Errorf("PubAll notif type: got %q, want turnended", n.GetType())
				}
				m.pubAllCount++
				m.pubAllCh <- struct{}{}
			},
		}
		gl := &servicetest.MockGameLoop{
			StopGameMock: func(g string) {
				m.stopCount++
				m.stopGameIDs = append(m.stopGameIDs, g)
				m.stopCh <- g
			},
		}
		clock := servicetest.NewFakeClock()
		uc := usecase.NewEmojixUsecase(nil, mgr, mwr, nil, mgn, gl, clock)
		_ = uc // NewEmojixUsecase installs the OnTurnEndHandler on gl
		return gl, clock, m
	}

	// runHandler spawns the captured handler (installed by NewEmojixUsecase) in
	// a goroutine and drives the fake clock until it returns.
	runHandler := func(t *testing.T, gl *servicetest.MockGameLoop, clock *servicetest.FakeClock) {
		t.Helper()
		done := make(chan struct{})
		go func() {
			defer close(done)
			gl.FireOnTurnEnd(context.Background(), gameID)
		}()
		driveClock(t, clock, done)
	}

	t.Run("happy path: one PubAll, one AddTurn, no StopGame", func(t *testing.T) {
		gl, clock, m := newUsecase(t,
			func(call int) (model.GameTurn, error) { return model.GameTurn{ID: "t-1"}, nil },
			func(call int) ([]model.Word, error) { return []model.Word{{ID: "w-1", Word: "Alpha"}}, nil },
			false,
		)
		runHandler(t, gl, clock)

		if m.pubAllCount != 1 {
			t.Errorf("PubAll count: got %d, want 1", m.pubAllCount)
		}
		if m.getAllCount != 1 {
			t.Errorf("GetAll count: got %d, want 1", m.getAllCount)
		}
		if m.addTurnCount != 1 {
			t.Errorf("AddTurn count: got %d, want 1", m.addTurnCount)
		}
		if m.stopCount != 0 {
			t.Errorf("StopGame count: got %d, want 0", m.stopCount)
		}
	})

	t.Run("first AddTurn fails, retry succeeds: two AddTurn, no StopGame", func(t *testing.T) {
		gl, clock, m := newUsecase(t,
			func(call int) (model.GameTurn, error) {
				if call == 1 {
					return model.GameTurn{}, errors.New("addturn failed")
				}
				return model.GameTurn{ID: "t-1"}, nil
			},
			func(call int) ([]model.Word, error) { return []model.Word{{ID: "w-1", Word: "Alpha"}}, nil },
			false,
		)
		runHandler(t, gl, clock)

		if m.pubAllCount != 1 {
			t.Errorf("PubAll count: got %d, want 1", m.pubAllCount)
		}
		if m.addTurnCount != 2 {
			t.Errorf("AddTurn count: got %d, want 2", m.addTurnCount)
		}
		if m.getAllCount != 2 {
			t.Errorf("GetAll count: got %d, want 2", m.getAllCount)
		}
		if m.stopCount != 0 {
			t.Errorf("StopGame count: got %d, want 0", m.stopCount)
		}
	})

	t.Run("both retries fail: one StopGame(gameID)", func(t *testing.T) {
		gl, clock, m := newUsecase(t,
			func(call int) (model.GameTurn, error) { return model.GameTurn{}, errors.New("addturn failed") },
			func(call int) ([]model.Word, error) { return []model.Word{{ID: "w-1", Word: "Alpha"}}, nil },
			true,
		)
		runHandler(t, gl, clock)

		if m.pubAllCount != 1 {
			t.Errorf("PubAll count: got %d, want 1", m.pubAllCount)
		}
		if m.addTurnCount != 2 {
			t.Errorf("AddTurn count: got %d, want 2", m.addTurnCount)
		}
		if m.stopCount != 1 {
			t.Errorf("StopGame count: got %d, want 1", m.stopCount)
		}
		if len(m.stopGameIDs) != 1 || m.stopGameIDs[0] != gameID {
			t.Errorf("StopGame gameID: got %v, want [%s]", m.stopGameIDs, gameID)
		}
	})

	t.Run("empty word list: two GetAll, zero AddTurn, one StopGame", func(t *testing.T) {
		// Exercises the T07 ErrNoWords guard inside onTurnEnd: pickGameWord
		// would panic on Intn(0) without the guard, so this stays a clean
		// error path that retries once then stops the game.
		gl, clock, m := newUsecase(t,
			func(call int) (model.GameTurn, error) {
				t.Error("AddTurn must not be called when there are no words")
				return model.GameTurn{}, nil
			},
			func(call int) ([]model.Word, error) { return []model.Word{}, nil },
			true,
		)
		runHandler(t, gl, clock)

		if m.pubAllCount != 1 {
			t.Errorf("PubAll count: got %d, want 1", m.pubAllCount)
		}
		if m.getAllCount != 2 {
			t.Errorf("GetAll count: got %d, want 2", m.getAllCount)
		}
		if m.addTurnCount != 0 {
			t.Errorf("AddTurn count: got %d, want 0", m.addTurnCount)
		}
		if m.stopCount != 1 {
			t.Errorf("StopGame count: got %d, want 1", m.stopCount)
		}
	})
}
