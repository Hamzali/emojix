#!/usr/bin/env zsh 
ls **/*.(go|html) | entr -r go run cmd/server/main.go
