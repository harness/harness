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
	go get -u github.com/PuerkitoBio/goquery
	go get -u github.com/russross/blackfriday
	GO15VENDOREXPERIMENT=1 go get -u github.com/go-swagger/go-swagger/...

gen: gen_static gen_template gen_migrations

gen_static:
	mkdir -p static/docs_gen/api static/docs_gen/build
	mkdir -p static/docs_gen/api static/docs_gen/plugin
	mkdir -p static/docs_gen/api static/docs_gen/setup
	mkdir -p static/docs_gen/api static/docs_gen/cli
	go generate github.com/drone/drone/static

gen_template:
	go generate github.com/drone/drone/template

gen_migrations:
	go generate github.com/drone/drone/store/migration

build:
	go build

build_static:
	go build --ldflags '-extldflags "-static" -X main.build=$(CI_BUILD_NUMBER)' -o drone_static

test:
	go test -cover $(PACKAGES)

# docker run --publish=3306:3306 -e MYSQL_DATABASE=test -e MYSQL_ALLOW_EMPTY_PASSWORD=yes  mysql:5.6.27
test_mysql:
	DATABASE_DRIVER="mysql" DATABASE_CONFIG="root@tcp(127.0.0.1:3306)/test?parseTime=true" go test github.com/drone/drone/store/datastore

# docker run --publish=5432:5432 postgres:9.4.5
test_postgres:
	DATABASE_DRIVER="postgres" DATABASE_CONFIG="host=127.0.0.1 user=postgres dbname=postgres sslmode=disable" go test github.com/drone/drone/store/datastore

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
	cp -a static/docs_gen/*   /drone/tmp/
	cp -a static/styles_gen   /drone/tmp/static/
	cp -a static/images       /drone/tmp/static/
