package repository

import (
	"context"
	"database/sql"
	"emojix/model"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func InitTestDB() (*sql.DB, error) {
	db, err := InitSqliteDB(":memory:")
	if err != nil {
		return nil, err
	}

	migrator, err := NewSQLiteMigrator(db, ":memory:", "../database/migrations")
	if err != nil {
		return nil, err
	}

	err = migrator.UpCmd()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func TestUserRepository(t *testing.T) {
	db, err := InitTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	cleanupDB := func() {
		_, err := db.Exec("DELETE FROM users;")
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Run("FindByID", func(t *testing.T) {
		cleanupDB()

		now := time.Now()
		now = time.UnixMicro(now.UnixMicro())
		_, err := db.Exec("INSERT INTO users (id, nickname, created_at, updated_at) VALUES ('some-id', 'some-nickname', ?, ?);", now.UnixMicro(), now.UnixMicro())
		if err != nil {
			t.Fatal(err)
		}

		user, err := repo.FindByID(context.Background(), "some-id")
		if err != nil {
			t.Fatal(err)
		}

		if user.ID != "some-id" {
			t.Errorf("expected id %s but got %s", "some-id", user.ID)
		}

		if user.Nickname != "some-nickname" {
			t.Errorf("expected nickname %s but got %s", "some-nickname", user.Nickname)
		}

		if user.CreatedAt.Compare(now) != 0 {
			t.Errorf("expected created_at %v but got %v", now, user.CreatedAt)
		}

		if user.UpdatedAt.Compare(now) != 0 {
			t.Errorf("expected updated_at %v but got %v", now, user.UpdatedAt)
		}
	})

	t.Run("CreateOrUpdate", func(t *testing.T) {
		t.Run("Update", func(t *testing.T) {
			cleanupDB()
			now := time.Now()
			now = time.UnixMicro(now.UnixMicro())
			_, err := db.Exec("INSERT INTO users (id, nickname, created_at, updated_at) VALUES ('some-id', 'some-nickname', ?, ?)", now.UnixMicro(), now.UnixMicro())
			if err != nil {
				t.Fatal(err)
			}

			userID := "some-id"
			params := UserCreateOrUpdateParams{
				Nickname: "new-nickname",
			}
			err = repo.CreateOrUpdate(context.Background(), userID, params)
			if err != nil {
				t.Fatal(err)
			}

			user, err := repo.FindByID(context.Background(), "some-id")
			if err != nil {
				t.Fatal(err)
			}

			if user.ID != "some-id" {
				t.Errorf("expected id %s but got %s", "some_id", user.ID)
			}

			if user.Nickname != "new-nickname" {
				t.Errorf("expected nickname %s but got %s", "new-nickname", user.Nickname)
			}

			if user.CreatedAt.Compare(now) != 0 {
				t.Errorf("expected created_at %v but got %v", now, user.CreatedAt)
			}

			// updates nickname
			if user.UpdatedAt.Compare(now) != 1 {
				t.Errorf("expected updated_at after %v but got %v", now, user.UpdatedAt)
			}
		})

		t.Run("Create", func(t *testing.T) {
			cleanupDB()

			now := time.Now()
			userID := "some-id"
			params := UserCreateOrUpdateParams{
				Nickname: "some-nickname",
			}
			err = repo.CreateOrUpdate(context.Background(), userID, params)
			if err != nil {
				t.Fatal(err)
			}

			user, err := repo.FindByID(context.Background(), "some-id")
			if err != nil {
				t.Fatal(err)
			}

			if user.ID != "some-id" {
				t.Errorf("expected id %s but got %s", "some-id", user.ID)
			}

			if user.Nickname != "some-nickname" {
				t.Errorf("expected nickname %s but got %s", "some-nickname", user.Nickname)
			}

			if user.CreatedAt.Compare(now) != 1 {
				t.Errorf("expected created_at after %v but got %v", now, user.CreatedAt)
			}

			// updates nickname
			if user.UpdatedAt.Compare(now) != 1 {
				t.Errorf("expected updated_at after %v but got %v", now, user.UpdatedAt)
			}
		})
	})
}

func TestWordRepository(t *testing.T) {
	db, err := InitTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewWordRepository(db)

	cleanupDB := func() {
		_, err := db.Exec("DELETE FROM words;")
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Run("GetAll", func(t *testing.T) {
		cleanupDB()

		words, err := repo.GetAll(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if len(words) != 0 {
			t.Errorf("expected 0 words but got %d", len(words))
		}

		_, err = db.Exec("INSERT INTO words (id, word, hint) VALUES ('1', 'word-1', 'hint-1'), ('2', 'word-2', 'hint-2');")
		if err != nil {
			t.Fatal(err)
		}

		words, err = repo.GetAll(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(words) != 2 {
			t.Errorf("expected 1 word but got %d", len(words))
		}

		// first word
		if words[0].ID != "1" {
			t.Errorf("expected id %s but got %s", "1", words[0].ID)
		}
		if words[0].Word != "word-1" {
			t.Errorf("expected word %s but got %s", "word-1", words[0].Word)
		}
		if words[0].Hint != "hint-1" {
			t.Errorf("expected hint %s but got %s", "hint-1", words[0].Hint)
		}

		// second word
		if words[1].ID != "2" {
			t.Errorf("expected id %s but got %s", "2", words[1].ID)
		}
		if words[1].Word != "word-2" {
			t.Errorf("expected word %s but got %s", "word-2", words[1].Word)
		}
		if words[1].Hint != "hint-2" {
			t.Errorf("expected hint %s but got %s", "hint-3", words[1].Hint)
		}
	})

	t.Run("FindByID", func(t *testing.T) {
		cleanupDB()

		_, err := db.Exec("INSERT INTO words (id, word, hint) VALUES ('1', 'word-1', 'hint-1');")
		if err != nil {
			t.Fatal(err)
		}

		word, err := repo.FindByID(context.Background(), "1")
		if err != nil {
			t.Fatal(err)
		}

		if word.ID != "1" {
			t.Errorf("expected id %s but got %s", "1", word.ID)
		}
		if word.Word != "word-1" {
			t.Errorf("expected word %s but got %s", "word-1", word.Word)
		}
		if word.Hint != "hint-1" {
			t.Errorf("expected hint %s but got %s", "hint-1", word.Hint)
		}
	})
}

func TestGameRepository(t *testing.T) {
	db, err := InitTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewGameRepository(db)

	cleanupDB := func() {
		_, err := db.Exec(`
			DELETE FROM games;
			DELETE FROM game_turns;
			DELETE FROM game_scores;
			DELETE FROM players;
			DELETE FROM users;
			DELETE FROM messages;
		`)
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Run("FindByID", func(t *testing.T) {
		cleanupDB()

		now := time.Now()
		now = time.UnixMicro(now.UnixMicro())
		_, err := db.Exec("INSERT INTO games (id, created_at, updated_at) VALUES ('game-id', ?, ?);", now.UnixMicro(), now.UnixMicro())
		if err != nil {
			t.Fatal(err)
		}

		game, err := repo.FindByID(context.Background(), "game-id")
		if err != nil {
			t.Fatal(err)
		}

		if game.ID != "game-id" {
			t.Errorf("expected id %s but got %s", "game-id", game.ID)
		}

		if game.CreatedAt.Compare(now) != 0 {
			t.Errorf("expected created_at %v but got %v", now, game.CreatedAt)
		}

		if game.UpdatedAt.Compare(now) != 0 {
			t.Errorf("expected updated_at %v but got %v", now, game.UpdatedAt)
		}
	})
	t.Run("Create", func(t *testing.T) {
		cleanupDB()

		now := time.Now()
		game, err := repo.Create(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if game.ID == "" {
			t.Errorf("expected id not to be empty but got %s", game.ID)
		}

		if game.CreatedAt.Compare(now) != 1 {
			t.Errorf("expected created_at after %v but got %v", now, game.CreatedAt)
		}

		if game.UpdatedAt.Compare(now) != 1 {
			t.Errorf("expected updated_at after %v but got %v", now, game.UpdatedAt)
		}
	})
	t.Run("AddPlayer", func(t *testing.T) {
		cleanupDB()

		now := time.Now()
		game, err := repo.Create(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec("INSERT INTO users (id, nickname, created_at, updated_at) VALUES ('user-id', 'user-nickname', ?, ?);", now.UnixMicro(), now.UnixMicro())
		if err != nil {
			t.Fatal(err)
		}

		err = repo.AddPlayer(context.Background(), game.ID, "user-id")
		if err != nil {
			t.Fatal(err)
		}

		players, err := repo.GetPlayers(context.Background(), game.ID)
		if err != nil {
			t.Fatal(err)
		}

		if len(players) != 1 {
			t.Errorf("expected 1 player but got %d", len(players))
		}

		// first player
		if players[0].ID != "user-id" {
			t.Errorf("expected id %s but got %s", "user-id", players[0].ID)
		}
		if players[0].Nickname != "user-nickname" {
			t.Errorf("expected nickname %s but got %s", "user-nickname", players[0].Nickname)
		}

		if players[0].State != "active" {
			t.Errorf("expected nickname %s but got %s", "active", players[0].State)
		}

		if players[0].JoinedAt.Compare(now) != 1 {
			t.Errorf("expected joined_at after %v but got %v", now, players[0].JoinedAt)
		}
	})

	t.Run("SetPlayerState", func(t *testing.T) {
		cleanupDB()

		now := time.Now()
		game, err := repo.Create(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec("INSERT INTO users (id, nickname, created_at, updated_at) VALUES ('user-id', 'user-nickname', ?, ?);", now.UnixMicro(), now.UnixMicro())
		if err != nil {
			t.Fatal(err)
		}

		err = repo.AddPlayer(context.Background(), game.ID, "user-id")
		if err != nil {
			t.Fatal(err)
		}

		err = repo.SetPlayerState(context.Background(), game.ID, "user-id", model.InactivePlayerState)
		if err != nil {
			t.Fatal(err)
		}

		players, err := repo.GetPlayers(context.Background(), game.ID)
		if err != nil {
			t.Fatal(err)
		}

		if len(players) != 1 {
			t.Errorf("expected 1 player but got %d", len(players))
		}

		// first player
		if players[0].ID != "user-id" {
			t.Errorf("expected id %s but got %s", "user-id", players[0].ID)
		}
		if players[0].Nickname != "user-nickname" {
			t.Errorf("expected nickname %s but got %s", "user-nickname", players[0].Nickname)
		}

		if players[0].State != "inactive" {
			t.Errorf("expected player state %s but got %s", "inactive", players[0].State)
		}

		if players[0].JoinedAt.Compare(now) != 1 {
			t.Errorf("expected joined_at after %v but got %v", now, players[0].JoinedAt)
		}

	})
	t.Run("GetPlayers", func(t *testing.T) {
		cleanupDB()

		now := time.Now()
		game, err := repo.Create(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec("INSERT INTO users (id, nickname, created_at, updated_at) VALUES ('user-id', 'user-nickname', ?, ?);", now.UnixMicro(), now.UnixMicro())
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec("INSERT INTO users (id, nickname, created_at, updated_at) VALUES ('user-id-2', 'user-nickname-2', ?, ?);", now.UnixMicro(), now.UnixMicro())
		if err != nil {
			t.Fatal(err)
		}

		err = repo.AddPlayer(context.Background(), game.ID, "user-id")
		if err != nil {
			t.Fatal(err)
		}

		err = repo.AddPlayer(context.Background(), game.ID, "user-id-2")
		if err != nil {
			t.Fatal(err)
		}

		players, err := repo.GetPlayers(context.Background(), game.ID)
		if err != nil {
			t.Fatal(err)
		}

		if len(players) != 2 {
			t.Errorf("expected 2 players but got %d", len(players))
		}

		// first player
		if players[0].ID != "user-id" {
			t.Errorf("expected id %s but got %s", "user-id", players[0].ID)
		}
		if players[0].Nickname != "user-nickname" {
			t.Errorf("expected nickname %s but got %s", "user-nickname", players[0].Nickname)
		}
		if players[0].JoinedAt.Compare(now) != 1 {
			t.Errorf("expected joined_at after %v but got %v", now, players[0].JoinedAt)
		}
		if players[0].State != "active" {
			t.Errorf("expected state %s but got %s", "active", players[0].State)
		}

		// second player
		if players[1].ID != "user-id-2" {
			t.Errorf("expected id %s but got %s", "user-id-2", players[1].ID)
		}
		if players[1].Nickname != "user-nickname-2" {
			t.Errorf("expected nickname %s but got %s", "user-nickname-2", players[1].Nickname)
		}
		if players[1].JoinedAt.Compare(now) != 1 {
			t.Errorf("expected joined_at after %v but got %v", now, players[1].JoinedAt)
		}
		if players[1].State != "active" {
			t.Errorf("expected state %s but got %s", "active", players[1].State)
		}
	})
	t.Run("SendMessage", func(t *testing.T) {
		cleanupDB()

		now := time.Now()
		game, err := repo.Create(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec("INSERT INTO users (id, nickname, created_at, updated_at) VALUES ('user-id', 'user-nickname', ?, ?);", now.UnixMicro(), now.UnixMicro())
		if err != nil {
			t.Fatal(err)
		}

		message, err := repo.SendMessage(context.Background(), game.ID, "turn_id", "user-id", "message_content")
		if err != nil {
			t.Fatal(err)
		}

		if message.ID == "" {
			t.Errorf("expected id not to be empty but got %s", message.ID)
		}

		if message.CreatedAt.Compare(now) != 1 {
			t.Errorf("expected created_at after %v but got %v", now, message.CreatedAt)
		}

		if message.PlayerID != "user-id" {
			t.Errorf("expected player_id %s but got %s", "user-id", message.PlayerID)
		}

		if message.Content != "message_content" {
			t.Errorf("expected content %s but got %s", "message_content", message.Content)
		}
	})
	t.Run("GetMessages", func(t *testing.T) {
		now := time.Now()
		cleanupDB()

		_, err = db.Exec("INSERT INTO users (id, nickname, created_at, updated_at) VALUES ('user-id', 'user-nickname', ?, ?);", now.UnixMicro(), now.UnixMicro())
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec("INSERT INTO users (id, nickname, created_at, updated_at) VALUES ('user-id-2', 'user-nickname-2', ?, ?);", now.UnixMicro(), now.UnixMicro())
		if err != nil {
			t.Fatal(err)
		}

		firstMsg, err := repo.SendMessage(context.Background(), "game-id", "turn-id", "user-id", "message content")
		if err != nil {
			t.Fatal(err)
		}

		secondMsg, err := repo.SendMessage(context.Background(), "game-id", "turn-id", "user-id-2", "second message content")
		if err != nil {
			t.Fatal(err)
		}

		messages, err := repo.GetMessages(context.Background(), "game-id")
		if err != nil {
			t.Fatal(err)
		}

		if len(messages) != 2 {
			t.Errorf("expected 1 message but got %d", len(messages))
		}

		expectedMsgs := []model.Message{firstMsg, secondMsg}
		for i, msg := range messages {
			expectedMsg := expectedMsgs[i]

			// first message
			if msg.ID != expectedMsg.ID {
				t.Errorf("expected id %s but got %s", expectedMsg.ID, msg.ID)
			}
			if msg.PlayerID != expectedMsg.PlayerID {
				t.Errorf("expected player_id %s but got %s", expectedMsg.PlayerID, msg.PlayerID)
			}
			if msg.Content != expectedMsg.Content {
				t.Errorf("expected content %s but got %s", expectedMsg.Content, msg.Content)
			}
			if msg.CreatedAt.Compare(expectedMsg.CreatedAt) != 0 {
				t.Errorf("expected created_at %v but got %v", expectedMsg.CreatedAt, msg.CreatedAt)
			}
		}
	})
	t.Run("AddTurn", func(t *testing.T) {
		cleanupDB()

		err = repo.AddTurn(context.Background(), "game-id", "word-id")
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("GetLatestTurn", func(t *testing.T) {
		now := time.Now()
		cleanupDB()

		err = repo.AddTurn(context.Background(), "game-id", "word-id-1")
		if err != nil {
			t.Fatal(err)
		}

		err = repo.AddTurn(context.Background(), "game-id", "word-id-2")
		if err != nil {
			t.Fatal(err)
		}

		err = repo.AddTurn(context.Background(), "game-id", "word-id-3")
		if err != nil {
			t.Fatal(err)
		}

		turn, err := repo.GetLatestTurn(context.Background(), "game-id")
		if err != nil {
			t.Fatal(err)
		}

		if turn.WordID != "word-id-3" {
			t.Errorf("expected word id %s but got %s", "word-id-3", turn.WordID)
		}

		if turn.CreatedAt.Compare(now) != 1 {
			t.Errorf("expected created_at after %v but got %v", now, turn.CreatedAt)
		}
	})
	t.Run("AddScore", func(t *testing.T) {
		cleanupDB()

		err = repo.AddScore(context.Background(), "game-id", "player-id", "message-id", "turn-id", 10)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("GetScores", func(t *testing.T) {
		now := time.Now()
		cleanupDB()

		err = repo.AddScore(context.Background(), "game-id", "player-id", "message-id", "turn-id", 10)
		if err != nil {
			t.Fatal(err)
		}

		err = repo.AddScore(context.Background(), "game-id", "player-id", "message-id", "turn-id", 20)
		if err != nil {
			t.Fatal(err)
		}

		err = repo.AddScore(context.Background(), "game-id", "player-id", "message-id", "turn-id", 30)
		if err != nil {
			t.Fatal(err)
		}

		scores, err := repo.GetScores(context.Background(), "game-id")
		if err != nil {
			t.Fatal(err)
		}

		if len(scores) != 3 {
			t.Fatalf("expected 3 scores but got %d", len(scores))
		}

		expectedScores := []model.Score{
			model.Score{GameID: "game-id", PlayerID: "player-id", MessageID: "message-id", TurnID: "turn-id", Score: 10, CreatedAt: now},
			model.Score{GameID: "game-id", PlayerID: "player-id", MessageID: "message-id", TurnID: "turn-id", Score: 20, CreatedAt: now},
			model.Score{GameID: "game-id", PlayerID: "player-id", MessageID: "message-id", TurnID: "turn-id", Score: 30, CreatedAt: now},
		}

		for i, score := range scores {
			expectedScore := expectedScores[i]

			if score.GameID != expectedScore.GameID {
				t.Errorf("index %d expected game id %s but got %s", i, expectedScore.GameID, score.GameID)
			}
			if score.PlayerID != expectedScore.PlayerID {
				t.Errorf("index %d expected player id %s but got %s", i, expectedScore.PlayerID, score.PlayerID)
			}
			if score.MessageID != expectedScore.MessageID {
				t.Errorf("index %d expected message id %s but got %s", i, expectedScore.MessageID, score.MessageID)
			}
			if score.TurnID != expectedScore.TurnID {
				t.Errorf("index %d expected turn id %s but got %s", i, expectedScore.TurnID, score.TurnID)
			}
			if score.Score != expectedScore.Score {
				t.Errorf("index %d expected score %d but got %d", i, expectedScore.Score, score.Score)
			}
			if score.CreatedAt.Compare(expectedScore.CreatedAt) != 1 {
				t.Errorf("index %d expected created_at after %v but got %v", i, expectedScore.CreatedAt, score.CreatedAt)
			}
		}
	})
}
