# Building remembrances-mcp

This document provides comprehensive instructions for building remembrances-mcp with both remote-only and embedded SurrealDB support.

## Build Modes

remembrances-mcp supports two build modes:

1. **Remote Mode** (Default): Connects to external SurrealDB server
   - ✅ No CGO required
   - ✅ Easy cross-compilation
   - ✅ Smaller binary size
   - ❌ Requires external SurrealDB server

2. **Embedded Mode**: Includes embedded SurrealDB database
   - ✅ No external server needed
   - ✅ Portable single binary
   - ❌ Requires CGO and Rust toolchain
   - ❌ Larger binary size
   - ❌ More complex build process

## Quick Start

### Building Remote Version

The simplest way to build for remote SurrealDB only:

```bash
make build
# or
CGO_ENABLED=0 go build -o bin/remembrances-mcp ./cmd/remembrances-mcp
```

### Building Embedded Version

Requires Rust toolchain and CGO:

```bash
# Install Rust if not already installed
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Build embedded version
make build-embedded
```

## Prerequisites

### For Remote Build
- Go 1.23 or later
- That's it!

### For Embedded Build
- Go 1.23 or later
- Rust toolchain (install from https://rustup.rs/)
- C compiler (gcc/clang)
- System dependencies:
  - **Ubuntu/Debian**: `build-essential clang libclang-dev llvm-dev pkg-config libssl-dev`
  - **macOS**: Xcode Command Line Tools + `brew install llvm`
  - **Windows**: Not officially supported yet

## Detailed Build Instructions

### 1. Remote Build (CGO_ENABLED=0)

This is the default and recommended method for most users:

```bash
# Using make
make build

# Or manually
CGO_ENABLED=0 go build \
  -ldflags="-s -w" \
  -o bin/remembrances-mcp \
  ./cmd/remembrances-mcp

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/remembrances-mcp-linux-amd64 ./cmd/remembrances-mcp
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o bin/remembrances-mcp-darwin-arm64 ./cmd/remembrances-mcp
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/remembrances-mcp-windows-amd64.exe ./cmd/remembrances-mcp
```

### 2. Embedded Build (CGO_ENABLED=1)

#### Step 1: Install Dependencies

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install -y build-essential clang libclang-dev llvm-dev pkg-config libssl-dev
```

**macOS:**
```bash
# Install Xcode Command Line Tools
xcode-select --install

# Install LLVM
brew install llvm

# Set LIBCLANG_PATH
export LIBCLANG_PATH="$(brew --prefix llvm)/lib"
```

**Install Rust:**
```bash
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source $HOME/.cargo/env
```

#### Step 2: Build Rust Library

```bash
# Clone and build the embedded library
./build-embedded.sh native
```

#### Step 3: Build Go Binary

```bash
# Using make
make build-embedded

# Or manually
CGO_ENABLED=1 go build \
  -tags embedded \
  -ldflags="-s -w" \
  -o bin/remembrances-mcp-embedded \
  ./cmd/remembrances-mcp
```

### 3. Cross-Compilation with Docker

For building embedded versions for multiple platforms:

```bash
# Build Docker image with all cross-compilation tools
docker build -t remembrances-mcp-builder -f Dockerfile.crossbuild .

# Build binaries in Docker
docker run --rm -v $(pwd):/workspace remembrances-mcp-builder \
  bash -c "make build-embedded"
```

### 4. Release Build with GoReleaser

Create production releases for all supported platforms:

```bash
# Install goreleaser
go install github.com/goreleaser/goreleaser@latest

# Create a release (requires git tag)
git tag -a v1.0.0 -m "Release v1.0.0"
make release

# Or create a snapshot release without git tag
make snapshot
```

GoReleaser will create:
- Remote binaries: `remembrances-mcp` for Linux, macOS, Windows (amd64, arm64)
- Embedded binaries: `remembrances-mcp-embedded` for Linux, macOS (amd64, arm64)

## Cross-Compilation Details

### Linux ARM64 from Linux x64

```bash
# Install cross-compiler
sudo apt-get install gcc-aarch64-linux-gnu g++-aarch64-linux-gnu

# Add Rust target
rustup target add aarch64-unknown-linux-gnu

# Build Rust library
./build-embedded.sh linux-arm64

# Build Go binary
CC=aarch64-linux-gnu-gcc \
CXX=aarch64-linux-gnu-g++ \
GOOS=linux GOARCH=arm64 CGO_ENABLED=1 \
go build -tags embedded -o bin/remembrances-mcp-linux-arm64 ./cmd/remembrances-mcp
```

### macOS from Linux (OSXCross)

OSXCross setup is complex and is best done using the provided Docker image:

```bash
docker build -t remembrances-mcp-builder -f Dockerfile.crossbuild .
docker run --rm -v $(pwd):/workspace remembrances-mcp-builder \
  bash -c "./build-embedded.sh darwin-amd64 && \
           make build-embedded GOOS=darwin GOARCH=amd64"
```

## Testing

```bash
# Test remote version
make test

# Test embedded version
make test-embedded

# Run specific test
go test -v ./internal/storage/...
```

## Usage

### Remote Version

```bash
# Start with remote SurrealDB
./bin/remembrances-mcp \
  --surrealdb-url ws://localhost:8000/rpc \
  --surrealdb-user root \
  --surrealdb-pass root
```

### Embedded Version

```bash
# Use embedded database
./bin/remembrances-mcp-embedded \
  --use-embedded-db \
  --db-path ./data/remembrances.db
```

## Troubleshooting

### CGO Errors

If you see errors like "CGO not enabled" or "C compiler not found":

```bash
# Ensure CGO is enabled
export CGO_ENABLED=1

# Ensure compiler is in PATH
which gcc  # or clang
```

### Rust Build Errors

If Rust build fails:

```bash
# Update Rust toolchain
rustup update

# Clean and rebuild
cd vendor/surrealdb-embedded-rust/surrealdb_embedded_rs
cargo clean
cargo build --release
```

### macOS LIBCLANG_PATH Issues

```bash
# Set LIBCLANG_PATH for LLVM
export LIBCLANG_PATH="$(brew --prefix llvm)/lib"

# Or for Xcode
export LIBCLANG_PATH=/Library/Developer/CommandLineTools/usr/lib
```

### Static Linking Errors

If you encounter linking errors with embedded builds:

```bash
# Try building without static linking
CGO_ENABLED=1 go build -tags embedded ./cmd/remembrances-mcp
```

## Binary Sizes

Typical binary sizes:

- **Remote version** (CGO_ENABLED=0): 15-25 MB (after UPX compression: 5-8 MB)
- **Embedded version** (CGO_ENABLED=1): 50-80 MB (static linking included)

## Performance Considerations

- Remote version has lower memory footprint but requires network overhead
- Embedded version has higher memory usage but eliminates network latency
- Embedded version provides better data locality and portability

## Contributing

When submitting PRs that modify build configuration:
1. Test both remote and embedded builds
2. Test on at least Linux and macOS if possible
3. Update this BUILD.md if adding new build options
4. Ensure `make test` and `make test-embedded` pass

## Support

For build issues:
1. Check this BUILD.md first
2. Search existing issues: https://github.com/madeindigio/remembrances-mcp/issues
3. Open a new issue with:
   - Your OS and architecture
   - Go version (`go version`)
   - Rust version (`cargo --version`)
   - Full error output
