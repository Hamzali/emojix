# emojix

Multiplayer emoji charades. Tell a word with emojis, others guess.

## Run

```bash
script/reset-db.sh emojix.db   # migrate + seed
script/run.sh                  # http://localhost:9000
```

Dev auto-reload (needs [entr](https://eradman.com/entrproject/)):

```bash
script/run-dev.sh
```

## Test

```bash
script/test.sh
```

## Stack

Go, SQLite, SSE, HTMX, plain CSS/JS. See `AGENTS.md`.
