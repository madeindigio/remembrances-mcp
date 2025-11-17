# Cross-Compilation Guide for remembrances-mcp

This guide explains how to cross-compile `remembrances-mcp` for multiple platforms (Linux, macOS, Windows) using Docker.

## Overview

The project uses a custom Docker image based on `goreleaser-cross` with additional tools for:
- **Rust cross-compilation** for building `surrealdb-embedded`
- **CMake and build tools** for compiling `llama.cpp` shared libraries
- **Cross-compilation toolchains** for multiple architectures

## Prerequisites

- Docker installed and running
- Sufficient disk space (~10GB for Docker image and build artifacts)
- Access to `/www/MCP/Remembrances/` directory with:
  - `go-llama.cpp` module
  - `surrealdb-embedded` module

## Quick Start

### 1. Build the Custom Docker Image

First, build the custom Docker image with Rust and build tools:

```bash
./scripts/build-docker-image.sh
```

This creates an image named `remembrances-mcp-builder:v1.23-rust` with all necessary tools.

### 2. Run Cross-Compilation

Use the custom image to build for all platforms:

```bash
export GORELEASER_CROSS_IMAGE=remembrances-mcp-builder:latest
./scripts/release-cross.sh --clean snapshot
```

Or skip building shared libraries (faster, but binaries won't have embedded functionality):

```bash
export GORELEASER_CROSS_IMAGE=remembrances-mcp-builder:latest
./scripts/release-cross.sh --skip-libs --clean snapshot
```

## Build Scripts

### `scripts/build-docker-image.sh`

Builds the custom Docker image with Rust and build tools.

**Options:**
- `-t, --tag TAG`: Specify image tag (default: v1.23-rust)
- `-n, --name NAME`: Specify image name (default: remembrances-mcp-builder)
- `--no-cache`: Build without cache
- `--push`: Push to registry after building

**Examples:**
```bash
# Build with default settings
./scripts/build-docker-image.sh

# Build with custom tag
./scripts/build-docker-image.sh --tag latest

# Build without cache
./scripts/build-docker-image.sh --no-cache
```

### `scripts/release-cross.sh`

Main script for cross-compiling the project.

**Commands:**
- `build`: Build binaries without releasing
- `snapshot`: Build snapshot release (no tag required)
- `release`: Build and create GitHub release

**Options:**
- `-c, --clean`: Clean before building
- `--skip-libs`: Skip building shared libraries
- `--libs-only`: Only build libraries, skip goreleaser
- `-v, --version VERSION`: Specify goreleaser-cross version

**Examples:**
```bash
# Build snapshot with custom image
export GORELEASER_CROSS_IMAGE=remembrances-mcp-builder:latest
./scripts/release-cross.sh snapshot

# Build without libraries (faster)
./scripts/release-cross.sh --skip-libs snapshot

# Only build shared libraries
./scripts/release-cross.sh --libs-only
```

### `scripts/build-libs-cross.sh`

Script that runs inside Docker to build shared libraries for all platforms.

This script:
1. Builds `llama.cpp` using CMake for each platform
2. Builds `surrealdb-embedded` using Cargo for each platform
3. Places libraries in `dist/libs/{platform}-{arch}/`

**Supported platforms:**
- `linux-amd64`
- `linux-arm64`
- `darwin-amd64` (macOS Intel)
- `darwin-arm64` (macOS Apple Silicon)
- `windows-amd64`
- `windows-arm64`

## Environment Variables

- `GORELEASER_CROSS_IMAGE`: Docker image to use (default: `ghcr.io/goreleaser/goreleaser-cross:v1.23`)
- `GORELEASER_CROSS_VERSION`: Version tag for default image (default: `v1.23`)
- `GITHUB_TOKEN`: GitHub token for releases
- `PROJECT_ROOT`: Project root directory (auto-detected)
- `LLAMA_CPP_DIR`: Path to go-llama.cpp (default: `/www/MCP/Remembrances/go-llama.cpp`)
- `SURREALDB_DIR`: Path to surrealdb-embedded (default: `/www/MCP/Remembrances/surrealdb-embedded`)

## Build Output

Build artifacts are placed in:
```
dist/
├── libs/                           # Shared libraries by platform
│   ├── linux-amd64/
│   ├── linux-arm64/
│   ├── darwin-amd64/
│   ├── darwin-arm64/
│   ├── windows-amd64/
│   └── windows-arm64/
└── outputs/
    └── dist/                       # Final release archives
        ├── remembrances-mcp_VERSION_linux_amd64.tar.gz
        ├── remembrances-mcp_VERSION_linux_arm64.tar.gz
        ├── remembrances-mcp_VERSION_darwin_amd64.tar.gz
        ├── remembrances-mcp_VERSION_darwin_arm64.tar.gz
        ├── remembrances-mcp_VERSION_windows_amd64.zip
        └── remembrances-mcp_VERSION_windows_arm64.zip
```

## Troubleshooting

### Docker Image Build Fails

If the Docker image build fails:
1. Ensure you have enough disk space
2. Try building without cache: `./scripts/build-docker-image.sh --no-cache`
3. Check Docker daemon is running: `docker info`

### Shared Library Build Fails

Common issues:
- **Rust not found**: Rebuild Docker image with `--no-cache`
- **CMake errors**: Check that llama.cpp submodules are initialized
- **Permission errors**: Ensure volumes are mounted with correct permissions

### GoReleaser Fails

If goreleaser fails during build:
1. Try skipping libraries first: `--skip-libs`
2. Check `go.mod` replace directives point to correct paths
3. Ensure vendor directory is synced: `go mod vendor`

## Custom Docker Image Details

The custom image (`Dockerfile.goreleaser-custom`) extends `goreleaser-cross:v1.23` with:

**Added Tools:**
- Rust 1.75.0 with rustup
- Cargo with cross-compilation targets
- CMake 3.x and Ninja build system
- libcurl development headers

**Rust Targets:**
- `x86_64-unknown-linux-gnu`
- `aarch64-unknown-linux-gnu`
- `x86_64-apple-darwin`
- `aarch64-apple-darwin`
- `x86_64-pc-windows-gnu`
- `aarch64-pc-windows-gnu`

## CI/CD Integration

For GitHub Actions or other CI systems:

```yaml
- name: Build custom Docker image
  run: ./scripts/build-docker-image.sh --tag ci-build

- name: Cross-compile release
  env:
    GORELEASER_CROSS_IMAGE: remembrances-mcp-builder:ci-build
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  run: ./scripts/release-cross.sh release
```

## Known Limitations

1. **macOS ARM64**: Requires osxcross with proper SDK
2. **Windows ARM64**: Limited testing, may need additional configuration
3. **Shared Libraries**: Some platforms may have missing dependencies

## Additional Resources

- [GoReleaser Documentation](https://goreleaser.com/)
- [goreleaser-cross GitHub](https://github.com/goreleaser/goreleaser-cross)
- [Rust Cross-Compilation Guide](https://rust-lang.github.io/rustup/cross-compilation.html)
