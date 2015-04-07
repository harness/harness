SHA := $(shell git rev-parse --short HEAD)
VERSION := 0.4.0-alpha

all: build

deps:
	go get -t -v ./...

test:
	go vet ./...
	go test -cover -short ./...

build:
	go build -ldflags "-X main.revision $(SHA) -X main.version $(VERSION).$(SHA)"

clean:
	find . -name "*.out" -delete
	rm -f drone
