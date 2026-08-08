package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "serve":
		err = serve(os.Args[2:])
	case "migrate":
		err = migrate(os.Args[2:])
	case "dev":
		err = dev(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `usage: go run ./cmd/emojix <command>

commands:
  serve              start the game server
  migrate <action>   db: up | reset | seed | fresh | create <name>
  dev                serve with auto-reload on .go/.gohtml changes

flags (serve, migrate, dev):
  -db string   sqlite file (default emojix.db)
`)
}
