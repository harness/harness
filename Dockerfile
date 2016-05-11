# Build the drone executable on a x64 Linux host:
#
#     go build --ldflags '-extldflags "-static"' -o drone
#
# Build the docker image:
#
#     docker build --rm=true -t drone/drone .

FROM busybox
EXPOSE 8000
ADD contrib/docker/etc/nsswitch.conf /etc/

# Pulled from centurylin/ca-certs source.
ADD https://raw.githubusercontent.com/CenturyLinkLabs/ca-certs-base-image/master/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENV DATABASE_DRIVER=sqlite3
ENV DATABASE_CONFIG=/var/lib/drone/drone.sqlite

ADD drone/drone /drone

# Alpine Linux doesn't use pam, which means that there is no /etc/nsswitch.conf,
# but Go and CGO rely on /etc/nsswitch.conf to check the order of DNS resolving.
# To fix this we just create /etc/nsswitch.conf and add the following line:
#RUN echo 'hosts: files mdns4_minimal [NOTFOUND=return] dns mdns4' >> /etc/nsswitch.conf

ENTRYPOINT ["/drone"]
CMD ["serve"]
