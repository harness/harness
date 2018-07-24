#!/bin/sh

# only execute this script as part of the pipeline.
[ -z "$CI" ] && echo "missing ci environment variable" && exit 2

# build a static binary with the build number and extra features.
go build -ldflags '-extldflags "-static" -X github.com/drone/drone/version.VersionDev=build.'${DRONE_BUILD_NUMBER} -o release/drone-server github.com/drone/drone/extras/cmd/drone-server
GOOS=linux GOARCH=amd64 CGO_ENABLED=0         go build -ldflags '-X github.com/drone/drone/version.VersionDev=build.'${DRONE_BUILD_NUMBER} -o release/drone-agent             github.com/drone/drone/cmd/drone-agent
GOOS=linux GOARCH=arm64 CGO_ENABLED=0         go build -ldflags '-X github.com/drone/drone/version.VersionDev=build.'${DRONE_BUILD_NUMBER} -o release/linux/arm64/drone-agent github.com/drone/drone/cmd/drone-agent
GOOS=linux GOARCH=arm   CGO_ENABLED=0 GOARM=7 go build -ldflags '-X github.com/drone/drone/version.VersionDev=build.'${DRONE_BUILD_NUMBER} -o release/linux/arm/drone-agent   github.com/drone/drone/cmd/drone-agent
