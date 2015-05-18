# Docker image for Drone's slack notification plugin
#
#     docker build --rm=true -t drone/drone-server .

FROM golang:1.4.2
ENV DRONE_SERVER_PORT :80

ADD . /gopath/src/github.com/drone/drone/
WORKDIR /gopath/src/github.com/drone/drone

RUN apt-get update                                                                       \
	&& apt-get install -y libsqlite3-dev                                                 \
	&& git clone git://github.com/gin-gonic/gin.git $GOPATH/src/github.com/gin-gonic/gin \
	&& go get -u github.com/jteeuwen/go-bindata/...                                      \
	&& make bindata deps                                                                 \
	&& make build

EXPOSE 80
ENTRYPOINT ["/gopath/src/github.com/drone/drone/bin/drone"]