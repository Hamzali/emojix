CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	nickname TEXT NOT NULL,
	created_at INT NOT NULL,
	updated_at INT NOT NULL
)
