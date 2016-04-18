#!/bin/bash

version=${1-"0.2.5"}
mkdir -p /drone/dist/binaries
gox -os="linux darwin windows" -output="/drone/dist/binaries/{{.Dir}}_{{.OS}}_{{.Arch}}" ./cmd/swagger

ls /drone/dist/binaries

cd /drone/dist
mkdir -p /drone/dist/linux/amd64/usr/bin
cp /drone/dist/binaries/swagger_linux_amd64 /drone/dist/linux/amd64/usr/bin/swagger
fpm -t deb -s dir -C /drone/dist/linux/amd64 -v "$version" -n swagger --license "ASL 2.0" -a x86_64 -m "ivan@flanders.co.nz" --url "https://goswagger.io" usr
fpm -t rpm -s dir -C /drone/dist/linux/amd64 -v "$version" -n swagger --license "ASL 2.0" -a x86_64 -m "ivan@flanders.co.nz" --url "https://goswagger.io" usr
