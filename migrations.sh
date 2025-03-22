set -a
. .env
set +a

go run ./cmd/migrations/main.go "$@"
