# Makefile macOS Cross-Compilation Fixes

**Date:** November 25, 2025  
**Status:** Completed ✅

## Summary

Fixed the Makefile and build system for macOS to support cross-compilation between arm64 (Apple Silicon) and x86_64 (Intel) architectures, including proper library linking for llama.cpp and surrealdb-embedded, and correct RPATH configuration.

## Problems Fixed

### 1. Home Directory Expansion (`~`)
**Problem:** The Makefile used `~` for paths which doesn't expand correctly in Make context.

**Solution:** Changed all `~` references to `$(HOME)`:
```makefile
# Before
GO_LLAMA_DIR := ~/www/MCP/Remembrances/go-llama.cpp

# After
GO_LLAMA_DIR ?= $(HOME)/www/MCP/Remembrances/go-llama.cpp
```

### 2. go.mod Replace Directives
**Problem:** Go doesn't expand `~` in `go.mod` replace directives.

**Solution:** Used absolute paths in `go.mod`:
```go
// Before
replace github.com/madeindigio/go-llama.cpp => ~/www/MCP/Remembrances/go-llama.cpp

// After
replace github.com/madeindigio/go-llama.cpp => /Users/digio/www/MCP/Remembrances/go-llama.cpp
```

### 3. Missing `NewFromURL` Function
**Problem:** `surrealdb.go` called `embedded.NewFromURL()` which didn't exist in the surrealdb-embedded library.

**Solution:** Added `NewFromURL()` function to `/Users/digio/www/MCP/Remembrances/surrealdb-embedded/surrealdb.go`:
- Supports URL schemes: `memory://`, `rocksdb://path`, `surrealkv://path`, `file://path`
- Automatically expands `~` in paths
- Parses URL format correctly

### 4. llama.cpp Submodule Detection
**Problem:** Check looked for wrong path structure.

**Solution:** Updated to check for `CMakeLists.txt`:
```makefile
# Before
@if [ ! -d "$(GO_LLAMA_DIR)/llama.cpp" ]; then

# After
@if [ ! -f "$(GO_LLAMA_DIR)/llama.cpp/CMakeLists.txt" ]; then
```

### 5. Cross-Compilation for x86_64 from arm64
**Problem:** CMake detected host CPU (apple-m3) when cross-compiling for x86_64.

**Solution:** Added `GGML_NATIVE=OFF` flag and architecture-specific settings:
```makefile
cmake -B build-$(TARGET_ARCH) llama.cpp \
    -DLLAMA_STATIC=OFF \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_OSX_ARCHITECTURES=x86_64 \
    -DCMAKE_C_FLAGS="-arch x86_64" \
    -DCMAKE_CXX_FLAGS="-arch x86_64" \
    -DGGML_NATIVE=OFF \
    -DLLAMA_METAL=OFF
```

### 6. Library Linking for Cross-Compilation
**Problem:** CGO tried to link arm64 libraries when building for x86_64.

**Solution:** Created architecture-specific distribution targets with correct library paths:
```makefile
dist-darwin-amd64:
    CGO_LDFLAGS="-L$(GO_LLAMA_DIR)/build-x86_64/bin -L$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/x86_64-apple-darwin/release ..."
```

### 7. RPATH Configuration for macOS
**Problem:** Binary couldn't find shared libraries at runtime. Two issues:
1. Go's `-ldflags="-r $ORIGIN"` doesn't work on macOS (Linux syntax)
2. `surrealdb_embedded_rs.dylib` was referenced with an absolute path

**Error message:**
```
dyld[36945]: Library not loaded: @rpath/libllama.dylib
  Referenced from: .../remembrances-mcp
  Reason: tried: '$ORIGIN/libllama.dylib' (no such file)
```

**Solution:** Use `install_name_tool` post-build to fix library references:
```makefile
# Add @executable_path to RPATH
install_name_tool -add_rpath @executable_path "$(BUILD_DIR)/$(BINARY_NAME)"

# Fix absolute paths to use @rpath
install_name_tool -change "/absolute/path/libsurrealdb_embedded_rs.dylib" \
    "@rpath/libsurrealdb_embedded_rs.dylib" "$(BUILD_DIR)/$(BINARY_NAME)"
```

**Result:** All libraries now reference `@rpath/libname.dylib` and the binary has `@executable_path` in its RPATH, so it finds libraries in the same directory.

## New Makefile Targets

### macOS Cross-Compilation
- `make build-darwin-arm64` - Build libraries for Apple Silicon
- `make build-darwin-amd64` - Build libraries for Intel
- `make build-darwin-universal` - Create Universal Binary (both architectures)
- `make dist-darwin-arm64` - Create arm64 distribution package
- `make dist-darwin-amd64` - Create x86_64 distribution package

### SurrealDB Cross-Compilation
- `make build-surrealdb-darwin-arm64` - Build for aarch64-apple-darwin
- `make build-surrealdb-darwin-amd64` - Build for x86_64-apple-darwin

### Utility
- `make check-env` - Shows build environment, paths, and library status

## Files Modified

1. `Makefile` - Major updates for macOS cross-compilation support and RPATH fixes
2. `go.mod` - Fixed replace directives with absolute paths
3. `/Users/digio/www/MCP/Remembrances/surrealdb-embedded/surrealdb.go` - Added `NewFromURL()` function

## Verification

### Native Build (arm64)
```bash
$ make build
✓ Build complete: build/remembrances-mcp

$ file build/remembrances-mcp
build/remembrances-mcp: Mach-O 64-bit executable arm64
```

### Cross-Compilation (x86_64)
```bash
$ make dist-darwin-amd64
✓ Distribution created in dist/darwin-amd64/

$ file dist/darwin-amd64/remembrances-mcp
dist/darwin-amd64/remembrances-mcp: Mach-O 64-bit executable x86_64

$ file dist/darwin-amd64/*.dylib
libllama.dylib: Mach-O 64-bit dynamically linked shared library x86_64
libsurrealdb_embedded_rs.dylib: Mach-O 64-bit dynamically linked shared library x86_64
```

### Runtime Test
```bash
$ cd build && ./remembrances-mcp --version
dev

# Verify library references
$ otool -L build/remembrances-mcp | grep -E "@rpath|libllama|libsurrealdb"
	@rpath/libllama.dylib
	@rpath/libggml.dylib
	@rpath/libggml-base.dylib
	@rpath/libsurrealdb_embedded_rs.dylib
```

## Distribution Structure

```
dist/darwin-amd64/
├── remembrances-mcp           # x86_64 binary
├── libllama.dylib             # x86_64
├── libggml.dylib              # x86_64
├── libggml-base.dylib         # x86_64
├── libggml-cpu.dylib          # x86_64
├── libggml-blas.dylib         # x86_64
├── libmtmd.dylib              # x86_64
├── libsurrealdb_embedded_rs.dylib  # x86_64
├── config.sample.yaml
├── config.sample.gguf.yaml
├── README.md
└── LICENSE.txt
```

## Usage Examples

```bash
# Build for native architecture (auto-detected)
make build

# Cross-compile for Intel Mac from Apple Silicon
make dist-darwin-amd64

# Build Universal Binary libraries
make build-darwin-universal

# Check current build environment
make check-env

# Override library paths
GO_LLAMA_DIR=/custom/path make build
```

## Notes

- Metal support is disabled for x86_64 builds (not supported on Intel Macs via cross-compilation)
- `libcommon` is a static library (.a), linked directly rather than as a shared library
- Rust targets must be added via `rustup target add` before cross-compiling surrealdb-embedded
- macOS uses `@executable_path` for RPATH (not `$ORIGIN` which is Linux-only)
- `install_name_tool` is used post-build to fix library references on macOS
- Binary can be run directly from build/ directory without setting `DYLD_LIBRARY_PATH`