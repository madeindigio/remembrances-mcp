# Cross-Compilation Guide for Remembrances-MCP

This guide explains how to build Remembrances-MCP for multiple platforms (Linux, macOS, Windows) with CGO support for llama.cpp embeddings.

## Table of Contents

- [Overview](#overview)
- [Requirements](#requirements)
- [Quick Start](#quick-start)
- [Building for Specific Platforms](#building-for-specific-platforms)
- [Multi-Platform Release](#multi-platform-release)
- [Troubleshooting](#troubleshooting)
- [Architecture Details](#architecture-details)

## Overview

Remembrances-MCP uses CGO to integrate with llama.cpp for local embedding generation. Cross-compiling CGO applications requires:

1. **Cross-compilers** for each target platform (CC/CXX)
2. **Static libraries** compiled for each target platform
3. **Proper build configuration** for each platform

We provide two approaches:

- **Option 1 (Recommended)**: Use `goreleaser-cross` Docker image with all cross-compilers pre-installed
- **Option 2**: Manual setup with native compilers on each platform

## Requirements

### For Docker-based Cross-Compilation (Recommended)

- **Docker** installed and running
- **8GB+ RAM** (for building llama.cpp)
- **10GB+ disk space** (for build artifacts)
- **Linux host** (amd64 or arm64)

### For Manual Building

- **Go 1.21+**
- **CMake 3.15+**
- **Platform-specific compilers**:
  - Linux: gcc, g++
  - macOS: clang, clang++ (via osxcross for cross-compilation)
  - Windows: mingw-w64

## Quick Start

### Build All Platforms at Once

```bash
# Build static libraries for all platforms and create releases
make release-multi-snapshot
```

This will:
1. Build llama.cpp static libraries for all platforms
2. Cross-compile Go binaries for all platforms
3. Create compressed archives
4. Generate checksums

**Output**: `dist/outputs/dist/` directory with binaries for:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

### Build Snapshot (No Git Tags Required)

```bash
# Test build without releasing
make release-multi-snapshot
```

### Create Actual Release

```bash
# Requires GITHUB_TOKEN environment variable
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxx
make release-multi
```

## Building for Specific Platforms

### Build Only Static Libraries

Build llama.cpp static library for a specific platform:

```bash
# Linux x86_64
make llama-deps-linux-amd64

# Linux ARM64
make llama-deps-linux-arm64

# macOS Intel
make llama-deps-darwin-amd64

# macOS Apple Silicon (M1/M2)
make llama-deps-darwin-arm64

# Windows x86_64
make llama-deps-windows-amd64

# All platforms
make llama-deps-all
```

**Output**: `go-llama.cpp/libbinding-<platform>.a`

### Build Go Binary for Specific Platform

After building the static library, you can build the Go binary:

```bash
# Using Docker (goreleaser-cross)
docker run --rm --privileged \
  -v $(pwd):/go/src/github.com/madeindigio/remembrances-mcp \
  -w /go/src/github.com/madeindigio/remembrances-mcp \
  ghcr.io/goreleaser/goreleaser-cross:v1.21 \
  --config .goreleaser-multiplatform.yml \
  build --single-target --snapshot --clean
```

## Multi-Platform Release

### Using Makefile (Recommended)

```bash
# Build snapshot (no Git tags/releases)
make release-multi-snapshot

# Create release (uploads to GitHub)
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxx
make release-multi
```

### Manual Docker Command

```bash
# 1. Build all static libraries
make llama-deps-all

# 2. Run goreleaser-cross
docker run --rm --privileged \
  -v $(pwd):/go/src/github.com/madeindigio/remembrances-mcp \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -w /go/src/github.com/madeindigio/remembrances-mcp \
  -e GITHUB_TOKEN \
  ghcr.io/goreleaser/goreleaser-cross:v1.21 \
  release --config .goreleaser-multiplatform.yml --snapshot --clean --skip=sign
```

### Without Docker (Native Build)

```bash
# Only builds for current platform
make build

# Or use standard goreleaser
make release
```

## Troubleshooting

### Docker Permission Issues

```bash
# Add your user to docker group
sudo usermod -aG docker $USER
newgrp docker

# Or run with sudo
sudo make release-multi-snapshot
```

### Build Failures

**Problem**: `libbinding-<platform>.a not found`

**Solution**: Build the static library first:
```bash
make llama-deps-<platform>
```

**Problem**: CMake errors about missing compilers

**Solution**: Ensure you're using the goreleaser-cross Docker image:
```bash
docker pull ghcr.io/goreleaser/goreleaser-cross:v1.21
```

**Problem**: Out of memory during llama.cpp build

**Solution**: 
- Increase Docker memory limit to 8GB+
- Reduce parallel build threads: Edit `build-static-multi.sh` and change `nproc` to a lower number

### Cross-Compiler Issues

**Problem**: Undefined symbols when linking

**Solution**: 
- Verify the correct toolchain is used (`CC`, `CXX`)
- Check that all object files are extracted from static archives
- Rebuild static library: `make clean && make llama-deps-<platform>`

### Platform-Specific Issues

#### macOS

**Issue**: Framework linking errors

**Solution**: Ensure `CGO_LDFLAGS` includes macOS frameworks:
```
-framework Accelerate -framework Foundation -framework Metal -framework MetalKit -framework MetalPerformanceShaders
```

#### Windows

**Issue**: DLL dependencies

**Solution**: Use static linking flag:
```
-extldflags=-static
```

#### Linux ARM64

**Issue**: `qemu-aarch64` errors in Docker

**Solution**: Install QEMU:
```bash
docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
```

## Architecture Details

### Directory Structure

```
remembrances-mcp/
├── .goreleaser.yml                    # Single-platform config (Linux only)
├── .goreleaser-multiplatform.yml      # Multi-platform config (all platforms)
├── Makefile                           # Build automation
├── go-llama.cpp/
│   ├── scripts/
│   │   ├── build-static.sh           # Single platform build
│   │   └── build-static-multi.sh     # Multi-platform build
│   ├── build/
│   │   ├── linux-amd64/              # Build artifacts per platform
│   │   ├── linux-arm64/
│   │   ├── darwin-amd64/
│   │   ├── darwin-arm64/
│   │   └── windows-amd64/
│   ├── libbinding-linux-amd64.a      # Static libraries per platform
│   ├── libbinding-linux-arm64.a
│   ├── libbinding-darwin-amd64.a
│   ├── libbinding-darwin-arm64.a
│   └── libbinding-windows-amd64.a
└── dist/outputs/dist/                # Final release artifacts
```

### Build Process Flow

```
1. Build llama.cpp static library for target platform
   ├── Configure CMake with target toolchain
   ├── Compile llama.cpp core
   ├── Compile binding.cpp
   └── Create combined static archive

2. Cross-compile Go binary
   ├── Set CGO_ENABLED=1
   ├── Set CC/CXX to target compilers
   ├── Link with platform-specific static library
   └── Apply platform-specific flags

3. Post-processing
   ├── Strip symbols (-s -w)
   ├── Compress with UPX (optional)
   └── Create archives (tar.gz/zip)
```

### Platform-Specific Compilers

| Platform       | Arch   | CC                        | CXX                         |
|----------------|--------|---------------------------|-----------------------------|
| Linux          | amd64  | x86_64-linux-gnu-gcc      | x86_64-linux-gnu-g++        |
| Linux          | arm64  | aarch64-linux-gnu-gcc     | aarch64-linux-gnu-g++       |
| macOS          | amd64  | o64-clang                 | o64-clang++                 |
| macOS          | arm64  | oa64-clang                | oa64-clang++                |
| Windows        | amd64  | x86_64-w64-mingw32-gcc    | x86_64-w64-mingw32-g++      |

### CGO Environment Variables

Each platform build requires specific CGO environment variables:

```bash
# Example for Linux ARM64
CGO_ENABLED=1
CC=aarch64-linux-gnu-gcc
CXX=aarch64-linux-gnu-g++
CGO_CFLAGS="-I./go-llama.cpp/llama.cpp -I./go-llama.cpp"
CGO_CXXFLAGS="-I./go-llama.cpp/llama.cpp -I./go-llama.cpp -I./go-llama.cpp/llama.cpp/common -I./go-llama.cpp/common"
CGO_LDFLAGS="-L./go-llama.cpp/build/linux-arm64 -lbinding -lm -lstdc++"
```

## Advanced Usage

### Custom Optimizations

Edit `go-llama.cpp/scripts/build-static-multi.sh` to customize compiler flags:

```bash
# Add AVX2 support for Intel CPUs
CFLAGS="$CFLAGS -mavx2 -mfma"

# Add NEON support for ARM
CFLAGS="$CFLAGS -mfpu=neon"

# Enable LTO
CFLAGS="$CFLAGS -flto"
LDFLAGS="$LDFLAGS -flto"
```

### GPU Support

To enable GPU acceleration in llama.cpp:

```bash
# CUDA support (NVIDIA)
cmake -DLLAMA_CUBLAS=ON ...

# Metal support (macOS)
cmake -DLLAMA_METAL=ON ...

# OpenCL support
cmake -DLLAMA_CLBLAST=ON ...
```

### Verify Binary

```bash
# Check binary architecture
file dist/outputs/dist/remembrances-mcp_linux_amd64_v1/remembrances-mcp

# Check dynamic dependencies
ldd dist/outputs/dist/remembrances-mcp_linux_amd64_v1/remembrances-mcp

# Test binary
./dist/outputs/dist/remembrances-mcp_linux_amd64_v1/remembrances-mcp --version
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
        with:
          submodules: recursive

      - name: Build all platforms
        run: make llama-deps-all

      - name: Run goreleaser
        uses: docker://ghcr.io/goreleaser/goreleaser-cross:v1.21
        with:
          args: release --config .goreleaser-multiplatform.yml --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Performance Considerations

### Build Times (Approximate)

| Platform       | CMake Config | Compilation | Total  |
|----------------|--------------|-------------|--------|
| Linux amd64    | 30s          | 3-5 min     | ~5 min |
| Linux arm64    | 30s          | 5-8 min     | ~8 min |
| macOS amd64    | 40s          | 4-6 min     | ~7 min |
| macOS arm64    | 40s          | 4-6 min     | ~7 min |
| Windows amd64  | 30s          | 4-6 min     | ~6 min |

**Total for all platforms**: ~30-40 minutes (parallel)

### Binary Sizes

| Platform       | Uncompressed | UPX Compressed |
|----------------|--------------|----------------|
| Linux amd64    | ~9 MB        | ~3 MB          |
| Linux arm64    | ~9 MB        | ~3 MB          |
| macOS amd64    | ~10 MB       | N/A*           |
| macOS arm64    | ~10 MB       | N/A*           |
| Windows amd64  | ~9 MB        | ~3 MB          |

*UPX not recommended for macOS due to codesigning issues

## References

- [goreleaser-cross](https://github.com/goreleaser/goreleaser-cross)
- [llama.cpp](https://github.com/ggerganov/llama.cpp)
- [GoReleaser CGO Cookbook](https://goreleaser.com/cookbooks/cgo-and-crosscompiling/)
- [osxcross](https://github.com/tpoechtrager/osxcross)

## Support

For issues or questions:
- GitHub Issues: https://github.com/madeindigio/remembrances-mcp/issues
- Documentation: https://github.com/madeindigio/remembrances-mcp/docs

## License

MIT License - See LICENSE file for details