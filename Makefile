SHA := $(shell git rev-parse --short HEAD)
VERSION := 0.4.0-alpha

all: concat bindata build

deps:
	go get -u github.com/jteeuwen/go-bindata/...
	go get -t -v ./...

test:
	go vet ./...
	go test -cover -short ./...

build:
	go build -ldflags "-X main.revision $(SHA) -X main.version $(VERSION).$(SHA)"

clean:
	find . -name "*.out" -delete
	rm -f drone
	rm -f bindata.go

concat:
	cat server/static/scripts/drone.js         \
		server/static/scripts/services/*.js    \
		server/static/scripts/filters/*.js     \
		server/static/scripts/controllers/*.js \
		server/static/scripts/term.js          > server/static/drone.js

bindata_debug:
	go-bindata --debug server/static/...

bindata:
	go-bindata server/static/...