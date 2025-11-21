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
export CGO_LDFLAGS := -L$(GO_LLAMA_DIR) -L$(GO_LLAMA_DIR)/build/bin -L$(GO_LLAMA_DIR)/build/common -L$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release -lllama -lcommon -lggml -lggml-base -lsurrealdb_embedded_rs $(LLAMA_LDFLAGS)

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
	@echo "Multi-variant library builds:"
	@echo "  make build-libs-all-variants - Build llama.cpp for all GPU types"
	@echo "  make build-libs-cuda         - Build llama.cpp with CUDA → build/libs/cuda/"
	@echo "  make build-libs-hipblas      - Build llama.cpp with ROCm → build/libs/hipblas/"
	@echo "  make build-libs-metal        - Build llama.cpp with Metal → build/libs/metal/"
	@echo "  make build-libs-openblas     - Build llama.cpp with OpenBLAS → build/libs/openblas/"
	@echo "  make build-libs-cpu          - Build llama.cpp CPU-only → build/libs/cpu/"
	@echo ""
	@echo "Multi-variant binary builds:"
	@echo "  make build-variant VARIANT=cuda  - Build single variant binary (remembrances-mcp-cuda)"
	@echo "  make build-all-variants          - Build all variant binaries (cpu, cuda, hipblas, etc.)"
	@echo ""
	@echo "Distribution packaging:"
	@echo "  make dist-variant VARIANT=cuda   - Package single variant with libraries as zip"
	@echo "  make dist-all                    - Package all variants as separate zip files"
	@echo ""
	@echo "Cross-compilation targets:"
	@echo "  make build-cross        - Cross-compile for all platforms using Docker"
	@echo "  make build-libs-cross   - Build only shared libraries for cross-compilation"
	@echo "  make release-cross      - Create a cross-platform release"
	@echo ""
	@echo "Build options:"
	@echo "  BUILD_TYPE=metal    - Build with Metal GPU support (macOS)"
	@echo "  BUILD_TYPE=cuda     - Build with CUDA support (Linux, recommended)"
	@echo "  BUILD_TYPE=cublas   - Build with CUDA support (Linux, deprecated, use cuda)"
	@echo "  BUILD_TYPE=hipblas  - Build with ROCm support (Linux)"
	@echo "  BUILD_TYPE=openblas - Build with OpenBLAS support"
	@echo ""
	@echo "Examples:"
	@echo "  make build                      # Build with default settings"
	@echo "  make BUILD_TYPE=cuda build      # Build with CUDA support (recommended)"
	@echo "  make build-all-variants         # Build all GPU variant binaries"
	@echo "  make dist-all                   # Package all variants for distribution"
	@echo "  make run                        # Build and run"
	@echo "  make build-cross                # Cross-compile for all platforms"
	@echo "  make release-cross              # Create GitHub release"

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
	@if [ ! -f "$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release/libsurrealdb_embedded_rs.so" ] && \
	   [ ! -f "$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release/libsurrealdb_embedded_rs.dylib" ]; then \
		echo "surrealdb-embedded not built. Building now..."; \
		cd $(SURREALDB_EMBEDDED_DIR) && make build-rust; \
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
	@# Copy SurrealDB embedded library from Rust build directory
	@echo "Copying SurrealDB embedded library..."
	@cp $(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release/libsurrealdb_embedded_rs.so $(BUILD_DIR)/ 2>/dev/null || \
	 cp $(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release/libsurrealdb_embedded_rs.dylib $(BUILD_DIR)/ 2>/dev/null || \
	 echo "⚠ Warning: SurrealDB embedded library not found"
	@# Copy ALL llama.cpp shared libraries (.so and .dylib)
	@# This includes libraries for CUDA, Metal, ROCm, Vulkan, etc.
	@echo "Copying all llama.cpp shared libraries..."
	@find $(GO_LLAMA_DIR)/build/bin -type f \( -name "*.so" -o -name "*.dylib" \) -exec cp {} $(BUILD_DIR)/ \; 2>/dev/null || true
	@# Also check other common locations in the build directory
	@find $(GO_LLAMA_DIR)/build/src -type f \( -name "*.so" -o -name "*.dylib" \) -exec cp {} $(BUILD_DIR)/ \; 2>/dev/null || true
	@find $(GO_LLAMA_DIR)/build/common -type f \( -name "*.so" -o -name "*.dylib" \) -exec cp {} $(BUILD_DIR)/ \; 2>/dev/null || true
	@find $(GO_LLAMA_DIR)/build/ggml -type f \( -name "*.so" -o -name "*.dylib" \) -exec cp {} $(BUILD_DIR)/ \; 2>/dev/null || true
	@echo "Shared libraries copied successfully"
	@ls -lh $(BUILD_DIR)/libsurrealdb_embedded_rs.* 2>/dev/null && echo "✓ SurrealDB embedded library copied" || echo "⚠ SurrealDB embedded library not found in build/"
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

# Build llama.cpp with specific variant and copy to build/libs/{variant}/
build-libs-variant:
	@if [ -z "$(VARIANT)" ]; then \
		echo "Error: VARIANT not specified"; \
		echo "Usage: make build-libs-variant VARIANT=cuda"; \
		exit 1; \
	fi
	@echo "Building llama.cpp with $(VARIANT) support..."
	@mkdir -p $(BUILD_DIR)/libs/$(VARIANT)
	@# Clean previous llama.cpp build
	@cd $(GO_LLAMA_DIR) && rm -rf build && rm -f prepare *.o *.a
	@# Build with specific variant
	@if [ "$(VARIANT)" = "cpu" ]; then \
		cd $(GO_LLAMA_DIR) && \
		cmake -B build llama.cpp -DLLAMA_STATIC=OFF -DCMAKE_BUILD_TYPE=Release && \
		cmake --build build --config Release -j; \
	else \
		cd $(GO_LLAMA_DIR) && BUILD_TYPE=$(VARIANT) \
		cmake -B build llama.cpp -DLLAMA_STATIC=OFF -DCMAKE_BUILD_TYPE=Release && \
		cmake --build build --config Release -j; \
	fi
	@# Copy all shared libraries to variant directory
	@echo "Copying libraries to $(BUILD_DIR)/libs/$(VARIANT)/"
	@find $(GO_LLAMA_DIR)/build -type f \( -name "*.so" -o -name "*.so.*" -o -name "*.dylib" \) \
		-exec cp {} $(BUILD_DIR)/libs/$(VARIANT)/ \; 2>/dev/null || true
	@# Create variant info file
	@echo "Variant: $(VARIANT)" > $(BUILD_DIR)/libs/$(VARIANT)/BUILD_INFO.txt
	@echo "Built: $$(date)" >> $(BUILD_DIR)/libs/$(VARIANT)/BUILD_INFO.txt
	@echo "Platform: $(PLATFORM)" >> $(BUILD_DIR)/libs/$(VARIANT)/BUILD_INFO.txt
	@echo "Arch: $(UNAME_M)" >> $(BUILD_DIR)/libs/$(VARIANT)/BUILD_INFO.txt
	@ls -lh $(BUILD_DIR)/libs/$(VARIANT)/
	@echo "✓ $(VARIANT) libraries built successfully"

# Build CUDA variant
build-libs-cuda:
	@echo "Building CUDA variant..."
	@$(MAKE) build-libs-variant VARIANT=cuda

# Build HIP/ROCm variant
build-libs-hipblas:
	@echo "Building HIPBlas (ROCm) variant..."
	@$(MAKE) build-libs-variant VARIANT=hipblas

# Build Metal variant (macOS)
build-libs-metal:
	@echo "Building Metal variant..."
	@$(MAKE) build-libs-variant VARIANT=metal

# Build OpenBLAS variant
build-libs-openblas:
	@echo "Building OpenBLAS variant..."
	@$(MAKE) build-libs-variant VARIANT=openblas

# Build CPU-only variant
build-libs-cpu:
	@echo "Building CPU-only variant..."
	@$(MAKE) build-libs-variant VARIANT=cpu

# Build all available variants for current platform
build-libs-all-variants:
	@echo "Building all library variants for $(PLATFORM)..."
	@echo ""
ifeq ($(PLATFORM),darwin)
	@# macOS: CPU and Metal
	@echo "=== Building CPU variant ==="
	@$(MAKE) build-libs-cpu
	@echo ""
	@echo "=== Building Metal variant ==="
	@$(MAKE) build-libs-metal
	@echo ""
	@echo "✓ All macOS variants built successfully!"
else ifeq ($(PLATFORM),linux)
	@# Linux: CPU, CUDA, HIPBlas, OpenBLAS
	@echo "=== Building CPU variant ==="
	@$(MAKE) build-libs-cpu
	@echo ""
	@if command -v nvcc >/dev/null 2>&1; then \
		echo "=== Building CUDA variant ==="; \
		$(MAKE) build-libs-cuda; \
		echo ""; \
	else \
		echo "⚠ Skipping CUDA (nvcc not found)"; \
	fi
	@if [ -d "/opt/rocm" ]; then \
		echo "=== Building HIPBlas variant ==="; \
		$(MAKE) build-libs-hipblas; \
		echo ""; \
	else \
		echo "⚠ Skipping HIPBlas (ROCm not found)"; \
	fi
	@if pkg-config --exists openblas 2>/dev/null || [ -f "/usr/include/openblas/cblas.h" ]; then \
		echo "=== Building OpenBLAS variant ==="; \
		$(MAKE) build-libs-openblas; \
		echo ""; \
	else \
		echo "⚠ Skipping OpenBLAS (not found)"; \
	fi
	@echo "✓ All Linux variants built successfully!"
endif
	@echo ""
	@echo "Libraries available in:"
	@find $(BUILD_DIR)/libs -type f -name "BUILD_INFO.txt" -exec echo "  - {}" \;
	@echo ""
	@echo "To use a specific variant, copy libraries from build/libs/{variant}/ to build/"

# Build a single variant binary with specific name (e.g., remembrances-mcp-cuda)
build-variant: surrealdb-embedded
	@if [ -z "$(VARIANT)" ]; then \
		echo "Error: VARIANT not specified"; \
		echo "Usage: make build-variant VARIANT=cuda"; \
		exit 1; \
	fi
	@echo "Building variant binary: $(BINARY_NAME)-$(VARIANT)"
	@# First build the libraries for this variant
	@$(MAKE) build-libs-variant VARIANT=$(VARIANT)
	@# Build the Go binary with variant-specific name
	@mkdir -p $(BUILD_DIR)
	go build -mod=mod -v $(GO_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-$(VARIANT) ./cmd/remembrances-mcp
	@# Copy SurrealDB embedded library
	@echo "Copying SurrealDB embedded library..."
	@cp $(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release/libsurrealdb_embedded_rs.so $(BUILD_DIR)/ 2>/dev/null || \
	 cp $(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release/libsurrealdb_embedded_rs.dylib $(BUILD_DIR)/ 2>/dev/null || \
	 echo "⚠ Warning: SurrealDB embedded library not found"
	@# Copy variant-specific llama.cpp libraries to build root
	@echo "Copying $(VARIANT) libraries to build directory..."
	@cp $(BUILD_DIR)/libs/$(VARIANT)/*.so $(BUILD_DIR)/ 2>/dev/null || true
	@cp $(BUILD_DIR)/libs/$(VARIANT)/*.dylib $(BUILD_DIR)/ 2>/dev/null || true
	@echo "✓ Variant binary built: $(BUILD_DIR)/$(BINARY_NAME)-$(VARIANT)"

# Build all variant binaries
build-all-variants:
	@echo "Building all variant binaries for $(PLATFORM)..."
	@echo ""
ifeq ($(PLATFORM),darwin)
	@# macOS: CPU and Metal
	@echo "=== Building CPU variant binary ==="
	@$(MAKE) build-variant VARIANT=cpu
	@echo ""
	@echo "=== Building Metal variant binary ==="
	@$(MAKE) build-variant VARIANT=metal
	@echo ""
	@echo "✓ All macOS variant binaries built successfully!"
else ifeq ($(PLATFORM),linux)
	@# Linux: CPU, CUDA, HIPBlas, OpenBLAS
	@echo "=== Building CPU variant binary ==="
	@$(MAKE) build-variant VARIANT=cpu
	@echo ""
	@if command -v nvcc >/dev/null 2>&1; then \
		echo "=== Building CUDA variant binary ==="; \
		$(MAKE) build-variant VARIANT=cuda; \
		echo ""; \
	else \
		echo "⚠ Skipping CUDA (nvcc not found)"; \
	fi
	@if [ -d "/opt/rocm" ]; then \
		echo "=== Building HIPBlas variant binary ==="; \
		$(MAKE) build-variant VARIANT=hipblas; \
		echo ""; \
	else \
		echo "⚠ Skipping HIPBlas (ROCm not found)"; \
	fi
	@if pkg-config --exists openblas 2>/dev/null || [ -f "/usr/include/openblas/cblas.h" ]; then \
		echo "=== Building OpenBLAS variant binary ==="; \
		$(MAKE) build-variant VARIANT=openblas; \
		echo ""; \
	else \
		echo "⚠ Skipping OpenBLAS (not found)"; \
	fi
	@echo "✓ All Linux variant binaries built successfully!"
endif
	@echo ""
	@echo "Variant binaries available in $(BUILD_DIR)/:"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)-* 2>/dev/null || echo "  (none found)"

# Package a single variant with its libraries as a zip file
dist-variant:
	@if [ -z "$(VARIANT)" ]; then \
		echo "Error: VARIANT not specified"; \
		echo "Usage: make dist-variant VARIANT=cuda"; \
		exit 1; \
	fi
	@echo "Packaging $(VARIANT) variant for distribution..."
	@# Ensure the variant binary exists
	@if [ ! -f "$(BUILD_DIR)/$(BINARY_NAME)-$(VARIANT)" ]; then \
		echo "Building $(VARIANT) variant first..."; \
		$(MAKE) build-variant VARIANT=$(VARIANT); \
	fi
	@# Create dist directory structure
	@mkdir -p dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M)
	@# Copy variant binary with default name
	@cp $(BUILD_DIR)/$(BINARY_NAME)-$(VARIANT) dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M)/$(BINARY_NAME)
	@# Copy variant-specific libraries
	@cp $(BUILD_DIR)/libs/$(VARIANT)/*.so dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M)/ 2>/dev/null || true
	@cp $(BUILD_DIR)/libs/$(VARIANT)/*.dylib dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M)/ 2>/dev/null || true
	@# Copy SurrealDB embedded library
	@echo "Copying SurrealDB embedded library to distribution..."
	@cp $(BUILD_DIR)/libsurrealdb_embedded_rs.so dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M)/ 2>/dev/null || \
	 cp $(BUILD_DIR)/libsurrealdb_embedded_rs.dylib dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M)/ 2>/dev/null || \
	 echo "⚠ Warning: SurrealDB embedded library not found for distribution"
	@# Copy documentation and configs
	@cp README.md dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M)/ 2>/dev/null || true
	@cp LICENSE.txt dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M)/ 2>/dev/null || true
	@cp config.sample.yaml dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M)/ 2>/dev/null || true
	@cp config.sample.gguf.yaml dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M)/ 2>/dev/null || true
	@cp run-remembrances.sh dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M)/ 2>/dev/null || true
	@# Create variant info file
	@echo "Variant: $(VARIANT)" > dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M)/VARIANT_INFO.txt
	@echo "Platform: $(PLATFORM)" >> dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M)/VARIANT_INFO.txt
	@echo "Architecture: $(UNAME_M)" >> dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M)/VARIANT_INFO.txt
	@echo "Built: $$(date)" >> dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M)/VARIANT_INFO.txt
	@# Create zip archive
	@cd dist-variants && zip -r $(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M).zip $(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M)
	@# Clean up temporary directory
	@rm -rf dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M)
	@echo "✓ Package created: dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M).zip"
	@ls -lh dist-variants/$(BINARY_NAME)-$(VARIANT)-$(PLATFORM)-$(UNAME_M).zip

# Package all variants for distribution
dist-all:
	@echo "Packaging all variants for distribution..."
	@mkdir -p dist-variants
ifeq ($(PLATFORM),darwin)
	@# macOS: CPU and Metal
	@$(MAKE) dist-variant VARIANT=cpu
	@$(MAKE) dist-variant VARIANT=metal
else ifeq ($(PLATFORM),linux)
	@# Linux: CPU and available GPU variants
	@$(MAKE) dist-variant VARIANT=cpu
	@if command -v nvcc >/dev/null 2>&1; then \
		$(MAKE) dist-variant VARIANT=cuda; \
	fi
	@if [ -d "/opt/rocm" ]; then \
		$(MAKE) dist-variant VARIANT=hipblas; \
	fi
	@if pkg-config --exists openblas 2>/dev/null || [ -f "/usr/include/openblas/cblas.h" ]; then \
		$(MAKE) dist-variant VARIANT=openblas; \
	fi
endif
	@echo ""
	@echo "✓ All variant packages created in dist-variants/:"
	@ls -lh dist-variants/*.zip 2>/dev/null || echo "  (no packages found)"

# Package all library variants for distribution (legacy target)
package-libs-all:
	@echo "Packaging all library variants..."
	@mkdir -p dist/libs
	@for variant in $(BUILD_DIR)/libs/*; do \
		if [ -d "$$variant" ]; then \
			variant_name=$$(basename $$variant); \
			echo "Packaging $$variant_name..."; \
			tar -czf dist/libs/llama-cpp-$$variant_name-$(PLATFORM)-$(UNAME_M).tar.gz \
				-C $(BUILD_DIR)/libs $$variant_name; \
		fi \
	done
	@echo "✓ All variants packaged in dist/libs/"
	@ls -lh dist/libs/

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@go clean
	@echo "Clean complete"

# Clean variant libraries only
clean-libs-variants:
	@echo "Cleaning variant libraries..."
	@rm -rf $(BUILD_DIR)/libs
	@echo "Variant libraries cleaned"

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

# Cross-compile for multiple platforms using Docker (recommended)
build-cross:
	@echo "Cross-compiling for multiple platforms using goreleaser-cross..."
	@./scripts/release-cross.sh snapshot

# Build only shared libraries for cross-compilation
build-libs-cross:
	@echo "Building shared libraries for cross-compilation..."
	@./scripts/release-cross.sh --libs-only

# Create a release with cross-compilation
release-cross:
	@echo "Creating cross-platform release..."
	@./scripts/release-cross.sh release

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
