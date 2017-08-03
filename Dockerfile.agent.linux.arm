# docker build --rm -t drone/drone .

FROM centurylink/ca-certs
ENV GODEBUG=netdns=go
ENV DRONE_PLATFORM=linux/arm
ADD release/linux/arm/drone-agent /bin/

ENTRYPOINT ["/bin/drone-agent"]
