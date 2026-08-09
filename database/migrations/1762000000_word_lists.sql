CREATE TABLE IF NOT EXISTS word_lists (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL
);

ALTER TABLE words ADD COLUMN list_id TEXT REFERENCES word_lists(id);

ALTER TABLE games ADD COLUMN list_id TEXT REFERENCES word_lists(id);

-- Recreate game_turns so word_id can be NULL until the teller picks,
-- and so each turn carries the 3 options + teller.
CREATE TABLE game_turns_new (
	id TEXT PRIMARY KEY,
	game_id TEXT NOT NULL,
	word_id TEXT,
	teller_id TEXT NOT NULL,
	option_a TEXT NOT NULL,
	option_b TEXT NOT NULL,
	option_c TEXT NOT NULL,
	created_at INT NOT NULL,
	started_at INT,
	FOREIGN KEY (game_id) REFERENCES games(id),
	FOREIGN KEY (word_id) REFERENCES words(id)
);

INSERT INTO game_turns_new (id, game_id, word_id, teller_id, option_a, option_b, option_c, created_at, started_at)
SELECT id, game_id, word_id, '', word_id, word_id, word_id, created_at, created_at FROM game_turns;

DROP TABLE game_turns;
ALTER TABLE game_turns_new RENAME TO game_turns;
