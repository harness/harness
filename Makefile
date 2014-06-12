SHA := $(shell git rev-parse --short HEAD)

all: rice amberc lessc build

deps:
	go list github.com/drone/drone/... | xargs go get -t 

build:
	go build -o debian/drone/usr/local/bin/drone  -ldflags "-X main.revision $(SHA)" github.com/drone/drone/client
	go build -o debian/drone/usr/local/bin/droned -ldflags "-X main.revision $(SHA)" github.com/drone/drone/server

test:
	go vet ./...
	go test -cover -short ./...

clean:
	@find ./ -name '*.out'    | xargs rm  # remove go coverage output
	@find ./ -name '*.sqlite' | xargs rm  # remove sqlite databases
	rm -rf debian/drone/usr/local/bin/drone
	rm -rf debian/drone/usr/local/bin/droned
	rm -rf debian/drone.deb

	#cd cmd/droned/static   && rice clean
	#cd cmd/droned/template && rice clean

rice:
	cd server               && rice embed
	#cd server/template/html && rice embed

amberc:
	@for f in server/template/*.amber; do $$GOPATH/bin/amberc -pp=true "$$f" > "$${f%.amber}.html"; done
	@mkdir -p server/template/html
	@mv server/template/*.html server/template/html

lessc:
	@lessc server/static/styles/drone.less > server/static/styles/drone.css

uglify:
	yui-compressor --type='css' -o 'server/static/styles/drone.min.css' server/static/styles/drone.css

# npm install -g uglifycss
# npm install -g uglify-js
# npm install -g less

# creates a debian package for drone
# to install `sudo dpkg -i drone.deb`
dpkg:
	mkdir -p debian/drone/usr/local/bin
	-dpkg-deb --build debian/drone

run:
	@cd server && go run main.go conf.gomake