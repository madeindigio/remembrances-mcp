#!/bin/bash

# Start Remembrances-MCP with kelindar/search embeddings
# Using CPU-only (no GPU/Vulkan)

export GOMEM_SURREALDB_START_CMD="surreal start --user root --pass root surrealkv:///www/Remembrances/programming"
export GOMEM_LLAMA_GPU_LAYERS=0

/www/MCP/remembrances-mcp/remembrances-mcp \
  --surrealdb-url ws://localhost:8000 \
  --llama-model-path /www/Remembrances/nomic-embed-text-v1.5.Q4_K_M.gguf \
  --llama-gpu-layers 0 \
  --knowledge-base /www/MCP/remembrances-mcp/.serena/memories
