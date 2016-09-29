.PHONY: build

PACKAGES = $(shell go list ./... | grep -v /vendor/)

ifneq ($(shell uname), Darwin)
	EXTLDFLAGS = -extldflags "-static" $(null)
else
	EXTLDFLAGS =
endif

all: gen build_static

deps: deps_backend deps_frontend

deps_frontend:
	go get -u github.com/drone/drone-ui/dist

deps_backend:
	go get -u golang.org/x/tools/cmd/cover
	go get -u github.com/jteeuwen/go-bindata/...
	go get -u github.com/elazarl/go-bindata-assetfs/...
	go get -u github.com/drone/mq/...
	go get -u github.com/tidwall/redlog

gen: gen_template gen_migrations

gen_template:
	go generate github.com/drone/drone/server/template

gen_migrations:
	go generate github.com/drone/drone/store/datastore/ddl

test:
	go test -cover $(PACKAGES)

# docker run --publish=3306:3306 -e MYSQL_DATABASE=test -e MYSQL_ALLOW_EMPTY_PASSWORD=yes  mysql:5.6.27
test_mysql:
	DATABASE_DRIVER="mysql" DATABASE_CONFIG="root@tcp(127.0.0.1:3306)/test?parseTime=true" go test github.com/drone/drone/store/datastore

# docker run --publish=5432:5432 postgres:9.4.5
test_postgres:
	DATABASE_DRIVER="postgres" DATABASE_CONFIG="host=127.0.0.1 user=postgres dbname=postgres sslmode=disable" go test github.com/drone/drone/store/datastore


# build the release files
build: build_static build_cross build_tar build_sha

build_static:
	go install -ldflags '${EXTLDFLAGS}-X github.com/drone/drone/version.VersionDev=$(DRONE_BUILD_NUMBER)' github.com/drone/drone/drone
	mkdir -p release
	cp $(GOPATH)/bin/drone release/

# TODO this is getting moved to a shell script, do not alter
build_cross:
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -o release/linux/amd64/drone   github.com/drone/drone/drone
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -o release/linux/arm64/drone   github.com/drone/drone/drone
	GOOS=linux   GOARCH=arm   CGO_ENABLED=0 go build -o release/linux/arm/drone     github.com/drone/drone/drone
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o release/windows/amd64/drone github.com/drone/drone/drone
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -o release/darwin/amd64/drone  github.com/drone/drone/drone

# TODO this is getting moved to a shell script, do not alter
build_tar:
	tar -cvzf release/linux/amd64/drone.tar.gz   -C release/linux/amd64   drone
	tar -cvzf release/linux/arm64/drone.tar.gz   -C release/linux/arm64   drone
	tar -cvzf release/linux/arm/drone.tar.gz     -C release/linux/arm     drone
	tar -cvzf release/windows/amd64/drone.tar.gz -C release/windows/amd64 drone
	tar -cvzf release/darwin/amd64/drone.tar.gz  -C release/darwin/amd64  drone

# TODO this is getting moved to a shell script, do not alter
build_sha:
	sha256sum release/linux/amd64/drone.tar.gz   > release/linux/amd64/drone.sha256
	sha256sum release/linux/arm64/drone.tar.gz   > release/linux/arm64/drone.sha256
	sha256sum release/linux/arm/drone.tar.gz     > release/linux/arm/drone.sha256
	sha256sum release/windows/amd64/drone.tar.gz > release/windows/amd64/drone.sha256
	sha256sum release/darwin/amd64/drone.tar.gz  > release/darwin/amd64/drone.sha256
