FROM golang:1.4.2
ENV DRONE_SERVER_PORT :80

ADD . /go/src/github.com/drone/drone/
ADD ./dist/drone/etc/drone/drone.toml /etc/drone/drone.toml
RUN mkdir -p /var/lib/drone
WORKDIR /go/src/github.com/drone/drone

RUN apt-get update                                                                 \
	&& apt-get install -y libsqlite3-dev                                       \
	&& go get -u github.com/jteeuwen/go-bindata/...                            \
	&& make bindata deps                                                       \
	&& make build

EXPOSE 80
ENTRYPOINT ["/go/src/github.com/drone/drone/bin/drone"]
