SELFPKG := github.com/drone/drone
VERSION := 0.2
SHA := $(shell git rev-parse --short HEAD)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
PKGS := \
build \
build/buildfile \
build/docker \
build/dockerfile \
build/proxy \
build/repo \
build/script \
channel \
database \
database/encrypt \
database/migrate/testing \
database/testing \
handler/testing \
mail \
model \
plugin/deploy \
plugin/publish \
queue
PKGS := $(addprefix github.com/drone/drone/pkg/,$(PKGS))

GODEPSPATH := $(PWD)/Godeps/_workspace

.PHONY: all build build-dist bump-deps clean deps dpkg embed godep install js rice run test $(PKGS)

all: build

build: deps embed
	go build -o bin/drone -ldflags "-X main.version $(VERSION)dev-$(SHA)" $(SELFPKG)/cmd/drone
	go build -o bin/droned -ldflags "-X main.version $(VERSION)dev-$(SHA)" $(SELFPKG)/cmd/droned

bump-deps: godep
	go get -u -t -v ./...
	godep save ./...

deps:
	go get -t -v ./...

js:
	cd cmd/droned/assets && find js -name "*.js" ! -name '.*' ! -name "main.js" -exec cat {} \; > js/main.js

# Embed static assets
embed: js rice
	cd cmd/droned   && rice embed
	cd pkg/template && rice embed

$(PKGS):
	go test -v $@

install:
	cp deb/drone/etc/init/drone.conf /etc/init/drone.conf
	test -f /etc/default/drone || cp deb/drone/etc/default/drone /etc/default/drone
	cd bin && install -t /usr/local/bin drone
	cd bin && install -t /usr/local/bin droned
	mkdir -p /var/lib/drone

# creates a debian package for drone
# to install `sudo dpkg -i drone.deb`
dpkg: build-dist
	mkdir -p deb/drone/usr/local/bin
	mkdir -p deb/drone/var/lib/drone
	mkdir -p deb/drone/var/cache/drone
	cp bin/drone  deb/drone/usr/local/bin
	cp bin/droned deb/drone/usr/local/bin
	-dpkg-deb --build deb/drone

run:
	bin/droned --port=":8080" --datasource="drone.sqlite"

godep:
	go get github.com/tools/godep


build-dist: PATH := $(GODEPSPATH)/bin:$(GOPATH)/bin:$(PATH)
build-dist: GOPATH := $(GODEPSPATH):$(GOPATH)
build-dist: embed
	go build -o bin/drone -ldflags "-X main.version $(VERSION)-$(SHA)" $(SELFPKG)/cmd/drone
	go build -o bin/droned -ldflags "-X main.version $(VERSION)-$(SHA)" $(SELFPKG)/cmd/droned

rice: PATH := $(GODEPSPATH)/bin:$(GOPATH)/bin:$(PATH)
rice: GOPATH := $(GODEPSPATH):$(GOPATH)
rice:
	go get github.com/GeertJohan/go.rice/rice

test: PATH := $(GODEPSPATH)/bin:$(GOPATH)/bin:$(PATH)
test: GOPATH := $(GODEPSPATH):$(GOPATH)
test: $(PKGS)

clean: PATH := $(GODEPSPATH)/bin:$(GOPATH)/bin:$(PATH)
clean: GOPATH := $(GODEPSPATH):$(GOPATH)
clean: rice
	cd cmd/droned   && rice clean
	cd pkg/template && rice clean
	rm -rf cmd/drone/drone
	rm -rf cmd/droned/droned
	rm -rf cmd/droned/drone.sqlite
	rm -rf bin/drone
	rm -rf bin/droned
	rm -rf deb/drone.deb
	rm -rf usr/local/bin/drone
	rm -rf usr/local/bin/droned
	rm -rf drone.sqlite
	rm -rf /tmp/drone.sqlite

