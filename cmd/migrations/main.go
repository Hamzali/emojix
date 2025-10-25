package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	_ "modernc.org/sqlite"
)

const baseDir = "./database/migrations"

type Migration struct {
	Name      string
	AppliedAt string
}

type Migrator struct {
	db               *sql.DB
	migrationFiles   []string
	appliedMigration []Migration
}

var dbname string = os.Getenv("DBNAME")

func NewMigrator() (*Migrator, error) {

	db, err := sql.Open("sqlite", dbname)

	if err != nil {
		return nil, err
	}

	log.Printf("connected to %s database", dbname)

	migrator := Migrator{db: db}
	err = migrator.init()

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

	migrationsDir, err := os.ReadDir(baseDir)
	if err != nil {
		return result, err
	}

	for _, migrationFile := range migrationsDir {
		result = append(result, migrationFile.Name())
	}

	return result, nil
}

func (m *Migrator) createCmd() error {
	if len(os.Args) != 3 {
		return errors.New("migration name is missing")
	}

	migrationName := os.Args[2]
	if migrationName == "" {
		return errors.New("invalid migration name")
	}
	filename := fmt.Sprintf("%d_%s.sql", time.Now().Unix(), migrationName)

	err := os.WriteFile(path.Join(baseDir, filename), []byte("-- write your migration here"), 0777)

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
func (m *Migrator) upCmd() error {

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
	}

	return nil

}
func (m *Migrator) applyMigration(migrationName string) error {

	content, err := os.ReadFile(path.Join(baseDir, migrationName))
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

func (m *Migrator) resetCmd() error {
	err := os.Remove(dbname)
	return err
}

func (m *Migrator) seedCmd() error {
	content, err := os.ReadFile("seed.sql")
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

func (m *Migrator) ExecuteCmd() error {

	cmd := os.Args[1]

	switch cmd {
	case "create":
		return m.createCmd()
	case "up":
		return m.upCmd()
	case "reset":
		return m.resetCmd()
	case "seed":
		return m.seedCmd()
	default:
		return errors.New("invalid cmd")
	}
}

func main() {
	migartor, err := NewMigrator()
	if err != nil {
		log.Fatalln(err)
	}

	err = migartor.ExecuteCmd()
	if err != nil {
		log.Fatalln(err)
	}
}
