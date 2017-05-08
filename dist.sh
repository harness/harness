#!/bin/sh
set -e
set -v

GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X github.com/drone/drone/version.VersionDev=build.${DRONE_BUILD_NUMBER}" -o release/linux/amd64/drone   github.com/drone/drone/drone
GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-X github.com/drone/drone/version.VersionDev=build.${DRONE_BUILD_NUMBER}" -o release/linux/arm64/drone   github.com/drone/drone/drone
GOOS=linux   GOARCH=arm   CGO_ENABLED=0 go build -ldflags "-X github.com/drone/drone/version.VersionDev=build.${DRONE_BUILD_NUMBER}" -o release/linux/arm/drone     github.com/drone/drone/drone
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X github.com/drone/drone/version.VersionDev=build.${DRONE_BUILD_NUMBER}" -o release/windows/amd64/drone github.com/drone/drone/drone
GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X github.com/drone/drone/version.VersionDev=build.${DRONE_BUILD_NUMBER}" -o release/darwin/amd64/drone  github.com/drone/drone/drone

tar -cvzf release/linux/amd64/drone.tar.gz   -C release/linux/amd64   drone
tar -cvzf release/linux/arm64/drone.tar.gz   -C release/linux/arm64   drone
tar -cvzf release/linux/arm/drone.tar.gz     -C release/linux/arm     drone
tar -cvzf release/windows/amd64/drone.tar.gz -C release/windows/amd64 drone
tar -cvzf release/darwin/amd64/drone.tar.gz  -C release/darwin/amd64  drone

sha256sum release/linux/amd64/drone.tar.gz   > release/linux/amd64/drone.sha256
sha256sum release/linux/arm64/drone.tar.gz   > release/linux/arm64/drone.sha256
sha256sum release/linux/arm/drone.tar.gz     > release/linux/arm/drone.sha256
sha256sum release/windows/amd64/drone.tar.gz > release/windows/amd64/drone.sha256
sha256sum release/darwin/amd64/drone.tar.gz  > release/darwin/amd64/drone.sha256
