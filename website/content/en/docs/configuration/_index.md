---
title: "Configuration"
linkTitle: "Configuration"
weight: 2
description: >
  Configure Remembrances MCP for your needs
---

## Configuration Methods

Remembrances MCP can be configured using:

1. **YAML configuration file** (recommended)
2. **Environment variables**
3. **Command-line flags**

Priority: CLI flags > Environment variables > YAML file > Defaults

## YAML Configuration

Create a configuration file at:
- **Linux**: `~/.config/remembrances/config.yaml`
- **macOS**: `~/Library/Application Support/remembrances/config.yaml`

Or specify a custom path with `--config`:

```yaml
# Database configuration
db-path: "./remembrances.db"

# GGUF embeddings (recommended)
gguf-model-path: "./nomic-embed-text-v1.5.Q4_K_M.gguf"
gguf-threads: 8
gguf-gpu-layers: 32

# Alternative: Ollama
# ollama-url: "http://localhost:11434"
# ollama-model: "nomic-embed-text"

# Alternative: OpenAI
# openai-key: "sk-..."
# openai-model: "text-embedding-3-large"

# Transport options
sse: false
sse-addr: ":3000"
http: false
http-addr: ":8080"

# Knowledge base
knowledge-base: "./kb"
```

## Environment Variables

All options can be set via environment variables with `GOMEM_` prefix:

```bash
export GOMEM_GGUF_MODEL_PATH="./model.gguf"
export GOMEM_GGUF_THREADS=8
export GOMEM_GGUF_GPU_LAYERS=32
export GOMEM_DB_PATH="./data.db"
```

## Key Configuration Options

### Database

- `db-path`: Path to embedded SurrealDB (default: `./remembrances.db`)
- `surrealdb-url`: URL for remote SurrealDB instance
- `surrealdb-user`: Username (default: `root`)
- `surrealdb-pass`: Password (default: `root`)

### GGUF Embeddings

- `gguf-model-path`: Path to GGUF model file
- `gguf-threads`: Number of threads (0 = auto-detect)
- `gguf-gpu-layers`: GPU layers to offload (0 = CPU only)

### Transport

- `sse`: Enable SSE transport
- `sse-addr`: SSE server address (default: `:3000`)
- `http`: Enable HTTP JSON API
- `http-addr`: HTTP server address (default: `:8080`)

## Example Configurations

### Local GGUF with GPU

```yaml
gguf-model-path: "./nomic-embed-text-v1.5.Q4_K_M.gguf"
gguf-threads: 8
gguf-gpu-layers: 32
db-path: "./remembrances.db"
```

### Ollama Integration

```yaml
ollama-url: "http://localhost:11434"
ollama-model: "nomic-embed-text"
db-path: "./remembrances.db"
```

### HTTP API Mode

```yaml
http: true
http-addr: ":8080"
gguf-model-path: "./model.gguf"
gguf-gpu-layers: 32
```

## See Also

- [GGUF Models](../gguf-models/) - Model selection and optimization
- [MCP API](../mcp-api/) - Available tools and endpoints
