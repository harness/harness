# syntax=docker/dockerfile:1

# =============================================================================
# Stage 1: Base tools installation (rarely changes, excellent caching)
# =============================================================================
FROM harness0.harness.io/oci/docker_artifacts/hhi/harnessos-alpine:1@sha256:290a37c85b4284d2c11fa0778e5da79643d912a1f5a57d6f701b77644503b7e7 AS hhi-ca-certificates
RUN apk add --no-cache 'ca-certificates>=20261020-r0'

FROM harness0.harness.io/oci/docker_artifacts/hhi/go:1.26.10@sha256:718fddf012b3a305e936ae9964e5a642036bcb7b4c6c0786e2f78f238d637db8 AS patched-dev-go
FROM node:22.23.4-bookworm@sha256:63538dfe497cc29dfafdda25ada09b46db68a3f4cc205abf1f5d6bf0913f3384 AS patched-dev-node

FROM mcr.microsoft.com/devcontainers/go:1-1.23@sha256:35e445a87924ae04c171d7e52e2a913050dfefbacda3756ef7a307db035704d5 AS tools
COPY --from=hhi-ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/hhi/certs/ca-certificates.crt
ENV SSL_CERT_FILE=/etc/hhi/certs/ca-certificates.crt SSL_CERT_DIR=/etc/hhi/certs
RUN rm -rf /usr/local/go
COPY --from=patched-dev-go /usr/lib/go /usr/local/go
COPY --from=patched-dev-node /usr/local/ /usr/local/
ENV PATH=/usr/local/go/bin:/usr/local/bin:$PATH GOLANG_VERSION=1.26.10 GOTOOLCHAIN=local


# Install system packages in single layer for better caching
RUN apt update && apt install -y \
    lsb-release \
    curl \
    gpg \
    protobuf-compiler \
    git-lfs \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Go tools (these rarely change), separately to avoid memory issues
RUN export GOMAXPROCS=1 && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.1
RUN export GOMAXPROCS=1 && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

ENV PATH="${PATH}:$(go env GOPATH)/bin"

# =============================================================================
# Stage 2: External services installation (moderate caching)
# =============================================================================
FROM tools AS services

ARG TARGETARCH

# Install redis
RUN curl -fsSL https://packages.redis.io/gpg | gpg --dearmor -o /usr/share/keyrings/redis-archive-keyring.gpg && \
    chmod 644 /usr/share/keyrings/redis-archive-keyring.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/redis-archive-keyring.gpg] https://packages.redis.io/deb $(lsb_release -cs) main" | tee /etc/apt/sources.list.d/redis.list && \
    apt-get update -y && \
    apt-get install redis -y && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Install gcloud and kubectl
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg && \
    apt-get update && \
    apt-get install -y \
        google-cloud-cli \
        kubectl \
        google-cloud-cli-gke-gcloud-auth-plugin \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install binary tools with architecture support
RUN curl -fsSL https://raw.githubusercontent.com/metalbear-co/mirrord/main/scripts/install.sh | bash

# Install yq with architecture detection
RUN curl -Lo /usr/local/bin/yq https://github.com/mikefarah/yq/releases/download/v4.34.1/yq_linux_${TARGETARCH} && \
    chmod +x /usr/local/bin/yq

# Configure Go to handle private modules
RUN go env -w GOPRIVATE=$(go env GOPRIVATE),git0.harness.io

# =============================================================================
# Stage 3: Code preparation and build (secure - credentials cleaned in same layer)
# =============================================================================
FROM services AS builder

ARG BRANCH

# Arg to track commit hash — cache busts only when this changes
ARG COMMIT_SHA

WORKDIR /root

RUN --mount=type=secret,id=harness_code_secret_harness0,env=HARNESS_CODE_SECRET_HARNESS0 \
    --mount=type=secret,id=harness_code_user,env=HARNESS_CODE_USER \
    --mount=type=secret,id=github_secret,env=GITHUB_SECRET \
    --mount=type=secret,id=github_user,env=GITHUB_USER \
    echo $COMMIT_SHA > /commit.txt && \
    git config --global credential.helper store && \
    echo "https://${HARNESS_CODE_USER}:${HARNESS_CODE_SECRET_HARNESS0}@git0.harness.io" >> ~/.git-credentials && \
    echo "https://${GITHUB_USER}:${GITHUB_SECRET}@github.com" >> ~/.git-credentials && \
    echo "@harness:registry=https://npm.pkg.github.com" > ~/.npmrc && \
    echo "//npm.pkg.github.com/:_authToken=${GITHUB_SECRET}" >> ~/.npmrc && \
    echo "always-auth=true" >> ~/.npmrc && \
    echo "machine git0.harness.io login git password ${HARNESS_CODE_SECRET_HARNESS0}" >> ~/.netrc && \
    git clone -b ${BRANCH} https://git0.harness.io/l7B_kbSEQD2wjrM7PShm5w/PROD/Harness_Commons/gitness.git && \
    cd /root/gitness && \
    git lfs install && git lfs pull && \
    make init && \
    make dep && \
    make tools && \
    make web-build && \
    make build && \
    rm -f ~/.git-credentials && git config --global --unset credential.helper && \
    sed -i 's|//npm.pkg.github.com/:_authToken=.*|//npm.pkg.github.com/:_authToken=xxx|' ~/.npmrc && \
    sed -i '/machine git0\.harness\.io/d' ~/.netrc

WORKDIR /root/gitness
COPY --from=hhi-ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
