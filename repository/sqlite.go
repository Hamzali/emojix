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

func NewUnitOfWorkFactory(db *sql.DB) UnitOfWorkFactory {
	return &sqliteUnitOfWorkFactory{db}
}

type sqliteUnitOfWorkFactory struct {
	db *sql.DB
}

func (uowf *sqliteUnitOfWorkFactory) New(ctx context.Context) (UnitOfWork, error) {
	tx, err := uowf.db.BeginTx(ctx, nil)

	if err != nil {
		return nil, err
	}

	return &sqliteUnitOfWork{tx}, nil
}

type sqliteUnitOfWork struct {
	tx *sql.Tx
}

// Commit implements UnitOfWork.
func (uow *sqliteUnitOfWork) Commit() error {
	return uow.tx.Commit()
}

// Rollback implements UnitOfWork.
func (uow *sqliteUnitOfWork) Rollback() error {
	return uow.tx.Rollback()
}

// GameRepository implements UnitOfWork.
func (uow *sqliteUnitOfWork) GameRepository() GameRepository {
	return NewGameRepository(uow.tx)
}

func InitSqliteDB(fileName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", fileName)
	if err != nil {
		return db, err
	}

	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return nil, err
	}

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

	row := r.db.QueryRowContext(ctx, "SELECT id, list_id, created_at, updated_at FROM games WHERE id = ?", id)

	err := row.Err()

	game := model.Game{}

	if err != nil {
		return game, err
	}

	var createdAt, updatedAt int64
	var listID sql.NullString

	err = row.Scan(&game.ID, &listID, &createdAt, &updatedAt)

	if err != nil {
		return game, err
	}

	game.ListID = listID.String
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

func (r *sqliteGameRepository) Create(ctx context.Context, listID string) (model.Game, error) {
	id, err := generateRandomID()
	if err != nil {
		return model.Game{}, err
	}

	game := model.Game{
		ID:        id,
		ListID:    listID,
		UpdatedAt: time.Now(),
		CreatedAt: time.Now(),
	}

	_, err = r.db.ExecContext(ctx, "INSERT INTO games (id, list_id, updated_at, created_at) VALUES (?, ?, ?, ?)", game.ID, game.ListID, game.UpdatedAt.Unix(), game.CreatedAt.Unix())

	if err != nil {
		return model.Game{}, err
	}

	return game, nil
}

func (r *sqliteGameRepository) SetPlayerState(ctx context.Context, gameID string, userID string, state model.PlayerState) error {
	_, err := r.db.ExecContext(
		ctx,
		"UPDATE players SET state = ? WHERE game_id = ? AND player_id = ?",
		state, gameID, userID,
	)

	if err != nil {
		return err
	}

	return nil
}

func (r *sqliteGameRepository) AddPlayer(ctx context.Context, gameID string, userID string) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO players (game_id,  player_id, state, joined_at) VALUES (?, ?, ?, ?)", gameID, userID, model.ActivePlayerState, time.Now().UnixMicro())

	if err != nil {
		return err
	}

	return nil
}

func (r *sqliteGameRepository) GetPlayers(ctx context.Context, gameID string) ([]model.Player, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.nickname, p.state, p.joined_at
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
		err = rows.Scan(&player.ID, &player.Nickname, &player.State, &joinedAt)
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
	now = time.UnixMicro(now.UnixMicro())
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
	row := r.db.QueryRowContext(ctx, `
		SELECT id, word_id, teller_id, option_a, option_b, option_c, emoji_hint, created_at, started_at
		FROM game_turns WHERE game_id = ? ORDER BY created_at DESC LIMIT 1`, gameID)

	err := row.Err()

	turn := model.GameTurn{GameID: gameID}

	if err != nil {
		return turn, err
	}

	var createdAt int64
	var wordID sql.NullString
	var startedAt sql.NullInt64
	err = row.Scan(&turn.ID, &wordID, &turn.TellerID, &turn.OptionA, &turn.OptionB, &turn.OptionC, &turn.EmojiHint, &createdAt, &startedAt)

	if err != nil {
		return turn, err
	}

	turn.WordID = wordID.String
	turn.CreatedAt = time.UnixMicro(createdAt)
	if startedAt.Valid {
		turn.StartedAt = time.UnixMicro(startedAt.Int64)
	}

	return turn, nil
}

func (r *sqliteGameRepository) AddTurn(ctx context.Context, params AddTurnParams) (model.GameTurn, error) {
	id, err := generateRandomID()
	if err != nil {
		return model.GameTurn{}, err
	}

	turn := model.GameTurn{
		ID:        id,
		GameID:    params.GameID,
		TellerID:  params.TellerID,
		OptionA:   params.OptionA,
		OptionB:   params.OptionB,
		OptionC:   params.OptionC,
		CreatedAt: time.Now(),
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO game_turns (id, game_id, word_id, teller_id, option_a, option_b, option_c, created_at, started_at)
		 VALUES (?, ?, NULL, ?, ?, ?, ?, ?, NULL)`,
		id, params.GameID, params.TellerID, params.OptionA, params.OptionB, params.OptionC, turn.CreatedAt.UnixMicro(),
	)
	if err != nil {
		return model.GameTurn{}, err
	}

	return turn, nil
}

func (r *sqliteGameRepository) SetTurnWord(ctx context.Context, turnID string, wordID string, emojiHint string) error {
	now := time.Now().UnixMicro()
	_, err := r.db.ExecContext(ctx,
		`UPDATE game_turns SET word_id = ?, emoji_hint = ?, started_at = ? WHERE id = ? AND word_id IS NULL`,
		wordID, emojiHint, now, turnID,
	)
	return err
}

func (r *sqliteGameRepository) SetTurnEmojiHint(ctx context.Context, turnID string, emojiHint string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE game_turns SET emoji_hint = ? WHERE id = ?`,
		emojiHint, turnID,
	)
	return err
}

func (r *sqliteGameRepository) CountTurns(ctx context.Context, gameID string) (int, error) {
	row := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM game_turns WHERE game_id = ?`, gameID)
	var n int
	err := row.Scan(&n)
	return n, err
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

func (r *sqliteWordRepository) GetLists(ctx context.Context) ([]model.WordList, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, title FROM word_lists ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lists := []model.WordList{}
	for rows.Next() {
		var list model.WordList
		if err = rows.Scan(&list.ID, &list.Title); err != nil {
			return nil, err
		}
		lists = append(lists, list)
	}
	return lists, rows.Err()
}

func (r *sqliteWordRepository) GetUnusedByList(ctx context.Context, listID, gameID string) ([]model.Word, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT w.id, w.list_id, w.word, w.hint
		FROM words w
		WHERE w.list_id = ?
		  AND w.id NOT IN (
		    SELECT word_id FROM game_turns
		    WHERE game_id = ? AND word_id IS NOT NULL
		  )`, listID, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	words := []model.Word{}
	for rows.Next() {
		var word model.Word
		var lid sql.NullString
		err = rows.Scan(&word.ID, &lid, &word.Word, &word.Hint)
		if err != nil {
			return nil, err
		}
		word.ListID = lid.String
		words = append(words, word)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return words, nil
}

func (r *sqliteWordRepository) FindByID(ctx context.Context, id string) (model.Word, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id, list_id, word, hint FROM words WHERE id = ?", id)

	err := row.Err()

	word := model.Word{}

	if err != nil {
		return word, err
	}

	var listID sql.NullString
	err = row.Scan(&word.ID, &listID, &word.Word, &word.Hint)
	if err != nil {
		return word, err
	}
	word.ListID = listID.String

	return word, nil
}
