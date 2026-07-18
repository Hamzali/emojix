#!/usr/bin/env sh
set -e
echo "fmt check"
[ -z "$(gofmt -l .)" ] || { echo "gofmt needed:"; gofmt -l .; exit 1; }
echo "vet"
go vet ./...
echo "test (race + cover)"
go test -race -cover -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -1