#!/usr/bin/env sh

echo "Updating cmd/gitrpcserver/wire.go"
go run github.com/google/wire/cmd/wire gen github.com/harness/gitness/cmd/gitrpcserver