.PHONY: vendor

PACKAGES = $(shell go list ./... | grep -v /vendor/)

all: gen build

deps:
	go get golang.org/x/tools/cmd/cover
	go get golang.org/x/tools/cmd/vet
	go get github.com/kr/vexp
	go get github.com/eknkc/amber/amberc
	go get github.com/eknkc/amber
	go get github.com/jteeuwen/go-bindata/...
	go get github.com/elazarl/go-bindata-assetfs/...
	go get github.com/dchest/jsmin
	go get github.com/franela/goblin
	go get github.com/go-swagger/go-swagger
	go get github.com/PuerkitoBio/goquery
	go get github.com/russross/blackfriday

gen: gen_static gen_template gen_migrations

gen_static:
	mkdir -p static/docs_gen/api static/docs_gen/build
	go generate github.com/drone/drone/static

gen_template:
	go generate github.com/drone/drone/template

gen_migrations:
	go generate github.com/drone/drone/shared/database

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
