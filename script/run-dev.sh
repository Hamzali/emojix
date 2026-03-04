#!/usr/bin/env zsh
ls **/*.(go|gohtml) | entr -r go run cmd/server/main.go
