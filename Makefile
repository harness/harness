SHA := $(shell git rev-parse --short HEAD)

all: build

deps:
	# npm install -g uglify-js
	# npm install -g less
	# npm -g install karma
	# npm -g install karma-jasmine
	# npm -g install karma-chrome-launcher
	# npm -g install karma-phantomjs-launcher 
	go get github.com/GeertJohan/go.rice/rice
	go list github.com/drone/drone/... | xargs go get -t -v

build:
	mkdir -p debian/drone/usr/local/bin
	go build -o debian/drone/usr/local/bin/drone  -ldflags "-X main.revision $(SHA)" github.com/drone/drone/client
	go build -o debian/drone/usr/local/bin/droned -ldflags "-X main.revision $(SHA)" github.com/drone/drone/server

test:
	go vet ./...
	go test -cover -short ./...

run:
	@cd server && go run main.go

clean:
	@find . -name "*.out"         -delete # remove go coverage output
	@find . -name "*.sqlite"      -delete # remove sqlite databases
	@find . -name '*.rice-box.go' -delete # remove go rice files & embedded content
	#@find . -name '*.css' -delete
	@rm -r debian/drone/usr/local/bin debian/drone.deb server/server client/client server/template/html

dpkg: rice build deb

# embeds content in go source code so that it is compiled
# and packaged inside the go binary file.
rice:
	cd server && rice embed

lessc:
	lessc server/app/styles/drone.less server/app/styles/drone.css
	lessc --clean-css server/app/styles/drone.less server/app/styles/drone.min.css

# creates a debian package for drone to install
# `sudo dpkg -i drone.deb`
deb:
	mkdir -p debian/drone/usr/local/bin
	mkdir -p debian/drone/var/lib/drone
	dpkg-deb --build debian/drone

deploy:
	scp -i $$DRONE_STAGING_KEY debian/drone.deb $$DRONE_STAGING_USER@$$DRONE_STAGING_HOST:/tmp
	ssh -i $$DRONE_STAGING_KEY $$DRONE_STAGING_USER@$$DRONE_STAGING_HOST -- dpkg -i /tmp/drone.deb