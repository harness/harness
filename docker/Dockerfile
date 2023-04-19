### Build operator
FROM golang:1.19-alpine as builder

RUN apk update \
    && apk add --no-cache protoc build-base git

# Setup workig dir
WORKDIR /app

# Access to private repos
ARG GITHUB_ACCESS_TOKEN
RUN git config --global url."https://${GITHUB_ACCESS_TOKEN}:x-oauth-basic@github.com/harness".insteadOf "https://github.com/harness"
RUN go env -w GOPRIVATE=github.com/harness/*

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
ARG GITNESS_VERSION_MAJOR
ARG GITNESS_VERSION_MINOR
ARG GITNESS_VERSION_PATCH
RUN make harness-build

### Pull CA Certs
FROM alpine:latest as cert-image

RUN apk --update add ca-certificates

### Create final image
FROM alpine/git:2.36.3 as final

RUN adduser -u 1001 -D -h /app iamuser

WORKDIR /app

COPY --from=cert-image /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/gitness /app/gitness

RUN chown -R 1001:1001 /app
RUN chmod -R 700 /app/gitness

EXPOSE 3000
EXPOSE 3001

USER 1001

ENTRYPOINT [ "/app/gitness", "server" ]