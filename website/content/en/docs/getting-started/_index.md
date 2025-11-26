---
title: "Getting Started"
linkTitle: "Getting Started"
weight: 1
description: >
  Install and run Remembrances MCP in minutes
---

## Prerequisites

- Go 1.20 or later
- Git
- (Optional) CUDA/ROCm for GPU acceleration

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/madeindigio/remembrances-mcp.git
cd remembrances-mcp
```

### 2. Build the Project

```bash
make build
```

This will:
- Install Go dependencies
- Compile llama.cpp with GPU support (if available)
- Build the `remembrances-mcp` binary

### 3. Download a GGUF Model

Download the recommended nomic-embed-text-v1.5 model:

```bash
wget https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf
```

Other recommended models:
- **nomic-embed-text-v1.5** (768 dimensions) - Best balance
- **all-MiniLM-L6-v2** (384 dimensions) - Faster, smaller

### 4. Run the Server

```bash
./run-remembrances.sh \
  --gguf-model-path ./nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-threads 8 \
  --gguf-gpu-layers 32
```

The server will start in stdio mode, ready to accept MCP connections.

## Quick Test

Test the server with a simple fact:

```bash
# In another terminal, use the MCP client
echo '{"method":"tools/call","params":{"name":"remembrance_save_fact","arguments":{"key":"test","value":"Hello World"}}}' | ./remembrances-mcp --gguf-model-path ./nomic-embed-text-v1.5.Q4_K_M.gguf
```

## Next Steps

- [Configure the server](../configuration/) for your needs
- [Learn about GGUF models](../gguf-models/) and optimization
- [Explore the MCP API](../mcp-api/) and available tools
