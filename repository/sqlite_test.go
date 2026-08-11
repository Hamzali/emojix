package repository

import (
	"context"
	"database/sql"
	"emojix/model"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// newTestDB returns a freshly migrated in-memory database for a single test.
// It reuses newMemoryDB (single connection, so the in-memory database stays
// consistent) and applies the real migrations. Each subtest gets its own
// database, so no manual cleanup between subtests is needed.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := newMemoryDB(t)

	migrator, err := NewSQLiteMigrator(db, ":memory:", "../database/migrations")
	if err != nil {
		t.Fatalf("new migrator: %v", err)
	}
	if err := migrator.UpCmd(); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	return db
}

func TestUserRepository(t *testing.T) {
	t.Run("FindByID", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewUserRepository(db)

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
			db := newTestDB(t)
			repo := NewUserRepository(db)

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
			db := newTestDB(t)
			repo := NewUserRepository(db)

			now := time.Now()
			userID := "some-id"
			params := UserCreateOrUpdateParams{
				Nickname: "some-nickname",
			}
			err := repo.CreateOrUpdate(context.Background(), userID, params)
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

func seedList(t *testing.T, db *sql.DB, listID, title string) {
	t.Helper()
	_, err := db.Exec("INSERT INTO word_lists (id, title) VALUES (?, ?)", listID, title)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWordRepository(t *testing.T) {
	t.Run("GetLists", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewWordRepository(db)

		lists, err := repo.GetLists(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(lists) != 0 {
			t.Errorf("expected 0 lists, got %d", len(lists))
		}

		seedList(t, db, "l1", "Action")
		seedList(t, db, "l2", "Sci-Fi")

		lists, err = repo.GetLists(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(lists) != 2 {
			t.Fatalf("expected 2 lists, got %d", len(lists))
		}
		// ordered by title
		if lists[0].Title != "Action" || lists[1].Title != "Sci-Fi" {
			t.Errorf("unexpected order: %+v", lists)
		}
	})

	t.Run("GetUnusedByList", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewWordRepository(db)
		seedList(t, db, "l1", "Action")

		_, err := db.Exec(`INSERT INTO words (id, list_id, word, hint) VALUES
			('1', 'l1', 'word-1', 'hint-1'),
			('2', 'l1', 'word-2', 'hint-2'),
			('3', 'l1', 'word-3', 'hint-3')`)
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.Exec("INSERT INTO games (id, list_id, created_at, updated_at) VALUES ('g1', 'l1', 1, 1)")
		if err != nil {
			t.Fatal(err)
		}

		words, err := repo.GetUnusedByList(context.Background(), "l1", "g1")
		if err != nil {
			t.Fatal(err)
		}
		if len(words) != 3 {
			t.Fatalf("expected 3 unused, got %d", len(words))
		}

		// mark word 1 as used
		_, err = db.Exec(`INSERT INTO game_turns (id, game_id, word_id, teller_id, option_a, option_b, option_c, created_at, started_at)
			VALUES ('t1', 'g1', '1', 'teller', '1', '2', '3', 1, 1)`)
		if err != nil {
			t.Fatal(err)
		}

		words, err = repo.GetUnusedByList(context.Background(), "l1", "g1")
		if err != nil {
			t.Fatal(err)
		}
		if len(words) != 2 {
			t.Fatalf("expected 2 unused after play, got %d", len(words))
		}
		for _, w := range words {
			if w.ID == "1" {
				t.Error("used word still returned")
			}
		}
	})

	t.Run("FindByID", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewWordRepository(db)
		seedList(t, db, "l1", "Action")

		_, err := db.Exec("INSERT INTO words (id, list_id, word, hint) VALUES ('1', 'l1', 'word-1', 'hint-1');")
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
		if word.ListID != "l1" {
			t.Errorf("expected list_id l1 got %s", word.ListID)
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
	t.Run("FindByID", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewGameRepository(db)

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
		db := newTestDB(t)
		repo := NewGameRepository(db)
		seedList(t, db, "list-1", "Test List")

		now := time.Now()
		game, err := repo.Create(context.Background(), "list-1")
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
		db := newTestDB(t)
		repo := NewGameRepository(db)
		seedList(t, db, "list-1", "Test List")

		now := time.Now()
		game, err := repo.Create(context.Background(), "list-1")
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
		db := newTestDB(t)
		repo := NewGameRepository(db)
		seedList(t, db, "list-1", "Test List")

		now := time.Now()
		game, err := repo.Create(context.Background(), "list-1")
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
		db := newTestDB(t)
		repo := NewGameRepository(db)
		seedList(t, db, "list-1", "Test List")

		now := time.Now()
		game, err := repo.Create(context.Background(), "list-1")
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
		db := newTestDB(t)
		repo := NewGameRepository(db)
		seedList(t, db, "list-1", "Test List")

		now := time.Now()
		game, err := repo.Create(context.Background(), "list-1")
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec("INSERT INTO users (id, nickname, created_at, updated_at) VALUES ('user-id', 'user-nickname', ?, ?);", now.UnixMicro(), now.UnixMicro())
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec("INSERT INTO words (id, word, hint) VALUES ('word-id', 'word', 'hint');")
		if err != nil {
			t.Fatal(err)
		}

		turn, err := repo.AddTurn(context.Background(), AddTurnParams{GameID: game.ID, TellerID: "teller", OptionA: "word-id", OptionB: "word-id", OptionC: "word-id"})
		if err != nil {
			t.Fatal(err)
		}

		message, err := repo.SendMessage(context.Background(), game.ID, turn.ID, "user-id", "message_content")
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
		db := newTestDB(t)
		repo := NewGameRepository(db)

		now := time.Now()

		_, err := db.Exec("INSERT INTO users (id, nickname, created_at, updated_at) VALUES ('user-id', 'user-nickname', ?, ?);", now.UnixMicro(), now.UnixMicro())
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec("INSERT INTO users (id, nickname, created_at, updated_at) VALUES ('user-id-2', 'user-nickname-2', ?, ?);", now.UnixMicro(), now.UnixMicro())
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec("INSERT INTO games (id, created_at, updated_at) VALUES ('game-id', ?, ?);", now.UnixMicro(), now.UnixMicro())
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec("INSERT INTO words (id, word, hint) VALUES ('word-id', 'word', 'hint');")
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec("INSERT INTO game_turns (id, game_id, word_id, teller_id, option_a, option_b, option_c, created_at, started_at) VALUES ('turn-id', 'game-id', 'word-id', 'teller', 'word-id', 'word-id', 'word-id', ?, ?);", now.UnixMicro(), now.UnixMicro())
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
	t.Run("AddTurn and SetTurnWord", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewGameRepository(db)

		now := time.Now()
		_, err := db.Exec("INSERT INTO games (id, created_at, updated_at) VALUES ('game-id', ?, ?);", now.UnixMicro(), now.UnixMicro())
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec("INSERT INTO words (id, word, hint) VALUES ('word-id', 'word', 'hint'), ('w2', 'word2', 'h2'), ('w3', 'word3', 'h3');")
		if err != nil {
			t.Fatal(err)
		}

		turn, err := repo.AddTurn(context.Background(), AddTurnParams{
			GameID: "game-id", TellerID: "teller-1",
			OptionA: "word-id", OptionB: "w2", OptionC: "w3",
		})
		if err != nil {
			t.Fatal(err)
		}
		if turn.WordID != "" {
			t.Errorf("new turn should have empty word_id, got %q", turn.WordID)
		}
		if turn.TellerID != "teller-1" {
			t.Errorf("teller: got %q", turn.TellerID)
		}

		if err := repo.SetTurnWord(context.Background(), turn.ID, "w2", "🍎🍌"); err != nil {
			t.Fatal(err)
		}
		got, err := repo.GetLatestTurn(context.Background(), "game-id")
		if err != nil {
			t.Fatal(err)
		}
		if got.WordID != "w2" {
			t.Errorf("word_id after pick: got %q want w2", got.WordID)
		}
		if got.EmojiHint != "🍎🍌" {
			t.Errorf("emoji_hint after pick: got %q want 🍎🍌", got.EmojiHint)
		}
		if got.StartedAt.IsZero() {
			t.Error("started_at should be set after pick")
		}

		if err := repo.SetTurnEmojiHint(context.Background(), turn.ID, "🍎🍌🔥"); err != nil {
			t.Fatal(err)
		}
		got, err = repo.GetLatestTurn(context.Background(), "game-id")
		if err != nil {
			t.Fatal(err)
		}
		if got.EmojiHint != "🍎🍌🔥" {
			t.Errorf("emoji_hint after append: got %q want 🍎🍌🔥", got.EmojiHint)
		}
	})
	t.Run("GetLatestTurn", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewGameRepository(db)

		now := time.Now()

		_, err := db.Exec("INSERT INTO games (id, created_at, updated_at) VALUES ('game-id', ?, ?);", now.UnixMicro(), now.UnixMicro())
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec("INSERT INTO words (id, word, hint) VALUES ('word-id-1', 'word', 'hint'), ('word-id-2', 'word', 'hint'), ('word-id-3', 'word', 'hint');")
		if err != nil {
			t.Fatal(err)
		}

		_, err = repo.AddTurn(context.Background(), AddTurnParams{GameID: "game-id", TellerID: "teller", OptionA: "word-id-1", OptionB: "word-id-1", OptionC: "word-id-1"})
		if err != nil {
			t.Fatal(err)
		}

		_, err = repo.AddTurn(context.Background(), AddTurnParams{GameID: "game-id", TellerID: "teller", OptionA: "word-id-2", OptionB: "word-id-2", OptionC: "word-id-2"})
		if err != nil {
			t.Fatal(err)
		}

		third, err := repo.AddTurn(context.Background(), AddTurnParams{GameID: "game-id", TellerID: "teller", OptionA: "word-id-3", OptionB: "word-id-3", OptionC: "word-id-3"})
		if err != nil {
			t.Fatal(err)
		}

		turn, err := repo.GetLatestTurn(context.Background(), "game-id")
		if err != nil {
			t.Fatal(err)
		}

		if turn.ID != third.ID {
			t.Errorf("expected latest turn id %s but got %s", third.ID, turn.ID)
		}
		if turn.OptionA != "word-id-3" {
			t.Errorf("expected option_a word-id-3 got %s", turn.OptionA)
		}

		if turn.CreatedAt.Compare(now) != 1 {
			t.Errorf("expected created_at after %v but got %v", now, turn.CreatedAt)
		}

		n, err := repo.CountTurns(context.Background(), "game-id")
		if err != nil {
			t.Fatal(err)
		}
		if n != 3 {
			t.Errorf("CountTurns: got %d want 3", n)
		}
	})
	t.Run("AddScore", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewGameRepository(db)

		insertScoreParents(t, db, "game-id", "player-id", "message-id", "turn-id", "word-id")

		err := repo.AddScore(context.Background(), "game-id", "player-id", "message-id", "turn-id", 10)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("GetScores", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewGameRepository(db)

		now := time.Now()
		insertScoreParents(t, db, "game-id", "player-id", "message-id", "turn-id", "word-id")

		err := repo.AddScore(context.Background(), "game-id", "player-id", "message-id", "turn-id", 10)
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
			{GameID: "game-id", PlayerID: "player-id", MessageID: "message-id", TurnID: "turn-id", Score: 10, CreatedAt: now},
			{GameID: "game-id", PlayerID: "player-id", MessageID: "message-id", TurnID: "turn-id", Score: 20, CreatedAt: now},
			{GameID: "game-id", PlayerID: "player-id", MessageID: "message-id", TurnID: "turn-id", Score: 30, CreatedAt: now},
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

// insertScoreParents inserts the full FK parent chain (game, user, word, turn,
// message) needed by game_scores. It panics on error because the parent rows
// are a precondition, not the thing under test.
func insertScoreParents(t *testing.T, db *sql.DB, gameID, playerID, messageID, turnID, wordID string) {
	t.Helper()
	now := time.Now().UnixMicro()

	mustExec := func(query string, args ...any) {
		_, err := db.Exec(query, args...)
		if err != nil {
			t.Fatalf("insert parent: %v", err)
		}
	}

	mustExec("INSERT INTO games (id, created_at, updated_at) VALUES (?, ?, ?);", gameID, now, now)
	mustExec("INSERT INTO users (id, nickname, created_at, updated_at) VALUES (?, ?, ?, ?);", playerID, "nick", now, now)
	mustExec("INSERT INTO words (id, word, hint) VALUES (?, ?, ?);", wordID, "word", "hint")
	mustExec(`INSERT INTO game_turns (id, game_id, word_id, teller_id, option_a, option_b, option_c, created_at, started_at)
		VALUES (?, ?, ?, 'teller', ?, ?, ?, ?, ?);`, turnID, gameID, wordID, wordID, wordID, wordID, now, now)
	mustExec("INSERT INTO messages (id, game_id, player_id, turn_id, content, created_at) VALUES (?, ?, ?, ?, ?, ?);", messageID, gameID, playerID, turnID, "content", now)
}

func TestForeignKeysEnforced(t *testing.T) {
	db := newTestDB(t)

	t.Run("PRAGMA foreign_keys is ON", func(t *testing.T) {
		var v string
		if err := db.QueryRow("PRAGMA foreign_keys").Scan(&v); err != nil {
			t.Fatal(err)
		}
		if v != "1" {
			t.Fatalf("expected PRAGMA foreign_keys = 1 but got %s", v)
		}
	})

	t.Run("players references games and users", func(t *testing.T) {
		_, err := db.Exec("INSERT INTO players (game_id, player_id, state, joined_at) VALUES ('NoSuch', 'NoSuch', 'active', 0);")
		if err == nil {
			t.Fatalf("expected FK violation inserting orphan players row")
		}
	})

	t.Run("game_turns references games and words", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO game_turns (id, game_id, word_id, teller_id, option_a, option_b, option_c, created_at)
			VALUES ('NoSuch', 'NoSuch', 'NoSuch', 't', 'a', 'b', 'c', 0);`)
		if err == nil {
			t.Fatalf("expected FK violation inserting orphan game_turns row")
		}
	})

	t.Run("messages references games, users and turns", func(t *testing.T) {
		_, err := db.Exec("INSERT INTO messages (id, game_id, player_id, turn_id, content, created_at) VALUES ('NoSuch', 'NoSuch', 'NoSuch', 'NoSuch', 'c', 0);")
		if err == nil {
			t.Fatalf("expected FK violation inserting orphan messages row")
		}
	})

	t.Run("game_scores references everything", func(t *testing.T) {
		_, err := db.Exec("INSERT INTO game_scores (game_id, player_id, message_id, turn_id, score, created_at) VALUES ('NoSuch', 'NoSuch', 'NoSuch', 'NoSuch', 0, 0);")
		if err == nil {
			t.Fatalf("expected FK violation inserting orphan game_scores row")
		}
	})
}
