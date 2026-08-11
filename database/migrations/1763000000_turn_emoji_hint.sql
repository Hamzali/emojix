-- Per-turn live emoji board. Seeded from word.hint on pick; teller appends during play.
ALTER TABLE game_turns ADD COLUMN emoji_hint TEXT NOT NULL DEFAULT '';
