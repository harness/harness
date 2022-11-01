#!/usr/bin/env sh

echo "Updating harness.wire_gen.go"
go run github.com/google/wire/cmd/wire gen -tags=harness -output_file_prefix="harness." github.com/harness/gitness/cli/server
perl -ni -e 'print unless /go:generate/' cli/server/harness.wire_gen.go
perl -i -pe's/\+build !wireinject/\+build !wireinject,harness/g' cli/server/harness.wire_gen.go
perl -i -pe's/go:build !wireinject/go:build !wireinject && harness/g' cli/server/harness.wire_gen.go