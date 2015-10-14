.PHONY: vendor docs

PACKAGES = $(shell go list ./... | grep -v /vendor/)

all: gen build

deps:
	go get -u golang.org/x/tools/cmd/cover
	go get -u golang.org/x/tools/cmd/vet
	go get -u github.com/kr/vexp
	go get -u github.com/eknkc/amber/...
	go get -u github.com/eknkc/amber
	go get -u github.com/jteeuwen/go-bindata/...
	go get -u github.com/elazarl/go-bindata-assetfs/...
	go get -u github.com/dchest/jsmin
	go get -u github.com/franela/goblin
	go get -u github.com/go-swagger/go-swagger/...
	go get -u github.com/PuerkitoBio/goquery
	go get -u github.com/russross/blackfriday

gen: gen_static gen_template gen_migrations

gen_static:
	mkdir -p static/docs_gen/api static/docs_gen/build
	mkdir -p static/docs_gen/api static/docs_gen/plugin
	mkdir -p static/docs_gen/api static/docs_gen/setup
	go generate github.com/drone/drone/static

gen_template:
	go generate github.com/drone/drone/template

gen_migrations:
	go generate github.com/drone/drone/shared/database

build:
	go build

build_static:
	go build --ldflags '-extldflags "-static" -X main.version=$(CI_BUILD_NUMBER)' -o drone_static

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

docs:
	mkdir -p /drone/tmp/docs
	mkdir -p /drone/tmp/static
	cp -a static/docs_gen/*   /drone/tmp/docs/
	cp -a static/styles_gen   /drone/tmp/static/
	cp -a static/images       /drone/tmp/static/
