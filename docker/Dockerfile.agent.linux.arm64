FROM drone/ca-certs
ENV GODEBUG=netdns=go
ENV DRONE_RUNNER_OS=linux
ENV DRONE_RUNNER_ARCH=arm64
ENV DRONE_RUNNER_PLATFORM=linux/arm64
ENV DRONE_RUNNER_CAPACITY=1
ENV DRONE_RUNNER_VARIANT=v8
ADD release/linux/arm64/drone-agent /bin/

LABEL com.centurylinklabs.watchtower.stop-signal="SIGINT"

ENTRYPOINT ["/bin/drone-agent"]
