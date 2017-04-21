# docker build --rm -t drone/drone .

FROM busybox
EXPOSE 8000

# Pulled from centurylin/ca-certs source.
ADD https://raw.githubusercontent.com/CenturyLinkLabs/ca-certs-base-image/master/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENV DATABASE_DRIVER=sqlite3
ENV DATABASE_CONFIG=/var/lib/drone/drone.sqlite
ENV GODEBUG=netdns=go

ADD release/drone /drone

ENTRYPOINT ["/drone"]
CMD ["server"]
