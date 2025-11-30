.PHONY: all build clean test llama-cpp llama-cpp-clean help \
	docker-build-cuda docker-push-cuda docker-run-cuda docker-stop-cuda \
	docker-build-cpu docker-push-cpu docker-run-cpu docker-stop-cpu \
	docker-download-model docker-prepare-cuda docker-prepare-cpu docker-login docker-help

# Default target
all: build

# Variables - Use $(HOME) instead of ~ for proper expansion in make
# These can be overridden via environment variables or command line
GO_LLAMA_DIR ?= $(HOME)/www/MCP/Remembrances/go-llama.cpp
SURREALDB_EMBEDDED_DIR ?= $(HOME)/www/MCP/Remembrances/surrealdb-embedded
BUILD_DIR := build
BINARY_NAME := remembrances-mcp

# Version information from git
# Get latest tag (version), default to "dev" if no tags exist
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
# Get current commit hash (short)
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
# Get full commit hash
COMMIT_HASH_FULL := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
# Check if working directory is dirty
GIT_DIRTY := $(shell git diff --quiet 2>/dev/null || echo "-dirty")
# Build version string (append -dirty if there are uncommitted changes)
BUILD_VERSION := $(VERSION)$(GIT_DIRTY)

# Go ldflags for version injection
VERSION_PKG := github.com/madeindigio/remembrances-mcp/pkg/version
GO_VERSION_LDFLAGS := -X $(VERSION_PKG).Version=$(BUILD_VERSION) -X $(VERSION_PKG).CommitHash=$(COMMIT_HASH)

# Detect OS and architecture
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# Normalize architecture names
ifeq ($(UNAME_M),x86_64)
	ARCH := amd64
else ifeq ($(UNAME_M),amd64)
	ARCH := amd64
else ifeq ($(UNAME_M),arm64)
	ARCH := arm64
else ifeq ($(UNAME_M),aarch64)
	ARCH := arm64
else
	ARCH := $(UNAME_M)
endif

# Platform-specific settings
ifeq ($(UNAME_S),Darwin)
	# macOS
	PLATFORM := darwin
	LLAMA_LDFLAGS := -framework Accelerate -framework Foundation -framework Metal -framework MetalKit -framework MetalPerformanceShaders
	BUILD_TYPE ?= metal
	# macOS library extension
	LIB_EXT := dylib
	# RPATH for macOS - use @executable_path
	RPATH_FLAG := -Wl,-rpath,@executable_path
	# Go linker RPATH option for macOS
	RPATH_OPTION := -r @executable_path
	# Go linker flags for macOS (RPATH only, no version - use GO_ALL_LDFLAGS for full flags)
	GO_LDFLAGS := -ldflags="-r @executable_path"
else ifeq ($(UNAME_S),Linux)
	# Linux
	PLATFORM := linux
	LLAMA_LDFLAGS := -lm -lstdc++ -lpthread
	BUILD_TYPE ?=
	# Linux library extension
	LIB_EXT := so
	# RPATH for Linux
	RPATH_FLAG := -Wl,-rpath,$$ORIGIN
	# Go linker RPATH option for Linux
	RPATH_OPTION := -r \$$ORIGIN
	# Go linker flags for Linux (RPATH only, no version - use GO_ALL_LDFLAGS for full flags)
	GO_LDFLAGS := -ldflags="-r \$$ORIGIN"
else
	$(error Unsupported platform: $(UNAME_S))
endif

# CGO flags for linking with llama.cpp and surrealdb-embedded
export CGO_ENABLED := 1
export CGO_CFLAGS := -I$(GO_LLAMA_DIR) -I$(GO_LLAMA_DIR)/llama.cpp -I$(GO_LLAMA_DIR)/llama.cpp/common -I$(GO_LLAMA_DIR)/llama.cpp/ggml/include -I$(GO_LLAMA_DIR)/llama.cpp/include -I$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/include
export CGO_LDFLAGS := -L$(GO_LLAMA_DIR) -L$(GO_LLAMA_DIR)/build/bin -L$(GO_LLAMA_DIR)/build/common -L$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release -lllama -lcommon -lggml -lggml-base -lsurrealdb_embedded_rs $(LLAMA_LDFLAGS)

# Go linker flags to set RPATH - platform-specific, defined above
# GO_LDFLAGS is set per-platform in the ifeq blocks above

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
	@echo "  make check-env          - Show build environment and library status"
	@echo ""
	@echo "macOS Cross-compilation (arm64 ↔ x86_64):"
	@echo "  make build-darwin-arm64    - Build libraries for Apple Silicon (arm64)"
	@echo "  make build-darwin-amd64    - Build libraries for Intel (x86_64)"
	@echo "  make build-darwin-universal - Build Universal Binary (both architectures)"
	@echo "  make dist-darwin-arm64     - Create distribution package for arm64"
	@echo "  make dist-darwin-amd64     - Create distribution package for x86_64"
	@echo ""
	@echo "SurrealDB cross-compilation:"
	@echo "  make build-surrealdb-darwin-arm64  - Build surrealdb-embedded for arm64"
	@echo "  make build-surrealdb-darwin-amd64  - Build surrealdb-embedded for x86_64"
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
	@echo "Cross-compilation targets (Docker):"
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
	@echo "Environment variables:"
	@echo "  GO_LLAMA_DIR          - Path to go-llama.cpp (default: \$$HOME/www/MCP/Remembrances/go-llama.cpp)"
	@echo "  SURREALDB_EMBEDDED_DIR - Path to surrealdb-embedded (default: \$$HOME/www/MCP/Remembrances/surrealdb-embedded)"
	@echo ""
	@echo "Docker (GitHub Container Registry):"
	@echo "  make docker-help            - Show detailed Docker usage"
	@echo "  make docker-prepare-cpu     - Build CPU binary + download GGUF model"
	@echo "  make docker-build-cpu       - Build lightweight Docker image (~350MB)"
	@echo "  make docker-prepare-cuda    - Build CUDA binary + download GGUF model"
	@echo "  make docker-build-cuda      - Build CUDA Docker image (~3GB)"
	@echo "  make docker-push-cpu        - Push CPU image to ghcr.io"
	@echo "  make docker-push-cuda       - Push CUDA image to ghcr.io"
	@echo "  make docker-run-cpu         - Run container (no GPU needed)"
	@echo "  make docker-run-cuda        - Run container with GPU support"
	@echo ""
	@echo "Examples:"
	@echo "  make build                      # Build with default settings (native arch)"
	@echo "  make BUILD_TYPE=cuda build      # Build with CUDA support (Linux)"
	@echo "  make build-darwin-amd64         # Cross-compile for Intel Mac"
	@echo "  make dist-darwin-arm64          # Create arm64 distribution"
	@echo "  make build-darwin-universal     # Create Universal Binary libraries"
	@echo "  make build-all-variants         # Build all GPU variant binaries"
	@echo "  make dist-all                   # Package all variants for distribution"
	@echo "  make docker-prepare-cpu && make docker-build-cpu    # Build CPU Docker image"
	@echo "  make docker-prepare-cuda && make docker-build-cuda  # Build CUDA Docker image"
	@echo "  make run                        # Build and run"
	@echo "  make check-env                  # Show current build environment"

# Build llama.cpp library
llama-cpp:
	@echo "Checking llama.cpp library..."
	@echo "  GO_LLAMA_DIR: $(GO_LLAMA_DIR)"
	@echo "  Platform: $(PLATFORM) / $(ARCH)"
	@# Check if the llama.cpp submodule exists (look for CMakeLists.txt inside llama.cpp/)
	@if [ ! -f "$(GO_LLAMA_DIR)/llama.cpp/CMakeLists.txt" ]; then \
		echo "Error: llama.cpp submodule not found at $(GO_LLAMA_DIR)/llama.cpp"; \
		echo "Please run: cd $(GO_LLAMA_DIR) && git submodule update --init --recursive"; \
		exit 1; \
	fi
	@# Check for already built libraries (platform-specific extension)
	@if [ ! -f "$(GO_LLAMA_DIR)/build/bin/libllama.$(LIB_EXT)" ]; then \
		echo "llama.cpp not built. Building now for $(PLATFORM)/$(ARCH)..."; \
		echo "Note: If this fails, please build manually:"; \
		echo "  cd $(GO_LLAMA_DIR) && cmake -B build llama.cpp -DLLAMA_STATIC=OFF && cmake --build build --config Release -j"; \
		cd "$(GO_LLAMA_DIR)" && BUILD_TYPE=$(BUILD_TYPE) cmake -B build llama.cpp -DLLAMA_STATIC=OFF -DCMAKE_BUILD_TYPE=Release && cmake --build build --config Release -j; \
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
	@echo "  SURREALDB_EMBEDDED_DIR: $(SURREALDB_EMBEDDED_DIR)"
	@if [ ! -d "$(SURREALDB_EMBEDDED_DIR)" ]; then \
		echo "Error: surrealdb-embedded not found at $(SURREALDB_EMBEDDED_DIR)"; \
		exit 1; \
	fi
	@if [ ! -f "$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release/libsurrealdb_embedded_rs.$(LIB_EXT)" ]; then \
		echo "surrealdb-embedded not built for $(PLATFORM)/$(ARCH). Building now..."; \
		cd "$(SURREALDB_EMBEDDED_DIR)" && make build-rust; \
	else \
		echo "surrealdb-embedded library already built"; \
	fi
	@echo "surrealdb-embedded library ready"
	@# Copy library to build directory
	@mkdir -p $(BUILD_DIR)
	@echo "Copying SurrealDB embedded library to $(BUILD_DIR)/..."
	@cp "$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release/libsurrealdb_embedded_rs.$(LIB_EXT)" "$(BUILD_DIR)/" 2>/dev/null && \
		echo "✓ SurrealDB embedded library copied to $(BUILD_DIR)/" || \
		echo "⚠ Warning: Could not copy SurrealDB embedded library"

# Build the main project
build: llama-cpp surrealdb-embedded
	@echo "Building $(BINARY_NAME) with GGUF and embedded SurrealDB support..."
	@echo "  Target: $(PLATFORM)/$(ARCH)"
	@echo "  Version: $(BUILD_VERSION)"
	@echo "  Commit: $(COMMIT_HASH)"
	@mkdir -p $(BUILD_DIR)
	go build -mod=mod -v -ldflags="$(RPATH_OPTION) $(GO_VERSION_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/remembrances-mcp
	@echo "Copying shared libraries to build directory..."
	@# Copy SurrealDB embedded library from Rust build directory
	@echo "Copying SurrealDB embedded library..."
	@cp "$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release/libsurrealdb_embedded_rs.$(LIB_EXT)" "$(BUILD_DIR)/" 2>/dev/null || \
	 echo "⚠ Warning: SurrealDB embedded library not found"
	@# Copy ALL llama.cpp shared libraries (platform-specific extension)
	@# This includes libraries for CUDA, Metal, ROCm, Vulkan, etc.
	@echo "Copying all llama.cpp shared libraries (*.$(LIB_EXT))..."
	@find "$(GO_LLAMA_DIR)/build/bin" -type f -name "*.$(LIB_EXT)" -exec cp {} "$(BUILD_DIR)/" \; 2>/dev/null || true
	@# Also check other common locations in the build directory
	@find "$(GO_LLAMA_DIR)/build/src" -type f -name "*.$(LIB_EXT)" -exec cp {} "$(BUILD_DIR)/" \; 2>/dev/null || true
	@find "$(GO_LLAMA_DIR)/build/common" -type f -name "*.$(LIB_EXT)" -exec cp {} "$(BUILD_DIR)/" \; 2>/dev/null || true
	@find "$(GO_LLAMA_DIR)/build/ggml" -type f -name "*.$(LIB_EXT)" -exec cp {} "$(BUILD_DIR)/" \; 2>/dev/null || true
	@echo "Shared libraries copied successfully"
	@ls -lh $(BUILD_DIR)/libsurrealdb_embedded_rs.* 2>/dev/null && echo "✓ SurrealDB embedded library copied" || echo "⚠ SurrealDB embedded library not found in build/"
	@ls -lh $(BUILD_DIR)/libllama.* 2>/dev/null && echo "✓ llama.cpp libraries copied" || echo "⚠ llama.cpp libraries not found in build/"
ifeq ($(PLATFORM),darwin)
	@# Fix RPATH for macOS - add @executable_path and update library references
	@echo "Fixing macOS library paths..."
	@install_name_tool -add_rpath @executable_path "$(BUILD_DIR)/$(BINARY_NAME)" 2>/dev/null || true
	@# Fix any absolute paths to surrealdb library
	@for lib_path in $$(otool -L "$(BUILD_DIR)/$(BINARY_NAME)" | grep surrealdb_embedded_rs | grep -v "@rpath" | awk '{print $$1}'); do \
		echo "  Fixing reference: $$lib_path -> @rpath/libsurrealdb_embedded_rs.dylib"; \
		install_name_tool -change "$$lib_path" "@rpath/libsurrealdb_embedded_rs.dylib" "$(BUILD_DIR)/$(BINARY_NAME)"; \
	done
	@echo "✓ macOS library paths fixed"
endif
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
	@echo "Building llama.cpp with $(VARIANT) support for $(PLATFORM)/$(ARCH)..."
	@mkdir -p $(BUILD_DIR)/libs/$(VARIANT)
	@# Clean previous llama.cpp build
	@cd "$(GO_LLAMA_DIR)" && rm -rf build && rm -f prepare *.o *.a
	@# Build with specific variant
	@if [ "$(VARIANT)" = "cpu" ]; then \
		cd "$(GO_LLAMA_DIR)" && \
		cmake -B build llama.cpp -DLLAMA_STATIC=OFF -DCMAKE_BUILD_TYPE=Release && \
		cmake --build build --config Release -j; \
	else \
		cd "$(GO_LLAMA_DIR)" && BUILD_TYPE=$(VARIANT) \
		cmake -B build llama.cpp -DLLAMA_STATIC=OFF -DCMAKE_BUILD_TYPE=Release && \
		cmake --build build --config Release -j; \
	fi
	@# Copy all shared libraries to variant directory (platform-specific)
	@echo "Copying libraries to $(BUILD_DIR)/libs/$(VARIANT)/"
	@find "$(GO_LLAMA_DIR)/build" -type f -name "*.$(LIB_EXT)" \
		-exec cp {} "$(BUILD_DIR)/libs/$(VARIANT)/" \; 2>/dev/null || true
	@# Create variant info file
	@echo "Variant: $(VARIANT)" > $(BUILD_DIR)/libs/$(VARIANT)/BUILD_INFO.txt
	@echo "Built: $$(date)" >> $(BUILD_DIR)/libs/$(VARIANT)/BUILD_INFO.txt
	@echo "Platform: $(PLATFORM)" >> $(BUILD_DIR)/libs/$(VARIANT)/BUILD_INFO.txt
	@echo "Arch: $(ARCH)" >> $(BUILD_DIR)/libs/$(VARIANT)/BUILD_INFO.txt
	@ls -lh $(BUILD_DIR)/libs/$(VARIANT)/
	@echo "✓ $(VARIANT) libraries built successfully"

# macOS-specific: Build for specific architecture (arm64 or amd64)
build-darwin-arch:
	@if [ -z "$(TARGET_ARCH)" ]; then \
		echo "Error: TARGET_ARCH not specified"; \
		echo "Usage: make build-darwin-arch TARGET_ARCH=arm64"; \
		echo "       make build-darwin-arch TARGET_ARCH=x86_64"; \
		exit 1; \
	fi
	@if [ "$(PLATFORM)" != "darwin" ]; then \
		echo "Error: This target is only for macOS"; \
		exit 1; \
	fi
	@echo "Building llama.cpp for macOS $(TARGET_ARCH)..."
	@mkdir -p $(BUILD_DIR)/libs/darwin-$(TARGET_ARCH)
	@# Clean and build for specific architecture
	@cd "$(GO_LLAMA_DIR)" && rm -rf build-$(TARGET_ARCH)
	@# Set architecture-specific flags for cross-compilation
	@if [ "$(TARGET_ARCH)" = "x86_64" ]; then \
		echo "Cross-compiling for Intel x86_64..."; \
		cd "$(GO_LLAMA_DIR)" && \
		cmake -B build-$(TARGET_ARCH) llama.cpp \
			-DLLAMA_STATIC=OFF \
			-DCMAKE_BUILD_TYPE=Release \
			-DCMAKE_OSX_ARCHITECTURES=x86_64 \
			-DCMAKE_C_FLAGS="-arch x86_64" \
			-DCMAKE_CXX_FLAGS="-arch x86_64" \
			-DGGML_NATIVE=OFF \
			-DLLAMA_METAL=OFF && \
		cmake --build build-$(TARGET_ARCH) --config Release -j; \
	else \
		echo "Building for Apple Silicon arm64..."; \
		cd "$(GO_LLAMA_DIR)" && \
		cmake -B build-$(TARGET_ARCH) llama.cpp \
			-DLLAMA_STATIC=OFF \
			-DCMAKE_BUILD_TYPE=Release \
			-DCMAKE_OSX_ARCHITECTURES=arm64 \
			-DLLAMA_METAL=ON && \
		cmake --build build-$(TARGET_ARCH) --config Release -j; \
	fi
	@# Copy libraries
	@find "$(GO_LLAMA_DIR)/build-$(TARGET_ARCH)" -type f -name "*.dylib" \
		-exec cp {} "$(BUILD_DIR)/libs/darwin-$(TARGET_ARCH)/" \; 2>/dev/null || true
	@echo "✓ macOS $(TARGET_ARCH) libraries built"

# Build for macOS arm64 (Apple Silicon)
build-darwin-arm64:
	@echo "Building llama.cpp for macOS arm64 (Apple Silicon)..."
	@$(MAKE) build-darwin-arch TARGET_ARCH=arm64
	@echo "Building surrealdb-embedded for macOS arm64..."
	@$(MAKE) build-surrealdb-darwin-arm64

# Build for macOS amd64 (Intel)
build-darwin-amd64:
	@echo "Building llama.cpp for macOS amd64 (Intel)..."
	@$(MAKE) build-darwin-arch TARGET_ARCH=x86_64
	@echo "Building surrealdb-embedded for macOS amd64..."
	@$(MAKE) build-surrealdb-darwin-amd64

# Build Universal Binary for macOS (both arm64 and x86_64)
build-darwin-universal: build-darwin-arm64 build-darwin-amd64
	@echo "Creating Universal Binary for macOS..."
	@mkdir -p $(BUILD_DIR)/libs/darwin-universal
	@# Combine llama.cpp libraries using lipo
	@echo "Combining llama.cpp libraries..."
	@for lib in $(BUILD_DIR)/libs/darwin-arm64/*.dylib; do \
		libname=$$(basename $$lib); \
		if [ -f "$(BUILD_DIR)/libs/darwin-x86_64/$$libname" ]; then \
			echo "Creating universal $$libname..."; \
			lipo -create \
				"$(BUILD_DIR)/libs/darwin-arm64/$$libname" \
				"$(BUILD_DIR)/libs/darwin-x86_64/$$libname" \
				-output "$(BUILD_DIR)/libs/darwin-universal/$$libname"; \
		else \
			echo "Copying $$libname (arm64 only)..."; \
			cp "$$lib" "$(BUILD_DIR)/libs/darwin-universal/"; \
		fi \
	done
	@# Combine surrealdb-embedded library if both exist
	@echo "Combining surrealdb-embedded libraries..."
	@if [ -f "$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/aarch64-apple-darwin/release/libsurrealdb_embedded_rs.dylib" ] && \
	   [ -f "$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/x86_64-apple-darwin/release/libsurrealdb_embedded_rs.dylib" ]; then \
		lipo -create \
			"$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/aarch64-apple-darwin/release/libsurrealdb_embedded_rs.dylib" \
			"$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/x86_64-apple-darwin/release/libsurrealdb_embedded_rs.dylib" \
			-output "$(BUILD_DIR)/libs/darwin-universal/libsurrealdb_embedded_rs.dylib"; \
		echo "✓ Universal surrealdb_embedded_rs.dylib created"; \
	else \
		echo "⚠ Cannot create universal surrealdb library - missing one or both architectures"; \
		cp "$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release/libsurrealdb_embedded_rs.dylib" \
			"$(BUILD_DIR)/libs/darwin-universal/" 2>/dev/null || true; \
	fi
	@echo "✓ Universal Binary libraries created in $(BUILD_DIR)/libs/darwin-universal/"
	@ls -lh $(BUILD_DIR)/libs/darwin-universal/

# Build surrealdb-embedded for specific Rust target
build-surrealdb-target:
	@if [ -z "$(RUST_TARGET)" ]; then \
		echo "Error: RUST_TARGET not specified"; \
		echo "Usage: make build-surrealdb-target RUST_TARGET=aarch64-apple-darwin"; \
		exit 1; \
	fi
	@echo "Building surrealdb-embedded for $(RUST_TARGET)..."
	@cd "$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs" && \
		rustup target add $(RUST_TARGET) 2>/dev/null || true && \
		cargo build --release --target $(RUST_TARGET)
	@mkdir -p $(BUILD_DIR)/libs/surrealdb-$(RUST_TARGET)
	@cp "$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/$(RUST_TARGET)/release/libsurrealdb_embedded_rs.dylib" \
		"$(BUILD_DIR)/libs/surrealdb-$(RUST_TARGET)/" 2>/dev/null || \
	 cp "$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/$(RUST_TARGET)/release/libsurrealdb_embedded_rs.so" \
		"$(BUILD_DIR)/libs/surrealdb-$(RUST_TARGET)/" 2>/dev/null || \
	 echo "⚠ Could not copy surrealdb library for $(RUST_TARGET)"
	@echo "✓ surrealdb-embedded built for $(RUST_TARGET)"

# Build surrealdb-embedded for macOS arm64
build-surrealdb-darwin-arm64:
	@echo "Building surrealdb-embedded for macOS arm64 (Apple Silicon)..."
	@$(MAKE) build-surrealdb-target RUST_TARGET=aarch64-apple-darwin

# Build surrealdb-embedded for macOS amd64
build-surrealdb-darwin-amd64:
	@echo "Building surrealdb-embedded for macOS amd64 (Intel)..."
	@$(MAKE) build-surrealdb-target RUST_TARGET=x86_64-apple-darwin

# Build complete distribution for macOS arm64
dist-darwin-arm64: build-darwin-arm64
	@echo "Creating distribution for macOS arm64..."
	@echo "  Version: $(BUILD_VERSION), Commit: $(COMMIT_HASH)"
	@mkdir -p dist/darwin-arm64
	@# Build Go binary for arm64 with correct library paths
	@# libcommon is static (.a), so we link it directly
	CGO_CFLAGS="-I$(GO_LLAMA_DIR) -I$(GO_LLAMA_DIR)/llama.cpp -I$(GO_LLAMA_DIR)/llama.cpp/common -I$(GO_LLAMA_DIR)/llama.cpp/ggml/include -I$(GO_LLAMA_DIR)/llama.cpp/include -I$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/include" \
	CGO_LDFLAGS="-L$(GO_LLAMA_DIR)/build-arm64/bin -L$(GO_LLAMA_DIR)/build-arm64/common -L$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/aarch64-apple-darwin/release -lllama -lggml -lggml-base -lsurrealdb_embedded_rs $(GO_LLAMA_DIR)/build-arm64/common/libcommon.a $(LLAMA_LDFLAGS) -lc++" \
		GOARCH=arm64 GOOS=darwin go build -mod=mod -v -ldflags="$(GO_VERSION_LDFLAGS)" -o dist/darwin-arm64/$(BINARY_NAME) ./cmd/remembrances-mcp
	@# Copy libraries
	@cp $(BUILD_DIR)/libs/darwin-arm64/*.dylib dist/darwin-arm64/ 2>/dev/null || \
	 find "$(GO_LLAMA_DIR)/build-arm64" -name "*.dylib" -exec cp {} dist/darwin-arm64/ \; 2>/dev/null || \
	 find "$(GO_LLAMA_DIR)/build/bin" -name "*.dylib" -exec cp {} dist/darwin-arm64/ \; 2>/dev/null || true
	@cp "$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/aarch64-apple-darwin/release/libsurrealdb_embedded_rs.dylib" \
		dist/darwin-arm64/ 2>/dev/null || \
	 cp "$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release/libsurrealdb_embedded_rs.dylib" \
		dist/darwin-arm64/ 2>/dev/null || true
	@cp README.md LICENSE.txt config.sample.yaml config.sample.gguf.yaml dist/darwin-arm64/ 2>/dev/null || true
	@# Fix RPATH for macOS
	@echo "Fixing macOS library paths for arm64 distribution..."
	@install_name_tool -add_rpath @executable_path dist/darwin-arm64/$(BINARY_NAME) 2>/dev/null || true
	@for lib_path in $$(otool -L dist/darwin-arm64/$(BINARY_NAME) | grep surrealdb_embedded_rs | grep -v "@rpath" | awk '{print $$1}'); do \
		install_name_tool -change "$$lib_path" "@rpath/libsurrealdb_embedded_rs.dylib" dist/darwin-arm64/$(BINARY_NAME); \
	done
	@echo "✓ Distribution created in dist/darwin-arm64/"
	@ls -lh dist/darwin-arm64/

# Build complete distribution for macOS amd64
dist-darwin-amd64: build-darwin-amd64
	@echo "Creating distribution for macOS amd64..."
	@echo "  Version: $(BUILD_VERSION), Commit: $(COMMIT_HASH)"
	@mkdir -p dist/darwin-amd64
	@# Build Go binary for amd64 with x86_64-specific library paths
	@# Note: Metal is disabled for x86_64, libcommon is static (.a) so we link it directly
	CGO_CFLAGS="-I$(GO_LLAMA_DIR) -I$(GO_LLAMA_DIR)/llama.cpp -I$(GO_LLAMA_DIR)/llama.cpp/common -I$(GO_LLAMA_DIR)/llama.cpp/ggml/include -I$(GO_LLAMA_DIR)/llama.cpp/include -I$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/include" \
	CGO_LDFLAGS="-L$(GO_LLAMA_DIR)/build-x86_64/bin -L$(GO_LLAMA_DIR)/build-x86_64/common -L$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/x86_64-apple-darwin/release -lllama -lggml -lggml-base -lsurrealdb_embedded_rs $(GO_LLAMA_DIR)/build-x86_64/common/libcommon.a -framework Accelerate -framework Foundation -lc++" \
		GOARCH=amd64 GOOS=darwin go build -mod=mod -v -ldflags="$(GO_VERSION_LDFLAGS)" -o dist/darwin-amd64/$(BINARY_NAME) ./cmd/remembrances-mcp
	@# Copy libraries
	@cp $(BUILD_DIR)/libs/darwin-x86_64/*.dylib dist/darwin-amd64/ 2>/dev/null || \
	 find "$(GO_LLAMA_DIR)/build-x86_64" -name "*.dylib" -exec cp {} dist/darwin-amd64/ \; 2>/dev/null || true
	@cp "$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/x86_64-apple-darwin/release/libsurrealdb_embedded_rs.dylib" \
		dist/darwin-amd64/ 2>/dev/null || true
	@cp README.md LICENSE.txt config.sample.yaml config.sample.gguf.yaml dist/darwin-amd64/ 2>/dev/null || true
	@# Fix RPATH for macOS
	@echo "Fixing macOS library paths for amd64 distribution..."
	@install_name_tool -add_rpath @executable_path dist/darwin-amd64/$(BINARY_NAME) 2>/dev/null || true
	@for lib_path in $$(otool -L dist/darwin-amd64/$(BINARY_NAME) | grep surrealdb_embedded_rs | grep -v "@rpath" | awk '{print $$1}'); do \
		install_name_tool -change "$$lib_path" "@rpath/libsurrealdb_embedded_rs.dylib" dist/darwin-amd64/$(BINARY_NAME); \
	done
	@echo "✓ Distribution created in dist/darwin-amd64/"
	@ls -lh dist/darwin-amd64/

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
	@echo "  Architecture: $(UNAME_M) (normalized: $(ARCH))"
	@echo "  Build Type: $(BUILD_TYPE)"
	@echo "  Library Extension: $(LIB_EXT)"
	@echo "  Go Version: $$(go version)"
	@echo "  CGO Enabled: $(CGO_ENABLED)"
	@echo "  llama.cpp Dir: $(GO_LLAMA_DIR)"
	@echo "  SurrealDB Dir: $(SURREALDB_EMBEDDED_DIR)"
	@echo "  CGO_CFLAGS: $(CGO_CFLAGS)"
	@echo "  CGO_LDFLAGS: $(CGO_LDFLAGS)"
	@echo ""
	@echo "Version Information:"
	@echo "  Version: $(VERSION)"
	@echo "  Build Version: $(BUILD_VERSION)"
	@echo "  Commit Hash: $(COMMIT_HASH)"
	@echo "  Commit (full): $(COMMIT_HASH_FULL)"
	@echo ""
	@echo "Library Status:"
	@echo "  llama.cpp: $$(ls $(GO_LLAMA_DIR)/build/bin/libllama.$(LIB_EXT) 2>/dev/null && echo 'Found' || echo 'Not built')"
	@echo "  surrealdb: $$(ls $(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release/libsurrealdb_embedded_rs.$(LIB_EXT) 2>/dev/null && echo 'Found' || echo 'Not built')"
ifeq ($(PLATFORM),darwin)
	@echo ""
	@echo "macOS Cross-compilation targets:"
	@echo "  make build-darwin-arm64     - Build for Apple Silicon"
	@echo "  make build-darwin-amd64     - Build for Intel"
	@echo "  make build-darwin-universal - Build Universal Binary"
endif

# ═══════════════════════════════════════════════════════════════════════════════
# Docker Configuration and Targets
# ═══════════════════════════════════════════════════════════════════════════════

# Docker configuration
DOCKER_REGISTRY := ghcr.io
DOCKER_ORG := madeindigio
DOCKER_IMAGE := remembrances-mcp
DOCKER_TAG_CUDA := cuda
DOCKER_TAG_CPU := cpu
DOCKER_FULL_IMAGE := $(DOCKER_REGISTRY)/$(DOCKER_ORG)/$(DOCKER_IMAGE)

# GGUF Model configuration (same as install.sh)
GGUF_MODEL_URL := https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf?download=true
GGUF_MODEL_NAME := nomic-embed-text-v1.5.Q4_K_M.gguf
MODELS_DIR := models

# Docker run configuration (can be overridden)
DOCKER_DATA_PATH ?= $(HOME)/.local/share/remembrances/data
DOCKER_KB_PATH ?= $(HOME)/.local/share/remembrances/knowledge-base
DOCKER_PORT ?= 8080

docker-help:
	@echo ""
	@echo "═══════════════════════════════════════════════════════════════"
	@echo "           Docker Build & Publish (GitHub Container Registry)"
	@echo "═══════════════════════════════════════════════════════════════"
	@echo ""
	@echo "Two image variants available:"
	@echo "  CPU  - Lightweight (~350MB), no GPU required, uses debian:bookworm-slim"
	@echo "  CUDA - GPU accelerated (~3GB), requires NVIDIA GPU + Container Toolkit"
	@echo ""
	@echo "CPU Image (Recommended for most users):"
	@echo "  make docker-prepare-cpu     - Build CPU binary + download model"
	@echo "  make docker-build-cpu       - Build lightweight Docker image"
	@echo "  make docker-push-cpu        - Push CPU image to GHCR"
	@echo "  make docker-run-cpu         - Run container (no GPU needed)"
	@echo "  make docker-stop-cpu        - Stop CPU container"
	@echo ""
	@echo "CUDA Image (For GPU acceleration):"
	@echo "  make docker-prepare-cuda    - Build CUDA binary + download model"
	@echo "  make docker-build-cuda      - Build CUDA Docker image"
	@echo "  make docker-push-cuda       - Push CUDA image to GHCR"
	@echo "  make docker-run-cuda        - Run container with --gpus all"
	@echo "  make docker-stop-cuda       - Stop CUDA container"
	@echo ""
	@echo "Common targets:"
	@echo "  make docker-download-model  - Download GGUF embedding model (~260MB)"
	@echo "  make docker-login           - Login to GitHub Container Registry"
	@echo ""
	@echo "Configuration (override via environment or command line):"
	@echo "  DOCKER_DATA_PATH=$(DOCKER_DATA_PATH)"
	@echo "  DOCKER_KB_PATH=$(DOCKER_KB_PATH)"
	@echo "  DOCKER_PORT=$(DOCKER_PORT)"
	@echo ""
	@echo "Example workflows:"
	@echo "  # CPU (lightweight)"
	@echo "  make docker-prepare-cpu && make docker-build-cpu && make docker-run-cpu"
	@echo ""
	@echo "  # CUDA (GPU)"
	@echo "  make docker-prepare-cuda && make docker-build-cuda && make docker-run-cuda"
	@echo ""
	@echo "Run with custom paths:"
	@echo "  make docker-run-cpu DOCKER_DATA_PATH=/my/data DOCKER_KB_PATH=/my/kb DOCKER_PORT=9090"
	@echo ""

# Download GGUF model for Docker image
docker-download-model:
	@echo "Downloading GGUF embedding model..."
	@mkdir -p $(MODELS_DIR)
	@if [ -f "$(MODELS_DIR)/$(GGUF_MODEL_NAME)" ]; then \
		echo "Model already exists: $(MODELS_DIR)/$(GGUF_MODEL_NAME)"; \
		ls -lh $(MODELS_DIR)/$(GGUF_MODEL_NAME); \
	else \
		echo "Downloading $(GGUF_MODEL_NAME) (~260MB)..."; \
		curl -fsSL --progress-bar -o "$(MODELS_DIR)/$(GGUF_MODEL_NAME)" "$(GGUF_MODEL_URL)"; \
		echo "✓ Model downloaded to $(MODELS_DIR)/$(GGUF_MODEL_NAME)"; \
		ls -lh $(MODELS_DIR)/$(GGUF_MODEL_NAME); \
	fi

# Prepare everything needed for Docker CPU build
docker-prepare-cpu: docker-download-model
	@echo ""
	@echo "Building CPU variant for Docker..."
	@$(MAKE) dist-variant VARIANT=cpu
	@echo ""
	@echo "Extracting CPU variant for Docker build..."
	@cd dist-variants && unzip -o $(BINARY_NAME)-cpu-$(PLATFORM)-$(UNAME_M).zip
	@echo ""
	@echo "✓ Docker CPU preparation complete!"
	@echo "  Binary: dist-variants/$(BINARY_NAME)-cpu-$(PLATFORM)-$(UNAME_M)/"
	@ls -la dist-variants/$(BINARY_NAME)-cpu-$(PLATFORM)-$(UNAME_M)/
	@echo ""
	@echo "  Model: $(MODELS_DIR)/$(GGUF_MODEL_NAME)"
	@echo ""
	@echo "Next step: make docker-build-cpu"

# Prepare everything needed for Docker CUDA build
docker-prepare-cuda: docker-download-model
	@echo ""
	@echo "Building CUDA variant for Docker..."
	@echo "NOTE: This requires llama.cpp built with CUDA support (libggml-cuda.so)"
	@$(MAKE) dist-variant VARIANT=cuda
	@echo ""
	@echo "Extracting CUDA variant for Docker build..."
	@cd dist-variants && unzip -o $(BINARY_NAME)-cuda-$(PLATFORM)-$(UNAME_M).zip
	@echo ""
	@# Verify CUDA library exists
	@if [ ! -f "dist-variants/$(BINARY_NAME)-cuda-$(PLATFORM)-$(UNAME_M)/libggml-cuda.so" ]; then \
		echo "⚠ WARNING: libggml-cuda.so not found!"; \
		echo "  Your build may be CPU-only. For real CUDA support:"; \
		echo "  1. Install CUDA toolkit"; \
		echo "  2. Rebuild with: BUILD_TYPE=cuda make build-libs-cuda"; \
		echo "  3. Then: make dist-variant VARIANT=cuda"; \
		echo ""; \
	fi
	@echo "✓ Docker CUDA preparation complete!"
	@echo "  Binary: dist-variants/$(BINARY_NAME)-cuda-$(PLATFORM)-$(UNAME_M)/"
	@ls -la dist-variants/$(BINARY_NAME)-cuda-$(PLATFORM)-$(UNAME_M)/
	@echo ""
	@echo "  Model: $(MODELS_DIR)/$(GGUF_MODEL_NAME)"
	@echo ""
	@echo "Next step: make docker-build-cuda"

# Distribution directory variables
DIST_DIR_CPU := dist-variants/$(BINARY_NAME)-cpu-$(PLATFORM)-$(UNAME_M)
DIST_DIR_CUDA := dist-variants/$(BINARY_NAME)-cuda-$(PLATFORM)-$(UNAME_M)

# Build Docker image - CPU version (lightweight)
docker-build-cpu:
	@echo "Building Docker image: $(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CPU)"
	@echo "  Distribution dir: $(DIST_DIR_CPU)"
	@echo ""
	@# Verify prerequisites
	@if [ ! -d "$(DIST_DIR_CPU)" ]; then \
		echo "Error: CPU variant not found at $(DIST_DIR_CPU)"; \
		echo "Run 'make docker-prepare-cpu' first."; \
		exit 1; \
	fi
	@if [ ! -f "$(MODELS_DIR)/$(GGUF_MODEL_NAME)" ]; then \
		echo "Error: GGUF model not found. Run 'make docker-download-model' first."; \
		exit 1; \
	fi
	@# Build the image
	docker build \
		-f docker/Dockerfile.cpu \
		-t $(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CPU) \
		-t $(DOCKER_FULL_IMAGE):$(VERSION)-cpu \
		-t $(DOCKER_FULL_IMAGE):latest \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT_HASH) \
		--build-arg DIST_DIR=$(DIST_DIR_CPU) \
		.
	@echo ""
	@echo "✓ Docker CPU image built successfully!"
	@echo "  $(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CPU)"
	@echo "  $(DOCKER_FULL_IMAGE):$(VERSION)-cpu"
	@echo "  $(DOCKER_FULL_IMAGE):latest"
	@echo ""
	@echo "Image size:"
	@docker images $(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CPU) --format "  {{.Size}}"
	@echo ""
	@echo "Next step: make docker-login && make docker-push-cpu"
	@echo "Or run locally: make docker-run-cpu"

# Build Docker image - CUDA version (GPU accelerated)
docker-build-cuda:
	@echo "Building Docker image: $(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CUDA)"
	@echo "  Distribution dir: $(DIST_DIR_CUDA)"
	@echo ""
	@# Verify prerequisites
	@if [ ! -d "$(DIST_DIR_CUDA)" ]; then \
		echo "Error: CUDA variant not found at $(DIST_DIR_CUDA)"; \
		echo "Run 'make docker-prepare-cuda' first."; \
		exit 1; \
	fi
	@if [ ! -f "$(MODELS_DIR)/$(GGUF_MODEL_NAME)" ]; then \
		echo "Error: GGUF model not found. Run 'make docker-download-model' first."; \
		exit 1; \
	fi
	@# Build the image
	docker build \
		-f docker/Dockerfile.cuda \
		-t $(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CUDA) \
		-t $(DOCKER_FULL_IMAGE):$(VERSION)-cuda \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT_HASH) \
		--build-arg DIST_DIR=$(DIST_DIR_CUDA) \
		.
	@echo ""
	@echo "✓ Docker CUDA image built successfully!"
	@echo "  $(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CUDA)"
	@echo "  $(DOCKER_FULL_IMAGE):$(VERSION)-cuda"
	@echo ""
	@echo "Image size:"
	@docker images $(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CUDA) --format "  {{.Size}}"
	@echo ""
	@echo "Next step: make docker-login && make docker-push-cuda"
	@echo "Or run locally: make docker-run-cuda"

# Login to GitHub Container Registry
docker-login:
	@echo "Logging in to GitHub Container Registry..."
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "Error: GITHUB_TOKEN environment variable not set."; \
		echo ""; \
		echo "To create a token:"; \
		echo "  1. Go to https://github.com/settings/tokens"; \
		echo "  2. Create a token with 'write:packages' scope"; \
		echo "  3. Export it: export GITHUB_TOKEN=ghp_xxxx"; \
		echo ""; \
		exit 1; \
	fi
	@echo "$$GITHUB_TOKEN" | docker login $(DOCKER_REGISTRY) -u $(DOCKER_ORG) --password-stdin
	@echo "✓ Logged in to $(DOCKER_REGISTRY)"

# Push Docker CPU image to GitHub Container Registry
docker-push-cpu: docker-login
	@echo "Pushing CPU Docker image to GitHub Container Registry..."
	docker push $(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CPU)
	docker push $(DOCKER_FULL_IMAGE):$(VERSION)-cpu
	docker push $(DOCKER_FULL_IMAGE):latest
	@echo ""
	@echo "✓ CPU images pushed successfully!"
	@echo "  $(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CPU)"
	@echo "  $(DOCKER_FULL_IMAGE):$(VERSION)-cpu"
	@echo "  $(DOCKER_FULL_IMAGE):latest"
	@echo ""
	@echo "Pull with: docker pull $(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CPU)"

# Push Docker CUDA image to GitHub Container Registry
docker-push-cuda: docker-login
	@echo "Pushing CUDA Docker image to GitHub Container Registry..."
	docker push $(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CUDA)
	docker push $(DOCKER_FULL_IMAGE):$(VERSION)-cuda
	@echo ""
	@echo "✓ CUDA images pushed successfully!"
	@echo "  $(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CUDA)"
	@echo "  $(DOCKER_FULL_IMAGE):$(VERSION)-cuda"
	@echo ""
	@echo "Pull with: docker pull $(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CUDA)"

# Run Docker container - CPU version (no GPU required)
docker-run-cpu:
	@echo "Starting Remembrances-MCP container (CPU)..."
	@echo ""
	@echo "Configuration:"
	@echo "  Data path: $(DOCKER_DATA_PATH)"
	@echo "  Knowledge base: $(DOCKER_KB_PATH)"
	@echo "  Port: $(DOCKER_PORT)"
	@echo ""
	@# Create directories if they don't exist
	@mkdir -p $(DOCKER_DATA_PATH) $(DOCKER_KB_PATH)
	@# Run container
	docker run -d \
		--name remembrances-mcp-cpu \
		-p $(DOCKER_PORT):8080 \
		-v $(DOCKER_DATA_PATH):/data \
		-v $(DOCKER_KB_PATH):/knowledge-base \
		--restart unless-stopped \
		$(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CPU)
	@echo ""
	@echo "✓ Container started!"
	@echo "  Name: remembrances-mcp-cpu"
	@echo "  API: http://localhost:$(DOCKER_PORT)"
	@echo "  Data: $(DOCKER_DATA_PATH)"
	@echo "  Knowledge Base: $(DOCKER_KB_PATH)"
	@echo ""
	@echo "Commands:"
	@echo "  View logs: docker logs -f remembrances-mcp-cpu"
	@echo "  Stop: make docker-stop-cpu"
	@echo "  Shell: docker exec -it remembrances-mcp-cpu /bin/bash"

# Stop Docker CPU container
docker-stop-cpu:
	@echo "Stopping Remembrances-MCP CPU container..."
	-docker stop remembrances-mcp-cpu
	-docker rm remembrances-mcp-cpu
	@echo "✓ CPU container stopped and removed"

# Run Docker container - CUDA version (GPU accelerated)
docker-run-cuda:
	@echo "Starting Remembrances-MCP container (CUDA)..."
	@echo ""
	@echo "Configuration:"
	@echo "  Data path: $(DOCKER_DATA_PATH)"
	@echo "  Knowledge base: $(DOCKER_KB_PATH)"
	@echo "  Port: $(DOCKER_PORT)"
	@echo "  GPU: all"
	@echo ""
	@# Create directories if they don't exist
	@mkdir -p $(DOCKER_DATA_PATH) $(DOCKER_KB_PATH)
	@# Run container with GPU support
	docker run -d \
		--name remembrances-mcp-cuda \
		--gpus all \
		-p $(DOCKER_PORT):8080 \
		-v $(DOCKER_DATA_PATH):/data \
		-v $(DOCKER_KB_PATH):/knowledge-base \
		--restart unless-stopped \
		$(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CUDA)
	@echo ""
	@echo "✓ Container started with GPU support!"
	@echo "  Name: remembrances-mcp-cuda"
	@echo "  API: http://localhost:$(DOCKER_PORT)"
	@echo "  Data: $(DOCKER_DATA_PATH)"
	@echo "  Knowledge Base: $(DOCKER_KB_PATH)"
	@echo ""
	@echo "Commands:"
	@echo "  View logs: docker logs -f remembrances-mcp-cuda"
	@echo "  Stop: make docker-stop-cuda"
	@echo "  Shell: docker exec -it remembrances-mcp-cuda /bin/bash"

# Stop Docker CUDA container
docker-stop-cuda:
	@echo "Stopping Remembrances-MCP CUDA container..."
	-docker stop remembrances-mcp-cuda
	-docker rm remembrances-mcp-cuda
	@echo "✓ CUDA container stopped and removed"

# Show Docker container status
docker-status:
	@echo "Remembrances-MCP Container Status:"
	@echo ""
	@docker ps -a --filter "name=remembrances-mcp" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}\t{{.Image}}"
	@echo ""
	@echo "Recent logs (CPU container):"
	@docker logs --tail 5 remembrances-mcp-cpu 2>/dev/null || echo "  (CPU container not running)"
	@echo ""
	@echo "Recent logs (CUDA container):"
	@docker logs --tail 5 remembrances-mcp-cuda 2>/dev/null || echo "  (CUDA container not running)"

# Clean all Docker artifacts
docker-clean:
	@echo "Cleaning all Docker artifacts..."
	-docker stop remembrances-mcp-cpu 2>/dev/null || true
	-docker stop remembrances-mcp-cuda 2>/dev/null || true
	-docker rm remembrances-mcp-cpu 2>/dev/null || true
	-docker rm remembrances-mcp-cuda 2>/dev/null || true
	-docker rmi $(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CPU) 2>/dev/null || true
	-docker rmi $(DOCKER_FULL_IMAGE):$(DOCKER_TAG_CUDA) 2>/dev/null || true
	-docker rmi $(DOCKER_FULL_IMAGE):$(VERSION)-cpu 2>/dev/null || true
	-docker rmi $(DOCKER_FULL_IMAGE):$(VERSION)-cuda 2>/dev/null || true
	-docker rmi $(DOCKER_FULL_IMAGE):latest 2>/dev/null || true
	@echo "✓ Docker artifacts cleaned"
