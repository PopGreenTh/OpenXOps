FROM golang:1.22-bookworm

RUN apt-get update \
    && apt-get install -y --no-install-recommends curl unzip ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Task
RUN curl -sSfL https://taskfile.dev/install.sh | sh -s -- -d -b /usr/local/bin v3.40.1

# Delve
RUN go install github.com/go-delve/delve/cmd/dlv@latest \
    && mv /go/bin/dlv /usr/local/bin/dlv

# AWS CLI v2
RUN arch="$(uname -m)" \
    && case "$arch" in \
         x86_64) awsarch=x86_64 ;; \
         aarch64) awsarch=aarch64 ;; \
         *) echo "unsupported arch $arch" >&2 && exit 1 ;; \
       esac \
    && curl -sSfL "https://awscli.amazonaws.com/awscli-exe-linux-${awsarch}.zip" -o /tmp/aws.zip \
    && unzip -q /tmp/aws.zip -d /tmp \
    && /tmp/aws/install \
    && rm -rf /tmp/aws /tmp/aws.zip

WORKDIR /workspaces/OpenXOps
