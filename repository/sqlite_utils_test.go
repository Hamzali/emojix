package repository

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// newMemoryDB opens an in-memory sqlite database constrained to a single open
// connection so the shared in-memory database is visible across every
// transaction/statement issued through this *sql.DB.
func newMemoryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := InitSqliteDB(":memory:")
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	return db
}

// newTestMigrator builds a Migrator pointing at a fresh temp migrations dir
// populated with the given files (name -> content). It returns the migrator and
// the base dir it was configured with.
func newTestMigrator(t *testing.T, db *sql.DB, files map[string]string) (*Migrator, string) {
	t.Helper()
	basedir := filepath.Join(t.TempDir(), "migrations")
	if err := os.MkdirAll(basedir, 0755); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(basedir, name), []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	m, err := NewSQLiteMigrator(db, ":memory:", basedir)
	if err != nil {
		t.Fatalf("new migrator: %v", err)
	}
	return m, basedir
}

// appliedNames queries the migrations table directly and returns the set of
// applied migration names.
func appliedNames(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query("SELECT name FROM migrations ORDER BY name;")
	if err != nil {
		t.Fatalf("query migrations: %v", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan migration name: %v", err)
		}
		names = append(names, n)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}
	return names
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var count int
	err := db.QueryRow(
		"SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?;", name,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query sqlite_master for %s: %v", name, err)
	}
	return count == 1
}

func TestMigrator_setupMigrationTable(t *testing.T) {
	db := newMemoryDB(t)
	defer db.Close()

	m, _ := newTestMigrator(t, db, nil)

	// NewSQLiteMigrator must have created the migrations table.
	if !tableExists(t, m.db, "migrations") {
		t.Fatal("expected migrations table to exist after NewSQLiteMigrator")
	}
}

func TestMigrator_readAppliedMigrations(t *testing.T) {
	db := newMemoryDB(t)
	defer db.Close()

	m, _ := newTestMigrator(t, db, map[string]string{
		"0001_create_t.sql": "CREATE TABLE t (id INTEGER);",
	})

	if got := len(m.appliedMigration); got != 0 {
		t.Fatalf("expected 0 applied migrations initially, got %d", got)
	}

	if err := m.UpCmd(); err != nil {
		t.Fatalf("UpCmd: %v", err)
	}

	if got := len(m.appliedMigration); got != 1 {
		t.Fatalf("expected 1 applied migration after UpCmd, got %d", got)
	}
	if m.appliedMigration[0].Name != "0001_create_t.sql" {
		t.Errorf("expected applied name %q, got %q", "0001_create_t.sql", m.appliedMigration[0].Name)
	}
}

func TestMigrator_isMigrationApplied(t *testing.T) {
	db := newMemoryDB(t)
	defer db.Close()

	m, _ := newTestMigrator(t, db, map[string]string{
		"0001_create_t.sql": "CREATE TABLE t (id INTEGER);",
	})

	if m.isMigrationApplied("0001_create_t.sql") {
		t.Fatal("expected migration not applied before UpCmd")
	}
	if m.isMigrationApplied("does_not_exist.sql") {
		t.Fatal("non-existent migration should not be reported as applied")
	}

	if err := m.UpCmd(); err != nil {
		t.Fatalf("UpCmd: %v", err)
	}

	// appliedMigration is loaded at construction time; refresh in-memory state
	// by re-reading from the db to assert the on-disk truth.
	m.appliedMigration, _ = m.readAppliedMigrations()
	if !m.isMigrationApplied("0001_create_t.sql") {
		t.Error("expected migration to be applied after UpCmd")
	}
	if m.isMigrationApplied("9999_not_applied.sql") {
		t.Error("non-existent migration should not be reported as applied")
	}
}

func TestMigrator_readLocalMigrationFiles(t *testing.T) {
	db := newMemoryDB(t)
	defer db.Close()

	m, _ := newTestMigrator(t, db, map[string]string{
		"0001_a.sql": "CREATE TABLE a (id INTEGER);",
		"0002_b.sql": "CREATE TABLE b (id INTEGER);",
	})

	// os.ReadDir returns entries sorted by filename.
	want := []string{"0001_a.sql", "0002_b.sql"}
	if len(m.migrationFiles) != len(want) {
		t.Fatalf("expected %d files, got %d (%v)", len(want), len(m.migrationFiles), m.migrationFiles)
	}
	for i, name := range want {
		if m.migrationFiles[i] != name {
			t.Errorf("index %d: expected %q, got %q", i, name, m.migrationFiles[i])
		}
	}
}

func TestMigrator_applyMigration(t *testing.T) {
	db := newMemoryDB(t)
	defer db.Close()

	m, _ := newTestMigrator(t, db, map[string]string{
		"0001_create_t.sql": "CREATE TABLE t (id INTEGER);",
	})

	if err := m.applyMigration("0001_create_t.sql"); err != nil {
		t.Fatalf("applyMigration: %v", err)
	}

	// The SQL took effect: table t now exists.
	if !tableExists(t, db, "t") {
		t.Error("expected table t to exist after applyMigration")
	}

	// A row was recorded in the migrations table.
	names := appliedNames(t, db)
	if len(names) != 1 || names[0] != "0001_create_t.sql" {
		t.Errorf("expected migrations to contain %q, got %v", "0001_create_t.sql", names)
	}
}

func TestMigrator_UpCmd_happyAndIdempotent(t *testing.T) {
	db := newMemoryDB(t)
	defer db.Close()

	m, _ := newTestMigrator(t, db, map[string]string{
		"0001_create_a.sql": "CREATE TABLE a (id INTEGER);",
		"0002_create_b.sql": "CREATE TABLE b (id INTEGER);",
	})

	if err := m.UpCmd(); err != nil {
		t.Fatalf("UpCmd first call: %v", err)
	}

	if !tableExists(t, db, "a") || !tableExists(t, db, "b") {
		t.Fatal("expected both tables a and b to exist after UpCmd")
	}

	names := appliedNames(t, db)
	if len(names) != 2 {
		t.Fatalf("expected 2 applied migrations, got %v", names)
	}

	// Second call must be a no-op: no extra migration rows, no error.
	if err := m.UpCmd(); err != nil {
		t.Fatalf("UpCmd second call: %v", err)
	}
	names = appliedNames(t, db)
	if len(names) != 2 {
		t.Errorf("expected still 2 applied migrations after idempotent call, got %v", names)
	}
}

func TestMigrator_UpCmd_partialFailurePinsCurrentBehavior(t *testing.T) {
	db := newMemoryDB(t)
	defer db.Close()

	m, _ := newTestMigrator(t, db, map[string]string{
		"0001_create_a.sql": "CREATE TABLE a (id INTEGER);",
		// malformed SQL: references a table that does not exist.
		"0002_bad.sql": "INSERT INTO no_such_table VALUES (1);",
	})

	err := m.UpCmd()
	if err == nil {
		t.Fatal("expected UpCmd to return an error on a malformed migration")
	}

	// The first, well-formed migration committed before the failure.
	if !tableExists(t, db, "a") {
		t.Error("expected table a to exist from the first applied migration")
	}
	names := appliedNames(t, db)
	if len(names) != 1 || names[0] != "0001_create_a.sql" {
		t.Errorf("expected only 0001_create_a.sql recorded as applied, got %v", names)
	}
}

func TestMigrator_applyMigration_failureRollsBack(t *testing.T) {
	db := newMemoryDB(t)
	defer db.Close()

	m, _ := newTestMigrator(t, db, map[string]string{
		"0001_bad.sql": "INSERT INTO no_such_table VALUES (1);",
	})

	if err := m.applyMigration("0001_bad.sql"); err == nil {
		t.Fatal("expected applyMigration to error on malformed SQL")
	}

	// No INSERT INTO migrations should have happened.
	if names := appliedNames(t, db); len(names) != 0 {
		t.Errorf("expected no applied migrations after failure, got %v", names)
	}
}

func TestMigrator_ResetCmd(t *testing.T) {
	dir := t.TempDir()
	dbname := filepath.Join(dir, "test.db")

	// Create the file so ResetCmd has something to remove.
	if err := os.WriteFile(dbname, []byte{}, 0644); err != nil {
		t.Fatalf("write db file: %v", err)
	}

	db, err := InitSqliteDB(dbname)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	basedir := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(basedir, 0755); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	m, err := NewSQLiteMigrator(db, dbname, basedir)
	if err != nil {
		t.Fatalf("new migrator: %v", err)
	}

	if err := m.ResetCmd(); err != nil {
		t.Fatalf("ResetCmd first call: %v", err)
	}
	if _, err := os.Stat(dbname); !os.IsNotExist(err) {
		t.Fatalf("expected db file removed after ResetCmd, got err=%v", err)
	}

	// Idempotent: removing a missing file returns nil.
	if err := m.ResetCmd(); err != nil {
		t.Fatalf("ResetCmd second call on missing file: %v", err)
	}
}

func TestMigrator_SeedCmd_appliesSeed(t *testing.T) {
	db := newMemoryDB(t)
	defer db.Close()

	m, basedir := newTestMigrator(t, db, map[string]string{
		// words table is needed by the seed.
		"0001_words.sql": "CREATE TABLE words (id TEXT PRIMARY KEY, word TEXT, hint TEXT);",
	})
	if err := m.UpCmd(); err != nil {
		t.Fatalf("UpCmd: %v", err)
	}

	// Place seed.sql in the parent dir of the migration basedir, which is where
	// the fixed SeedCmd looks (filepath.Dir(basedir)/seed.sql).
	seedPath := filepath.Join(filepath.Dir(basedir), "seed.sql")
	seedSQL := "INSERT INTO words (id, word, hint) VALUES ('seed-1', 'seeded', '🌱');"
	if err := os.WriteFile(seedPath, []byte(seedSQL), 0644); err != nil {
		t.Fatalf("write seed: %v", err)
	}

	if err := m.SeedCmd(); err != nil {
		t.Fatalf("SeedCmd: %v", err)
	}

	var word string
	if err := db.QueryRow("SELECT word FROM words WHERE id = 'seed-1';").Scan(&word); err != nil {
		t.Fatalf("query seed row: %v", err)
	}
	if word != "seeded" {
		t.Errorf("expected seeded row, got word=%q", word)
	}
}

func TestMigrator_SeedCmd_missingFile(t *testing.T) {
	db := newMemoryDB(t)
	defer db.Close()

	m, _ := newTestMigrator(t, db, nil)

	err := m.SeedCmd()
	if err == nil {
		t.Fatal("expected SeedCmd to error when seed.sql is missing")
	}
	if !strings.Contains(err.Error(), "seed.sql") {
		t.Errorf("expected error to mention seed.sql, got %q", err.Error())
	}
}

func TestMigrator_CreateCmd(t *testing.T) {
	db := newMemoryDB(t)
	defer db.Close()

	m, basedir := newTestMigrator(t, db, nil)

	if err := m.CreateCmd("add_words"); err != nil {
		t.Fatalf("CreateCmd: %v", err)
	}

	entries, err := os.ReadDir(basedir)
	if err != nil {
		t.Fatalf("read basedir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 created migration file, got %d", len(entries))
	}

	filename := entries[0].Name()
	if !strings.HasSuffix(filename, "_add_words.sql") {
		t.Errorf("expected filename to end with _add_words.sql, got %q", filename)
	}

	body, err := os.ReadFile(filepath.Join(basedir, filename))
	if err != nil {
		t.Fatalf("read created migration: %v", err)
	}
	if string(body) != "-- write your migration here" {
		t.Errorf("expected placeholder body, got %q", string(body))
	}
}

func TestMigrator_CreateCmd_emptyName(t *testing.T) {
	db := newMemoryDB(t)
	defer db.Close()

	m, _ := newTestMigrator(t, db, nil)

	if err := m.CreateCmd(""); err == nil {
		t.Fatal("expected error for empty migration name")
	}
}
