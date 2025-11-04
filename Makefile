# Makefile for remembrances-mcp
# Supports building with and without embedded SurrealDB support

.PHONY: help build build-remote build-embedded test clean docker-build release

# Default target
help:
	@echo "Available targets:"
	@echo "  make build            - Build standard version (remote SurrealDB only)"
	@echo "  make build-embedded   - Build with embedded SurrealDB support (requires CGO)"
	@echo "  make test             - Run tests"
	@echo "  make clean            - Clean build artifacts"
	@echo "  make docker-build     - Build cross-platform binaries using Docker"
	@echo "  make release          - Create release with goreleaser"
	@echo ""
	@echo "Environment variables:"
	@echo "  GOOS              - Target OS (linux, darwin, windows)"
	@echo "  GOARCH            - Target architecture (amd64, arm64)"

# Build remote-only version (no CGO required)
build: build-remote

build-remote:
	@echo "Building remote-only version..."
	CGO_ENABLED=0 go build -o bin/remembrances-mcp ./cmd/remembrances-mcp

# Build embedded version (requires CGO and Rust)
build-embedded:
	@echo "Building embedded version..."
	@if ! command -v cargo > /dev/null; then \
		echo "Error: Rust/Cargo not found. Please install from https://rustup.rs/"; \
		exit 1; \
	fi
	@echo "Building Rust library..."
	./build-embedded.sh native
	@echo "Building Go binary with embedded support..."
	CGO_ENABLED=1 go build -tags embedded -o bin/remembrances-mcp-embedded ./cmd/remembrances-mcp

# Run tests
test:
	go test -v ./...

test-embedded:
	CGO_ENABLED=1 go test -tags embedded -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf dist/
	rm -rf vendor/surrealdb-embedded-rust/target/
	go clean

# Build using Docker for cross-compilation
docker-build:
	@echo "Building Docker image for cross-compilation..."
	docker build -t remembrances-mcp-builder -f Dockerfile.crossbuild .
	@echo "Building binaries in Docker..."
	docker run --rm -v $(PWD):/workspace remembrances-mcp-builder \
		bash -c "make build-embedded"

# Create release with goreleaser
release:
	@if ! command -v goreleaser > /dev/null; then \
		echo "Error: goreleaser not found. Install with: go install github.com/goreleaser/goreleaser@latest"; \
		exit 1; \
	fi
	goreleaser release --clean

# Create snapshot release (no git tags required)
snapshot:
	@if ! command -v goreleaser > /dev/null; then \
		echo "Error: goreleaser not found. Install with: go install github.com/goreleaser/goreleaser@latest"; \
		exit 1; \
	fi
	goreleaser release --snapshot --clean

# Install locally
install: build
	go install ./cmd/remembrances-mcp

install-embedded: build-embedded
	cp bin/remembrances-mcp-embedded $(GOPATH)/bin/

# Development build with race detector
dev:
	go build -race -o bin/remembrances-mcp-dev ./cmd/remembrances-mcp

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run ./...

# Update dependencies
deps:
	go mod download
	go mod tidy
