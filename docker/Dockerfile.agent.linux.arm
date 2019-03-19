FROM drone/ca-certs
ENV GODEBUG=netdns=go
ENV DRONE_RUNNER_OS=linux
ENV DRONE_RUNNER_ARCH=arm
ENV DRONE_RUNNER_PLATFORM=linux/arm
ENV DRONE_RUNNER_CAPACITY=1
ENV DRONE_RUNNER_VARIANT=v7
ADD release/linux/arm/drone-agent /bin/

LABEL com.centurylinklabs.watchtower.stop-signal="SIGINT"

ENTRYPOINT ["/bin/drone-agent"]
