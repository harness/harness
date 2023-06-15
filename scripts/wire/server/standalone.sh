#!/usr/bin/env sh

echo "Updating cmd/gitness/wire_gen.go"
go run github.com/google/wire/cmd/wire gen github.com/harness/gitness/cmd/gitness