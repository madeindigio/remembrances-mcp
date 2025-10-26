# Embeddings Migration Guide: llama.cpp → kelindar/search

## 🎯 Overview

Remembrances-MCP has migrated from `go-llama.cpp` to `kelindar/search` for embedding generation. This change brings several important improvements:

### ✅ Benefits

1. **BERT Model Support**: Now supports BERT-based embedding models (like `nomic-embed-text`) in GGUF format
2. **No CGO Required**: Uses `purego` to call native libraries without cgo, simplifying builds and cross-compilation
3. **Simpler Build Process**: No need to compile llama.cpp static libraries for each platform
4. **Better Compatibility**: Works with a wider range of embedding models
5. **Faster Builds**: No C/C++ compilation required

### ⚠️ Breaking Changes

- **Architecture Support**: The old implementation only supported LLaMA/Mistral architectures. The new one supports BERT.
- **Configuration Variables**: New environment variables introduced (see below)
- **Build Process**: Simplified (no more `llama-deps` targets)

---

## 🔧 Configuration Changes

### Old Configuration (DEPRECATED)

```bash
# Old llama.cpp configuration
export LLAMA_MODEL_PATH="/path/to/model.gguf"
export LLAMA_DIMENSION=768
export LLAMA_THREADS=4           # No longer used
export LLAMA_GPU_LAYERS=0
export LLAMA_CONTEXT=512         # No longer used
```

### New Configuration (RECOMMENDED)

```bash
# New kelindar/search configuration
export SEARCH_MODEL_PATH="/path/to/nomic-embed-text-v1.5.Q4_K_M.gguf"
export SEARCH_DIMENSION=768
export SEARCH_GPU_LAYERS=0       # 0 = CPU, >0 = GPU layers
```

### Automatic Migration

The system automatically migrates old `LLAMA_*` variables to `SEARCH_*` internally, so **existing configurations will continue to work**. However, we recommend updating to the new variables.

---

## 📦 Supported Models

### ✅ Supported (BERT-based GGUF)

- **nomic-embed-text-v1.5** (recommended)
- **nomic-embed-text-v2**
- **all-MiniLM-L6-v2**
- **BERT-base-uncased**
- Other BERT-based models in GGUF format

### ❌ Not Supported

- LLaMA/Mistral models (use for text generation, not embeddings)
- Non-GGUF formats
- Models with incompatible architectures

---

## 🚀 Getting Started

### 1. Download a Compatible Model

```bash
# Example: Download nomic-embed-text-v1.5 (Q4_K_M quantization)
cd /path/to/models
wget https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf
```

### 2. Update Configuration

```bash
# Set environment variables
export SEARCH_MODEL_PATH="/path/to/models/nomic-embed-text-v1.5.Q4_K_M.gguf"
export SEARCH_DIMENSION=768
export SEARCH_GPU_LAYERS=0  # Use 0 for CPU, or number of layers for GPU
```

### 3. Install Native Library (REQUIRED)

⚠️ **IMPORTANT**: kelindar/search requires a native library (`libllama_go.so` / `libllama_go.dylib` / `llamago.dll`) to be installed on your system. The application uses `purego` to dynamically load this library at runtime.

For **Linux** and **Windows**, kelindar/search provides pre-compiled binaries with Vulkan GPU support in the `dist/` directory of the [kelindar/search repository](https://github.com/kelindar/search/tree/main/dist).

```bash
# Download and install the library

# Linux (amd64)
wget https://github.com/kelindar/search/raw/main/dist/libllama_go.so
sudo cp libllama_go.so /usr/lib/
sudo ldconfig

# macOS (you'll need to compile - see below)
# Pre-built binaries not available for macOS

# Windows (amd64)
# Download llamago.dll from the dist/ folder
# Copy to C:\Windows\System32\ or add to PATH
```

For **macOS** or custom builds, you **MUST** compile from source - see [Compilation Guide](#compilation-guide) below.

**Verification**: After installation, verify the library is loadable:
```bash
# Linux
ldconfig -p | grep libllama_go

# macOS
ls -l /usr/local/lib/libllama_go.dylib
```

### 4. Build and Run

```bash
# Install dependencies
go mod download

# Build (no cgo required!)
make build

# Run with new configuration
./dist/remembrances-mcp \
  --search-model-path /path/to/nomic-embed-text-v1.5.Q4_K_M.gguf \
  --search-dimension 768
```

---

## 🔨 Compilation Guide

### Prerequisites

- Go 1.23.4 or later
- C/C++ compiler (gcc/g++ or Visual Studio)
- CMake

### Compile kelindar/search Native Library

If the pre-compiled binaries don't work for your platform, compile from source:

#### Linux

```bash
cd /tmp
git clone https://github.com/kelindar/search.git
cd search
git submodule update --init --recursive

mkdir build && cd build
cmake -DBUILD_SHARED_LIBS=ON -DCMAKE_BUILD_TYPE=Release -DCMAKE_CXX_COMPILER=g++ -DCMAKE_C_COMPILER=gcc ..
cmake --build . --config Release

# Install
sudo cp libllama_go.so /usr/lib/
```

#### macOS

```bash
cd /tmp
git clone https://github.com/kelindar/search.git
cd search
git submodule update --init --recursive

mkdir build && cd build
cmake -DBUILD_SHARED_LIBS=ON -DCMAKE_BUILD_TYPE=Release ..
cmake --build . --config Release

# Install
sudo cp libllama_go.dylib /usr/local/lib/
```

#### Windows

```bash
# Using Visual Studio Build Tools
cd C:\Temp
git clone https://github.com/kelindar/search.git
cd search
git submodule update --init --recursive

mkdir build && cd build
cmake -DCMAKE_BUILD_TYPE=Release ..
cmake --build . --config Release

# Install (copy to system PATH)
copy bin\llamago.dll C:\Windows\System32\
```

#### GPU Support (Vulkan)

```bash
# Make sure Vulkan SDK is installed
# https://vulkan.lunarg.com/

# Then add -DGGML_VULKAN=ON to cmake
cmake -DCMAKE_BUILD_TYPE=Release -DGGML_VULKAN=ON ..
cmake --build . --config Release
```

---

## 🧪 Testing

### Basic Test

```bash
# Build
make build

# Run with test configuration
./dist/remembrances-mcp \
  --search-model-path ./models/nomic-embed-text-v1.5.Q4_K_M.gguf \
  --search-dimension 768 \
  --log-level debug
```

### Verify Embeddings

```go
package main

import (
    "context"
    "fmt"
    "github.com/madeindigio/remembrances-mcp/pkg/embedder"
)

func main() {
    // Create embedder
    emb, err := embedder.NewSearchEmbedder(
        "/path/to/nomic-embed-text-v1.5.Q4_K_M.gguf",
        0, // GPU layers
    )
    if err != nil {
        panic(err)
    }
    defer emb.Close()

    // Generate embedding
    ctx := context.Background()
    embedding, err := emb.EmbedQuery(ctx, "test query")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Embedding dimension: %d\n", len(embedding))
    fmt.Printf("First 5 values: %v\n", embedding[:5])
}
```

---

## 🐛 Troubleshooting

### Error: "unknown model architecture: 'nomic-bert'"

**This was the old error with llama.cpp.** With kelindar/search, this is now resolved. If you still see this:
- Make sure you're using the new build (check with `./dist/remembrances-mcp --version`)
- Verify `SEARCH_MODEL_PATH` is set correctly
- Ensure you downloaded a GGUF model (not safetensors or other formats)

### Error: "failed to load model"

Check:
1. Model file exists and is readable
2. Model is in GGUF format
3. Model is a BERT-based embedding model (not a text generation model)

### Error: "Library 'libllama_go.so' not found" or similar

The native library is missing or not in the search path. This is the most common issue.

**Solutions:**

1. **Download pre-compiled binary** (Linux/Windows):
   ```bash
   # Linux
   wget https://github.com/kelindar/search/raw/main/dist/libllama_go.so
   sudo cp libllama_go.so /usr/lib/
   sudo ldconfig
   ```

2. **Compile from source** (macOS or custom builds):
   See [Compilation Guide](#compilation-guide) below

3. **Verify installation**:
   ```bash
   # Linux
   ldconfig -p | grep libllama_go
   
   # macOS
   ls -l /usr/local/lib/libllama_go.dylib
   
   # Windows
   where llamago.dll
   ```

4. **Set LD_LIBRARY_PATH** (temporary workaround):
   ```bash
   export LD_LIBRARY_PATH=/path/to/library:$LD_LIBRARY_PATH
   ```

**Note**: The library MUST be installed before running the application. Unlike the old llama.cpp implementation, this is a runtime dependency.

### Performance Issues

1. **Enable GPU**: Set `SEARCH_GPU_LAYERS` to a positive number
2. **Use Quantized Models**: Q4_K_M or Q8_0 variants are faster
3. **Check Model Size**: Smaller models (MiniLM) are faster than larger ones (BERT-base)

---

## 📚 Additional Resources

- [kelindar/search Repository](https://github.com/kelindar/search)
- [Nomic Embed Text Models](https://huggingface.co/nomic-ai)
- [GGUF Format Documentation](https://github.com/ggerganov/ggml/blob/master/docs/gguf.md)
- [llama.cpp Documentation](https://github.com/ggerganov/llama.cpp)

---

## 🤝 Migration Checklist

- [ ] Download compatible BERT GGUF model
- [ ] Update environment variables from `LLAMA_*` to `SEARCH_*`
- [ ] Install pre-compiled binaries (or compile from source)
- [ ] Update build scripts/CI/CD to remove llama.cpp compilation steps
- [ ] Test embedding generation with new setup
- [ ] Update documentation/deployment guides
- [ ] Remove old `go-llama.cpp` directory (optional cleanup)

---

## 💡 FAQ

### Q: Why does the app panic about "Library not found"?

**A:** kelindar/search requires a native library (`libllama_go.so` / `.dylib` / `.dll`) to be installed on your system. Download from the [dist/ folder](https://github.com/kelindar/search/tree/main/dist) or compile from source. See installation instructions above.

### Q: Can I still use my old llama.cpp models?

**A:** No. LLaMA/Mistral models are text generation models, not embedding models. You need BERT-based models for embeddings.

### Q: Do I need to recompile for each platform?

**A:** No! Unlike the old setup, kelindar/search uses purego (no cgo), so binaries are portable. Pre-compiled native libraries are provided for Linux/Windows.

### Q: What happened to the `LLAMA_THREADS` and `LLAMA_CONTEXT` variables?

**A:** They are no longer needed. kelindar/search handles threading automatically.

### Q: Is GPU support available?

**A:** Yes! Set `SEARCH_GPU_LAYERS` to enable GPU acceleration. However, you need to compile the native library with Vulkan support (`-DGGML_VULKAN=ON`) or use the pre-compiled binaries that already include it.

### Q: What's the default embedding dimension?

**A:** 768 for most BERT models, but MiniLM uses 384. Always verify with your specific model.

### Q: Can I distribute my binary without the native library?

**A:** No. Unlike pure Go applications, the native library (`libllama_go.so` / `.dylib` / `.dll`) must be distributed alongside your binary or installed on the target system. This is a runtime dependency that cannot be statically linked.

---

## 📝 Changelog

### v2.0.0 (2025-01-XX)

- **BREAKING**: Migrated from `go-llama.cpp` to `kelindar/search`
- **NEW**: Support for BERT models (nomic-embed-text, MiniLM, etc.)
- **NEW**: No cgo required (uses purego)
- **IMPROVED**: Simplified build process (no C++ compilation)
- **DEPRECATED**: `LLAMA_*` environment variables (use `SEARCH_*` instead)
- **REMOVED**: `llama-deps*` Make targets
- **REMOVED**: `go-llama.cpp` submodule dependency

---

**Questions or issues?** Open an issue on GitHub or check existing discussions.