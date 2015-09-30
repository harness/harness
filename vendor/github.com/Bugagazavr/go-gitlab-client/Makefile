all:deps test

deps:
	go get github.com/stretchr/testify
	go get ./...

test:
	go test -cover -short ./...
