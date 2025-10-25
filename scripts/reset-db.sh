rm -f emojix.db
DBNAME=emojix.db go run cmd/migrations/main.go up
