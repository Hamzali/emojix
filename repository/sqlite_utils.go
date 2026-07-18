package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"
)

type Migration struct {
	Name      string
	AppliedAt string
}

type Migrator struct {
	basedir          string
	dbname           string
	db               *sql.DB
	migrationFiles   []string
	appliedMigration []Migration
}

func NewSQLiteMigrator(db *sql.DB, dbname string, basedir string) (*Migrator, error) {
	migrator := Migrator{basedir: basedir, dbname: dbname, db: db}
	err := migrator.init()

	return &migrator, err
}

func (m *Migrator) init() error {
	err := m.setupMigrationTable()
	if err != nil {
		return err
	}

	m.appliedMigration, err = m.readAppliedMigrations()
	if err != nil {
		return err
	}

	m.migrationFiles, err = m.readLocalMigrationFiles()
	if err != nil {
		return err
	}

	return nil
}

func (m *Migrator) setupMigrationTable() error {
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			name TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		);
	`)
	return err
}

func (m *Migrator) readAppliedMigrations() ([]Migration, error) {
	appliedMigrations := []Migration{}
	rows, err := m.db.Query("select * from migrations order by name;")
	if err != nil {
		return appliedMigrations, err
	}

	defer rows.Close()

	for rows.Next() {
		migration := Migration{}

		err := rows.Scan(&migration.Name, &migration.AppliedAt)

		if err != nil {
			return appliedMigrations, err
		}

		appliedMigrations = append(appliedMigrations, migration)
	}

	return appliedMigrations, nil
}

func (m *Migrator) readLocalMigrationFiles() ([]string, error) {
	result := []string{}

	migrationsDir, err := os.ReadDir(m.basedir)
	if err != nil {
		return result, err
	}

	for _, migrationFile := range migrationsDir {
		result = append(result, migrationFile.Name())
	}

	return result, nil
}

// CreateCmd creates a new migration file named "<unix>_<name>.sql" in the migration
// dir. The name is passed explicitly rather than read from os.Args so the method is
// testable and free of hidden global-state dependencies; the CLI wrapper in
// cmd/migrations passes the argument it parses from os.Args[2].
func (m *Migrator) CreateCmd(name string) error {
	if name == "" {
		return errors.New("invalid migration name")
	}
	filename := fmt.Sprintf("%d_%s.sql", time.Now().Unix(), name)

	err := os.WriteFile(path.Join(m.basedir, filename), []byte("-- write your migration here"), 0777)

	if err != nil {
		return err
	}

	return nil
}

func (m *Migrator) isMigrationApplied(name string) bool {
	for _, mg := range m.appliedMigration {
		if mg.Name == name {
			return true
		}
	}

	return false
}
func (m *Migrator) UpCmd() error {

	for _, mf := range m.migrationFiles {
		log.Printf("applying migration %s", mf)
		if m.isMigrationApplied(mf) {
			log.Printf("migration %s is already applied", mf)
			continue
		}
		err := m.applyMigration(mf)
		if err != nil {
			log.Printf("failed to apply %s\n", mf)
			return err
		}
		// Keep the in-memory record in sync so a subsequent UpCmd in the same
		// process is a no-op rather than re-applying already-applied files.
		m.appliedMigration = append(m.appliedMigration, Migration{Name: mf})
	}

	return nil

}
func (m *Migrator) applyMigration(migrationName string) error {

	content, err := os.ReadFile(path.Join(m.basedir, migrationName))
	if err != nil {
		return err
	}
	migrationSql := string(content)

	tx, err := m.db.Begin()
	defer tx.Rollback()

	if err != nil {
		return err
	}

	_, err = tx.Exec(migrationSql)

	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO migrations (name, applied_at) VALUES (?, ?)", migrationName, time.Now().Format(time.RFC1123))

	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (m *Migrator) ResetCmd() error {
	err := os.Remove(m.dbname)

	if os.IsNotExist(err) {
		return nil
	}

	return err
}

func (m *Migrator) SeedCmd() error {
	log.Printf("applying seed.sql")
	content, err := os.ReadFile(filepath.Join(filepath.Dir(m.basedir), "seed.sql"))
	if err != nil {
		return err
	}
	seedSql := string(content)

	tx, err := m.db.Begin()
	defer tx.Rollback()
	if err != nil {
		return err
	}

	_, err = tx.Exec(seedSql)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}
