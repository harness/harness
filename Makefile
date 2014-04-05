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
database/migrate \
database/testing \
mail \
model \
plugin/deploy \
queue
PKGS := $(addprefix github.com/drone/drone/pkg/,$(PKGS))
.PHONY : test $(PKGS) godep rice Godeps/Godeps.json

all: embed build

godep:
	go get github.com/tools/godep

rice:
	go get github.com/GeertJohan/go.rice/rice

Godeps/Godeps.json: godep
	# can switch to copy=false once https://github.com/tools/godep/issues/28 is resolved
	# or if drone no longer has deps that use bzr
	godep save -copy=true $(PKGS)
	rm -r Godeps/_workspace Godeps/Readme # undo copying, give up dep tracking for bzr deps

deps: go-gitlab-client godep
	go get -d ./...
	godep restore

embed: js rice
	cd cmd/droned   && rice embed
	cd pkg/template && rice embed

js:
	cd cmd/droned/assets && find js -name "*.js" ! -name '.*' ! -name "main.js" -exec cat {} \; > js/main.js

build:
	cd cmd/drone  && go build -ldflags "-X main.version $(SHA)" -o ../../bin/drone
	cd cmd/droned && go build -ldflags "-X main.version $(SHA)" -o ../../bin/droned

test: $(PKGS)

$(PKGS):
	go test -v $@

install:
	cp deb/drone/etc/init/drone.conf /etc/init/drone.conf
	test -f /etc/default/drone || cp deb/drone/etc/default/drone /etc/default/drone
	cd bin && install -t /usr/local/bin drone
	cd bin && install -t /usr/local/bin droned
	mkdir -p /var/lib/drone

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

# creates a debian package for drone
# to install `sudo dpkg -i drone.deb`
dpkg:
	mkdir -p deb/drone/usr/local/bin
	mkdir -p deb/drone/var/lib/drone
	mkdir -p deb/drone/var/cache/drone
	cp bin/drone  deb/drone/usr/local/bin
	cp bin/droned deb/drone/usr/local/bin
	-dpkg-deb --build deb/drone

run:
	bin/droned --port=":8080" --datasource="drone.sqlite"

go-gitlab-client:
	rm -rf $$GOPATH/src/github.com/plouc/go-gitlab-client
	git clone -b raw-request https://github.com/fudanchii/go-gitlab-client $$GOPATH/src/github.com/plouc/go-gitlab-client
