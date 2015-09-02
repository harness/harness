.PHONY: dist

SHA := $(shell git rev-parse --short HEAD)
VERSION := 0.4.0-alpha

all: build

build:
	go run make.go bindata build


# Execute the database test suite against mysql 5.5
#
# You can launch a mysql container locally for testing:
# docker run -rm -e MYSQL_ALLOW_EMPTY_PASSWORD=yes -e MYSQL_DATABASE=test -p 3306:3306 mysql:5.5
test_mysql:
	mysql -P 3306 --protocol=tcp -u root -e 'create database if not exists test;'
	TEST_DRIVER="mysql" TEST_DATASOURCE="root@tcp(127.0.0.1:3306)/test" go test -short github.com/drone/drone/pkg/store/builtin
	mysql -P 3306 --protocol=tcp -u root -e 'drop database test;'

run:
	bin/drone --debug

# installs the drone binaries into bin
install:
	install -t /usr/local/bin bin/drone
	install -t /usr/local/bin bin/drone-agent

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
