# Docker image for Drone's git-clone plugin
#
#     docker build --rm=true -t drone/drone-build .

FROM library/golang:1.4

# copy the local package files to the container's workspace.
#ADD . /go/src/github.com/drone/drone-build/

# build the program inside the container.
#RUN go get github.com/drone/drone-build/... && \
#    go install github.com/drone/drone-build


ADD drone-build /go/bin/

# run the git-clone plugin when the container starts
ENTRYPOINT ["/go/bin/drone-build"]
