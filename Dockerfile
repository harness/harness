FROM golang:1.13 as droneserver

COPY . /go/
# compile the server using the cgo
RUN GOPATH="" GOOS=linux GOARCH=amd64 go build -tags "nolimit" -ldflags "-extldflags \"-static\"" -o release/linux/amd64/drone-server github.com/drone/drone/cmd/drone-server

FROM alpine:3.9 as alpine
RUN apk add -U --no-cache ca-certificates

FROM alpine:3.9

EXPOSE 80 443
VOLUME /data

RUN [ ! -e /etc/nsswitch.conf ] && echo 'hosts: files dns' > /etc/nsswitch.conf

ENV GODEBUG netdns=go
ENV XDG_CACHE_HOME /data
ENV DRONE_DATABASE_DRIVER sqlite3
ENV DRONE_DATABASE_DATASOURCE /data/database.sqlite
ENV DRONE_RUNNER_OS=linux
ENV DRONE_RUNNER_ARCH=amd64
ENV DRONE_SERVER_PORT=:80
ENV DRONE_SERVER_HOST=localhost
ENV DRONE_DATADOG_ENABLED=true
ENV DRONE_DATADOG_ENDPOINT=https://tats.drone.ci/api/v1/series

COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=droneserver /go/release/linux/amd64/drone-server /bin/
ENTRYPOINT ["/bin/drone-server"]
