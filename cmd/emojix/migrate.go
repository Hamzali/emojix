package main

import (
	"database/sql"
	"emojix/repository"
	"flag"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

const migrationsDir = "./database/migrations"

func migrate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("migrate needs an action: up | reset | seed | fresh | create <name>")
	}
	action := args[0]
	rest := args[1:]

	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	dbName := fs.String("db", "emojix.db", "sqlite file")

	// create takes a name before flags: migrate create add_foo -db x.db
	var createName string
	if action == "create" {
		if len(rest) < 1 {
			return fmt.Errorf("migrate create needs a name")
		}
		createName = rest[0]
		rest = rest[1:]
	}
	if err := fs.Parse(rest); err != nil {
		return err
	}

	switch action {
	case "reset":
		err := os.Remove(*dbName)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	case "fresh":
		_ = os.Remove(*dbName)
		return withMigrator(*dbName, func(m *repository.Migrator) error {
			if err := m.UpCmd(); err != nil {
				return err
			}
			return m.SeedCmd()
		})
	case "up", "seed", "create":
		return withMigrator(*dbName, func(m *repository.Migrator) error {
			switch action {
			case "up":
				return m.UpCmd()
			case "seed":
				return m.SeedCmd()
			default:
				return m.CreateCmd(createName)
			}
		})
	default:
		return fmt.Errorf("unknown migrate action %q", action)
	}
}

func withMigrator(dbName string, fn func(*repository.Migrator) error) error {
	db, err := sql.Open("sqlite", dbName)
	if err != nil {
		return err
	}
	defer db.Close()

	m, err := repository.NewSQLiteMigrator(db, dbName, migrationsDir)
	if err != nil {
		return err
	}
	return fn(m)
}
