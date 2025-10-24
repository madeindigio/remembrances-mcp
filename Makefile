# Remembrances-MCP Makefile
# This Makefile handles building the remembrances-mcp server with static llama.cpp library

.PHONY: build clean test deps llama-deps llama-deps-all llama-deps-all-parallel llama-deps-linux-amd64 llama-deps-linux-arm64 llama-deps-darwin-amd64 llama-deps-darwin-arm64 llama-deps-windows-amd64 release release-multi release-multi-fast install format lint help

# Default target
all: build

# Variables
BINARY_NAME=remembrances-mcp
BUILD_DIR=./dist
CMD_DIR=./cmd/$(BINARY_NAME)
LLAMA_DIR=./go-llama.cpp
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X github.com/madeindigio/remembrances-mcp/pkg/version.Version=$(VERSION) -X github.com/madeindigio/remembrances-mcp/pkg/version.CommitHash=$(COMMIT)"

# Go build flags
GO_BUILD_FLAGS=-a -installsuffix cgo
GO_BUILD_ENV=CGO_ENABLED=1

# Platform detection
UNAME_S=$(shell uname -s)
UNAME_M=$(shell uname -m)

# Set platform-specific flags
ifeq ($(UNAME_S),Darwin)
	ifeq ($(UNAME_M),arm64)
		GO_BUILD_FLAGS+=-target darwin/arm64
		LLAMA_TARGET=darwin-arm64
	else
		GO_BUILD_FLAGS+=-target darwin/amd64
		LLAMA_TARGET=darwin-amd64
	endif
	LLAMA_FRAMEWORKS=-framework Accelerate -framework Foundation -framework Metal -framework MetalKit -framework MetalPerformanceShaders
else ifeq ($(UNAME_S),Linux)
	GO_BUILD_FLAGS+=-target linux/$(shell go env GOARCH)
	LLAMA_TARGET=linux-$(shell go env GOARCH)
	LLAMA_FRAMEWORKS=
endif

# Build llama.cpp static library for current platform
llama-deps:
	@echo "Building llama.cpp static library for $(LLAMA_TARGET)..."
	@cd $(LLAMA_DIR) && \
		CGO_CFLAGS="-I./llama.cpp -I./llama.cpp/common" \
		CGO_CXXFLAGS="-I./llama.cpp -I./llama.cpp/common" \
		CGO_LDFLAGS="-L. -lbinding -lm -lstdc++ $(LLAMA_FRAMEWORKS)" \
		./scripts/build-static.sh $(shell echo $(LLAMA_TARGET) | cut -d'-' -f1) $(shell echo $(LLAMA_TARGET) | cut -d'-' -f2)

# Build llama.cpp static library for specific platforms
llama-deps-linux-amd64:
	@echo "Building llama.cpp static library for linux-amd64..."
	@cd $(LLAMA_DIR) && chmod +x scripts/build-static-multi.sh && ./scripts/build-static-multi.sh linux amd64

llama-deps-linux-arm64:
	@echo "Building llama.cpp static library for linux-arm64..."
	@cd $(LLAMA_DIR) && chmod +x scripts/build-static-multi.sh && ./scripts/build-static-multi.sh linux arm64

llama-deps-darwin-amd64:
	@echo "Building llama.cpp static library for darwin-amd64..."
	@cd $(LLAMA_DIR) && chmod +x scripts/build-static-multi.sh && ./scripts/build-static-multi.sh darwin amd64

llama-deps-darwin-arm64:
	@echo "Building llama.cpp static library for darwin-arm64..."
	@cd $(LLAMA_DIR) && chmod +x scripts/build-static-multi.sh && ./scripts/build-static-multi.sh darwin arm64

llama-deps-windows-amd64:
	@echo "Building llama.cpp static library for windows-amd64..."
	@cd $(LLAMA_DIR) && chmod +x scripts/build-static-multi.sh && ./scripts/build-static-multi.sh windows amd64

# Build llama.cpp static libraries for all platforms (sequential)
llama-deps-all: llama-deps-linux-amd64 llama-deps-linux-arm64 llama-deps-darwin-amd64 llama-deps-darwin-arm64 llama-deps-windows-amd64
	@echo "All platform static libraries built successfully!"

# Build llama.cpp static libraries for all platforms (parallel - FAST)
llama-deps-all-parallel:
	@echo "Building llama.cpp static libraries in parallel..."
	@chmod +x scripts/build-llama-parallel.sh
	@./scripts/build-llama-parallel.sh

# Build llama.cpp with clean and verify
llama-deps-all-parallel-clean:
	@echo "Building llama.cpp static libraries in parallel (clean build)..."
	@chmod +x scripts/build-llama-parallel.sh
	@./scripts/build-llama-parallel.sh --clean --verify

# Build the main binary
build: llama-deps
	@echo "Building $(BINARY_NAME) for $(LLAMA_TARGET)..."
	@mkdir -p $(BUILD_DIR)
	@$(GO_BUILD_ENV) \
		CGO_CFLAGS="-I$(LLAMA_DIR)/llama.cpp -I$(LLAMA_DIR)/llama.cpp/common" \
		CGO_CXXFLAGS="-I$(LLAMA_DIR)/llama.cpp -I$(LLAMA_DIR)/llama.cpp/common" \
		CGO_LDFLAGS="-L$(LLAMA_DIR) -lbinding -lm -lstdc++ $(LLAMA_FRAMEWORKS)" \
		go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

# Development build (faster, no optimization)
dev:
	@echo "Building $(BINARY_NAME) for development..."
	@mkdir -p $(BUILD_DIR)
	@$(GO_BUILD_ENV) \
		CGO_CFLAGS="-I$(LLAMA_DIR)/llama.cpp -I$(LLAMA_DIR)/llama.cpp/common" \
		CGO_CXXFLAGS="-I$(LLAMA_DIR)/llama.cpp -I$(LLAMA_DIR)/llama.cpp/common" \
		CGO_LDFLAGS="-L$(LLAMA_DIR) -lbinding -lm -lstdc++ $(LLAMA_FRAMEWORKS)" \
		go build -race $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

# Install dependencies
deps:
	@echo "Installing Go dependencies..."
	@go mod download
	@go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@cd $(LLAMA_DIR) && make clean 2>/dev/null || true
	@rm -rf $(LLAMA_DIR)/build
	@rm -f $(LLAMA_DIR)/libbinding.a
	@rm -f $(LLAMA_DIR)/llama.cpp/*.o

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

# Multi-platform release build (requires goreleaser-cross Docker image)
release-multi:
	@chmod +x scripts/release-multiplatform.sh
	@./scripts/release-multiplatform.sh release

# Multi-platform snapshot build (no release)
release-multi-snapshot:
	@chmod +x scripts/release-multiplatform.sh
	@SKIP_CONFIRM=true ./scripts/release-multiplatform.sh snapshot

# Multi-platform release build with parallel library builds (FAST)
release-multi-fast:
	@echo "Pre-building llama.cpp libraries in parallel..."
	@$(MAKE) llama-deps-all-parallel
	@chmod +x scripts/release-multiplatform.sh
	@SKIP_CONFIRM=true ./scripts/release-multiplatform.sh release prebuilt

# Multi-platform snapshot build with parallel library builds (FAST)
release-multi-snapshot-fast:
	@echo "Pre-building llama.cpp libraries in parallel..."
	@$(MAKE) llama-deps-all-parallel
	@chmod +x scripts/release-multiplatform.sh
	@SKIP_CONFIRM=true ./scripts/release-multiplatform.sh snapshot prebuilt

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
		--llama-model-path ./models/test.gguf \
		--llama-dimension 768 \
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
	@echo "  Platform: $(LLAMA_TARGET)"
	@echo "  Go Version: $(shell go version)"
	@echo "  CGO Enabled: $(shell go env CGO_ENABLED)"

# Help target
help:
	@echo "Available targets:"
	@echo ""
	@echo "Building:"
	@echo "  build                         - Build the binary with llama.cpp static library"
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
	@echo "  llama-deps                    - Build llama.cpp static library for current platform"
	@echo "  llama-deps-linux-amd64        - Build llama.cpp for Linux x86_64"
	@echo "  llama-deps-linux-arm64        - Build llama.cpp for Linux ARM64"
	@echo "  llama-deps-darwin-amd64       - Build llama.cpp for macOS Intel"
	@echo "  llama-deps-darwin-arm64       - Build llama.cpp for macOS Apple Silicon"
	@echo "  llama-deps-windows-amd64      - Build llama.cpp for Windows x86_64"
	@echo "  llama-deps-all                - Build llama.cpp for all platforms (sequential)"
	@echo "  llama-deps-all-parallel       - Build llama.cpp for all platforms (PARALLEL - FAST)"
	@echo "  llama-deps-all-parallel-clean - Build llama.cpp for all platforms (parallel, clean)"
	@echo ""
	@echo "Releases:"
	@echo "  build-all                     - Build for all platforms (goreleaser)"
	@echo "  release                       - Create release build (single platform)"
	@echo "  release-multi                 - Create multi-platform release (Docker)"
	@echo "  release-multi-snapshot        - Create multi-platform snapshot (Docker)"
	@echo "  release-multi-fast            - Multi-platform release with parallel builds (FAST)"
	@echo "  release-multi-snapshot-fast   - Multi-platform snapshot with parallel builds (FAST)"
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
	@echo "Tip: Use *-parallel or *-fast targets for 50% faster builds!"
