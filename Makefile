.PHONY: dist

SHA := $(shell git rev-parse --short HEAD)
VERSION := 0.4.0-alpha

all: concat bindata build

deps:
	go get -t -v ./...

test:
	go vet ./...
	go test -cover -short ./...

build:
	mkdir -p bin
	go build -o bin/drone -ldflags "-X main.revision $(SHA) -X main.version $(VERSION).$(SHA)"

clean:
	find . -name "*.out" -delete
	rm -f drone
	rm -f bindata.go

concat:
	cat server/static/scripts/drone.js         \
		server/static/scripts/services/*.js    \
		server/static/scripts/filters/*.js     \
		server/static/scripts/controllers/*.js \
		server/static/scripts/term.js          > server/static/scripts/drone.min.js

bindata_deps:
	go get github.com/jteeuwen/go-bindata/...

bindata_debug:
	$$GOPATH/bin/go-bindata --debug server/static/...

bindata:
	$$GOPATH/bin/go-bindata server/static/...

# creates a debian package for drone
# to install `sudo dpkg -i drone.deb`
dist:
	mkdir -p dist/drone/usr/local/bin
	mkdir -p dist/drone/var/lib/drone
	mkdir -p dist/drone/var/cache/drone
	cp bin/drone dist/drone/usr/local/bin
	-dpkg-deb --build dist/drone
