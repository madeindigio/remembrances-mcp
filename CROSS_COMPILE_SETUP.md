# Cross-Compilation Setup Summary

## Date: 2025-11-17

This document summarizes all changes made to enable cross-compilation for the `remembrances-mcp` project.

## Identified Issues

During the initial execution of `./scripts/release-cross.sh`, the following issues were encountered:

### 1. Library Compilation Script
- **Issue**: The command `bash scripts/build-libs-cross.sh` failed because goreleaser-cross did not recognize the command.
- **Solution**: Added `--entrypoint /bin/bash` to the docker run command in `build_shared_libraries()`.

### 2. Duplicate Replace Directives in go.mod
- **Issue**: Two `replace` directives pointed to the same directory, causing a Go error:
```
replace github.com/madeindigio/go-llama.cpp => /www/MCP/Remembrances/go-llama.cpp
replace github.com/go-skynet/go-llama.cpp => /www/MCP/Remembrances/go-llama.cpp
```
- **Solution**: Removed the duplicate directive for `go-skynet/go-llama.cpp`.

### 3. Docker Volumes Not Mounted
- **Issue**: GoReleaser could not access `/www/MCP/Remembrances/` for local modules.
- **Solution**: Added volume mount in `run_goreleaser()`:
```bash
-v "/www/MCP/Remembrances:/www/MCP/Remembrances"
```

### 4. CURL Dependency in llama.cpp
- **Issue**: CMake required CURL, which was not available in the container.
- **Solution**: Disabled CURL in CMake flags:
```bash
-DLLAMA_CURL=OFF
```

### 5. Outdated Vendor Directory
- **Issue**: The vendor directory was not synchronized with go.mod.
- **Solution**: Added `go mod vendor` to the before hooks in `.goreleaser.yml`.

### 6. Rust/Cargo Not Available
- **Issue**: The goreleaser-cross container did not have Rust installed to compile surrealdb-embedded.
- **Solution**: Created a custom Docker image with Rust.

### 7. Missing macOS Tools
- **Issue**: `install_name_tool` not available for macOS cross-compilation.
- **Status**: Pending - the custom image should resolve this with osxcross.

## Created Files

### 1. `docker/Dockerfile.goreleaser-custom`
Custom Dockerfile extending `goreleaser-cross:v1.23` with:
- Rust 1.75.0 and rustup
- Rust cross-compilation targets
- CMake and build tools
- libcurl for compiling llama.cpp

### 2. `scripts/build-docker-image.sh`
Script to build the custom Docker image with options for:
- Specifying a custom tag
- Building without cache
- Pushing to a registry

### 3. `docs/CROSS_COMPILE.md`
Comprehensive documentation on:
- How to use cross-compilation
- Script descriptions
- Environment variables
- Troubleshooting
- CI/CD integration

## Modified Files

### 1. `scripts/release-cross.sh`
**Changes:**
- Added `GORELEASER_CROSS_IMAGE` variable to use custom image
- Updated `build_shared_libraries()` with `--entrypoint /bin/bash`
- Updated `run_goreleaser()` to mount `/www/MCP/Remembrances` volume
- All Docker image references now use `${GORELEASER_CROSS_IMAGE}`
- Added fault tolerance in library compilation

### 2. `scripts/build-libs-cross.sh`
**Changes:**
- Added `-DLLAMA_CURL=OFF` flag in cmake_flags for all platforms

### 3. `go.mod`
**Changes:**
- Removed `replace github.com/go-skynet/go-llama.cpp => /www/MCP/Remembrances/go-llama.cpp` directive

### 4. `.goreleaser.yml`
**Changes:**
- Added `go mod vendor` to before hooks

## Usage

### Option 1: With Custom Image (Recommended)

```bash
# 1. Build the custom image
./scripts/build-docker-image.sh

# 2. Run cross-compilation
export GORELEASER_CROSS_IMAGE=remembrances-mcp-builder:latest
./scripts/release-cross.sh --clean snapshot
```

### Option 2: Without Shared Libraries (Faster)

```bash
# Only build Go binaries without shared libraries
./scripts/release-cross.sh --skip-libs --clean snapshot
```

### Option 3: Libraries Only

```bash
# Only build the shared libraries
export GORELEASER_CROSS_IMAGE=remembrances-mcp-builder:latest
./scripts/release-cross.sh --libs-only
```

## Environment Variables

```bash
# Specify custom Docker image
export GORELEASER_CROSS_IMAGE=remembrances-mcp-builder:v1.23-rust

# Or use a specific goreleaser-cross version
export GORELEASER_CROSS_VERSION=v1.22

# For GitHub releases
export GITHUB_TOKEN=your_token_here
```

## Output Structure

```
dist/
├── libs/                           # Shared libraries per platform
│   ├── linux-amd64/
│   │   ├── libllama.so
│   │   ├── libggml.so
│   │   └── libsurrealdb_embedded_rs.so
│   ├── linux-arm64/
│   ├── darwin-amd64/
│   ├── darwin-arm64/
│   ├── windows-amd64/
│   └── windows-arm64/
└── outputs/
    └── dist/                       # Release files
        ├── remembrances-mcp_VERSION_linux_amd64.tar.gz
        ├── remembrances-mcp_VERSION_linux_arm64.tar.gz
        ├── remembrances-mcp_VERSION_darwin_amd64.tar.gz
        ├── remembrances-mcp_VERSION_darwin_arm64.tar.gz
        ├── remembrances-mcp_VERSION_windows_amd64.zip
        ├── remembrances-mcp_VERSION_windows_arm64.zip
        └── checksums.txt
```

## Supported Platforms

- ✅ Linux AMD64
- ✅ Linux ARM64
- ⚠️  macOS AMD64 (requires configured osxcross)
- ⚠️  macOS ARM64 (requires configured osxcross)
- ⚠️  Windows AMD64 (basic compilation works, libraries pending)
- ⚠️  Windows ARM64 (experimental support)

## Next Steps

1. **Test custom Docker image** - Verify Rust and all tools work
2. **Build shared libraries** - Run `--libs-only` to check
3. **Full compilation** - Run full snapshot for all platforms
4. **Validate binaries** - Test binaries on each platform
5. **CI/CD** - Integrate into GitHub Actions for automated builds

## References

- [GoReleaser Cross](https://github.com/goreleaser/goreleaser-cross)
- [Rust Linux Darwin Builder](https://github.com/joseluisq/rust-linux-darwin-builder)
- [llama.cpp](https://github.com/ggerganov/llama.cpp)
- [SurrealDB Embedded](https://surrealdb.com/)