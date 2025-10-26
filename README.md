# Remembrances-MCP

Remembrances-MCP is a Go-based MCP server that provides long-term memory capabilities to AI agents. It supports multiple memory layers (key-value, vector/RAG, graph database) using SurrealDB, and can manage knowledge bases via Markdown files.

## Features

- MCP server for AI agent memory
- SurrealDB support (embedded or external)
- Knowledge base management with Markdown files
- Embedding generation via Ollama (local), OpenAI API, or kelindar/search (embedded BERT models)
- Multiple transport options: stdio (default), SSE, and HTTP JSON API

## Supported Platforms

Cross-platform support with pre-built binaries for:

- **Linux**: x86_64 (amd64), ARM64
- **macOS**: Intel (x86_64), Apple Silicon (ARM64)
- **Windows**: x86_64 (amd64)

All binaries use kelindar/search (no cgo required) for standalone deployment with BERT model support.

## 🚀 Quick Start

### Pre-built Binaries

Download the latest release for your platform from [GitHub Releases](https://github.com/madeindigio/remembrances-mcp/releases).

#### Native Library Requirement

**Important:** This application uses `kelindar/search` for BERT embeddings, which requires a native library at runtime:
- **Linux**: `libllama_go.so`
- **macOS**: `libllama_go.dylib`
- **Windows**: `llama_go.dll`

**Installation Options:**

1. **Automatic Installation (Recommended):**
   ```bash
   # Download the native library for your platform from:
   # https://github.com/kelindar/search/releases/latest
   
   # Linux/macOS - use the installation script:
   sudo ./scripts/install-with-library.sh
   ```

2. **Manual Installation:**
   ```bash
   # Linux
   sudo cp libllama_go.so /usr/local/lib/
   sudo ldconfig
   
   # macOS
   sudo cp libllama_go.dylib /usr/local/lib/
   
   # Windows
   # Copy llama_go.dll to C:\Windows\System32\ or add to PATH
   ```

3. **Local Directory (No Installation):**
   ```bash
   # Place the library in the same directory as the binary
   # and set the library path:
   export LD_LIBRARY_PATH=.:$LD_LIBRARY_PATH  # Linux
   export DYLD_LIBRARY_PATH=.:$DYLD_LIBRARY_PATH  # macOS
   ```

For detailed information about static compilation and deployment strategies, see [docs/KELINDAR_SEARCH_STATIC_LINKING.md](docs/KELINDAR_SEARCH_STATIC_LINKING.md).

### Build from Source

```bash
# Quick build (current platform, ~1 minute - no C++ compilation!)
make build

# Multi-platform build (all platforms, ~2-3 minutes)
make release-multi-snapshot
```

See [BUILD_QUICK_START.md](docs/BUILD_QUICK_START.md) for detailed build instructions.

## Usage

Run the server with CLI flags or environment variables:

```bash
go run ./cmd/remembrances-mcp/main.go [flags]
```

### CLI Flags

- `--sse` (default: false): Enable SSE transport
- `--sse-addr` (default: :3000): Address to bind SSE transport (host:port). Can also be set via `GOMEM_SSE_ADDR`.
- `--http` (default: false): Enable HTTP JSON API transport
- `--http-addr` (default: :8080): Address to bind HTTP transport (host:port). Can also be set via `GOMEM_HTTP_ADDR`.
- `--rest-api-serve`: Enable REST API server
- `--knowledge-base`: Path to knowledge base directory
- `--db-path`: Path to embedded SurrealDB database (default: ./remembrances.db)
- `--surrealdb-url`: URL for remote SurrealDB instance
- `--surrealdb-user`: SurrealDB username (default: root)
- `--surrealdb-pass`: SurrealDB password (default: root)
- `--surrealdb-namespace`: SurrealDB namespace (default: test)
- `--surrealdb-database`: SurrealDB database (default: test)
- `--ollama-url`: Ollama server URL (default: http://localhost:11434)
- `--ollama-model`: Ollama model for embeddings
- `--openai-key`: OpenAI API key
- `--openai-url`: OpenAI base URL (default: https://api.openai.com/v1)
- `--openai-model`: OpenAI model for embeddings (default: text-embedding-3-large)
- `--search-model-path`: Path to .gguf BERT model file (e.g., nomic-embed-text-v1.5.Q4_K_M.gguf)
- `--search-dimension`: Dimension of embeddings (default: 768)
- `--search-gpu-layers`: Number of GPU layers to offload (0 = CPU only)
- `--llama-model-path`: DEPRECATED - use --search-model-path instead
- `--llama-dimension`: DEPRECATED - use --search-dimension instead
- `--llama-gpu-layers`: DEPRECATED - use --search-gpu-layers instead

- `--surrealdb-start-cmd`: Optional command to start an external SurrealDB instance when an initial connection cannot be established. Can also be set via `GOMEM_SURREALDB_START_CMD`.

### Environment Variables

All flags can be set via environment variables prefixed with `GOMEM_` and dashes replaced by underscores. For example:

- `GOMEM_SSE`
- `GOMEM_SSE_ADDR` (e.g. `:3000` or `0.0.0.0:3000`)
- `GOMEM_HTTP`
- `GOMEM_HTTP_ADDR` (e.g. `:8080` or `0.0.0.0:8080`)
- `GOMEM_REST_API_SERVE`
- `GOMEM_KNOWLEDGE_BASE`
- `GOMEM_DB_PATH`
- `GOMEM_SURREALDB_URL`
- `GOMEM_SURREALDB_USER`
- `GOMEM_SURREALDB_PASS`
- `GOMEM_SURREALDB_NAMESPACE`
- `GOMEM_SURREALDB_DATABASE`
- `GOMEM_OLLAMA_URL`
- `GOMEM_OLLAMA_MODEL`
- `GOMEM_OPENAI_KEY`
- `GOMEM_OPENAI_URL`
- `GOMEM_OPENAI_MODEL`
- `GOMEM_SEARCH_MODEL_PATH` (recommended for BERT models)
- `GOMEM_SEARCH_DIMENSION`
- `GOMEM_SEARCH_GPU_LAYERS`
- `GOMEM_LLAMA_MODEL_PATH` (deprecated - auto-migrated to search)
- `GOMEM_LLAMA_DIMENSION` (deprecated)
- `GOMEM_LLAMA_GPU_LAYERS` (deprecated)
- `GOMEM_LLAMA_CONTEXT`

Additionally, there is an optional environment variable/flag to help auto-start a local SurrealDB when the server cannot connect at startup:

- `GOMEM_SURREALDB_START_CMD` / `--surrealdb-start-cmd`

Example usage (start command provided via env):

```bash
export GOMEM_SURREALDB_START_CMD="surreal start --user root --pass root surrealkv:///path/to/surreal_data"
go run ./cmd/remembrances-mcp/main.go --knowledge-base ./kb

# Start SSE transport on a custom address via CLI flag
go run ./cmd/remembrances-mcp/main.go --sse --sse-addr=":3000"

# Or via environment variable
GOMEM_SSE=true GOMEM_SSE_ADDR=":3000" go run ./cmd/remembrances-mcp/main.go --sse

# Start HTTP JSON API transport
go run ./cmd/remembrances-mcp/main.go --http --http-addr=":8080"

# Or via environment variable
GOMEM_HTTP=true GOMEM_HTTP_ADDR=":8080" go run ./cmd/remembrances-mcp/main.go
```

### Transport Options

The server supports three transport modes:

1. **stdio (default)**: Standard input/output for MCP protocol communication
2. **SSE**: Server-Sent Events for web-based clients 
3. **HTTP**: Simple HTTP JSON API for direct REST-style access

#### HTTP Transport Endpoints

When using `--http`, the server exposes these endpoints:

- `GET /health` - Health check endpoint
- `GET /mcp/tools` - List available MCP tools
- `POST /mcp/tools/call` - Call an MCP tool

Example HTTP usage:

```bash
# List available tools
curl http://localhost:8080/mcp/tools

# Call a tool
curl -X POST http://localhost:8080/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name": "remembrance_save_fact", "arguments": {"key": "test", "value": "example"}}'
```

Behavior: when the program starts it will attempt to connect to SurrealDB. If the connection fails and a start command was provided, the program will spawn the provided command (using `/bin/sh -c "<cmd>"`), stream its stdout/stderr to the running process, and poll the database connection for up to 30 seconds with exponential backoff. If the database becomes available the server continues startup. If starting the command fails or the database remains unreachable after the timeout, the program logs a descriptive error and exits.

## Requirements

- Go 1.20+
- SurrealDB (embedded or external)
- Ollama (optional, for local embeddings)
- OpenAI API key (optional, for cloud embeddings)
- llama.cpp (optional, for embedded embeddings)

## Embedding Providers

### Priority Order
The system automatically selects embedding providers in this priority order:
1. **llama.cpp** (if `--llama-model-path` is provided)
2. **Ollama** (if `--ollama-url` is provided)
3. **OpenAI** (if `--openai-key` is provided)

### llama.cpp Embedded Embeddings

llama.cpp provides **local, private, and cost-effective** embedding generation without requiring external services.

#### Benefits
- **Privacy**: All processing happens locally on your machine
- **Offline Capability**: No internet connection required after model download
- **Cost Efficiency**: No API charges
- **Performance**: Fast local inference with optional GPU acceleration
- **Flexibility**: Use any compatible .gguf embedding model

#### Configuration
```bash
# Using CLI flags
./remembrances-mcp \
  --llama-model-path ./models/all-MiniLM-L6-v2.gguf \
  --llama-dimension 384 \
  --llama-context 512 \
  --surrealdb-start-cmd 'surreal start --user root --pass root ws://localhost:8000'

# Using environment variables
export GOMEM_LLAMA_MODEL_PATH=./models/all-MiniLM-L6-v2.gguf
export GOMEM_LLAMA_DIMENSION=384
export GOMEM_LLAMA_CONTEXT=512
export GOMEM_SURREALDB_START_CMD='surreal start --user root --pass root ws://localhost:8000'
./remembrances-mcp
```

#### Context Size Impact

The `--llama-context` parameter significantly affects performance and resource usage:

| Context Size | Memory Usage | Speed | Use Cases | Example Text |
|--------------|---------------|--------|------------|--------------|
| 128-256 | ~50MB | ⚡ Very Fast | Single words, short phrases | "database", "user", "error" |
| 512 | ~100MB | 🚀 Fast | Complete sentences, short paragraphs | "The user database connection failed due to invalid credentials" |
| 1024 | ~200MB | 🐢 Moderate | Long paragraphs, technical documents | "The system attempted to connect to PostgreSQL database using provided credentials, but authentication failed because the user account had been locked due to multiple failed login attempts" |
| 2048+ | ~400MB+ | 🐌 Slow | Full documents, articles, books | Complete technical documentation or research papers |

**Performance Trade-offs:**
- **Smaller Context**: Faster processing, lower memory usage, ideal for short texts
- **Larger Context**: Better semantic understanding, higher memory usage, ideal for long documents

#### Recommended Models

| Model | Dimensions | Size | Use Case | Download Command |
|--------|-------------|-------|-----------|-----------------|
| all-MiniLM-L6-v2 | 384 | General purpose, fast inference | `wget https://huggingface.co/TheBloke/all-MiniLM-L6-v2-GGUF/resolve/main/all-MiniLM-L6-v2.Q4_K_M.gguf` |
| nomic-embed-text | 768 | High quality, balanced | `wget https://huggingface.co/TheBloke/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf` |
| bge-large-en-v1.5 | 1024 | English text, high accuracy | `wget https://huggingface.co/TheBloke/bge-large-en-v1.5-GGUF/resolve/main/bge-large-en-v1.5.Q4_K_M.gguf` |
| multilingual-e5-large | 1024 | Multilingual support | `wget https://huggingface.co/TheBloke/multilingual-e5-large-GGUF/resolve/main/multilingual-e5-large.Q4_K_M.gguf` |

**Important Note**: The vector dimension must match your database schema. The current schema is configured for 768 dimensions. Use models with 768 dimensions or modify the schema accordingly.

#### GPU Acceleration

For NVIDIA GPUs with CUDA support:

```bash
./remembrances-mcp \
  --llama-model-path ./models/model.gguf \
  --llama-gpu-layers 20 \
  --llama-threads 4
```

- `--llama-gpu-layers 0`: CPU only (default)
- `--llama-gpu-layers 20`: Offload 20 layers to GPU
- `--llama-gpu-layers -1`: Offload all possible layers to GPU

#### Thread Configuration

```bash
# Auto-detect CPU cores (recommended)
--llama-threads 0

# Manual configuration
--llama-threads 4  # Use 4 CPU threads
--llama-threads 8  # Use 8 CPU threads
```

## Build

```bash
go mod tidy
go build -o remembrances-mcp ./cmd/remembrances-mcp
```

## Example

```bash
GOMEM_OPENAI_KEY=sk-xxx \
GOMEM_DB_PATH=./data.db \
go run ./cmd/remembrances-mcp/main.go --knowledge-base ./kb --rest-api-serve
```

## License

See [LICENSE.txt](LICENSE.txt).

## Tasks

### build

Build the project

```bash
go mod tidy
go build -o dist/remembrances-mcp ./cmd/remembrances-mcp
```

### starts-surrealdb

Starts the SurrealDB instance

interactive: true
```bash
surreal start --user root --pass root surrealkv:///www/MCP/remembrances-mcp/surreal_data
```

### run-tests

Runs the test suite

interactive: true

```bash
./tests/run_all.sh
```

### tag

Deploys a new tag for the repo.

Specify major/minor/patch with VERSION

Env: PRERELEASE=0, VERSION=minor, FORCE_VERSION=0
Inputs: VERSION, PRERELEASE, FORCE_VERSION


```
# https://github.com/unegma/bash-functions/blob/main/update.sh

CURRENT_VERSION=`git describe --abbrev=0 --tags 2>/dev/null`
CURRENT_VERSION_PARTS=(${CURRENT_VERSION//./ })
VNUM1=${CURRENT_VERSION_PARTS[0]}
# remove v
VNUM1=${VNUM1:1}
VNUM2=${CURRENT_VERSION_PARTS[1]}
VNUM3=${CURRENT_VERSION_PARTS[2]}

if [[ $VERSION == 'major' ]]
then
  VNUM1=$((VNUM1+1))
  VNUM2=0
  VNUM3=0
elif [[ $VERSION == 'minor' ]]
then
  VNUM2=$((VNUM2+1))
  VNUM3=0
elif [[ $VERSION == 'patch' ]]
then
  VNUM3=$((VNUM3+1))
else
  echo "Invalid version"
  exit 1
fi

NEW_TAG="v$VNUM1.$VNUM2.$VNUM3"

# if command convco is available, use it to check the version
if command -v convco &> /dev/null
then
  # if the version is a prerelease, add the prerelease tag
  if [[ $PRERELEASE == '1' ]]
  then
    NEW_TAG=v$(convco version -b --prerelease)
  else
    NEW_TAG=v$(convco version -b)
  fi
fi

# if $FORCE_VERSION is different to 0 then use it as the version
if [[ $FORCE_VERSION != '0' ]]
then
  NEW_TAG=v$FORCE_VERSION
fi

echo Adding git tag with version ${NEW_TAG}
git tag ${NEW_TAG}
git push origin ${NEW_TAG}
```

### changelog

Generate a changelog for the repo.

```
convco changelog > CHANGELOG.md
git add CHANGELOG.md
git commit -m "Update changelog"
git push
```

### release

Releasing a new version into the repo.

```
goreleaser release --clean --skip sign
```

### release-snapshot

Releasing a new snapshot version into the repo.

```
goreleaser release --snapshot --skip sign --clean
```