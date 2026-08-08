# emojix

Multiplayer emoji charades. Tell a word with emojis, others guess.

## Run

```bash
go run ./cmd/emojix migrate fresh   # reset + up + seed
go run ./cmd/emojix serve           # http://localhost:9000
go run ./cmd/emojix dev             # serve + reload on .go/.gohtml changes
```

## Test

```bash
gofmt -l .
go vet ./...
go test -race -cover ./...
```

## Migrate

```bash
go run ./cmd/emojix migrate up
go run ./cmd/emojix migrate create add_something
go run ./cmd/emojix migrate seed
go run ./cmd/emojix migrate reset
```

All migrate/serve/dev commands accept `-db path` (default `emojix.db`).

## Stack

Go, SQLite, SSE, HTMX, plain CSS/JS. See `AGENTS.md`.
