.PHONY: all build clean run-tracker run-peer test lint fmt help docker-build docker-push deploy

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Binary names
TRACKER_BINARY=tracker
PEER_BINARY=peer
DOWNLOAD_BINARY=p2p-download

# Directories
BIN_DIR=bin
TRACKER_DIR=services/tracker
PEER_DIR=services/peer
DOWNLOAD_DIR=services/peer/cmd/download

all: build

## build: Build all binaries
build: build-tracker build-peer build-download

## build-tracker: Build tracker server
build-tracker:
	@echo "Building tracker..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$(TRACKER_BINARY) ./$(TRACKER_DIR)/cmd

## build-peer: Build peer node
build-peer:
	@echo "Building peer..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$(PEER_BINARY) ./$(PEER_DIR)/cmd

## build-download: Build download CLI tool
build-download:
	@echo "Building p2p-download..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$(DOWNLOAD_BINARY) ./$(DOWNLOAD_DIR)

## run-tracker: Run tracker server
run-tracker:
	@echo "Running tracker server..."
	$(GOCMD) run ./$(TRACKER_DIR)/cmd -addr :8080

## run-peer: Run peer node (use PEER_PORT and PEER_ID env vars)
run-peer:
	@echo "Running peer node..."
	$(GOCMD) run ./$(PEER_DIR)/cmd

## test: Run all tests
test:
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## lint: Run linter
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

## fmt: Format code
fmt:
	$(GOFMT) -s -w .

## tidy: Tidy go modules
tidy:
	$(GOMOD) tidy

## clean: Clean build files
clean:
	$(GOCLEAN)
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html
	rm -rf demo_data

## demo: Run demo script
demo: build
	@echo "Running demo..."
	./scripts/demo.sh

## run-peer1: Run first peer (seeder)
run-peer1: build
	./$(BIN_DIR)/$(PEER_BINARY) -port 6881 -data ./data1 -tracker http://localhost:8080

## run-peer2: Run second peer (leecher)
run-peer2: build
	./$(BIN_DIR)/$(PEER_BINARY) -port 6882 -data ./data2 -tracker http://localhost:8080

## build-peer-macos: Build peer for macOS ARM64 (Apple Silicon)
build-peer-macos:
	@echo "Building peer for macOS ARM64..."
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BIN_DIR)/peer-darwin-arm64 ./$(PEER_DIR)/cmd

## build-peer-windows: Build peer for Windows AMD64
build-peer-windows:
	@echo "Building peer for Windows AMD64..."
	@mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BIN_DIR)/peer-windows-amd64.exe ./$(PEER_DIR)/cmd

## build-peer-linux: Build peer for Linux AMD64
build-peer-linux:
	@echo "Building peer for Linux AMD64..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BIN_DIR)/peer-linux-amd64 ./$(PEER_DIR)/cmd

## build-peer-all: Build peer for all platforms
build-peer-all: build-peer-macos build-peer-windows build-peer-linux
	@echo "All peer binaries built!"
	@ls -lh $(BIN_DIR)/peer-*

## build-download-linux: Build p2p-download for Linux AMD64
build-download-linux:
	@echo "Building p2p-download for Linux AMD64..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BIN_DIR)/p2p-download-linux-amd64 ./$(DOWNLOAD_DIR)

## build-download-macos: Build p2p-download for macOS ARM64
build-download-macos:
	@echo "Building p2p-download for macOS ARM64..."
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BIN_DIR)/p2p-download-darwin-arm64 ./$(DOWNLOAD_DIR)

## build-download-all: Build p2p-download for all platforms
build-download-all: build-download-linux build-download-macos
	@echo "All p2p-download binaries built!"
	@ls -lh $(BIN_DIR)/p2p-download-*

## docker-build: Build Docker images
docker-build:
	@echo "Building Docker images..."
	docker build --platform linux/amd64 -t registry.idist.dev/p2p-system/p2p-tracker:latest -f docker/tracker.Dockerfile .
	docker build --platform linux/amd64 -t registry.idist.dev/p2p-system/p2p-peer:latest -f docker/peer.Dockerfile .

## docker-push: Push Docker images to registry
docker-push:
	@echo "Pushing Docker images..."
	./scripts/build-push.sh

## deploy: Deploy to Kubernetes
deploy:
	@echo "Deploying to Kubernetes..."
	./scripts/deploy.sh

## help: Show this help
help:
	@echo "Available commands:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
