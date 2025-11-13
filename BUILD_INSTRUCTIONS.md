# Build Instructions for GGUF Support

This document provides step-by-step instructions for building Remembrances-MCP with GGUF embedding support.

## Prerequisites

### Required

1. **Go 1.21 or later**
   ```bash
   go version  # Should show 1.21 or higher
   ```

2. **C/C++ Compiler**
   - **Linux**: gcc/g++
     ```bash
     sudo apt-get install build-essential  # Debian/Ubuntu
     sudo yum groupinstall "Development Tools"  # RHEL/CentOS
     ```
   - **macOS**: Xcode Command Line Tools
     ```bash
     xcode-select --install
     ```

3. **Make**
   ```bash
   make --version  # Should be available
   ```

4. **Git** (with submodules support)
   ```bash
   git --version
   ```

### Optional (for GPU acceleration)

- **macOS**: Metal support is built-in
- **Linux with NVIDIA GPU**: CUDA Toolkit 11.0+
- **Linux with AMD GPU**: ROCm 5.0+
- **OpenBLAS**: For CPU optimization

## Initial Setup

### 1. Clone the Repository

```bash
git clone https://github.com/madeindigio/remembrances-mcp.git
cd remembrances-mcp
```

### 2. Initialize go-llama.cpp Submodules

The go-llama.cpp library is located at `/www/MCP/Remembrances/go-llama.cpp` and uses git submodules:

```bash
cd /www/MCP/Remembrances/go-llama.cpp
git submodule update --init --recursive
cd -
```

This will download the llama.cpp library which is required for GGUF support.

## Build Process

### Option 1: Using Makefile (Recommended)

The Makefile handles all the complexity of building llama.cpp and linking it with Go.

#### Basic Build (CPU on Linux, Metal on macOS)

```bash
make build
```

This will:
1. Build llama.cpp library with default settings
2. Set appropriate CGO flags
3. Compile the Go application
4. Create binary at `build/remembrances-mcp`

#### Build with GPU Support

**macOS with Metal (default)**:
```bash
make BUILD_TYPE=metal build
```

**Linux with CUDA (NVIDIA)**:
```bash
make BUILD_TYPE=cublas build
```

**Linux with ROCm (AMD)**:
```bash
make BUILD_TYPE=hipblas build
```

**With OpenBLAS**:
```bash
make BUILD_TYPE=openblas build
```

#### Check Build Environment

```bash
make check-env
```

This will display your build configuration including CGO flags.

### Option 2: Manual Build

If you prefer to build manually or the Makefile doesn't work for your setup:

#### Step 1: Build llama.cpp

```bash
cd /www/MCP/Remembrances/go-llama.cpp

# Default build
make libbinding.a

# Or with specific build type
BUILD_TYPE=cublas make libbinding.a  # CUDA
BUILD_TYPE=hipblas make libbinding.a  # ROCm
BUILD_TYPE=metal make libbinding.a    # Metal
BUILD_TYPE=openblas make libbinding.a # OpenBLAS
```

#### Step 2: Set CGO Environment Variables

**Linux**:
```bash
export CGO_ENABLED=1
export GO_LLAMA_DIR=/www/MCP/Remembrances/go-llama.cpp
export CGO_CFLAGS="-I$GO_LLAMA_DIR -I$GO_LLAMA_DIR/llama.cpp -I$GO_LLAMA_DIR/llama.cpp/common -I$GO_LLAMA_DIR/llama.cpp/ggml/include -I$GO_LLAMA_DIR/llama.cpp/include"
export CGO_LDFLAGS="-L$GO_LLAMA_DIR -L$GO_LLAMA_DIR/build/bin -L$GO_LLAMA_DIR/build/common -lllama -lcommon -lggml -lggml-base -lm -lstdc++ -lpthread"
```

**macOS**:
```bash
export CGO_ENABLED=1
export GO_LLAMA_DIR=/www/MCP/Remembrances/go-llama.cpp
export CGO_CFLAGS="-I$GO_LLAMA_DIR -I$GO_LLAMA_DIR/llama.cpp -I$GO_LLAMA_DIR/llama.cpp/common -I$GO_LLAMA_DIR/llama.cpp/ggml/include -I$GO_LLAMA_DIR/llama.cpp/include"
export CGO_LDFLAGS="-L$GO_LLAMA_DIR -L$GO_LLAMA_DIR/build/bin -L$GO_LLAMA_DIR/build/common -lllama -lcommon -lggml -lggml-base -lm -lstdc++ -lpthread -framework Accelerate -framework Foundation -framework Metal -framework MetalKit -framework MetalPerformanceShaders"
```

#### Step 3: Build Go Application

```bash
cd /www/MCP/remembrances-mcp
go build -v -o build/remembrances-mcp ./cmd/remembrances-mcp
```

## Verification

### Test the Build

```bash
./build/remembrances-mcp --version
```

### Download a Test Model

```bash
# Create models directory
mkdir -p models

# Download nomic-embed-text-v1.5 Q4_K_M (recommended)
wget https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf -P models/
```

### Run a Quick Test

```bash
# Test with CPU only
./build/remembrances-mcp \
  --gguf-model-path ./models/nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-threads 4

# Test with GPU (if available)
./build/remembrances-mcp \
  --gguf-model-path ./models/nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-threads 4 \
  --gguf-gpu-layers 32
```

### Run the Test Script

```bash
./scripts/test-gguf.sh ./models/nomic-embed-text-v1.5.Q4_K_M.gguf 8 32
```

## Troubleshooting

### Issue: "llama.cpp submodule not found"

**Solution**:
```bash
cd /www/MCP/Remembrances/go-llama.cpp
git submodule update --init --recursive
```

### Issue: "undefined reference to..." errors

**Solution**: This usually means CGO flags are not set correctly. Verify:
```bash
make check-env
```

Also ensure llama.cpp is built:
```bash
cd /www/MCP/Remembrances/go-llama.cpp
make libbinding.a
```

### Issue: "ld: library not found"

**Solution**: The llama.cpp libraries are not in the expected location. Rebuild:
```bash
make clean-all
make build
```

### Issue: CGO compilation takes very long

**Solution**: This is normal for the first build as llama.cpp is being compiled. Subsequent builds will be faster.

### Issue: "failed to load GGUF model"

**Solutions**:
1. Verify model file exists and is readable
2. Check model is valid GGUF format
3. Ensure sufficient RAM for model size
4. Try with a smaller quantized model (Q4_K_M instead of Q8_0)

### Issue: Out of memory when loading model

**Solutions**:
1. Use a more quantized model (Q2_K, Q4_K_M)
2. Reduce GPU layers: `--gguf-gpu-layers 0`
3. Close other applications
4. Check available RAM: `free -h` (Linux) or Activity Monitor (macOS)

### Issue: Slow embedding generation

**Solutions**:
1. Increase threads: `--gguf-threads $(nproc)`
2. Enable GPU layers: `--gguf-gpu-layers 32` (or higher)
3. Use a more quantized model
4. Verify GPU is being used (check GPU usage during inference)

## Platform-Specific Notes

### macOS

- Metal GPU acceleration is enabled by default
- Make sure Xcode Command Line Tools are installed
- For best performance, set `--gguf-gpu-layers 99` to offload all layers

### Linux with NVIDIA GPU

1. Install CUDA Toolkit:
   ```bash
   # Check NVIDIA driver
   nvidia-smi
   
   # Install CUDA (example for Ubuntu)
   wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/cuda-keyring_1.0-1_all.deb
   sudo dpkg -i cuda-keyring_1.0-1_all.deb
   sudo apt-get update
   sudo apt-get install cuda
   ```

2. Build with CUDA:
   ```bash
   make BUILD_TYPE=cublas build
   ```

3. Set appropriate GPU layers based on VRAM

### Linux with AMD GPU

1. Install ROCm:
   ```bash
   # Follow official ROCm installation guide for your distro
   # https://rocmdocs.amd.com/en/latest/Installation_Guide/Installation-Guide.html
   ```

2. Build with ROCm:
   ```bash
   make BUILD_TYPE=hipblas build
   ```

### Docker Build

Coming soon: Docker images with pre-built llama.cpp for easy deployment.

## Clean Build

If you encounter issues, try a clean build:

```bash
# Clean everything
make clean-all

# Rebuild from scratch
make build
```

## Development Build

For development with race detector:

```bash
make build-dev
./build/remembrances-mcp-dev
```

## Cross-Compilation

Cross-compilation requires llama.cpp to be built for each target platform. This is complex and not fully supported yet. Use platform-native builds for now.

## Next Steps

After successful build:

1. **Configure your embedder**: See `config.sample.yaml` for examples
2. **Run tests**: `GGUF_TEST_MODEL_PATH=./models/model.gguf go test ./pkg/embedder`
3. **Read full docs**: See `docs/GGUF_EMBEDDINGS.md`
4. **Start using**: Run the server with your GGUF model!

## Getting Help

If you encounter issues not covered here:

1. Check `docs/GGUF_EMBEDDINGS.md` for detailed documentation
2. Review `GGUF_IMPLEMENTATION_SUMMARY.md` for technical details
3. Open an issue on GitHub with:
   - Output of `make check-env`
   - Full error message
   - Your OS and architecture
   - Go version

## Build System Commands Reference

```bash
make build           # Build with GGUF support
make llama-cpp       # Build llama.cpp only
make llama-cpp-clean # Clean llama.cpp build
make clean           # Clean Go build artifacts
make clean-all       # Deep clean (including llama.cpp)
make test            # Run tests
make run             # Build and run
make check-env       # Display build environment
make help            # Show all available commands
```

## Success!

If everything works, you should see:

```bash
$ ./build/remembrances-mcp --version
v0.40.0 (commit_hash)

$ ./build/remembrances-mcp --gguf-model-path ./models/model.gguf
Loading GGUF model: ./models/model.gguf
Model loaded successfully
Embedding dimension: 768
Server started...
```

Congratulations! You're ready to use GGUF embeddings in Remembrances-MCP! 🎉
