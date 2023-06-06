### Build web
FROM node:16 as web

# Create app directory
WORKDIR /usr/src/app

COPY web/package.json ./
COPY web/yarn.lock ./

RUN --mount=type=secret,id=npmrc,target=/root/.npmrc yarn

# If you are building your code for production
# RUN npm ci --omit=dev

COPY ./web .

RUN yarn build && \
    yarn cache clean

### Build gitness
FROM golang:1.19-alpine as builder

RUN apk update \
    && apk add --no-cache protoc build-base git

# Setup workig dir
WORKDIR /app

# Access to private repos
ARG GITHUB_ACCESS_TOKEN
RUN git config --global url."https://${GITHUB_ACCESS_TOKEN}:x-oauth-basic@github.com/harness".insteadOf "https://github.com/harness"
RUN git config --global --add safe.directory '/app'
RUN go env -w GOPRIVATE=github.com/harness/*

# Get dependancies - will also be cached if we won't change mod/sum
COPY go.mod .
COPY go.sum .
COPY Makefile .
RUN make dep
RUN make tools
# COPY the source code as the last step
COPY . .

COPY --from=web /usr/src/app/dist /app/web/dist

# build
ARG GIT_COMMIT
ARG GITNESS_VERSION_MAJOR
ARG GITNESS_VERSION_MINOR
ARG GITNESS_VERSION_PATCH

# set required build flags
ARG sqlite
RUN if [[ -z "$sqlite" ]] ; then \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    make harness-build-pq \
; else \
    CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64 \
    make harness-build \
; fi

### Pull CA Certs
FROM alpine:latest as cert-image

RUN apk --update add ca-certificates

### Create final image
FROM us.gcr.io/platform-205701/ubi/ubi-go:8.7 as final

USER root
RUN mkdir /app && chown nobody:nobody /app

USER nobody
WORKDIR /app

COPY --chown=nobody:nobody --from=cert-image /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --chown=nobody:nobody --from=builder /app/gitness /app/gitness

RUN chmod -R 700 /app/gitness

EXPOSE 3000
EXPOSE 3001

ENTRYPOINT [ "/app/gitness", "server" ]