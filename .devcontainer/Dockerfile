FROM mcr.microsoft.com/devcontainers/go:1-1.23 as base

# Install nodejs
RUN sudo apt update && sudo apt install nodejs -y

# Install proto
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.1 &&  \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0 &&  \
    sudo apt install -y protobuf-compiler

ENV PATH="${PATH}:$(go env GOPATH)/bin"

# Install redis
RUN sudo apt-get install lsb-release curl gpg -y && \
    curl -fsSL https://packages.redis.io/gpg | sudo gpg --dearmor -o /usr/share/keyrings/redis-archive-keyring.gpg && \
    sudo chmod 644 /usr/share/keyrings/redis-archive-keyring.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/redis-archive-keyring.gpg] https://packages.redis.io/deb $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/redis.list && \
    sudo apt-get update -y && \
    sudo apt-get install redis -y

# Install gcloud
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list &&  \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg &&  \
    sudo apt-get update && \
    sudo apt-get install google-cloud-cli -y

# Install kubectl
RUN sudo apt-get install kubectl

# Install gcloud auth plugin
RUN sudo apt-get install google-cloud-cli-gke-gcloud-auth-plugin

# Install mirrord
RUN curl -fsSL https://raw.githubusercontent.com/metalbear-co/mirrord/main/scripts/install.sh | bash

# Install yq
RUN curl -Lo /usr/local/bin/yq https://github.com/mikefarah/yq/releases/download/v4.34.1/yq_linux_amd64 && \
    sudo chmod +x /usr/local/bin/yq

ARG HARNESS_CODE_SECRET_HARNESS0
ARG HARNESS_CODE_USER
ARG GITHUB_SECRET
ARG GITHUB_USER
ARG BRANCH

# Setup git credentials for cloning
RUN git config --global credential.helper store && \
    echo "https://${HARNESS_CODE_USER}:${HARNESS_CODE_SECRET_HARNESS0}@git0.harness.io" >> ~/.git-credentials && \
    echo "https://${GITHUB_USER}:${GITHUB_SECRET}@github.com" >> ~/.git-credentials

WORKDIR /root

# Clone the codebase
RUN git clone -b ${BRANCH} https://git0.harness.io/l7B_kbSEQD2wjrM7PShm5w/PROD/Harness_Commons/gitness.git

WORKDIR /root/gitness

## Initialiase repo
#RUN make init
#
## Build gitness
#RUN make build-gitness
#
## Build cde-manager
#RUN make build-cde-manager

# Remove git credentials
RUN rm -f ~/.git-credentials && git config --global --unset credential.helper


