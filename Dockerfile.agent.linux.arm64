# docker build --rm -t drone/drone .

FROM centurylink/ca-certs
ENV GODEBUG=netdns=go
ENV DRONE_PLATFORM=linux/arm64
ADD release/linux/arm64/drone-agent /bin/

ENTRYPOINT ["/bin/drone-agent"]
