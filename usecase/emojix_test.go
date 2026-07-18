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
			&service.MockGameLoop{},
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
			&service.MockGameLoop{},
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
		mgr := &repository.MockGameRepository{
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
		mwr := &repository.MockWordRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.Word, error) {
				return model.Word{ID: "some-word-id", Word: "Some Word", Hint: "Some Hint"}, nil
			},
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, mwr, nil, nil, &service.MockGameLoop{}, service.NewRealClock())

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
		mgr := &repository.MockGameRepository{
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
		mwr := &repository.MockWordRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.Word, error) {
				return model.Word{ID: "some-word-id", Word: "Some Word", Hint: "Some Hint"}, nil
			},
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, mwr, nil, nil, &service.MockGameLoop{}, service.NewRealClock())

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
		mgr := &repository.MockGameRepository{
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
		mwr := &repository.MockWordRepository{
			FindByIDMock: func(ctx context.Context, id string) (model.Word, error) {
				return model.Word{ID: "some-word-id", Word: "Some Word", Hint: "Some Hint"}, nil
			},
		}
		uc := usecase.NewEmojixUsecase(nil, mgr, mwr, nil, nil, &service.MockGameLoop{}, service.NewRealClock())

		gameState, err := uc.GameState(context.Background(), "some-game-id", "p-1")
		if err != nil {
			t.Fatal(err)
		}
		if !gameState.TurnEnded {
			t.Errorf("expected TurnEnded to be true when all active players guessed, got false")
		}
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
			&service.MockGameLoop{},
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
	ch := make(chan service.GameNotification)
	cleanupCalled := false
	mgn := &service.MockGameNotifier{
		SubMock: func(gameID, userID string) (chan service.GameNotification, func()) {
			err := assertCalledWithError("GameID", "some-game-id", gameID)
			if err != nil {
				t.Error(err)
			}

			err = assertCalledWithError("UserID", "some-user-id", userID)
			if err != nil {
				t.Error(err)
			}

			return ch, func() {

				cleanupCalled = true
			}
		},
	}
	emojixUsecase := usecase.NewEmojixUsecase(
		nil,
		nil,
		nil,
		nil,
		mgn,
		&service.MockGameLoop{},
		service.NewRealClock(),
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

	if cleanupCalled != true {
		t.Error("expected to unsubscribe")
	}
}

func TestGameState_TurnTimedOut(t *testing.T) {
	clock := service.NewFakeClock()
	turnStartedAt := clock.Now()

	mgr := &repository.MockGameRepository{
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

	mwr := &repository.MockWordRepository{
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
		&service.MockGameLoop{},
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

func newInitGameUsecase(t *testing.T, mur repository.UserRepository, mgr *repository.MockGameRepository, mwr *repository.MockWordRepository, gl *service.MockGameLoop, commitErr error, newErr error) (usecase.EmojixUsecase, *repository.MockUnitOfWork) {
	t.Helper()
	uow := &repository.MockUnitOfWork{
		GameRepositoryMock: mgr,
		CommitMock:          func() error { return commitErr },
		RollbackMock:        func() error { return nil },
	}
	factory := &repository.MockUnitOfWorkFactory{
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
		mgr := &repository.MockGameRepository{
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
		mwr := &repository.MockWordRepository{
			GetAllMock: func(ctx context.Context) ([]model.Word, error) {
				return []model.Word{{ID: "w1", Word: "Alpha"}, {ID: "w2", Word: "Beta"}}, nil
			},
		}

		committed := false
		var startGameID string
		var startDur time.Duration
		startCalled := make(chan struct{}, 1)
		gl := &service.MockGameLoop{
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
		mgr := &repository.MockGameRepository{}
		mwr := &repository.MockWordRepository{}
		gl := &service.MockGameLoop{}
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
		mgr := &repository.MockGameRepository{
			CreateMock: func(ctx context.Context) (model.Game, error) {
				return model.Game{}, errors.New("create failed")
			},
		}
		mwr := &repository.MockWordRepository{}
		gl := &service.MockGameLoop{}
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
		mgr := &repository.MockGameRepository{
			CreateMock: func(ctx context.Context) (model.Game, error) {
				return model.Game{ID: "game-2"}, nil
			},
			AddPlayerMock: func(ctx context.Context, gameID, playerID string) error { return nil },
			AddTurnMock: func(ctx context.Context, gameID, wordID string) (model.GameTurn, error) {
				return model.GameTurn{}, errors.New("addturn failed")
			},
		}
		mwr := &repository.MockWordRepository{
			GetAllMock: func(ctx context.Context) ([]model.Word, error) {
				return []model.Word{{ID: "w1", Word: "Alpha"}}, nil
			},
		}
		gl := &service.MockGameLoop{}
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
		mgr := &repository.MockGameRepository{
			CreateMock: func(ctx context.Context) (model.Game, error) {
				return model.Game{ID: "game-3"}, nil
			},
			AddPlayerMock: func(ctx context.Context, gameID, playerID string) error { return nil },
			AddTurnMock: func(ctx context.Context, gameID, wordID string) (model.GameTurn, error) {
				t.Error("AddTurn must not be called when there are no words")
				return model.GameTurn{}, nil
			},
		}
		mwr := &repository.MockWordRepository{
			GetAllMock: func(ctx context.Context) ([]model.Word, error) {
				return []model.Word{}, nil
			},
		}
		gl := &service.MockGameLoop{}
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
		mur := &repository.MockUserRepository{
			CreateOrUpdateMock: func(ctx context.Context, id string, params repository.UserCreateOrUpdateParams) error {
				gotID = id
				gotNick = params.Nickname
				createCalled <- struct{}{}
				return nil
			},
		}
		uc := usecase.NewEmojixUsecase(mur, nil, nil, nil, nil, &service.MockGameLoop{}, service.NewRealClock())

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
		mur := &repository.MockUserRepository{
			CreateOrUpdateMock: func(ctx context.Context, id string, params repository.UserCreateOrUpdateParams) error {
				return errors.New("persist failed")
			},
		}
		uc := usecase.NewEmojixUsecase(mur, nil, nil, nil, nil, &service.MockGameLoop{}, service.NewRealClock())

		_, err := uc.InitUser(context.Background())
		if err == nil {
			t.Fatal("expected error from CreateOrUpdate")
		}
	})
}
