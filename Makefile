.PHONY: all build clean test llama-cpp llama-cpp-clean help

# Default target
all: build

# Variables
GO_LLAMA_DIR := /www/MCP/Remembrances/go-llama.cpp
SURREALDB_EMBEDDED_DIR := /www/MCP/Remembrances/surrealdb-embedded
BUILD_DIR := build
BINARY_NAME := remembrances-mcp

# Detect OS and architecture
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# Platform-specific settings
ifeq ($(UNAME_S),Darwin)
	# macOS
	PLATFORM := darwin
	LLAMA_LDFLAGS := -framework Accelerate -framework Foundation -framework Metal -framework MetalKit -framework MetalPerformanceShaders
	BUILD_TYPE ?= metal
else ifeq ($(UNAME_S),Linux)
	# Linux
	PLATFORM := linux
	LLAMA_LDFLAGS := -lm -lstdc++ -lpthread
	BUILD_TYPE ?=
else
	$(error Unsupported platform: $(UNAME_S))
endif

# CGO flags for linking with llama.cpp and surrealdb-embedded
export CGO_ENABLED := 1
export CGO_CFLAGS := -I$(GO_LLAMA_DIR) -I$(GO_LLAMA_DIR)/llama.cpp -I$(GO_LLAMA_DIR)/llama.cpp/common -I$(GO_LLAMA_DIR)/llama.cpp/ggml/include -I$(GO_LLAMA_DIR)/llama.cpp/include -I$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/include
export CGO_LDFLAGS := -L$(GO_LLAMA_DIR) -L$(GO_LLAMA_DIR)/build/bin -L$(GO_LLAMA_DIR)/build/common -L$(SURREALDB_EMBEDDED_DIR) -lllama -lcommon -lggml -lggml-base -lsurrealdb_embedded_rs $(LLAMA_LDFLAGS)

# Go linker flags to set RPATH
GO_LDFLAGS := -ldflags="-r \$$ORIGIN"

help:
	@echo "Remembrances-MCP Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  make build              - Build the project with GGUF and embedded SurrealDB support"
	@echo "  make llama-cpp          - Build llama.cpp library"
	@echo "  make llama-cpp-clean    - Clean llama.cpp build artifacts"
	@echo "  make surrealdb-embedded - Build surrealdb-embedded library"
	@echo "  make clean              - Clean all build artifacts"
	@echo "  make test               - Run tests"
	@echo "  make run                - Build and run the application"
	@echo ""
	@echo "Build options:"
	@echo "  BUILD_TYPE=metal    - Build with Metal GPU support (macOS)"
	@echo "  BUILD_TYPE=cublas   - Build with CUDA support (Linux)"
	@echo "  BUILD_TYPE=hipblas  - Build with ROCm support (Linux)"
	@echo "  BUILD_TYPE=openblas - Build with OpenBLAS support"
	@echo ""
	@echo "Examples:"
	@echo "  make build                    # Build with default settings"
	@echo "  make BUILD_TYPE=cublas build  # Build with CUDA support"
	@echo "  make run                      # Build and run"

# Build llama.cpp library
llama-cpp:
	@echo "Checking llama.cpp library..."
	@if [ ! -d "$(GO_LLAMA_DIR)/llama.cpp" ]; then \
		echo "Error: llama.cpp submodule not found at $(GO_LLAMA_DIR)/llama.cpp"; \
		echo "Please run: cd $(GO_LLAMA_DIR) && git submodule update --init --recursive"; \
		exit 1; \
	fi
	@if [ ! -f "$(GO_LLAMA_DIR)/build/bin/libllama.so" ]; then \
		echo "llama.cpp not built. Building now..."; \
		echo "Note: If this fails, please build manually:"; \
		echo "  cd $(GO_LLAMA_DIR) && make libbinding.a"; \
		cd $(GO_LLAMA_DIR) && BUILD_TYPE=$(BUILD_TYPE) cmake -B build -DLLAMA_STATIC=OFF && cmake --build build --config Release -j; \
	else \
		echo "llama.cpp library already built at $(GO_LLAMA_DIR)/build/bin/"; \
	fi
	@echo "llama.cpp library ready"

# Clean llama.cpp build artifacts
llama-cpp-clean:
	@echo "Cleaning llama.cpp build artifacts..."
	@cd $(GO_LLAMA_DIR) && $(MAKE) clean || true
	@echo "llama.cpp cleaned"

# Build surrealdb-embedded library
surrealdb-embedded:
	@echo "Checking surrealdb-embedded library..."
	@if [ ! -d "$(SURREALDB_EMBEDDED_DIR)" ]; then \
		echo "Error: surrealdb-embedded not found at $(SURREALDB_EMBEDDED_DIR)"; \
		exit 1; \
	fi
	@if [ ! -f "$(SURREALDB_EMBEDDED_DIR)/libsurrealdb_embedded_rs.so" ]; then \
		echo "surrealdb-embedded not built. Building now..."; \
		cd $(SURREALDB_EMBEDDED_DIR) && make; \
	else \
		echo "surrealdb-embedded library already built"; \
	fi
	@echo "surrealdb-embedded library ready"

# Build the main project
build: llama-cpp surrealdb-embedded
	@echo "Building $(BINARY_NAME) with GGUF and embedded SurrealDB support..."
	@mkdir -p $(BUILD_DIR)
	go build -mod=mod -v $(GO_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/remembrances-mcp
	@echo "Copying shared libraries to build directory..."
	@cp $(SURREALDB_EMBEDDED_DIR)/libsurrealdb_embedded_rs.so $(BUILD_DIR)/ 2>/dev/null || true
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	@echo "Note: Using wrapper script to set LD_LIBRARY_PATH"
	./run-remembrances.sh

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@go clean
	@echo "Clean complete"

# Deep clean (including llama.cpp)
clean-all: clean llama-cpp-clean
	@echo "Deep clean complete"

# Install dependencies
deps:
	@echo "Installing Go dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies installed"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "Format complete"

# Lint code
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, skipping"; \
	fi

# Build for multiple platforms (requires goreleaser or cross-compilation setup)
build-release:
	@echo "Building release binaries..."
	@echo "Note: This requires llama.cpp to be built for each target platform"
	goreleaser build --snapshot --rm-dist

# Development build with race detector
build-dev: llama-cpp
	@echo "Building $(BINARY_NAME) with race detector..."
	@mkdir -p $(BUILD_DIR)
	go build -race -v $(GO_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-dev ./cmd/remembrances-mcp
	@echo "Development build complete: $(BUILD_DIR)/$(BINARY_NAME)-dev"

# Check build environment
check-env:
	@echo "Build Environment:"
	@echo "  Platform: $(PLATFORM)"
	@echo "  Architecture: $(UNAME_M)"
	@echo "  Build Type: $(BUILD_TYPE)"
	@echo "  Go Version: $$(go version)"
	@echo "  CGO Enabled: $(CGO_ENABLED)"
	@echo "  llama.cpp Dir: $(GO_LLAMA_DIR)"
	@echo "  CGO_CFLAGS: $(CGO_CFLAGS)"
	@echo "  CGO_LDFLAGS: $(CGO_LDFLAGS)"
