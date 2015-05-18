# Docker image for the Drone build agent
#
#     CGO_ENABLED=0 go build -a -tags netgo
#     docker build --rm=true -t drone/drone-agent .

FROM gliderlabs/alpine:3.1
RUN apk-install ca-certificates
ADD drone-agent /bin/
ENTRYPOINT ["/bin/drone-agent"]
