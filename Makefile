.PHONY: vendor

PACKAGES = $(shell go list ./... | grep -v /vendor/)

all: gen build

deps:
	go get golang.org/x/tools/cmd/cover
	go get golang.org/x/tools/cmd/vet
	go get -u github.com/kr/vexp
	go get -u github.com/eknkc/amber/amberc
	go get -u github.com/jteeuwen/go-bindata/...
	go get -u github.com/elazarl/go-bindata-assetfs/...
	go get -u github.com/dchest/jsmin

gen:
	go generate $(PACKAGES)

build:
	go build

build_static:
	go build --ldflags '-extldflags "-static"' -o drone_static

test:
	go test -cover $(PACKAGES)

deb:
	mkdir -p contrib/debian/drone/usr/local/bin
	mkdir -p contrib/debian/drone/var/lib/drone
	mkdir -p contrib/debian/drone/var/cache/drone
	cp drone contrib/debian/drone/usr/local/bin
	-dpkg-deb --build contrib/debian/drone

vendor:
	vexp
