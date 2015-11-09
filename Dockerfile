# Build the drone executable on a x64 Linux host:
#
#     go build --ldflags '-extldflags "-static"' -o drone_static
#
#
# Alternate command for Go 1.4 and older:
#
#     go build -a -tags netgo --ldflags '-extldflags "-static"' -o drone_static
#
#
# Build the docker image:
#
#     docker build --rm=true -t drone/drone .

FROM centurylink/ca-certs
EXPOSE 8000
ADD contrib/docker/etc/nsswitch.conf /etc/

ENV DATABASE_DRIVER=sqlite3
ENV DATABASE_CONFIG=/var/lib/drone/drone.sqlite

ADD drone_static /drone_static

ENTRYPOINT ["/drone_static"]
