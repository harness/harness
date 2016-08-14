# docker build --rm -t drone/drone .

FROM centurylink/ca-certs
EXPOSE 8000

ENV DATABASE_DRIVER=sqlite3
ENV DATABASE_CONFIG=/var/lib/drone/drone.sqlite
ENV GODEBUG=netdns=go

ADD release/drone /drone

ENTRYPOINT ["/drone"]
CMD ["server"]
