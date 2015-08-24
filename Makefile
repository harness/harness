.PHONY: dist

SHA := $(shell git rev-parse --short HEAD)
VERSION := 0.4.0-alpha

all: concat bindata build

deps:
	go get github.com/jteeuwen/go-bindata/...

test:
	go vet github.com/drone/drone/pkg/...
	go vet github.com/drone/drone/cmd/...
	go test -cover -short github.com/drone/drone/pkg/...

# Execute the database test suite against mysql 5.5
#
# You can launch a mysql container locally for testing:
# docker run -rm -e MYSQL_ALLOW_EMPTY_PASSWORD=yes -e MYSQL_DATABASE=test -p 3306:3306 mysql:5.5
test_mysql:
	mysql -P 3306 --protocol=tcp -u root -e 'create database if not exists test;'
	TEST_DRIVER="mysql" TEST_DATASOURCE="root@tcp(127.0.0.1:3306)/test" go test -short github.com/drone/drone/pkg/store/builtin
	mysql -P 3306 --protocol=tcp -u root -e 'drop database test;'

build:
	go build -o bin/drone       -ldflags "-X main.revision=$(SHA) -X main.version=$(VERSION).$(SHA)" github.com/drone/drone/cmd/drone-server
	go build -o bin/drone-agent -ldflags "-X main.revision=$(SHA) -X main.version=$(VERSION).$(SHA)" github.com/drone/drone/cmd/drone-agent

run:
	bin/drone-server --debug

clean:
	find . -name "*.out" -delete
	find . -name "*_bindata.go" -delete
	rm -f bin/drone*

concat:
	cat cmd/drone-server/static/scripts/drone.js       \
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

docker:
	docker build --file=cmd/drone-build/Dockerfile.alpine --rm=true -t drone/drone-build .

# creates a debian package for drone
# to install `sudo dpkg -i drone.deb`
dist:
	mkdir -p dist/drone/usr/local/bin
	mkdir -p dist/drone/var/lib/drone
	mkdir -p dist/drone/var/cache/drone
	cp bin/drone dist/drone/usr/local/bin
	-dpkg-deb --build dist/drone
