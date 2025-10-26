# Static Compilation Options for kelindar/search

## Executive Summary

**YES, it is possible to create a fork or wrapper for static compilation**, but there are multiple approaches with different trade-offs. This document analyzes three viable options for achieving true static linking in remembrances-mcp.

## TL;DR - Recommendations

| Approach | Effort | Complexity | Maintenance | Best For |
|----------|--------|------------|-------------|----------|
| **Hybrid Wrapper (Recommended)** | 1-2 days | Medium | Low | Production use |
| **Full Fork** | 3-5 days | High | High | Complete control |
| **Upstream PR** | 2-3 days | Medium-High | Shared | Community benefit |

**Recommendation**: Start with **Hybrid Wrapper** approach for immediate needs.

---

## Option 1: Hybrid Wrapper (RECOMMENDED)

### Concept

Add a CGO-based static backend alongside the existing purego backend, selectable via build tags.

### Architecture

```
pkg/embedder/
├── embedder.go           # Interface (unchanged)
├── factory.go            # Updated to support static builds
├── search.go             # Current purego implementation (default)
├── search_static.go      # New CGO implementation
│   // +build static
├── search_static.h       # C headers for static linking
└── libllama_go.a         # Static library (for static builds)
```

### Implementation

#### 1. search_static.go (New File)

```go
//go:build static
// +build static

package embedder

// #cgo CFLAGS: -I${SRCDIR}/../../dist/include
// #cgo LDFLAGS: -L${SRCDIR}/../../dist/lib -lllama_go -lstdc++ -lm -static
// #include <stdlib.h>
// #include "search_static.h"
import "C"
import (
    "context"
    "fmt"
    "sync"
    "unsafe"
)

type SearchStaticEmbedder struct {
    model       C.model_ptr
    modelPath   string
    dimension   int
    gpuLayers   int
    mu          sync.RWMutex
    initialized bool
}

func NewSearchEmbedder(modelPath string, gpuLayers int) (*SearchStaticEmbedder, error) {
    cPath := C.CString(modelPath)
    defer C.free(unsafe.Pointer(cPath))
    
    model := C.load_model(cPath, C.uint(gpuLayers))
    if model == nil {
        return nil, fmt.Errorf("failed to load model: %s", modelPath)
    }
    
    dimension := int(C.get_embedding_size(model))
    
    return &SearchStaticEmbedder{
        model:       model,
        modelPath:   modelPath,
        dimension:   dimension,
        gpuLayers:   gpuLayers,
        initialized: true,
    }, nil
}

func (s *SearchStaticEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    if !s.initialized {
        return nil, fmt.Errorf("embedder not initialized")
    }
    
    cText := C.CString(text)
    defer C.free(unsafe.Pointer(cText))
    
    embeddings := make([]float32, s.dimension)
    
    result := C.embed_text(s.model, cText, (*C.float)(unsafe.Pointer(&embeddings[0])))
    if result != 0 {
        return nil, fmt.Errorf("embedding failed with code: %d", result)
    }
    
    return embeddings, nil
}

func (s *SearchStaticEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
    embeddings := make([][]float32, len(texts))
    for i, text := range texts {
        emb, err := s.EmbedText(ctx, text)
        if err != nil {
            return nil, fmt.Errorf("failed to embed document %d: %w", i, err)
        }
        embeddings[i] = emb
    }
    return embeddings, nil
}

func (s *SearchStaticEmbedder) Dimension() int {
    return s.dimension
}

func (s *SearchStaticEmbedder) Close() error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if s.initialized && s.model != nil {
        C.free_model(s.model)
        s.initialized = false
    }
    return nil
}
```

#### 2. search_static.h (New File)

```c
#ifndef SEARCH_STATIC_H
#define SEARCH_STATIC_H

#ifdef __cplusplus
extern "C" {
#endif

typedef void* model_ptr;
typedef void* context_ptr;

// Load a model from file
model_ptr load_model(const char* path, unsigned int gpu_layers);

// Get embedding dimension
int get_embedding_size(model_ptr model);

// Embed text and write to output array
int embed_text(model_ptr model, const char* text, float* out_embeddings);

// Free model resources
void free_model(model_ptr model);

#ifdef __cplusplus
}
#endif

#endif // SEARCH_STATIC_H
```

#### 3. CMakeLists.txt (Modified)

```cmake
cmake_minimum_required(VERSION 3.14)
project(llama_go_lib VERSION 1.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

# Option to build static or shared library
option(BUILD_STATIC_LIB "Build static library for CGO" OFF)

if(BUILD_STATIC_LIB)
    set(BUILD_SHARED_LIBS OFF CACHE BOOL "Build static libraries" FORCE)
    add_subdirectory(llama.cpp)
    add_subdirectory(llama.cpp/common)
    add_library(llama_go STATIC llama-go.cpp/llama-go.cpp)
else()
    set(BUILD_SHARED_LIBS OFF CACHE BOOL "Build static libraries" FORCE)
    add_subdirectory(llama.cpp)
    add_subdirectory(llama.cpp/common)
    add_library(llama_go SHARED llama-go.cpp/llama-go.cpp)
endif()

target_link_libraries(llama_go PRIVATE llama common)
```

#### 4. Build Commands

```bash
# Dynamic build (default, no cgo)
go build -o remembrances-mcp ./cmd/remembrances-mcp

# Static build (with cgo)
# First, build static library
mkdir -p build-static && cd build-static
cmake -DBUILD_STATIC_LIB=ON ..
cmake --build . --config Release
cp lib/libllama_go.a ../dist/lib/
cd ..

# Then build Go binary
CGO_ENABLED=1 go build \
  -tags static \
  -ldflags='-extldflags "-static"' \
  -o remembrances-mcp-static \
  ./cmd/remembrances-mcp

# Verify static linking
ldd remembrances-mcp-static
# Should output: "not a dynamic executable"
```

### Advantages

✅ **Best of both worlds**: purego by default, static when needed  
✅ **Low maintenance**: No need to sync with upstream changes  
✅ **Flexible**: Users choose at build time  
✅ **Independent**: Doesn't require forking kelindar/search  
✅ **Testable**: Both backends can be tested separately  

### Disadvantages

❌ Two implementations to maintain  
❌ Requires C++ toolchain for static builds  
❌ More complex build process for static variant  
❌ Duplicate code (but minimal)  

### Estimated Effort

- **Initial implementation**: 1-2 days
- **Testing**: 0.5 day
- **Documentation**: 0.5 day
- **Total**: 2-3 days

---

## Option 2: Full Fork of kelindar/search

### Concept

Fork kelindar/search and replace purego with cgo throughout.

### Implementation Plan

1. **Fork Repository**
   ```bash
   git clone https://github.com/kelindar/search kelindar-search-static
   cd kelindar-search-static
   git remote add upstream https://github.com/kelindar/search
   ```

2. **Replace loader.go**
   ```go
   // loader_cgo.go (replaces loader.go)
   package search
   
   // #cgo LDFLAGS: -L${SRCDIR}/dist -lllama_go -lstdc++ -lm -static
   // #include "llama-go.h"
   import "C"
   import "unsafe"
   
   func loadModel(path string, gpuLayers uint32) uintptr {
       cPath := C.CString(path)
       defer C.free(unsafe.Pointer(cPath))
       return uintptr(C.load_model(cPath, C.uint(gpuLayers)))
   }
   
   // ... similar replacements for all functions
   ```

3. **Update All Function Calls**
   - Replace `purego.RegisterLibFunc` with direct C calls
   - Add C type conversions
   - Implement proper memory management

4. **Build Static Library**
   ```bash
   mkdir build && cd build
   cmake -DBUILD_SHARED_LIBS=OFF ..
   cmake --build . --config Release
   ```

5. **Update Go Module**
   ```bash
   # In go.mod
   module github.com/yourusername/kelindar-search-static
   
   # In remembrances-mcp/go.mod
   replace github.com/kelindar/search => github.com/yourusername/kelindar-search-static
   ```

### Advantages

✅ Complete control over implementation  
✅ Can optimize specifically for static linking  
✅ True single-file binary  
✅ No runtime library dependencies  

### Disadvantages

❌ Must maintain fork (sync with upstream)  
❌ Loses all purego benefits  
❌ Complex cross-compilation  
❌ Requires C++ toolchain always  
❌ Slower builds (~10x)  
❌ Community fragmentation  

### Estimated Effort

- **Initial fork and conversion**: 3-4 days
- **Testing all platforms**: 1-2 days
- **Documentation**: 1 day
- **Ongoing maintenance**: 2-4 hours/month
- **Total initial**: 5-7 days

---

## Option 3: Contribute Upstream PR

### Concept

Add static build support to original kelindar/search via Pull Request.

### Implementation Plan

1. **Design Proposal**
   - Write RFC/issue on kelindar/search
   - Propose build tag approach
   - Get maintainer feedback

2. **Implementation**
   ```go
   // loader_purego.go
   //go:build !static
   // +build !static
   
   package search
   // Current purego implementation
   
   // loader_cgo.go
   //go:build static
   // +build static
   
   package search
   // New cgo implementation
   ```

3. **Documentation**
   - Update README with static build instructions
   - Add build examples
   - Document trade-offs

4. **CI/CD Updates**
   - Add static build tests
   - Test on multiple platforms
   - Ensure both builds work

5. **Submit PR**
   - Create pull request
   - Address review feedback
   - Maintain until merged

### Advantages

✅ Benefits entire community  
✅ Official support in upstream  
✅ Shared maintenance burden  
✅ No fork to maintain  
✅ Industry best practices  

### Disadvantages

❌ Depends on maintainer acceptance  
❌ May take time for review  
❌ Must follow project conventions  
❌ Less control over design  

### Estimated Effort

- **Design and proposal**: 0.5 day
- **Implementation**: 2-3 days
- **Testing and CI**: 1 day
- **Review process**: Variable (weeks to months)
- **Total**: 3.5-4.5 days + waiting time

---

## Comparison Matrix

| Factor | Hybrid Wrapper | Full Fork | Upstream PR |
|--------|----------------|-----------|-------------|
| **Implementation Time** | 2-3 days | 5-7 days | 3.5-4.5 days |
| **Ongoing Maintenance** | Low | High | None |
| **Flexibility** | High | Very High | Medium |
| **Community Benefit** | Project only | Fork users | Everyone |
| **Risk** | Low | Medium | Low |
| **Cross-compilation** | Medium complexity | High complexity | Medium complexity |
| **Build Speed (dynamic)** | Fast | N/A | Fast |
| **Build Speed (static)** | Slow | Slow | Slow |
| **Binary Size (dynamic)** | Small | N/A | Small |
| **Binary Size (static)** | Large | Large | Large |
| **Requires C++ toolchain** | Only for static | Always | Only for static |

---

## Recommended Implementation Plan

### Phase 1: Proof of Concept (Day 1-2)

1. **Create prototype in local branch**
   ```bash
   git checkout -b feature/static-compilation
   ```

2. **Implement minimal CGO wrapper**
   - Single file: `pkg/embedder/search_static.go`
   - Basic functions: load, embed, free
   - Test with simple model

3. **Verify static linking**
   ```bash
   CGO_ENABLED=1 go build -tags static \
     -ldflags='-extldflags "-static"' \
     -o test-static
   ldd test-static  # Should say "not a dynamic executable"
   ```

### Phase 2: Full Implementation (Day 3-4)

1. **Complete CGO wrapper**
   - All embedder interface methods
   - Error handling
   - Memory management
   - Context support

2. **Update factory**
   ```go
   // pkg/embedder/factory.go
   func NewSearchEmbedder(path string, layers int) (Embedder, error) {
       // Dynamic build (default)
       return newSearchDynamic(path, layers)
   }
   
   func NewSearchStaticEmbedder(path string, layers int) (Embedder, error) {
       // Static build (cgo)
       return newSearchStatic(path, layers)
   }
   ```

3. **Build system updates**
   - Makefile targets for static builds
   - CMake configuration
   - Documentation

### Phase 3: Testing (Day 5)

1. **Functional tests**
   ```bash
   # Test dynamic build
   make build && ./remembrances-mcp --version
   
   # Test static build
   make build-static && ./remembrances-mcp-static --version
   ```

2. **Cross-platform tests**
   - Linux x86_64
   - Linux ARM64
   - macOS (Intel + Apple Silicon)
   - Windows (optional)

3. **Performance comparison**
   - Embedding speed
   - Memory usage
   - Binary size

### Phase 4: Documentation (Day 6)

1. **Build documentation**
   - Static vs dynamic trade-offs
   - Build instructions
   - Troubleshooting guide

2. **Update README**
   - Add static build section
   - Prerequisites
   - Examples

3. **Create migration guide**
   - For users wanting static builds
   - Platform-specific notes

---

## Code Examples

### Makefile Additions

```makefile
# Build with dynamic library (default)
.PHONY: build
build:
	go build -o remembrances-mcp ./cmd/remembrances-mcp

# Build static library first
.PHONY: build-static-lib
build-static-lib:
	mkdir -p build-static && cd build-static && \
	cmake -DBUILD_STATIC_LIB=ON .. && \
	cmake --build . --config Release && \
	cp lib/libllama_go.a ../dist/lib/

# Build Go binary with static linking
.PHONY: build-static
build-static: build-static-lib
	CGO_ENABLED=1 go build \
		-tags static \
		-ldflags='-extldflags "-static"' \
		-o remembrances-mcp-static \
		./cmd/remembrances-mcp

# Clean static builds
.PHONY: clean-static
clean-static:
	rm -rf build-static
	rm -f remembrances-mcp-static
	rm -f dist/lib/libllama_go.a

# Verify builds
.PHONY: verify
verify:
	@echo "=== Dynamic Build ==="
	@ldd remembrances-mcp || echo "Dynamic binary"
	@echo ""
	@echo "=== Static Build ==="
	@ldd remembrances-mcp-static || echo "Static binary (expected)"
```

### Build Script

```bash
#!/bin/bash
# scripts/build-static.sh

set -e

echo "Building static version of remembrances-mcp..."

# Check for required tools
command -v cmake >/dev/null 2>&1 || { echo "cmake required"; exit 1; }
command -v g++ >/dev/null 2>&1 || { echo "g++ required"; exit 1; }

# Build static library
echo "Step 1/3: Building static C++ library..."
mkdir -p build-static
cd build-static
cmake -DBUILD_STATIC_LIB=ON \
      -DCMAKE_BUILD_TYPE=Release \
      -DCMAKE_CXX_COMPILER=g++ \
      ..
cmake --build . --config Release
cd ..

# Copy to dist
echo "Step 2/3: Copying library..."
mkdir -p dist/lib dist/include
cp build-static/lib/libllama_go.a dist/lib/
cp llama-go.h dist/include/

# Build Go binary
echo "Step 3/3: Building Go binary..."
CGO_ENABLED=1 go build \
    -tags static \
    -ldflags='-extldflags "-static -lstdc++ -lm"' \
    -o remembrances-mcp-static \
    ./cmd/remembrances-mcp

echo ""
echo "✓ Build complete!"
echo "  Binary: ./remembrances-mcp-static"
echo ""

# Verify
echo "Verifying static linking..."
if ldd remembrances-mcp-static 2>&1 | grep -q "not a dynamic executable"; then
    echo "✓ Binary is statically linked"
else
    echo "⚠ Warning: Binary may have dynamic dependencies"
    ldd remembrances-mcp-static
fi

echo ""
echo "Binary size:"
ls -lh remembrances-mcp-static
```

---

## Decision Matrix

### When to Use Each Option

**Use Hybrid Wrapper if:**
- Need static builds occasionally
- Want to maintain simplicity
- Cross-compilation is important
- Community uses dynamic builds

**Use Full Fork if:**
- Need complete control
- Only use static builds
- Have resources for maintenance
- Significant customization needed

**Use Upstream PR if:**
- Want to benefit community
- Can wait for review process
- Follow project conventions
- Prefer official support

---

## Conclusion

**Recommended Approach: Hybrid Wrapper**

This provides the best balance of:
- ✅ Flexibility (both static and dynamic)
- ✅ Low maintenance
- ✅ Quick implementation
- ✅ No fork required
- ✅ Backward compatible

**Implementation Timeline:**
- Week 1: Proof of concept + implementation (Days 1-4)
- Week 2: Testing + documentation (Days 5-6)
- **Total**: ~6 days of focused work

**Next Steps:**
1. Review this document with team
2. Decide on approach
3. Create implementation branch
4. Follow phased plan above
5. Test thoroughly
6. Document for users

---

## References

- [CGO Documentation](https://pkg.go.dev/cmd/cgo)
- [Go Build Tags](https://pkg.go.dev/cmd/go#hdr-Build_constraints)
- [Static Linking in Go](https://mt165.co.uk/blog/static-link-go/)
- [kelindar/search Repository](https://github.com/kelindar/search)
- [llama.cpp Documentation](https://github.com/ggerganov/llama.cpp)

---

**Document Status:** Complete  
**Last Updated:** October 2024  
**Author:** Remembrances-MCP Team