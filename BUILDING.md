# Building Drone CI

## Install Go Dependencies

    $ go get ./...

## Server

If you plan on running the server binary in a Docker container,
run `export GOOS=linux GOARCH=amd64` first.

```
go build \
  -o release/drone-server \
  -ldflags "-extldflags -static -X github.com/drone/drone/version.VersionDev=build.$(date +'%s')" \
    github.com/drone/drone/cmd/drone-server
```

## Agent

```
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
  -o release/drone-agent \
    github.com/drone/drone/cmd/drone-agent
```

## Docker Images

Ensure the binaries have been built first (see above).

Build the server:

    $ docker build -t drone-server .

Build the agent:

    $ docker build -t drone-agent -f Dockerfile.agent .
