### Build operator
FROM golang:1.19 as builder

RUN dpkg --add-architecture amd64 \
    && apt update \
    && apt install -y --no-install-recommends protobuf-compiler

# Setup workig dir
WORKDIR /app

# Get dependancies - will also be cached if we won't change mod/sum
COPY go.mod .
COPY go.sum .
COPY Makefile .
RUN make dep
RUN make tools
# COPY the source code as the last step
COPY . .
# set required build flags
ENV CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64

# build
ARG GIT_COMMIT
ARG GITNESS_VERSION
RUN make harness-build

### Pull CA Certs
FROM alpine:latest as cert-image

RUN apk --update add ca-certificates

### Create final image
FROM debian:buster-slim as final

RUN useradd -u 1001 -m -d /app iamuser

RUN apt update \
    && apt install -y --no-install-recommends git

WORKDIR /app

COPY --from=cert-image /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/gitness /app/gitness

RUN chown -R 1001:1001 /app
RUN chmod -R 700 /app/gitness

EXPOSE 3000
EXPOSE 3001

USER 1001

ENTRYPOINT [ "/app/gitness", "server" ]