# docker build --rm -t drone/drone .

FROM centurylink/ca-certs
EXPOSE 8000 80 443

ENV DATABASE_DRIVER=sqlite3
ENV DATABASE_CONFIG=/var/lib/drone/drone.sqlite
ENV GODEBUG=netdns=go
ENV XDG_CACHE_HOME /etc/letsencrypt

ADD release/drone /drone

ENTRYPOINT ["/drone"]
CMD ["server"]
