FROM golang:1.4.2

ENV DRONE_SERVER_PORT :80
WORKDIR $GOPATH/src/github.com/drone/drone

EXPOSE 80

ENTRYPOINT ["/usr/local/bin/drone"]
CMD ["-config", "/tmp/drone.toml"]

RUN apt-get update                                                                       \
    && apt-get install -y libsqlite3-dev                                                 \
    && git clone git://github.com/gin-gonic/gin.git $GOPATH/src/github.com/gin-gonic/gin \
    && go get -u github.com/jteeuwen/go-bindata/...

RUN touch /tmp/drone.toml

ADD . .
RUN make bindata deps           \
    && make build               \
    && mv bin/* /usr/local/bin/ \
    && rm -rf bin cmd/drone-server/drone_bindata.go
