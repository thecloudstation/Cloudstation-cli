# Multi-stage Dockerfile for CloudStation Orchestrator
# This creates a production-ready, lightweight Docker image (~250-300MB)

# Stage 1: Nomad-Pack - Copy nomad-pack from official HashiCorp image
FROM hashicorp/nomad-pack:0.4.0 AS nomadpack

# Stage 2: Builder - Compile the cs binary for Linux
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go.mod and go.sum first for better layer caching
COPY go.mod go.sum ./

# Download dependencies (cached layer)
RUN go mod download

# Copy source code
COPY . .

# NOTE: Embedded packs dependency
# The builtin/nomadpack/packs/ directory must exist in the build context
# Run 'make sync-packs' before building Docker image to ensure packs are available
# The Go embed directive in builtin/nomadpack/embedded.go requires these files

# Build the cs binary with optimized flags
# -s -w removes debug info and symbol table for smaller binary
# CGO_ENABLED=0 creates a static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o cs \
    ./cmd/cloudstation

# Verify the binary works
RUN ./cs --version

# Stage 3: Runtime - Create the final minimal image
FROM alpine:3.19 AS runtime

# Set target architecture (for multi-arch builds)
ARG TARGETARCH=amd64

# Install system dependencies
# docker-cli: Required for csdocker and docker plugins
# git: Required for source code operations
# bash: Required for scripts and compatibility
# jq: Required for JSON processing
# curl: Required for HTTP operations and binary downloads
# ca-certificates: Required for HTTPS
# unzip: Required for nomad installation
# python3, py3-pip: Required for Azure CLI
# Build deps (gcc, musl-dev, etc.): Required temporarily for Azure CLI installation
RUN apk add --no-cache \
    docker-cli \
    git \
    bash \
    jq \
    curl \
    ca-certificates \
    unzip \
    python3 \
    py3-pip \
    gcc \
    musl-dev \
    python3-dev \
    libffi-dev \
    openssl-dev

# Install Docker Buildx plugin
# This is required for nixpacks and other builders that use BuildKit
RUN BUILDX_VERSION=v0.12.1 \
    && mkdir -p /usr/local/lib/docker/cli-plugins \
    && wget -q https://github.com/docker/buildx/releases/download/${BUILDX_VERSION}/buildx-${BUILDX_VERSION}.linux-${TARGETARCH} \
       -O /usr/local/lib/docker/cli-plugins/docker-buildx \
    && chmod +x /usr/local/lib/docker/cli-plugins/docker-buildx

# Install Azure CLI (using --break-system-packages for Alpine 3.19+)
RUN pip3 install --break-system-packages --no-cache-dir azure-cli

# Remove build dependencies to reduce image size
RUN apk del gcc musl-dev python3-dev libffi-dev openssl-dev

# Install nixpacks from official installation script (needs tar for extraction)
RUN apk add --no-cache tar \
    && curl -sSL https://nixpacks.com/install.sh | bash \
    && chmod +x /usr/local/bin/nixpacks

# Install mise (required by railpack 0.15.4+ for language runtime detection)
# Railpack 0.15.4 downloads its own mise to /tmp/railpack/mise/mise-{version}
# The downloaded binary is glibc-based, so we need glibc compatibility on Alpine
# Pre-install mise to system PATH AND to railpack's expected location
RUN apk add --no-cache gcompat \
    && curl -sSL https://mise.run | sh \
    && cp /root/.local/bin/mise /usr/local/bin/mise \
    && cp /root/.local/bin/mise /usr/bin/mise \
    && chmod +x /usr/local/bin/mise /usr/bin/mise \
    && mkdir -p /tmp/railpack/mise \
    && cp /root/.local/bin/mise /tmp/railpack/mise/mise-2025.12.12 \
    && chmod +x /tmp/railpack/mise/mise-2025.12.12 \
    && mise --version

# Install railpack from official installation script
RUN curl -sSL https://raw.githubusercontent.com/railwayapp/railpack/main/install.sh | bash \
    && chmod +x /usr/local/bin/railpack \
    && apk del tar

# Install nomad CLI from HashiCorp releases (latest stable version)
# Note: nomad binary requires glibc, so we install compatibility layer for Alpine (musl)
# Download architecture-appropriate binary based on TARGETARCH
RUN apk add --no-cache libc6-compat \
    && NOMAD_VERSION=1.10.5 \
    && if [ "$TARGETARCH" = "arm64" ]; then NOMAD_ARCH="arm64"; else NOMAD_ARCH="amd64"; fi \
    && curl -sSL https://releases.hashicorp.com/nomad/${NOMAD_VERSION}/nomad_${NOMAD_VERSION}_linux_${NOMAD_ARCH}.zip -o /tmp/nomad.zip \
    && unzip /tmp/nomad.zip -d /usr/local/bin \
    && chmod +x /usr/local/bin/nomad \
    && rm /tmp/nomad.zip

# Copy nomad-pack from official HashiCorp image
COPY --from=nomadpack /bin/nomad-pack /usr/local/bin/nomad-pack
RUN chmod +x /usr/local/bin/nomad-pack

# Copy cs binary from builder stage
COPY --from=builder /build/cs /usr/local/bin/cs

# Copy entrypoint scripts
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
COPY scripts/entrypoint-test.sh /usr/local/bin/test-entrypoint.sh
COPY scripts/entrypoint-noauth.sh /usr/local/bin/entrypoint-noauth.sh
COPY scripts/entrypoint-debug.sh /usr/local/bin/entrypoint-debug.sh
RUN chmod +x /usr/local/bin/entrypoint.sh /usr/local/bin/test-entrypoint.sh /usr/local/bin/entrypoint-noauth.sh /usr/local/bin/entrypoint-debug.sh

# Set environment variables
ENV PATH="/usr/local/bin:${PATH}"
ENV DOCKER_CONFIG="/root/.docker"
ENV HOME="/root"

# Create necessary directories
RUN mkdir -p /workspace && mkdir -p /root/.docker

# Set working directory
WORKDIR /workspace

# Run as root (required for docker socket access)
USER root

# Set entrypoint and default command
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["cs"]
