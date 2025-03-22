CREATE TABLE IF NOT EXISTS games (
	id TEXT PRIMARY KEY,
	created_at INT NOT NULL,
	updated_at INT NOT NULL
);

CREATE TABLE IF NOT EXISTS players (
	game_id TEXT NOT NULL,
	player_id TEXT NOT NULL,
	joined_at INT NOT NULL,
	FOREIGN KEY (game_id) REFERENCES games(id),
	FOREIGN KEY (player_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS messages (
	game_id TEXT NOT NULL,
	player_id TEXT,
	content TEXT,
	created_at INT NOT NULL,
	FOREIGN KEY (game_id) REFERENCES games(id),
	FOREIGN KEY (player_id) REFERENCES users(id)
);

