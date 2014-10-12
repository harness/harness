SHA := $(shell git rev-parse --short HEAD)

all: build

deps:
	# which npm && npm -g install uglify-js less
	go get github.com/GeertJohan/go.rice/rice
	go get github.com/franela/goblin
	go get -t -v ./...

test:
	go vet ./...
	go test -cover -short ./...

test_mysql:
	mysql -P 3306 --protocol=tcp -u root -e 'create database test;'
	TEST_DRIVER="mysql" TEST_DATASOURCE="root@tcp(127.0.0.1:3306)/test" go test -short github.com/drone/drone/server/datastore/database

test_postgres:
	TEST_DRIVER="postgres" TEST_DATASOURCE="host=127.0.0.1 user=postgres dbname=postgres sslmode=disable" go test -short github.com/drone/drone/server/datastore/database

build:
	mkdir -p packaging/root/usr/local/bin
	go build -o packaging/root/usr/local/bin/drone  -ldflags "-X main.revision $(SHA)" github.com/drone/drone/cli
	go build -o packaging/root/usr/local/bin/droned -ldflags "-X main.revision $(SHA)" github.com/drone/drone/server

install:
	install -t /usr/local/bin debian/drone/usr/local/bin/drone 
	install -t /usr/local/bin debian/drone/usr/local/bin/droned 

run:
	@go run server/main.go --config=$$HOME/.drone/config.toml

clean:
	find . -name "*.out" -delete
	rm -rf packaging/output
	rm -f packaging/root/usr/local/bin/drone
	rm -f packaging/root/usr/local/bin/droned
	mkdir -p packaging/output

lessc:
	lessc --clean-css server/app/styles/drone.less server/app/styles/drone.css

packages: clean build embed deb rpm

# embeds content in go source code so that it is compiled
# and packaged inside the go binary file.
embed:
	rice --import-path="github.com/drone/drone/server" append --exec="packaging/root/usr/local/bin/droned"

# creates a debian package for drone to install
# `sudo dpkg -i drone.deb`
deb:
	fpm -s dir -t deb -n drone -v 0.3 -p packaging/output/drone.deb \
		--deb-priority optional --category admin \
		--force \
		--deb-compression bzip2 \
	 	--after-install packaging/scripts/postinst.deb \
	 	--before-remove packaging/scripts/prerm.deb \
		--after-remove packaging/scripts/postrm.deb \
		--url https://github.com/drone/drone \
		--description "Drone continuous integration server" \
		-m "Brad Rydzewski <brad@drone.io>" \
		--license "Apache License 2.0" \
		--vendor "drone.io" -a amd64 \
		--config-files /etc/drone/drone.toml \
		packaging/root/=/

rpm:
	fpm -s dir -t rpm -n drone -v 0.3 -p packaging/output/drone.rpm \
		--rpm-compression bzip2 --rpm-os linux \
		--force \
	 	--after-install packaging/scripts/postinst.rpm \
	 	--before-remove packaging/scripts/prerm.rpm \
		--after-remove packaging/scripts/postrm.rpm \
		--url https://github.com/drone/drone \
		--description "Drone continuous integration server" \
		-m "Brad Rydzewski <brad@drone.io>" \
		--license "Apache License 2.0" \
		--vendor "drone.io" -a amd64 \
		--config-files /etc/drone/drone.toml \
		packaging/root/=/

# deploys drone to a staging server. this requires the following
# environment variables are set:
#
#   DRONE_STAGING_HOST -- the hostname or ip
#   DRONE_STAGING_USER -- the username used to login
#   DRONE_STAGING_KEY  -- the identity file path (~/.ssh/id_rsa)
deploy:
	scp -i $$DRONE_STAGING_KEY debian/drone.deb $$DRONE_STAGING_USER@$$DRONE_STAGING_HOST:/tmp
	ssh -i $$DRONE_STAGING_KEY $$DRONE_STAGING_USER@$$DRONE_STAGING_HOST -- dpkg -i /tmp/drone.deb