FROM mcr.microsoft.com/windows/nanoserver:1803
USER ContainerAdministrator

ENV GODEBUG=netdns=go
ENV DRONE_RUNNER_OS=windows
ENV DRONE_RUNNER_ARCH=amd64
ENV DRONE_RUNNER_PLATFORM=windows/amd64
ENV DRONE_RUNNER_KERNEL=1803
ENV DRONE_RUNNER_CAPACITY=1

ADD release/windows/1803/amd64/drone-controller.exe C:/drone-controller.exe
ENTRYPOINT [ "C:\\drone-controller.exe" ]
