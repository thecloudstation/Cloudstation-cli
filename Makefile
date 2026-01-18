.PHONY: build test clean clean-logs install fmt vet lint docker-build docker-push docker-test docker-shell sync-packs

# Binary name
BINARY=cs
VERSION?=0.1.0
BUILD_DIR=bin

# API URLs (override for custom deployments)
API_URL?=https://cst-cs-backend-gmlyovvq.cloud-station.io
AUTH_URL?=https://cs-auth.cloud-station.io

# Docker configuration
IMAGE_NAME?=cloudstation-orchestrator
IMAGE_TAG?=latest
REGISTRY?=acrbc001.azurecr.io

# Build flags - bake in version and API URLs
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.DefaultAPIURL=$(API_URL) -X main.DefaultAuthURL=$(AUTH_URL) -s -w"

# Sync embedded packs from cloudstation-packs repository
sync-packs:
	@echo "Syncing embedded packs..."
	@mkdir -p builtin/nomadpack/packs/cloudstation/templates
	@cp -r ../cloudstation-packs/packs/cloudstation/* builtin/nomadpack/packs/cloudstation/
	@echo "Packs synced successfully"

# Build the binary
build: sync-packs
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/cloudstation
	@echo "Binary built: $(BUILD_DIR)/$(BINARY)"

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -cover ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.txt coverage.html
	rm -rf builtin/goreleaser/logs/
	go clean

# Clean Claude cache/log files
clean-logs:
	@echo "Cleaning Claude cache logs..."
	rm -rf builtin/goreleaser/logs/
	rm -rf test-docker/logs/
	rm -rf test-docker/.claude/

# Install binary to $GOPATH/bin
install:
	@echo "Installing $(BINARY)..."
	go install $(LDFLAGS) ./cmd/cloudstation

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Run linter (if golangci-lint is installed)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed" && exit 1)
	golangci-lint run

# Run all quality checks
check: fmt vet test

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/cloudstation
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64 ./cmd/cloudstation
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/cloudstation
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/cloudstation
	@echo "Binaries built in $(BUILD_DIR)/"

# Docker targets

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .
	@echo "Image built: $(IMAGE_NAME):$(IMAGE_TAG)"
	@docker images $(IMAGE_NAME):$(IMAGE_TAG)

# Build Docker image for Linux AMD64
docker-build-linux-amd64:
	@echo "Building Docker image for Linux AMD64..."
	DOCKER_BUILDKIT=1 docker build --platform linux/amd64 -t $(IMAGE_NAME):$(IMAGE_TAG)-amd64 .
	@echo "Image built: $(IMAGE_NAME):$(IMAGE_TAG)-amd64"

# Build Docker image for Linux ARM64
docker-build-linux-arm64:
	@echo "Building Docker image for Linux ARM64..."
	DOCKER_BUILDKIT=1 docker build --platform linux/arm64 -t $(IMAGE_NAME):$(IMAGE_TAG)-arm64 .
	@echo "Image built: $(IMAGE_NAME):$(IMAGE_TAG)-arm64"

# Push Docker image to registry
docker-push: docker-build
	@echo "Tagging image for registry..."
	docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
	docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(REGISTRY)/$(IMAGE_NAME):$(VERSION)
	@echo "Logging in to Azure Container Registry..."
	az acr login --name acrbc001
	@echo "Pushing image to $(REGISTRY)..."
	docker push $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
	docker push $(REGISTRY)/$(IMAGE_NAME):$(VERSION)
	@echo "Image pushed: $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)"
	@echo "Image pushed: $(REGISTRY)/$(IMAGE_NAME):$(VERSION)"

# Test Docker image (verify all binaries are available)
docker-test:
	@echo "Testing Docker image..."
	@echo "Testing cs binary..."
	docker run --rm $(IMAGE_NAME):$(IMAGE_TAG) cs --version
	@echo "Testing nixpacks..."
	docker run --rm $(IMAGE_NAME):$(IMAGE_TAG) nixpacks --version
	@echo "Testing railpack..."
	docker run --rm $(IMAGE_NAME):$(IMAGE_TAG) railpack --version
	@echo "Testing nomad-pack..."
	docker run --rm $(IMAGE_NAME):$(IMAGE_TAG) nomad-pack version
	@echo "Testing nomad..."
	docker run --rm $(IMAGE_NAME):$(IMAGE_TAG) nomad version
	@echo "Testing docker..."
	docker run --rm $(IMAGE_NAME):$(IMAGE_TAG) docker --version
	@echo "Testing git..."
	docker run --rm $(IMAGE_NAME):$(IMAGE_TAG) git --version
	@echo "Testing Azure CLI..."
	docker run --rm $(IMAGE_NAME):$(IMAGE_TAG) az --version
	@echo "All tests passed!"

# Start an interactive shell in the Docker container
docker-shell:
	@echo "Starting interactive shell in Docker container..."
	docker run --rm -it \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(PWD):/workspace \
		$(IMAGE_NAME):$(IMAGE_TAG) \
		/bin/bash
