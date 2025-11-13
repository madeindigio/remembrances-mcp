#!/bin/bash
# Test script for GGUF embeddings support

set -e

echo "========================================"
echo "GGUF Embeddings Test Script"
echo "========================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if model path is provided
if [ -z "$1" ]; then
    echo -e "${RED}Error: Model path not provided${NC}"
    echo ""
    echo "Usage: $0 <path-to-gguf-model> [threads] [gpu-layers]"
    echo ""
    echo "Example:"
    echo "  $0 /path/to/nomic-embed-text-v1.5.Q4_K_M.gguf 8 32"
    echo ""
    echo "You can download a model with:"
    echo "  wget https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf"
    exit 1
fi

MODEL_PATH="$1"
THREADS="${2:-8}"
GPU_LAYERS="${3:-0}"

# Check if model exists
if [ ! -f "$MODEL_PATH" ]; then
    echo -e "${RED}Error: Model file not found: $MODEL_PATH${NC}"
    exit 1
fi

echo -e "${GREEN}Model found: $MODEL_PATH${NC}"
echo "Threads: $THREADS"
echo "GPU Layers: $GPU_LAYERS"
echo ""

# Check if llama.cpp is built
LLAMA_DIR="/www/MCP/Remembrances/go-llama.cpp"
if [ ! -f "$LLAMA_DIR/build/bin/libllama.a" ]; then
    echo -e "${YELLOW}Warning: llama.cpp library not found${NC}"
    echo "Building llama.cpp..."
    cd "$LLAMA_DIR"
    make libbinding.a
    cd - > /dev/null
    echo -e "${GREEN}llama.cpp built successfully${NC}"
    echo ""
fi

# Run the example
echo "========================================"
echo "Running GGUF embeddings example..."
echo "========================================"
echo ""

cd "$(dirname "$0")/.."

# Build the example if needed
if [ ! -f "examples/gguf_embeddings" ]; then
    echo "Building example..."
    export CGO_ENABLED=1
    export CGO_CFLAGS="-I$LLAMA_DIR -I$LLAMA_DIR/llama.cpp -I$LLAMA_DIR/llama.cpp/common -I$LLAMA_DIR/llama.cpp/ggml/include -I$LLAMA_DIR/llama.cpp/include"
    export CGO_LDFLAGS="-L$LLAMA_DIR -L$LLAMA_DIR/build/bin -L$LLAMA_DIR/build/common -lllama -lcommon -lggml -lggml-base -lm -lstdc++ -lpthread"

    if [[ "$OSTYPE" == "darwin"* ]]; then
        export CGO_LDFLAGS="$CGO_LDFLAGS -framework Accelerate -framework Foundation -framework Metal -framework MetalKit -framework MetalPerformanceShaders"
    fi

    go build -o examples/gguf_embeddings examples/gguf_embeddings.go
    echo -e "${GREEN}Example built successfully${NC}"
    echo ""
fi

# Run tests
echo "Test 1: Single embedding"
echo "------------------------"
./examples/gguf_embeddings \
    --model "$MODEL_PATH" \
    --threads "$THREADS" \
    --gpu-layers "$GPU_LAYERS" \
    --text "Hello, world! This is a test of GGUF embeddings."

echo ""
echo ""
echo "Test 2: Benchmark"
echo "------------------------"
./examples/gguf_embeddings \
    --model "$MODEL_PATH" \
    --threads "$THREADS" \
    --gpu-layers "$GPU_LAYERS" \
    --benchmark

echo ""
echo "========================================"
echo -e "${GREEN}All tests completed successfully!${NC}"
echo "========================================"
echo ""
echo "Next steps:"
echo "  1. Try running the full server with GGUF:"
echo "     ./build/remembrances-mcp --gguf-model-path $MODEL_PATH --gguf-threads $THREADS --gguf-gpu-layers $GPU_LAYERS"
echo ""
echo "  2. Run the Go tests:"
echo "     GGUF_TEST_MODEL_PATH=$MODEL_PATH go test -v ./pkg/embedder -run TestGGUF"
echo ""
echo "  3. Run benchmarks:"
echo "     GGUF_TEST_MODEL_PATH=$MODEL_PATH go test -bench=. ./pkg/embedder"
echo ""
