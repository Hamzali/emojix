#!/usr/bin/env sh
set -e
echo "fmt check"
[ -z "$(gofmt -l .)" ] || { echo "gofmt needed:"; gofmt -l .; exit 1; }
echo "vet"
go vet ./...
echo "test (race + cover)"
# -coverpkg=./... counts cross-package coverage (e.g. server tests exercising
# the usecase layer); per-package default undercounts the real number.
# -count=1 bypasses the test cache so the gate always runs the suite.
# -shuffle=on surfaces hidden test-order dependencies.
go test -race -cover -count=1 -shuffle=on -coverpkg=./... -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -1