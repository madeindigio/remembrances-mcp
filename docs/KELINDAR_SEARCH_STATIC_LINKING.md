# kelindar/search Static Compilation Analysis

## Executive Summary

This document analyzes the `kelindar/search` library architecture and explores options for static compilation of its native library component into Go binaries. The analysis concludes that **true static linking into the Go binary is not possible** with the current `purego`-based architecture, but several deployment strategies are available.

## Architecture Overview

### How kelindar/search Works

1. **Go Layer** (`kelindar/search` package)
   - Pure Go code using `github.com/ebitengine/purego`
   - No cgo required
   - Dynamic library loading at runtime

2. **Native Library** (`libllama_go.so/dylib/dll`)
   - C++ code wrapping llama.cpp functionality
   - Compiled as a shared library
   - Statically links its dependencies (llama.cpp, common)

3. **Runtime Loading**
   - `purego` uses `dlopen()` (Unix) or `LoadLibrary()` (Windows)
   - Searches standard system paths: `/usr/lib`, `/usr/local/lib`, etc.
   - Cannot embed library inside Go binary

### Build Process

From the `kelindar/search` repository:

```cmake
# Root CMakeLists.txt
set(BUILD_SHARED_LIBS OFF CACHE BOOL "Build static libraries" FORCE)
add_subdirectory(llama.cpp)
add_subdirectory(llama.cpp/common)
add_subdirectory(llama-go.cpp)

# llama-go.cpp/CMakeLists.txt
add_library(llama_go SHARED ${SOURCES})
target_link_libraries(llama_go PRIVATE
    llama       # Static llama library
    common      # Static common library
)
```

**Key Points:**
- Dependencies (llama.cpp, common) are built as **static libraries**
- Final output (libllama_go) is a **shared library**
- The shared library contains all dependencies statically linked
- Result: One `.so`/`.dylib`/`.dll` file with no external dependencies (except system libs)

## Why True Static Linking Is Not Possible

### Technical Limitations

1. **purego Architecture**
   ```go
   // From kelindar/search/loader.go
   func load(name string) (uintptr, error) {
       return purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
   }
   ```
   - `Dlopen()` requires a file path to a shared library
   - Operates at runtime, not compile time
   - Cannot accept embedded data

2. **No Memory-Based Loading**
   - `dlopen()` system call requires a filesystem path
   - Cannot load from memory buffers
   - Cannot load from embedded byte arrays

3. **cgo Alternative Would Defeat Purpose**
   - Using cgo would enable true static linking
   - But kelindar/search was designed to **avoid cgo**
   - Benefits of no-cgo approach:
     - Simpler cross-compilation
     - Faster builds
     - No C toolchain required
     - Better portability

## Available Deployment Strategies

### Strategy 1: Separate Distribution (Current Approach)

**Distribute two files:**
- `remembrances-mcp` (Go binary)
- `libllama_go.so` (native library)

**Installation:**
```bash
# Copy native library to system path
sudo cp libllama_go.so /usr/local/lib/
sudo ldconfig  # Update library cache (Linux)

# Or set library path
export LD_LIBRARY_PATH=/path/to/library:$LD_LIBRARY_PATH
```

**Pros:**
- Clean separation of concerns
- Easy to update either component
- Library can be shared by multiple programs

**Cons:**
- Two files to manage
- Requires system-level installation or PATH configuration

### Strategy 2: Embedded Extraction (Pseudo-Static)

**Embed library in Go binary, extract at runtime:**

```go
package main

import (
    _ "embed"
    "os"
    "path/filepath"
)

//go:embed libllama_go.so
var nativeLib []byte

func init() {
    // Extract to temporary location
    tmpDir := os.TempDir()
    libPath := filepath.Join(tmpDir, "libllama_go.so")
    
    if err := os.WriteFile(libPath, nativeLib, 0755); err != nil {
        panic(err)
    }
    
    // Set library path for purego to find
    os.Setenv("LD_LIBRARY_PATH", tmpDir+":"+os.Getenv("LD_LIBRARY_PATH"))
}
```

**Pros:**
- Single binary distribution
- Automatic extraction
- Works with purego

**Cons:**
- Writes to filesystem (temp directory)
- Larger binary size
- Potential permission issues
- Cleanup considerations
- Security concerns (temp file attacks)

### Strategy 3: Custom Wrapper Binary

**Create a wrapper script or binary:**

```bash
#!/bin/bash
# remembrances-mcp-launcher

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export LD_LIBRARY_PATH="${SCRIPT_DIR}:${LD_LIBRARY_PATH}"
exec "${SCRIPT_DIR}/remembrances-mcp" "$@"
```

**Pros:**
- Clean separation
- No temp file issues
- Easy to understand

**Cons:**
- Requires shell script
- Multiple files still needed

### Strategy 4: Docker/Container Distribution

**Package everything in a container:**

```dockerfile
FROM ubuntu:22.04
COPY libllama_go.so /usr/local/lib/
COPY remembrances-mcp /usr/local/bin/
RUN ldconfig
ENTRYPOINT ["/usr/local/bin/remembrances-mcp"]
```

**Pros:**
- Consistent deployment
- All dependencies included
- Platform-agnostic

**Cons:**
- Requires Docker/container runtime
- Larger distribution size

### Strategy 5: Platform-Specific Installers

**Create installers for each platform:**
- Linux: `.deb`, `.rpm`, or AppImage
- macOS: `.pkg` or `.app` bundle
- Windows: `.msi` or `.exe` installer

**Pros:**
- Native platform experience
- Handles library installation automatically
- Professional distribution

**Cons:**
- More complex build process
- Platform-specific maintenance

## Compilation Instructions

### Building the Native Library

```bash
# Clone kelindar/search
git clone --recurse-submodules https://github.com/kelindar/search
cd search

# Linux build
mkdir build && cd build
cmake -DBUILD_SHARED_LIBS=ON \
      -DCMAKE_BUILD_TYPE=Release \
      -DCMAKE_CXX_COMPILER=g++ \
      -DCMAKE_C_COMPILER=gcc ..
cmake --build . --config Release

# Result: build/lib/libllama_go.so

# macOS build (similar)
cmake -DCMAKE_BUILD_TYPE=Release ..
cmake --build . --config Release

# Windows build
cmake -DCMAKE_BUILD_TYPE=Release ..
cmake --build . --config Release
```

### GPU Support (Optional)

```bash
# Vulkan support (requires Vulkan SDK)
cmake -DCMAKE_BUILD_TYPE=Release -DGGML_VULKAN=ON ..
cmake --build . --config Release

# CUDA support
cmake -DCMAKE_BUILD_TYPE=Release -DGGML_CUDA=ON ..
cmake --build . --config Release
```

### Cross-Compilation

Cross-compiling the native library requires:
1. Target architecture C++ toolchain
2. Target architecture system libraries
3. CMake configuration for cross-compilation

**Example for Linux ARM64 from x86_64:**
```bash
sudo apt-get install g++-aarch64-linux-gnu

cmake -DCMAKE_SYSTEM_NAME=Linux \
      -DCMAKE_SYSTEM_PROCESSOR=aarch64 \
      -DCMAKE_C_COMPILER=aarch64-linux-gnu-gcc \
      -DCMAKE_CXX_COMPILER=aarch64-linux-gnu-g++ \
      -DCMAKE_BUILD_TYPE=Release ..
```

## Verification

### Check Static Linking Inside Shared Library

```bash
# Linux
ldd libllama_go.so
# Should only show system libraries (libc, libstdc++, etc.)
# Should NOT show libllama or libcommon

# macOS
otool -L libllama_go.dylib

# Windows
dumpbin /DEPENDENTS llama_go.dll
```

### Check Go Binary

```bash
# Linux
ldd remembrances-mcp
# Should show libllama_go.so as dependency

# Verify no cgo
go tool nm remembrances-mcp | grep cgo
# Should return nothing
```

## Recommendations for Remembrances-MCP

### Short-Term: Strategy 1 (Separate Distribution)

**Recommended approach:**
1. Distribute `remembrances-mcp` binary
2. Distribute `libllama_go.so` separately
3. Provide installation script:

```bash
#!/bin/bash
# install.sh

set -e

INSTALL_DIR="/usr/local"
BIN_DIR="${INSTALL_DIR}/bin"
LIB_DIR="${INSTALL_DIR}/lib"

echo "Installing remembrances-mcp..."
sudo cp remembrances-mcp "${BIN_DIR}/"
sudo chmod +x "${BIN_DIR}/remembrances-mcp"

echo "Installing native library..."
sudo cp libllama_go.so "${LIB_DIR}/"
sudo chmod 755 "${LIB_DIR}/libllama_go.so"

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    sudo ldconfig
fi

echo "Installation complete!"
echo "Run: remembrances-mcp --help"
```

### Long-Term: Strategy 5 (Platform Installers)

**For production distribution:**
- **Linux**: Create `.deb` and `.rpm` packages
- **macOS**: Create `.pkg` installer or `.app` bundle
- **Windows**: Create `.msi` installer

**Benefits:**
- Professional deployment
- Handles library paths automatically
- Integrates with system package managers
- Clean uninstallation

## Alternative: Fork kelindar/search with cgo

**If true static linking is absolutely required:**

1. Fork `kelindar/search`
2. Replace `purego` with cgo
3. Link statically against C++ code:

```go
// #cgo LDFLAGS: -L. -lllama_go -lstdc++ -static
// #include "llama-go.h"
import "C"
```

**Trade-offs:**
- ✅ True static binary
- ❌ Requires C++ compiler
- ❌ Complex cross-compilation
- ❌ Slower builds
- ❌ Loses purego benefits

## Conclusion

**Key Findings:**
1. kelindar/search already does maximum static linking (within the .so)
2. True static linking into Go binary requires cgo (defeats project goals)
3. Multiple viable deployment strategies exist
4. Separate distribution is simplest and most maintainable

**Recommendation:**
- Use **Strategy 1** (separate distribution) for development
- Consider **Strategy 5** (installers) for production
- Document library requirements clearly
- Provide installation scripts

The current approach with kelindar/search is sound and follows industry best practices for Go projects that interface with native code without cgo.

## References

- [kelindar/search GitHub](https://github.com/kelindar/search)
- [purego Documentation](https://github.com/ebitengine/purego)
- [Go Static Linking Guide](https://mt165.co.uk/blog/static-link-go/)
- [CMake Shared vs Static Libraries](https://cmake.org/cmake/help/latest/variable/BUILD_SHARED_LIBS.html)