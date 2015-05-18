# Docker image for Drone's slack notification plugin
#
#     CGO_ENABLED=0 go build -a -tags netgo
#     docker build --rm=true -t drone/drone-build .

FROM gliderlabs/alpine:3.1
RUN apk-install ca-certificates
ADD drone-build /bin/
ENTRYPOINT ["/bin/drone-build"]
