package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"emojix/model"
	"encoding/hex"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

type DBTX interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func InitDB(fileName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", fileName)
	if err != nil {
		return db, err
	}

	// TODO: current test are implemented without foreign key constraints
	// enable this when you comeback and improve tests
	// _, err = db.Exec("PRAGMA foreign_keys = ON;")
	// if err != nil {
	// 	return nil, err
	// }

	return db, nil
}

type sqliteUserRepository struct {
	db DBTX
}

func NewUserRepository(db DBTX) UserRepository {
	return &sqliteUserRepository{db: db}
}

func (r *sqliteUserRepository) FindByID(ctx context.Context, id string) (model.User, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id, nickname, created_at, updated_at FROM users WHERE id = ?", id)

	err := row.Err()

	user := model.User{}

	if err != nil {
		return user, err
	}

	var createdAt, updatedAt int64

	err = row.Scan(&user.ID, &user.Nickname, &createdAt, &updatedAt)

	if err != nil {
		return user, err
	}

	user.CreatedAt = time.UnixMicro(createdAt)
	user.UpdatedAt = time.UnixMicro(updatedAt)

	return user, nil
}

func (r *sqliteUserRepository) CreateOrUpdate(ctx context.Context, id string, params UserCreateOrUpdateParams) error {
	log.Println("CREATE OR UPDATE", id)
	row := r.db.QueryRowContext(ctx, "SELECT id FROM users WHERE id = ?", id)

	err := row.Err()

	if err != nil {
		return err
	}

	var dbID string
	err = row.Scan(&dbID)
	nowMs := time.Now().UnixMicro()

	if err == sql.ErrNoRows {
		_, err = r.db.ExecContext(ctx, "INSERT INTO users (id, nickname, created_at, updated_at) VALUES (?, ?, ?, ?)", id, params.Nickname, nowMs, nowMs)
		if err != nil {
			return err
		}

		return nil
	}

	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, "UPDATE users SET nickname = ?, updated_at = ? WHERE id = ?", params.Nickname, nowMs, id)
	if err != nil {
		return err
	}

	return nil
}

type sqliteGameRepository struct {
	db DBTX
}

func NewGameRepository(db DBTX) GameRepository {
	return &sqliteGameRepository{db: db}
}

func (r *sqliteGameRepository) FindByID(ctx context.Context, id string) (model.Game, error) {

	row := r.db.QueryRowContext(ctx, "SELECT id, created_at, updated_at FROM games WHERE id = ?", id)

	err := row.Err()

	game := model.Game{}

	if err != nil {
		return game, err
	}

	var createdAt, updatedAt int64

	err = row.Scan(&game.ID, &createdAt, &updatedAt)

	if err != nil {
		return game, err
	}

	game.CreatedAt = time.UnixMicro(createdAt)
	game.UpdatedAt = time.UnixMicro(updatedAt)

	return game, nil
}

func generateRandomID() (string, error) {
	// Create a byte slice of size 16 (128 bits)
	bytes := make([]byte, 16)

	// Fill the byte slice with random values
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Encode the bytes to a hexadecimal string
	return hex.EncodeToString(bytes), nil
}

func (r *sqliteGameRepository) Create(ctx context.Context) (model.Game, error) {
	id, err := generateRandomID()
	if err != nil {
		return model.Game{}, err
	}

	game := model.Game{
		ID:        id,
		UpdatedAt: time.Now(),
		CreatedAt: time.Now(),
	}

	_, err = r.db.ExecContext(ctx, "INSERT INTO games (id, updated_at, created_at) VALUES (?, ?, ?)", game.ID, game.UpdatedAt.Unix(), game.CreatedAt.Unix())

	if err != nil {
		return model.Game{}, err
	}

	return game, nil
}

func (r *sqliteGameRepository) AddPlayer(ctx context.Context, gameID string, userID string) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO players (game_id,  player_id, joined_at) VALUES (?, ?, ?)", gameID, userID, time.Now().UnixMicro())

	if err != nil {
		return err
	}

	return nil
}

func (r *sqliteGameRepository) GetPlayers(ctx context.Context, gameID string) ([]model.Player, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.nickname, p.joined_at
		FROM players p
		JOIN users u ON p.player_id = u.id
		WHERE p.game_id = ?`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	players := []model.Player{}
	for rows.Next() {
		var player model.Player
		var joinedAt int64
		err = rows.Scan(&player.ID, &player.Nickname, &joinedAt)
		if err != nil {
			return nil, err
		}
		player.JoinedAt = time.UnixMicro(joinedAt)
		players = append(players, player)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return players, nil
}

func (r *sqliteGameRepository) GetMessages(ctx context.Context, gameID string) ([]model.Message, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.id, m.player_id, m.turn_id, m.content, m.created_at
		FROM messages m
		WHERE m.game_id = ?`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []model.Message{}
	for rows.Next() {
		var msg model.Message
		var createdAt int64
		err = rows.Scan(&msg.ID, &msg.PlayerID, &msg.TurnID, &msg.Content, &createdAt)
		if err != nil {
			return nil, err
		}
		msg.CreatedAt = time.UnixMicro(createdAt)
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *sqliteGameRepository) SendMessage(ctx context.Context, gameID string, turnID string, userID string, content string) (model.Message, error) {
	id, err := generateRandomID()
	if err != nil {
		return model.Message{}, err
	}

	now := time.Now()
	_, err = r.db.ExecContext(
		ctx,
		"INSERT INTO messages (id, game_id, turn_id, player_id, content, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		id, gameID, turnID, userID, content, now.UnixMicro(),
	)

	if err != nil {
		return model.Message{}, err
	}

	return model.Message{
		ID:        id,
		PlayerID:  userID,
		TurnID:    turnID,
		Content:   content,
		CreatedAt: now,
	}, nil
}

func (r *sqliteGameRepository) GetLatestTurn(ctx context.Context, gameID string) (model.GameTurn, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id, word_id, created_at FROM game_turns WHERE game_id = ? ORDER BY created_at DESC LIMIT 1", gameID)

	err := row.Err()

	turn := model.GameTurn{}

	if err != nil {
		return turn, err
	}

	var createdAt int64
	err = row.Scan(&turn.ID, &turn.WordID, &createdAt)

	if err != nil {
		return turn, err
	}

	turn.CreatedAt = time.UnixMicro(createdAt)

	return turn, nil
}

func (r *sqliteGameRepository) AddTurn(ctx context.Context, gameID string, wordID string) error {
	id, err := generateRandomID()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, "INSERT INTO game_turns (id, game_id, word_id, created_at) VALUES (?, ?, ?, ?)", id, gameID, wordID, time.Now().UnixMicro())
	if err != nil {
		return err
	}

	return nil
}

func (r *sqliteGameRepository) GetScores(ctx context.Context, gameID string) ([]model.Score, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.player_id, s.message_id, s.game_id, s.turn_id, s.score, s.created_at
		FROM game_scores s
		WHERE s.game_id = ?`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	scores := []model.Score{}
	for rows.Next() {
		var score model.Score
		var createdAt int64
		err = rows.Scan(&score.PlayerID, &score.MessageID, &score.GameID, &score.TurnID, &score.Score, &createdAt)
		if err != nil {
			return nil, err
		}
		score.CreatedAt = time.UnixMicro(createdAt)
		scores = append(scores, score)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return scores, nil
}

func (r *sqliteGameRepository) AddScore(ctx context.Context, gameID string, userID string, messageID string, turnID string, score int) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO game_scores (game_id, player_id, message_id, turn_id, score, created_at) VALUES (?, ?, ?, ?, ?, ?)", gameID, userID, messageID, turnID, score, time.Now().UnixMicro())

	if err != nil {
		return err
	}

	return nil
}

// WORD REPOSITORY

type sqliteWordRepository struct {
	db DBTX
}

func NewWordRepository(db DBTX) WordRepository {
	return &sqliteWordRepository{db}
}

func (r *sqliteWordRepository) GetAll(ctx context.Context) ([]model.Word, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT w.id, w.word, w.hint
		FROM words w
		LIMIT 100;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	words := []model.Word{}
	for rows.Next() {
		var word model.Word
		err = rows.Scan(&word.ID, &word.Word, &word.Hint)
		if err != nil {
			return nil, err
		}
		words = append(words, word)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return words, nil
}

func (r *sqliteWordRepository) FindByID(ctx context.Context, id string) (model.Word, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id, word, hint FROM words WHERE id = ?", id)

	err := row.Err()

	word := model.Word{}

	if err != nil {
		return word, err
	}

	err = row.Scan(&word.ID, &word.Word, &word.Hint)
	if err != nil {
		return word, err
	}

	return word, nil
}
