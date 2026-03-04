export DBNAME=$1

go run cmd/migrations/main.go reset
go run cmd/migrations/main.go up
go run cmd/migrations/main.go seed
