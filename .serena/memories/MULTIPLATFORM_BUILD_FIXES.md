# Multiplatform Build System - Fix Documentation

**Date**: October 2024  
**Status**: ✅ RESOLVED  
**Platforms**: Linux (amd64, arm64), macOS (amd64, arm64)

## Overview

This document details the fixes applied to the multiplatform build system for Remembrances-MCP to enable successful cross-compilation using GoReleaser and Docker-based toolchains.

## Problem Statement

The `make release-multi-snapshot` command was failing with multiple errors:

1. GoReleaser version incompatibility
2. Interactive prompts blocking automated builds
3. Undefined `BUILD_NUMBER` in llama.cpp compilation
4. Missing Apple frameworks in cross-compilation environment
5. Incorrect archiver tools for macOS targets
6. CGO linking conflicts between platform-specific libraries
7. MinGW threading issues for Windows builds

## Fixes Applied

### 1. GoReleaser Version Compatibility

**Issue**: GoReleaser configuration files used `version: 2` but the Docker image `goreleaser-cross:v1.21` only supports `version: 1`.

**Fix**:
```diff
- version: 2
+ version: 1
```

**Files Modified**:
- `.goreleaser-multiplatform.yml`
- `.goreleaser-fast.yml`
- `.goreleaser.yml`

### 2. Non-Interactive Build Mode

**Issue**: Build script prompted for user confirmation, blocking automated CI/CD builds.

**Fix**: Added `SKIP_CONFIRM` environment variable support.

**File**: `scripts/release-multiplatform.sh`
```bash
# Check for non-interactive mode
SKIP_CONFIRM="${SKIP_CONFIRM:-false}"

# Confirm before proceeding (skip if SKIP_CONFIRM is set)
if [ "$SKIP_CONFIRM" != "true" ]; then
    read -p "Continue? (Y/n) " -n 1 -r
    # ... confirmation logic
else
    log_info "Skipping confirmation (SKIP_CONFIRM=true)"
fi
```

**Makefile Integration**:
```makefile
release-multi-snapshot:
	@chmod +x scripts/release-multiplatform.sh
	@SKIP_CONFIRM=true ./scripts/release-multiplatform.sh snapshot
```

### 3. BUILD_NUMBER Definition

**Issue**: llama.cpp compilation failed with `BUILD_NUMBER` undefined error.

**Fix**: Remove stale `build-info.h` and provide CMake definitions.

**File**: `go-llama.cpp/scripts/build-static-multi.sh`
```bash
# Fix BUILD_NUMBER and BUILD_COMMIT for build-info.h
export BUILD_NUMBER=0
export BUILD_COMMIT="unknown"

# Remove old build-info.h to force regeneration with correct BUILD_NUMBER
rm -f "llama.cpp/build-info.h"

# Configure with CMake
CMAKE_ARGS=(
    # ... other args
    "-DBUILD_NUMBER=0"
    "-DBUILD_COMMIT=unknown"
)
```

### 4. Apple Frameworks in Cross-Compilation

**Issue**: macOS frameworks (Accelerate, Foundation, Metal, etc.) not available in Linux Docker container.

**Fix**: Disabled Apple-specific features for cross-compilation builds.

**File**: `go-llama.cpp/scripts/build-static-multi.sh`
```bash
# Darwin platform configuration
CMAKE_SYSTEM_NAME="Darwin"
LDFLAGS="-pthread"
# Disable Apple frameworks for cross-compilation (not available in Docker)
DISABLE_APPLE_FRAMEWORKS="ON"

# Add platform-specific CMake arguments
if [ "$OS" = "darwin" ]; then
    # Disable Apple-specific features for cross-compilation
    if [ "$DISABLE_APPLE_FRAMEWORKS" = "ON" ]; then
        CMAKE_ARGS+=("-DLLAMA_ACCELERATE=OFF")
        CMAKE_ARGS+=("-DLLAMA_METAL=OFF")
    fi
fi
```

**File**: `.goreleaser-multiplatform.yml`
```yaml
# Darwin builds - removed framework dependencies
- CGO_LDFLAGS=-L./go-llama.cpp -lbinding-darwin-amd64 -lm -lstdc++
# Previously: ... -framework Accelerate -framework Foundation -framework Metal ...
```

### 5. Platform-Specific Archiver Tools

**Issue**: Used Linux `ar` command for all platforms, causing incompatible archive formats for macOS.

**Fix**: Use platform-specific archiver tools from cross-compilation toolchains.

**File**: `go-llama.cpp/scripts/build-static-multi.sh`
```bash
case "$OS" in
    "linux")
        case "$ARCH" in
            "amd64")
                AR="x86_64-linux-gnu-ar"
                ;;
            "arm64")
                AR="aarch64-linux-gnu-ar"
                ;;
        esac
        ;;
    "darwin")
        case "$ARCH" in
            "amd64")
                AR="x86_64-apple-darwin21.1-ar"  # osxcross toolchain
                ;;
            "arm64")
                AR="aarch64-apple-darwin21.1-ar"
                ;;
        esac
        ;;
esac

# Use platform-specific AR throughout the script
$AR x ../libllama.a
$AR rcs "$OUTPUT_LIB" "$TEMP_DIR"/*.o
```

### 6. CGO Linking Configuration

**Issue**: Hardcoded `-lbinding` in `llama.go` conflicted with platform-specific library flags from GoReleaser.

**Fix**: Removed hardcoded `-lbinding` from CGO pragmas; now provided by GoReleaser per-platform.

**File**: `go-llama.cpp/llama.go`
```go
// NOTE: -lbinding is omitted from CGO LDFLAGS and provided by GoReleaser/Makefile per-platform
// (e.g., -lbinding-linux-amd64). For local builds, set CGO_LDFLAGS environment variable:
// CGO_LDFLAGS="-L./go-llama.cpp -lbinding -lm -lstdc++"

// #cgo CXXFLAGS: -I${SRCDIR}/llama.cpp/common -I${SRCDIR}/llama.cpp
// #cgo LDFLAGS: -L${SRCDIR}/ -lm -lstdc++
// Previously: // #cgo LDFLAGS: -L${SRCDIR}/ -lbinding -lm -lstdc++
```

**Note**: GoReleaser provides platform-specific flags via environment variables:
- `CGO_LDFLAGS=-L./go-llama.cpp -lbinding-linux-amd64 -lm -lstdc++`
- `CGO_LDFLAGS=-L./go-llama.cpp -lbinding-darwin-arm64 -lm -lstdc++`
- etc.

### 7. C++ Standard Upgrade

**Issue**: MinGW threading support incomplete with C++11.

**Fix**: Upgraded to C++14 for better threading compatibility.

**File**: `go-llama.cpp/scripts/build-static-multi.sh`
```bash
# All platforms now use C++14
CXXFLAGS="-O3 -DNDEBUG -std=c++14 -fPIC ..."
# Previously: -std=c++11
```

### 8. Windows Build Status

**Issue**: MinGW in Docker container lacks complete POSIX threads support, causing compilation errors in llama.cpp threading code.

**Status**: ⚠️ **DISABLED TEMPORARILY**

**File**: `.goreleaser-multiplatform.yml`
```yaml
# Windows AMD64 - DISABLED: MinGW threading issues with llama.cpp
# TODO: Re-enable after fixing threading support or upgrading to newer llama.cpp
# - id: windows-amd64
#   main: ./cmd/remembrances-mcp
#   ...
```

**Error Details**:
```
error: 'std::mutex' is defined in header '<mutex>'; did you forget to '#include <mutex>'?
error: '__gthread_t' does not name a type
```

**Future Options**:
1. Update llama.cpp to version with better MinGW compatibility
2. Use different threading library (e.g., winpthreads)
3. Build Windows binaries natively on Windows
4. Use MSVC-based toolchain instead of MinGW

### 9. Git Safe Directory Configuration

**Issue**: Git ownership warnings in Docker container.

**Fix**: Configure git safe directories in build script.

**File**: `go-llama.cpp/scripts/build-static-multi.sh`
```bash
# Fix git ownership issues in Docker
if [ -d "./llama.cpp/.git" ]; then
    git config --global --add safe.directory "$(pwd)/llama.cpp" 2>/dev/null || true
    git config --global --add safe.directory "/go/src/github.com/madeindigio/remembrances-mcp/go-llama.cpp/llama.cpp" 2>/dev/null || true
fi
git config --global --add safe.directory "$(pwd)" 2>/dev/null || true
```

### 10. Missing LICENSE File

**Issue**: GoReleaser archive step failed looking for `LICENSE` file.

**Fix**: Created symlink from existing `LICENSE.txt`.

```bash
ln -sf LICENSE.txt LICENSE
```

## Build Results

### ✅ Successfully Built Platforms

| Platform | Architecture | Binary Size | Status |
|----------|-------------|-------------|---------|
| Linux | x86_64 (amd64) | 3.8M | ✅ Working |
| Linux | ARM64 | 3.6M | ✅ Working |
| macOS | x86_64 (Intel) | 3.9M | ✅ Working |
| macOS | ARM64 (Apple Silicon) | 3.6M | ✅ Working |
| Windows | x86_64 (amd64) | - | ⚠️ Disabled |

### Build Artifacts

Artifacts are generated in: `dist/outputs/dist/`

Example output:
```
remembrances-mcp_0.30.3-SNAPSHOT-116e4f8_Darwin_arm64.tar.gz
remembrances-mcp_0.30.3-SNAPSHOT-116e4f8_Darwin_x86_64.tar.gz
remembrances-mcp_0.30.3-SNAPSHOT-116e4f8_Linux_arm64.tar.gz
remembrances-mcp_0.30.3-SNAPSHOT-116e4f8_Linux_x86_64.tar.gz
checksums.txt
```

## Usage

### Snapshot Build (No GitHub Release)

```bash
make release-multi-snapshot
```

### Full Release Build (with GitHub Release)

```bash
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxx
make release-multi
```

### Fast Build (Pre-built Libraries)

```bash
# Pre-compile all platform libraries in parallel
make llama-deps-all-parallel

# Use pre-built libraries for faster build
make release-multi-snapshot-fast
```

### Manual Docker Build

```bash
docker run --rm --network=host \
  -v $(pwd):/go/src/github.com/madeindigio/remembrances-mcp \
  -w /go/src/github.com/madeindigio/remembrances-mcp \
  -e GIT_CONFIG_GLOBAL=/tmp/.gitconfig \
  ghcr.io/goreleaser/goreleaser-cross:v1.21 \
  release --config .goreleaser-multiplatform.yml --clean --snapshot --skip=sign
```

## Build Time

- **Standard build** (libraries compiled in Docker): ~2-3 minutes
- **Fast build** (pre-built libraries): ~30-60 seconds
- **Library pre-compilation** (parallel): ~1-2 minutes

## Known Limitations

1. **Windows builds disabled**: MinGW threading issues require resolution
2. **Apple frameworks unavailable**: Cross-compiled macOS binaries lack hardware acceleration (Accelerate, Metal)
3. **Local builds**: Require manual `CGO_LDFLAGS` configuration without GoReleaser

## Local Development

For local builds without GoReleaser:

```bash
# Set CGO flags manually
export CGO_LDFLAGS="-L./go-llama.cpp -lbinding -lm -lstdc++"

# Build llama.cpp library for current platform
cd go-llama.cpp
./scripts/build-static-multi.sh $(uname -s | tr '[:upper:]' '[:lower:]') $(uname -m)
cd ..

# Build Go binary
go build -o remembrances-mcp ./cmd/remembrances-mcp
```

## Testing

Verify builds work correctly:

```bash
# Extract and test each platform binary
tar xzf dist/outputs/dist/remembrances-mcp_*_Linux_x86_64.tar.gz
./remembrances-mcp --version

# Verify checksums
cd dist/outputs/dist
sha256sum -c checksums.txt
```

## References

- **GoReleaser Documentation**: https://goreleaser.com/
- **goreleaser-cross**: https://github.com/goreleaser/goreleaser-cross
- **llama.cpp**: https://github.com/ggerganov/llama.cpp
- **osxcross**: https://github.com/tpoechtrager/osxcross

## Future Improvements

1. **Enable Windows builds**: Investigate alternative threading solutions or upgrade llama.cpp
2. **Native builds**: Consider platform-specific GitHub Actions runners for native compilation
3. **Cache optimization**: Implement intelligent library caching system
4. **ARM optimization**: Enable NEON/SIMD optimizations for ARM builds
5. **Binary signing**: Implement code signing for macOS and Windows binaries
6. **SBOM generation**: Add Software Bill of Materials for security compliance

## Troubleshooting

### Build fails with "cannot find -lbinding"

**Cause**: Missing platform-specific library or symlink issue.

**Solution**:
```bash
# Check if libraries were built
ls -la go-llama.cpp/libbinding-*.a

# Rebuild libraries
make llama-deps-all-parallel
```

### Docker permission errors

**Cause**: File ownership conflicts between host and container.

**Solution**:
```bash
# Fix permissions
chmod -R u+w go-llama.cpp/build dist
```

### "dubious ownership in repository" warnings

**Cause**: Git security check in Docker container.

**Solution**: These warnings are handled automatically by the build script and can be safely ignored.

### GoReleaser version mismatch

**Ensure**: All `.goreleaser*.yml` files use `version: 1`

```bash
grep "^version:" .goreleaser*.yml
```

---

**Last Updated**: October 24, 2024  
**Maintained By**: Development Team  
**Status**: Production Ready (4/5 platforms)