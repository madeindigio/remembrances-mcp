# GGUF Embeddings Support

This document describes how to use local GGUF models for embeddings in Remembrances-MCP.

## Overview

Remembrances-MCP now supports loading local GGUF embedding models directly using the go-llama.cpp library. This provides:

- **Privacy**: All embeddings are generated locally, no data sent to external services
- **Performance**: Direct model inference without network latency
- **Cost**: No API costs for embedding generation
- **Flexibility**: Support for quantized models (Q4_K_M, Q8_0, etc.) to balance speed and accuracy

## Supported Models

The implementation supports GGUF embedding models based on:

- **Nomic Embed** (nomic-bert architecture)
  - nomic-embed-text-v1.5 (768 dimensions)
  - nomic-embed-text-v2 (768 dimensions)
  - nomic-embed-text-v2-moe (768 dimensions)
  
- **Qwen** embedding models
- Other BERT-based embedding models in GGUF format

## Installation

### Prerequisites

1. **Go 1.21+** installed
2. **C/C++ compiler** (gcc, clang, or MSVC)
3. **Make** (for building llama.cpp)
4. **Git** with submodules support

### Building

1. **Clone with submodules**:
   ```bash
   git clone --recurse-submodules https://github.com/madeindigio/remembrances-mcp
   cd remembrances-mcp
   ```

2. **Build the project**:
   ```bash
   # Default build (CPU only on Linux, Metal on macOS)
   make build
   
   # With CUDA support (Linux with NVIDIA GPU)
   make BUILD_TYPE=cublas build
   
   # With ROCm support (Linux with AMD GPU)
   make BUILD_TYPE=hipblas build
   
   # With OpenBLAS support
   make BUILD_TYPE=openblas build
   ```

3. **Check build environment**:
   ```bash
   make check-env
   ```

## Configuration

### Using CLI Flags

```bash
./build/remembrances-mcp \
  --gguf-model-path /path/to/nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-threads 8 \
  --gguf-gpu-layers 32
```

### Using Environment Variables

```bash
export GOMEM_GGUF_MODEL_PATH="/path/to/nomic-embed-text-v1.5.Q4_K_M.gguf"
export GOMEM_GGUF_THREADS=8
export GOMEM_GGUF_GPU_LAYERS=32

./build/remembrances-mcp
```

### Using YAML Configuration

Create a `config.yaml` file:

```yaml
# GGUF Local Model Configuration
gguf-model-path: "/path/to/nomic-embed-text-v1.5.Q4_K_M.gguf"
gguf-threads: 8        # Number of CPU threads (0 = auto-detect)
gguf-gpu-layers: 32    # Number of layers to offload to GPU (0 = CPU only)
```

Then run:

```bash
./build/remembrances-mcp --config config.yaml
```

## Configuration Parameters

### `gguf-model-path`
- **Type**: String
- **Required**: Yes (if using GGUF)
- **Description**: Path to the GGUF model file
- **Example**: `/www/models/nomic-embed-text-v1.5.Q4_K_M.gguf`

### `gguf-threads`
- **Type**: Integer
- **Default**: 0 (auto-detect, defaults to 8)
- **Description**: Number of CPU threads to use for inference
- **Recommendation**: Set to number of physical CPU cores for best performance

### `gguf-gpu-layers`
- **Type**: Integer
- **Default**: 0 (CPU only)
- **Description**: Number of model layers to offload to GPU
- **Notes**: 
  - Requires GPU support (Metal/CUDA/ROCm)
  - Higher values = more GPU usage, faster inference
  - Set to 99 or -1 to offload all layers
  - Set to 0 for CPU-only inference

## Model Priority

When multiple embedding configurations are present, the priority is:

1. **GGUF** (local model) - Highest priority
2. **Ollama** (local server)
3. **OpenAI** (remote API) - Lowest priority

This means if you configure a GGUF model, it will be used even if Ollama or OpenAI are also configured.

## Getting GGUF Models

### Download Pre-quantized Models

You can download GGUF models from Hugging Face:

```bash
# Using huggingface-cli
huggingface-cli download \
  nomic-ai/nomic-embed-text-v1.5-GGUF \
  nomic-embed-text-v1.5.Q4_K_M.gguf \
  --local-dir ./models

# Or download manually from:
# https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF
```

### Recommended Quantization Levels

- **Q4_K_M**: Good balance of size and quality (recommended for most uses)
- **Q8_0**: Higher quality, larger size
- **Q2_K**: Smallest size, lower quality (for resource-constrained environments)

## Performance Tips

### CPU Optimization

1. **Set appropriate thread count**:
   ```bash
   --gguf-threads $(nproc)  # Use all CPU cores
   ```

2. **Use quantized models**: Q4_K_M provides good performance with minimal quality loss

### GPU Acceleration

1. **Metal (macOS)**:
   ```bash
   # Build with Metal support (default on macOS)
   make BUILD_TYPE=metal build
   
   # Offload all layers to GPU
   --gguf-gpu-layers 99
   ```

2. **CUDA (NVIDIA)**:
   ```bash
   # Build with CUDA support
   make BUILD_TYPE=cublas build
   
   # Offload layers to GPU
   --gguf-gpu-layers 32
   ```

3. **ROCm (AMD)**:
   ```bash
   # Build with ROCm support
   make BUILD_TYPE=hipblas build
   
   # Offload layers to GPU
   --gguf-gpu-layers 32
   ```

## Examples

### Example 1: Basic CPU Usage

```bash
./build/remembrances-mcp \
  --gguf-model-path ./models/nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-threads 8
```

### Example 2: GPU Acceleration (macOS Metal)

```bash
./build/remembrances-mcp \
  --gguf-model-path ./models/nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-threads 4 \
  --gguf-gpu-layers 99
```

### Example 3: GPU Acceleration (Linux CUDA)

```bash
./build/remembrances-mcp \
  --gguf-model-path ./models/nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-threads 8 \
  --gguf-gpu-layers 32
```

### Example 4: Full Configuration

Create `config.yaml`:

```yaml
# Server configuration
mcp-http: true
mcp-http-addr: ":3000"
mcp-http-endpoint: "/mcp"

# Database configuration
surrealdb-url: "ws://localhost:8000"
surrealdb-user: "root"
surrealdb-pass: "root"
surrealdb-namespace: "production"
surrealdb-database: "remembrances"

# GGUF embeddings (takes priority)
gguf-model-path: "/www/models/nomic-embed-text-v1.5.Q4_K_M.gguf"
gguf-threads: 8
gguf-gpu-layers: 32

# Fallback to Ollama if GGUF fails
ollama-url: "http://localhost:11434"
ollama-model: "nomic-embed-text"
```

Run with:
```bash
./build/remembrances-mcp --config config.yaml
```

## Troubleshooting

### Build Issues

1. **"llama.cpp submodule not found"**:
   ```bash
   cd ~/www/MCP/Remembrances/go-llama.cpp
   git submodule update --init --recursive
   ```

2. **CGO compilation errors**:
   - Ensure you have a C++ compiler installed
   - Check CGO flags: `make check-env`
   - Try cleaning and rebuilding: `make clean-all && make build`

### Runtime Issues

1. **"failed to load GGUF model"**:
   - Verify model file exists and is readable
   - Check model file is a valid GGUF format
   - Ensure sufficient RAM for model

2. **Slow performance**:
   - Increase `gguf-threads` to match CPU cores
   - Use a more quantized model (Q2_K, Q4_K_M)
   - Enable GPU layers if available

3. **Out of memory**:
   - Use a more quantized model (Q4_K_M instead of Q8_0)
   - Reduce `gguf-gpu-layers` if using GPU
   - Ensure sufficient system RAM

### Model Compatibility

1. **"unknown model architecture"**:
   - Ensure llama.cpp is up to date
   - Check if model architecture is supported
   - Try a different model variant

## Architecture Details

The GGUF embedder implementation:

1. **Loading**: Model is loaded once at startup using go-llama.cpp
2. **Thread Safety**: Mutex-protected to ensure safe concurrent access
3. **Batching**: Processes multiple texts sequentially with internal parallelism
4. **Dimension Detection**: Automatically detects embedding dimensions from model
5. **Resource Management**: Proper cleanup on shutdown via `Close()` method

## API Usage

### From Go Code

```go
import "github.com/madeindigio/remembrances-mcp/pkg/embedder"

// Create GGUF embedder
cfg := embedder.GGUFConfig{
    ModelPath: "/path/to/model.gguf",
    Threads:   8,
    GPULayers: 32,
}

emb, err := embedder.NewGGUFEmbedderFromConfig(cfg)
if err != nil {
    log.Fatal(err)
}
defer emb.Close()

// Generate embeddings
ctx := context.Background()
texts := []string{"hello world", "example text"}
embeddings, err := emb.EmbedDocuments(ctx, texts)
if err != nil {
    log.Fatal(err)
}

// Get embedding dimension
dim := emb.Dimension()
fmt.Printf("Embedding dimension: %d\n", dim)
```

## Performance Benchmarks

Approximate performance on different hardware (nomic-embed-text-v1.5 Q4_K_M):

| Hardware | Config | Texts/sec |
|----------|--------|-----------|
| M1 Pro (Metal) | 8 threads, 99 GPU layers | ~200 |
| M1 Pro (CPU) | 8 threads | ~80 |
| RTX 3090 (CUDA) | 8 threads, 32 GPU layers | ~300 |
| Ryzen 9 5950X | 16 threads | ~150 |
| i7-10700K | 8 threads | ~100 |

*Benchmarks are approximate and depend on text length and system configuration*

## License

This implementation uses go-llama.cpp which is MIT licensed. See the go-llama.cpp directory for details.
