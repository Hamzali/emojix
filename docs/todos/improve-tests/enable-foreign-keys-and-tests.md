# T16 — Enable PRAGMA foreign_keys + FK-constraint tests

- **Status:** `[ ]`
- **Depends on:** —
- **Unblocks:** makes the schema's FKs actually mean something; surfaces the
  `players → users` ordering issue noted in init.sql.

## Problem

`repository/sqlite.go` `InitSqliteDB` has `PRAGMA foreign_keys = ON;` **commented out**,
with the note *"current tests are implemented without foreign key constraints"*. That
means every `FOREIGN KEY` in `database/migrations/1761420367_init.sql` is decorative: you
can insert a `player` for a non-existent `game`/`user`, a `message` for a non-existent
`turn`, a score for a non-existent message. False confidence.

Also, the init.sql defines `players` (which `REFERENCES users`) **before** the `users`
table is created. Works only because SQLite doesn't validate FKs at DDL time; enabling
would still work but the ordering is fragile and worth tidying.

DEVLOG (Oct 31) explicitly calls this out: *"sqlite by default does not respect foreign
keys and you have to explicitly tell to trigger related constraints on connection"*.

## Goal

Turn FK enforcement on and prove it via negative tests. Keep the existing repo tests green.

## Approach

1. **Uncomment** `_, err = db.Exec("PRAGMA foreign_keys = ON;")` in `InitSqliteDB`.
2. **Reorder** `init.sql` so referenced tables precede referencing ones: `users` and
   `words` and `games` first, then `game_turns` (refs games+words), then `players` (refs
   games+users), `messages` (refs games+users+turns), `game_scores` (refs everything).
   Optional quality-of-life; not strictly required for correctness once FKs are on, but
   removes the footgun.
3. **Add `sqlite_test.go` FK negative tests** (one per FK relationship), each asserting
   the insert fails:
   - Insert `players(game_id=NoSuch, player_id=NoSuch)` → error.
   - Insert `game_turns(game_id=NoSuch, word_id=NoSuch)` → error.
   - Insert `messages(game_id=NoSuch, player_id=NoSuch, turn_id=NoSuch)` → error.
   - Insert `game_scores` with all-fake FKs → error.
   - Positive controls: existing tests already insert valid FKs; ensure they still pass.
4. Check that the existing per-table tests (which build rows out of order sometimes) are
   still valid; if any test relies on inserting an orphan row, fix the test to insert the
   parent first.

## Out of scope

- Indexing / cascades / `ON DELETE` rules (backlog — the schema currently has none).
- Migration versioning changes.

## Acceptance criteria

- [ ] `PRAGMA foreign_keys = ON` is active in `InitSqliteDB` and its test.
- [ ] Reorder of `init.sql` (if applied) keeps the migration idempotent
      (`CREATE TABLE IF NOT EXISTS`) and `UpCmd` idempotent.
- [ ] At least 4 negative FK tests pass; they fail if the pragma is ever toggled off again
      (i.e. they genuinely depend on enforcement, not on a silent success).
- [ ] All existing `repository` tests still green.
- [ ] `go test -race ./repository/...` green.

## Files touched

- `repository/sqlite.go` (uncomment pragma)
- `database/migrations/1761420367_init.sql` (optional reorder)
- `repository/sqlite_test.go` (new FK tests; possibly fix any orphan-inserting test)

## Notes

- `PRAGMA foreign_keys` is **per-connection**, not per-DB. The in-memory test DB and the
  file DB both go through `InitSqliteDB`, so setting it there covers both. Verify with a
  `db.QueryRow("PRAGMA foreign_keys").Scan(&v)` in a test.
- modernc.org/sqlite honors `PRAGMA foreign_keys = ON` the same as cgo sqlite; this is the
  reason it's safe to enable here.