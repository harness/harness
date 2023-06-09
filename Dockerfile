# ---------------------------------------------------------#
#                     Build web image                      #
# ---------------------------------------------------------#
FROM node:16 as web

WORKDIR /usr/src/app

COPY web/package.json ./
COPY web/yarn.lock ./

ARG GITHUB_ACCESS_TOKEN

# If you are building your code for production
# RUN npm ci --omit=dev

COPY ./web .
COPY .npmrc /root/.npmrc

RUN yarn && yarn build && yarn cache clean

# ---------------------------------------------------------#
#                   Build gitness image                    #
# ---------------------------------------------------------#
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
ARG BUILD_TAGS

# set required build flags
RUN CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64 \
    BUILD_TAGS=${BUILD_TAGS} \
    make build

### Pull CA Certs
FROM alpine:latest as cert-image

RUN apk --update add ca-certificates

# ---------------------------------------------------------#
#                   Create final image                     #
# ---------------------------------------------------------#
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