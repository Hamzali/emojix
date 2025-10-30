package main

import (
	"database/sql"
	"emojix/repository"
	"errors"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	dbname := os.Getenv("DBNAME")
	if dbname == "" {
		dbname = "emojix.db"
	}

	db, err := sql.Open("sqlite", dbname)
	if err != nil {
		log.Fatal("db connection error")
	}
	defer db.Close()

	migartor, err := repository.NewSQLiteMigrator(db, dbname, "./database/migrations")
	if err != nil {
		log.Fatalln(err)
	}

	cmd := os.Args[1]

	switch cmd {
	case "create":
		err = migartor.CreateCmd()
	case "up":
		err = migartor.UpCmd()
	case "reset":
		err = migartor.ResetCmd()
	case "seed":
		err = migartor.SeedCmd()
	default:
		err = errors.New("invalid cmd")
	}

	if err != nil {
		log.Fatalln(err)
	}

	log.Println("done")
}
