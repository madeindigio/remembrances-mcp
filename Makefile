# Remembrances-MCP Makefile
# This Makefile handles building the remembrances-mcp server
# Now uses kelindar/search library (no cgo required for embeddings)

.PHONY: build clean test deps release release-multi install format lint help

# Default target
all: build

# Variables
BINARY_NAME=remembrances-mcp
BUILD_DIR=./dist
CMD_DIR=./cmd/$(BINARY_NAME)
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X github.com/madeindigio/remembrances-mcp/pkg/version.Version=$(VERSION) -X github.com/madeindigio/remembrances-mcp/pkg/version.CommitHash=$(COMMIT)"

# Go build flags - CGO not required for kelindar/search
GO_BUILD_FLAGS=-a
GO_BUILD_ENV=CGO_ENABLED=0

# Build the main binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@$(GO_BUILD_ENV) go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

# Development build (faster, no optimization)
dev:
	@echo "Building $(BINARY_NAME) for development..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build -race $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

# Install dependencies
deps:
	@echo "Installing Go dependencies..."
	@go mod download
	@go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@go clean

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

# Format code
format:
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

# Lint code
lint:
	@echo "Linting code..."
	@golangci-lint run

# Security scan
security:
	@echo "Running security scan..."
	@gosec ./...

# Build for all platforms (used by goreleaser)
build-all:
	@echo "Building for all platforms..."
	@$(MAKE) clean
	@goreleaser release --snapshot --skip sign --clean

# Release build (single platform)
release:
	@echo "Building release..."
	@$(MAKE) clean
	@goreleaser release --clean --skip sign

# Multi-platform release build
release-multi:
	@echo "Building multi-platform release..."
	@goreleaser release --clean

# Multi-platform snapshot build (no release)
release-multi-snapshot:
	@echo "Building multi-platform snapshot..."
	@goreleaser release --snapshot --clean --skip=publish

# Install binary to system
install: build
	@echo "Installing $(BINARY_NAME)..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

# Uninstall binary
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)

# Development setup
setup-dev:
	@echo "Setting up development environment..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Run with development configuration
run-dev: build
	@echo "Running $(BINARY_NAME) in development mode..."
	@$(BUILD_DIR)/$(BINARY_NAME) \
		--search-model-path ./models/nomic-embed-text-v1.5.Q4_K_M.gguf \
		--search-dimension 768 \
		--surrealdb-start-cmd 'surreal start --user root --pass root ws://localhost:8000'

# Docker build
docker-build:
	@echo "Building Docker image..."
	@docker build -t remembrances-mcp:$(VERSION) .

# Docker run
docker-run:
	@echo "Running Docker container..."
	@docker run -it --rm \
		-v $(PWD)/models:/app/models \
		-v $(PWD)/data:/app/data \
		-p 8000:8000 \
		remembrances-mcp:$(VERSION)

# Show build information
info:
	@echo "Build Information:"
	@echo "  Binary: $(BINARY_NAME)"
	@echo "  Version: $(VERSION)"
	@echo "  Commit: $(COMMIT)"
	@echo "  Go Version: $(shell go version)"
	@echo "  CGO Enabled: NO (uses kelindar/search via purego)"
	@echo "  Embedder: kelindar/search (supports BERT GGUF models)"

# Help target
help:
	@echo "Available targets:"
	@echo ""
	@echo "Building:"
	@echo "  build                         - Build the binary (no cgo required)"
	@echo "  dev                           - Build development version with race detection"
	@echo "  clean                         - Clean build artifacts"
	@echo ""
	@echo "Testing:"
	@echo "  test                          - Run tests"
	@echo "  test-coverage                 - Run tests with coverage"
	@echo "  format                        - Format code"
	@echo "  lint                          - Lint code"
	@echo "  security                      - Run security scan"
	@echo ""
	@echo "Dependencies:"
	@echo "  deps                          - Install Go dependencies"
	@echo ""
	@echo "Releases:"
	@echo "  build-all                     - Build for all platforms (goreleaser)"
	@echo "  release                       - Create release build"
	@echo "  release-multi                 - Create multi-platform release"
	@echo "  release-multi-snapshot        - Create multi-platform snapshot"
	@echo ""
	@echo "Installation:"
	@echo "  install                       - Install binary to /usr/local/bin"
	@echo "  uninstall                     - Remove binary from /usr/local/bin"
	@echo "  setup-dev                     - Setup development environment"
	@echo ""
	@echo "Running:"
	@echo "  run-dev                       - Run with development configuration"
	@echo "  docker-build                  - Build Docker image"
	@echo "  docker-run                    - Run Docker container"
	@echo ""
	@echo "Information:"
	@echo "  info                          - Show build information"
	@echo "  help                          - Show this help message"
	@echo ""
	@echo "Note: Now uses kelindar/search (no cgo/llama.cpp compilation needed)!"
