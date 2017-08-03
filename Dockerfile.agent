# docker build --rm -t drone/drone .

FROM centurylink/ca-certs
ENV GODEBUG=netdns=go
ADD release/drone-agent /bin/

ENTRYPOINT ["/bin/drone-agent"]
