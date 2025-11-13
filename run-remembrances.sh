#!/bin/bash
# Wrapper script to run remembrances-mcp with GGUF support
# This script sets up the correct library paths for llama.cpp

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Path to go-llama.cpp libraries
LLAMA_LIB_DIR="/www/MCP/Remembrances/go-llama.cpp/build/bin"
LLAMA_COMMON_DIR="/www/MCP/Remembrances/go-llama.cpp/build/common"

# Add llama.cpp library directories to LD_LIBRARY_PATH
export LD_LIBRARY_PATH="${LLAMA_LIB_DIR}:${LLAMA_COMMON_DIR}:${LD_LIBRARY_PATH}"

# Verify libraries exist
if [ ! -f "${LLAMA_LIB_DIR}/libllama.so" ]; then
    echo "Error: llama.cpp libraries not found at ${LLAMA_LIB_DIR}"
    echo "Please build llama.cpp first:"
    echo "  cd /www/MCP/Remembrances/go-llama.cpp"
    echo "  make libbinding.a"
    exit 1
fi

# Run the application with all passed arguments
exec "${SCRIPT_DIR}/build/remembrances-mcp" "$@"
