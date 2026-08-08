package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMigrateFresh(t *testing.T) {
	// tests run with cwd = package dir; migrate paths are repo-root relative.
	root := findGoMod(t)
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	dbPath := filepath.Join(t.TempDir(), "fresh.db")
	if err := migrate([]string{"fresh", "-db", dbPath}); err != nil {
		t.Fatalf("fresh: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM words").Scan(&n); err != nil {
		t.Fatalf("words table: %v", err)
	}
	if n == 0 {
		t.Fatal("expected seed words")
	}
}

func TestMigrateResetMissingIsOK(t *testing.T) {
	err := migrate([]string{"reset", "-db", filepath.Join(t.TempDir(), "nope.db")})
	if err != nil {
		t.Fatal(err)
	}
}

func findGoMod(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
