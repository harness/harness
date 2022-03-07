#!/bin/sh

echo "building docker images for ${GOOS}/${GOARCH} ..."

REPO="github.com/drone/drone"

# compile the server using the cgo
go build -ldflags "-extldflags \"-static\"" -o release/${GOOS}/${GOARCH}/drone-server ${REPO}/cmd/drone-server
