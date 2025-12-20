---
title: "Configuration"
linkTitle: "Configuration"
weight: 2
description: >
  Configure Remembrances according to your needs
---

## Configuration Methods

Remembrances can be configured using:

1. **YAML configuration file** (recommended)
2. **Environment variables**
3. **Command line flags**

Priority: CLI flags > Environment variables > YAML file > Default values

## YAML Configuration

Create a configuration file at:
- **Linux**: `~/.config/remembrances/config.yaml`
- **macOS**: `~/Library/Application Support/remembrances/config.yaml`

Or specify a custom path with `--config`:

```yaml
# Database configuration, only if using embedded SurrealDB, otherwise comment this line
db-path: "./remembrances.db"

# GGUF Embeddings (recommended for portability and privacy)
gguf-model-path: "./nomic-embed-text-v1.5.Q4_K_M.gguf"
gguf-threads: 8
gguf-gpu-layers: 32

# Alternative: Ollama, if you prefer to use Ollama for embeddings (uncomment the following lines, you will need to comment the previous GGUF ones)
# ollama-url: "http://localhost:11434"
# ollama-model: "nomic-embed-text"

# Alternative: OpenAI (if you prefer to use OpenAI embeddings API - less privacy, but easier to use and doesn't require powerful hardware)
# openai-key: "sk-..."
# openai-model: "text-embedding-3-large"

# Transport options
# MCP Streamable HTTP is the recommended network transport for MCP tools.
mcp-http: false
mcp-http-addr: "3000"
mcp-http-endpoint: "/mcp"

# JSON HTTP API (useful for integration with other systems or agents that don't natively support MCP)
http: false
http-addr: ":8080"

# Knowledge base, path to the folder with markdown files to index and use as knowledge base, markdown files will also be generated for each web that Remembrances is required to save information in the knowledge base
knowledge-base: "./kb"
```

## Environment Variables

All options can be configured through environment variables with the `GOMEM_` prefix:

```bash
# GOMEM_DB_PATH: Path to the embedded SurrealDB database (default: ./remembrances.db)
export GOMEM_DB_PATH="./data.db"

# GOMEM_GGUF_MODEL_PATH: Path to the GGUF model file for embeddings (recommended for portability and privacy)
export GOMEM_GGUF_MODEL_PATH="./model.gguf"

# GOMEM_GGUF_THREADS: Number of threads to use for GGUF processing (0 = auto-detect)
export GOMEM_GGUF_THREADS=8

# GOMEM_GGUF_GPU_LAYERS: Number of GPU layers to offload for accelerated processing (0 = CPU only)
export GOMEM_GGUF_GPU_LAYERS=32

# GOMEM_OLLAMA_URL: Ollama server URL for embeddings (alternative to GGUF)
export GOMEM_OLLAMA_URL="http://localhost:11434"

# GOMEM_OLLAMA_MODEL: Ollama model to use for embeddings (e.g. nomic-embed-text)
export GOMEM_OLLAMA_MODEL="nomic-embed-text"

# GOMEM_OPENAI_KEY: OpenAI API key for embeddings (less privacy, but easier to use)
export GOMEM_OPENAI_KEY="sk-..."

# GOMEM_OPENAI_MODEL: OpenAI model for embeddings (e.g. text-embedding-3-large)
export GOMEM_OPENAI_MODEL="text-embedding-3-large"

# GOMEM_MCP_HTTP: Enable MCP Streamable HTTP transport (default: false)
export GOMEM_MCP_HTTP=false

# GOMEM_MCP_HTTP_ADDR: MCP Streamable HTTP server address (default: :3000)
export GOMEM_MCP_HTTP_ADDR=":3000"

# GOMEM_MCP_HTTP_ENDPOINT: MCP Streamable HTTP endpoint path (default: /mcp)
export GOMEM_MCP_HTTP_ENDPOINT="/mcp"

# GOMEM_HTTP: Enable JSON HTTP API for integration with other systems (default: false)
export GOMEM_HTTP=false

# GOMEM_HTTP_ADDR: HTTP server address (default: :8080)
export GOMEM_HTTP_ADDR=":8080"

# GOMEM_KNOWLEDGE_BASE: Path to the folder with markdown files for the knowledge base
export GOMEM_KNOWLEDGE_BASE="./kb"

# GOMEM_SURREALDB_URL: URL for remote SurrealDB instance (if not using embedded)
export GOMEM_SURREALDB_URL=""

# GOMEM_SURREALDB_USER: User for remote SurrealDB (default: root)
export GOMEM_SURREALDB_USER="root"

# GOMEM_SURREALDB_PASS: Password for remote SurrealDB (default: root)
export GOMEM_SURREALDB_PASS="root"
```

## Key Configuration Options

### Database (embedded or remote, you will need a SurrealDB server if using remote mode)

- `db-path`: Path to embedded SurrealDB (default: `./remembrances.db`)

And to access a remote SurrealDB instance:
- `surrealdb-url`: URL for remote SurrealDB instance
- `surrealdb-user`: Username (default: `root`)
- `surrealdb-pass`: Password (default: `root`)

### GGUF Embeddings

- `gguf-model-path`: Path to the GGUF model file
- `gguf-threads`: Number of threads (0 = auto-detect)
- `gguf-gpu-layers`: GPU layers to offload (0 = CPU only)

### Transport

- `mcp-http`: Enable MCP Streamable HTTP transport
- `mcp-http-addr`: MCP Streamable HTTP server address (default: `:3000`)
- `mcp-http-endpoint`: MCP endpoint path (default: `/mcp`)

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
```

### Code-Specific Embedding Models

For optimal code search results when using the [Code Indexing](/docs/code-indexing/) feature, you can configure a dedicated embedding model specialized for code:

```yaml
# Use the same provider options as the main embedder
# GGUF (recommended for local, privacy-focused deployments)
code-gguf-model-path: "./coderankembed.Q4_K_M.gguf"

# OR Ollama
# code-ollama-model: "jina/jina-embeddings-v3"

# OR OpenAI
# code-openai-model: "text-embedding-3-large"
```

**Recommended Code Embedding Models:**

| Provider | Model | Notes |
|----------|-------|-------|
| GGUF | CodeRankEmbed (Q4_K_M) | Best for local, privacy-focused code search |
| Ollama | jina-embeddings-v3 | Good balance of quality and speed |
| OpenAI | text-embedding-3-large | High quality, cloud-based |

**Fallback Behavior:** If a code-specific model is not configured, Remembrances automatically uses your default embedding model for code indexing.

**Environment Variables:**
```bash
export GOMEM_CODE_GGUF_MODEL_PATH="./coderankembed.Q4_K_M.gguf"
export GOMEM_CODE_OLLAMA_MODEL="jina/jina-embeddings-v3"
export GOMEM_CODE_OPENAI_MODEL="text-embedding-3-large"
```

### Configuration through CLI parameters

You have all the above configuration available through command line flags. Check the help with:

```bash
remembrances-mcp --help
```
```
<file_path>
remembrances-mcp/website/content/en/docs/configuration/_index.md
</file_path>