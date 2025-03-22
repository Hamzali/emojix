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

func InitDB(fileName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", fileName)

	if err != nil {
		return db, err
	}

	return db, nil
}

type sqliteUserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
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

	user.CreatedAt = time.Unix(0, createdAt)
	user.UpdatedAt = time.Unix(0, updatedAt)

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

	if err == sql.ErrNoRows {
		_, err = r.db.ExecContext(ctx, "INSERT INTO users (id, nickname, created_at, updated_at) VALUES (?, ?, ?, ?)", id, params.Nickname, time.Now().Unix(), time.Now().Unix())
		if err != nil {
			return err
		}

		return nil
	}

	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, "UPDATE users SET nickname = ?, updated_at = ? WHERE id = ?", params.Nickname, time.Now().Unix(), id)
	if err != nil {
		return err
	}

	return nil
}

type sqliteGameRepository struct {
	db *sql.DB
}

func NewGameRepository(db *sql.DB) GameRepository {
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

	game.CreatedAt = time.Unix(0, createdAt)
	game.UpdatedAt = time.Unix(0, updatedAt)

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
	_, err := r.db.ExecContext(ctx, "INSERT INTO players (game_id,  player_id, joined_at) VALUES (?, ?, ?)", gameID, userID, time.Now().Unix())

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
		player.JoinedAt = time.Unix(0, joinedAt)
		players = append(players, player)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return players, nil
}

func (r *sqliteGameRepository) GetMessages(ctx context.Context, gameID string) ([]model.Message, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.player_id, m.content, m.created_at 
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
		err = rows.Scan(&msg.PlayerID, &msg.Content, &createdAt)
		if err != nil {
			return nil, err
		}
		msg.CreatedAt = time.Unix(0, createdAt)
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *sqliteGameRepository) SendMessage(ctx context.Context, gameID string, userID string, content string) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO messages (game_id, player_id, content, created_at) VALUES (?, ?, ?, ?)", gameID, userID, content, time.Now().Unix())

	if err != nil {
		return err
	}

	return nil
}
