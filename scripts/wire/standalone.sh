#!/usr/bin/env sh

echo "Updating standalone.wire_gen.go"
go run github.com/google/wire/cmd/wire gen -tags= -output_file_prefix="standalone." github.com/harness/gitness/cli/server
perl -ni -e 'print unless /go:generate/' cli/server/standalone.wire_gen.go
perl -i -pe's/\+build !wireinject/\+build !wireinject,!harness/g' cli/server/standalone.wire_gen.go
perl -i -pe's/go:build !wireinject/go:build !wireinject && !harness/g' cli/server/standalone.wire_gen.go