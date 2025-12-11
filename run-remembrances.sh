#!/bin/bash
# Wrapper script to run remembrances-mcp with GGUF support
# This script sets up the correct library paths for llama.cpp

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Use local build directory for libraries
BUILD_DIR="${SCRIPT_DIR}/build"

# Add build directory to LD_LIBRARY_PATH
export LD_LIBRARY_PATH="${BUILD_DIR}:${LD_LIBRARY_PATH}"

# Verify libraries exist
if [ ! -f "${BUILD_DIR}/libllama.so" ]; then
    echo "Error: llama.cpp libraries not found at ${BUILD_DIR}"
    echo "Please build the project first:"
    echo "  make BUILD_TYPE=cuda build"
    exit 1
fi

# Run the application with all passed arguments
exec "${SCRIPT_DIR}/build/remembrances-mcp" "$@"
