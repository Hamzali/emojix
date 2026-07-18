# T17 — Migrator tests + fix SeedCmd path

- **Status:** `[ ]`
- **Depends on:** —
- **Unblocks:** reliable local onboarding (`script/reset-db.sh`, `cmd/migrations`)

## Problem

`repository/sqlite_utils.go` `Migrator` is the entire local-DB lifecycle (create/apply
migrations, reset, seed) and has **no tests**. Two latent bugs hide behind that:

1. `SeedCmd` hardcodes `"./database/seed.sql"` instead of `path.Join(m.basedir,
   "seed.sql")`. The migration dir is configurable (`NewSQLiteMigrator(db, dbname,
   basedir)`), but `SeedCmd` ignores it. Running migrations from a non-root cwd (e.g. the
   `cmd/migrations` package dir) silently fails to find seed.sql.
2. `applyMigration`'s `defer tx.Rollback()` runs after `tx.Commit()` returns — harmless
   but the rollback-on-committed-tx returns an error that's swallowed; the `defer` pattern
   here is the usual "rollback unless commit" smell. Not a bug, but a test pinning the
   happy path makes any future cleanup safe.

`cmd/migrations/main.go` is a thin CLI wrapper; testable bits all live in `Migrator`.

## Goal

Cover migration apply/Up/Reset/Seed against a temp directory + in-memory/throwaway DB, and
fix `SeedCmd`'s path bug.

## Approach

1. **Fix `SeedCmd`** to read `path.Join(m.basedir, "seed.sql")` (or accept a seed path on
   `NewSQLiteMigrator`). Keep `./database/seed.sql` as the default when basedir is
   `./database/migrations` by deriving: `filepath.Dir(m.basedir)` + `/seed.sql`, or by
   accepting `m.basedir` to be the *database dir* rather than the migrations subdir. Pick
   the smallest change: add an explicit `seedDir` derived from `basedir`'s parent.
2. **Tests** in `repository/sqlite_utils_test.go`, each using a fresh temp dir
   (`t.TempDir()`) with a couple of hand-written `.sql` migration files containing
   `CREATE TABLE …`:

   - `setupMigrationTable` — assert the `migrations` table exists after `NewSQLiteMigrator`.
   - `readAppliedMigrations` — empty initially; after `UpCmd`, contains the applied files.
   - `isMigrationApplied` — true/false boundaries.
   - `readLocalMigrationFiles` — returns the files in the temp dir.
   - `applyMigration` — applies one file; row visible in `migrations` table; the file's SQL
     took effect (e.g. a table exists).
   - `UpCmd` — happy path: applies all unapplied files in order; idempotent (second call
     applies nothing); partial failure rolls back (a malformed file leaves earlier
     applied marks intact per the `tx` semantics — assert the exact current behavior and
     pin it).
   - `applyMigration` failure — SQL error → no `INSERT INTO migrations` happens; rollback
     leaves the migrations table unchanged.
   - `ResetCmd` — delete the file; idempotent if missing (returns nil per existing code).
   - `SeedCmd` — with the fixed path: creates a temp `seed.sql` in the expected location,
     applies it, asserts a seed row exists. Also a negative case: seed file missing →
     readable error.
   - `CreateCmd` — needs `os.Args` = `["migrations","create","name"]`; assert a new file
     `<unix>_<name>.sql` exists with the placeholder body. (Poke `os.Args` carefully and
     restore it.)

## Out of scope

- Adding a real migration tool dependency (sqlc, goose, …). Stays custom per the no-deps
  ethos.
- Multi-step migration semantics (column renames, data backfills) — current schema is
  single-migration; out of scope.

## Acceptance criteria

- [ ] `SeedCmd` reads seed.sql relative to its migration dir, not the cwd.
- [ ] All listed cases pass.
- [ ] `UpCmd` is idempotent under a multi-file temp dir.
- [ ] `go test -race ./repository/...` green; tests use `t.TempDir()` (cleaned by the
      framework).
- [ ] `cmd/migrations` still runs from `script/migrations.sh` unchanged.

## Files touched

- `repository/sqlite_utils.go` (`SeedCmd` path fix)
- `repository/sqlite_utils_test.go` (new)

## Notes

- `CreateCmd` reading `os.Args` is awkward to test; an alternative is to refactor to
  `CreateCmd(name string)`. Prefer the refactor if the test gets ugly — it's a tiny
  signature change and makes the function honest. Note the deviation from
  `os.Args`-driven sibling methods in a comment.
- Tests must not touch the project's real `emojix.db` — `t.TempDir()` + the `ResetCmd`
  path via a temp filename avoids that.