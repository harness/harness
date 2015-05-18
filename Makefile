.PHONY: dist

SHA := $(shell git rev-parse --short HEAD)
VERSION := 0.4.0-alpha

all: concat bindata build

deps:
	go get github.com/jteeuwen/go-bindata/...
	go get -t -v ./...

test:
	go vet ./...
	go test -cover -short ./...

build:
	go build -o bin/drone -ldflags "-X main.revision $(SHA) -X main.version $(VERSION).$(SHA)" github.com/drone/drone/cmd/drone-server

clean:
	find . -name "*.out" -delete
	rm -f drone
	rm -f bindata.go

concat:
	cat cmd/drone-server/static/scripts/drone.js         \
		cmd/drone-server/static/scripts/services/*.js    \
		cmd/drone-server/static/scripts/filters/*.js     \
		cmd/drone-server/static/scripts/controllers/*.js \
		cmd/drone-server/static/scripts/term.js          > cmd/drone-server/static/scripts/drone.min.js

# installs the drone binaries into bin
install:
	install -t /usr/local/bin bin/drone
	install -t /usr/local/bin bin/drone-agent

# embeds all the static files directly
# into the drone binary file
bindata:
	$$GOPATH/bin/go-bindata -o="cmd/drone-server/drone_bindata.go" cmd/drone-server/static/...

bindata_debug:
	$$GOPATH/bin/go-bindata --debug -o="cmd/drone-server/drone_bindata.go" cmd/drone-server/static/...

# creates a debian package for drone
# to install `sudo dpkg -i drone.deb`
dist:
	mkdir -p dist/drone/usr/local/bin
	mkdir -p dist/drone/var/lib/drone
	mkdir -p dist/drone/var/cache/drone
	cp bin/drone dist/drone/usr/local/bin
	-dpkg-deb --build dist/drone
