FROM golang:1.5

WORKDIR /go/src/github.com/drone/drone
ADD . /go/src/github.com/drone/drone

RUN mkdir -p /var/lib/drone
RUN apt-get update      \
	&& apt-get install -y libsqlite3-dev                                       

RUN    go get -u github.com/jteeuwen/go-bindata/...  \
    && go run make.go bindata                        \
    && go run make.go build

ENV DRONE_SERVER_PORT :80
EXPOSE 80
ENTRYPOINT ["/go/src/github.com/drone/drone/bin/drone"]
