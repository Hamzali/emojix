CREATE TABLE IF NOT EXISTS games (
	id TEXT PRIMARY KEY,
	created_at INT NOT NULL,
	updated_at INT NOT NULL
);

CREATE TABLE IF NOT EXISTS words (
	id TEXT PRIMARY KEY,
	word TEXT NOT NULL,
	hint TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS game_turns (
    id TEXT PRIMARY KEY,
	game_id TEXT NOT NULL,
	word_id TEXT NOT NULL,
	created_at INT NOT NULL,
	FOREIGN KEY (game_id) REFERENCES games(id),
	FOREIGN KEY (word_id) REFERENCES words(id)
);

CREATE TABLE IF NOT EXISTS players (
	game_id TEXT NOT NULL,
	player_id TEXT NOT NULL,
	joined_at INT NOT NULL,
	FOREIGN KEY (game_id) REFERENCES games(id),
	FOREIGN KEY (player_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
	game_id TEXT NOT NULL,
	player_id TEXT,
	turn_id TEXT,
	content TEXT,
	created_at INT NOT NULL,
	FOREIGN KEY (game_id) REFERENCES games(id),
	FOREIGN KEY (player_id) REFERENCES users(id),
	FOREIGN KEY (turn_id) REFERENCES game_turns(id)
);

CREATE TABLE IF NOT EXISTS game_scores (
	game_id TEXT NOT NULL,
	player_id TEXT NOT NULL,
	message_id TEXT NOT NULL,
	turn_id TEXT NOT NULL,
	score INT NOT NULL,
	created_at INT NOT NULL,
	FOREIGN KEY (game_id) REFERENCES games(id),
	FOREIGN KEY (player_id) REFERENCES users(id),
	FOREIGN KEY (message_id) REFERENCES messages(id),
	FOREIGN KEY (turn_id) REFERENCES game_turns(id)
);

CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	nickname TEXT NOT NULL,
	created_at INT NOT NULL,
	updated_at INT NOT NULL
)
