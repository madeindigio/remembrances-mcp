# Remembrances-MCP

Remembrances-MCP is a Go-based MCP server that provides long-term memory capabilities to AI agents. It supports multiple memory layers (key-value, vector/RAG, graph database) using SurrealDB, and can manage knowledge bases via Markdown files.

## Features

- MCP server for AI agent memory
- SurrealDB support (embedded or external)
- Knowledge base management with Markdown files
- **Embedding generation via:**
  - **GGUF models (local, privacy-first, GPU accelerated)** ⭐ NEW
  - Ollama (local server)
  - OpenAI API (remote)
- Multiple transport options: stdio (default), SSE, and HTTP JSON API

## 🚀 GGUF Embeddings (NEW)

Remembrances-MCP now supports loading local GGUF embedding models directly! This provides:

- **🔒 Privacy**: All embeddings generated locally, no data sent externally
- **⚡ Performance**: Direct model inference without network latency
- **💰 Cost**: No API costs for embedding generation
- **🎯 Flexibility**: Support for quantized models (Q4_K_M, Q8_0, etc.)
- **🖥️ GPU Acceleration**: Metal (macOS), CUDA (NVIDIA), ROCm (AMD)

### Quick Start with GGUF

```bash
# 1. Build the project (compiles llama.cpp automatically)
make build

# 2. Download a GGUF model
# Example: nomic-embed-text-v1.5 (768 dimensions)
wget https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf

# 3. Run with GGUF model (using wrapper script)
./run-remembrances.sh \
  --gguf-model-path ./nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-threads 8 \
  --gguf-gpu-layers 32
  
# Alternative: Set LD_LIBRARY_PATH manually
export LD_LIBRARY_PATH=~/www/MCP/Remembrances/go-llama.cpp/build/bin:$LD_LIBRARY_PATH
./build/remembrances-mcp \
  --gguf-model-path ./nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-threads 8 \
  --gguf-gpu-layers 32
```

**📖 Full Documentation**: See [docs/GGUF_EMBEDDINGS.md](docs/GGUF_EMBEDDINGS.md) for detailed instructions, performance tips, and troubleshooting.

## 🔍 Code Indexing System (NEW)

Remembrances-MCP includes a powerful **Code Indexing System** that uses Tree-sitter for multi-language AST parsing with semantic embeddings. This allows AI agents to:

- **Index codebases** across 14+ languages (Go, TypeScript, JavaScript, Python, Rust, Java, C/C++, PHP, Ruby, Swift, Kotlin, and more)
- **Search semantically** for code symbols using natural language queries
- **Navigate code** by finding definitions, references, and call hierarchies
- **Manipulate code** by renaming symbols across entire codebases

### Quick Start

```bash
# 1. Index a project
# Use the MCP tool: code_index_project
{
  "project_name": "my-project",
  "root_path": "/path/to/project",
  "languages": ["go", "typescript"]
}

# 2. Search for code
# Use: code_semantic_search
{
  "project_name": "my-project",
  "query": "function that handles user authentication"
}

# 3. Find symbol definitions
# Use: code_find_symbol
{
  "project_name": "my-project",
  "name": "UserService"
}
```

### Available Tools

| Category | Tools |
|----------|-------|
| **Indexing** | `code_index_project`, `code_index_status`, `code_list_projects`, `code_delete_project`, `code_reindex_file`, `code_get_project_stats`, `code_get_file_symbols` |
| **Search** | `code_semantic_search`, `code_find_symbol`, `code_find_references`, `code_find_implementations`, `code_get_call_hierarchy`, `code_hybrid_search` |
| **Manipulation** | `code_rename_symbol`, `code_get_symbol_body`, `code_replace_symbol_body`, `code_insert_symbol` |

### Supported Languages

Go, TypeScript, JavaScript, TSX, Python, Rust, Java, Kotlin, Swift, C, C++, Objective-C, PHP, Ruby, C#, Scala, Bash, YAML

**📖 Full Documentation**: 
- [docs/CODE_INDEXING.md](docs/CODE_INDEXING.md) - User Guide
- [docs/CODE_INDEXING_API.md](docs/CODE_INDEXING_API.md) - API Reference
- [docs/TREE_SITTER_LANGUAGES.md](docs/TREE_SITTER_LANGUAGES.md) - Language Support

## 💡 Tool Help System (how_to_use)

Remembrances-MCP includes an intelligent help system that provides on-demand documentation while minimizing initial context token consumption.

### Usage

```bash
# Get complete overview of all tools
how_to_use()

# Get documentation for a tool group
how_to_use("memory")    # Memory tools (facts, vectors, graph)
how_to_use("kb")        # Knowledge base tools
how_to_use("code")      # Code indexing tools

# Get documentation for a specific tool
how_to_use("remembrance_save_fact")
how_to_use("kb_add_document")
how_to_use("search_code")
```

### Benefits

- **~85% reduction** in initial context token consumption
- **On-demand documentation** - load only what you need
- **Comprehensive help** - full arguments, examples, and related tools

**📖 Full Documentation**: See [docs/TOOL_HELP_SYSTEM.md](docs/TOOL_HELP_SYSTEM.md)

## Usage

Run the server with CLI flags or environment variables:

```bash
go run ./cmd/remembrances-mcp/main.go [flags]
```

### Configuration File

The server can be configured using a YAML configuration file. If `--config` is not specified, the server will automatically look for a configuration file in the following standard locations:

- **Linux**: `~/.config/remembrances/config.yaml`
- **macOS**: `~/Library/Application Support/remembrances/config.yaml`

If no configuration file is found, the server will use environment variables and default values.

### CLI Flags

- `--config`: Path to YAML configuration file (optional, see above for automatic location)
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
- `--gguf-model-path`: Path to GGUF model file for local embeddings (NEW)
- `--gguf-threads`: Number of threads for GGUF model (0 = auto-detect) (NEW)
- `--gguf-gpu-layers`: Number of GPU layers for GGUF model (0 = CPU only) (NEW)
- `--ollama-url`: Ollama server URL (default: http://localhost:11434)
- `--ollama-model`: Ollama model for embeddings
- `--openai-key`: OpenAI API key
- `--openai-url`: OpenAI base URL (default: https://api.openai.com/v1)
- `--openai-model`: OpenAI model for embeddings (default: text-embedding-3-large)

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
- `GOMEM_GGUF_MODEL_PATH`
- `GOMEM_GGUF_THREADS`
- `GOMEM_GGUF_GPU_LAYERS`
- `GOMEM_OLLAMA_URL`
- `GOMEM_OLLAMA_MODEL`
- `GOMEM_OPENAI_KEY`
- `GOMEM_OPENAI_URL`
- `GOMEM_OPENAI_MODEL`
- `GOMEM_CODE_GGUF_MODEL_PATH` - GGUF model for code embeddings
- `GOMEM_CODE_OLLAMA_MODEL` - Ollama model for code embeddings
- `GOMEM_CODE_OPENAI_MODEL` - OpenAI model for code embeddings

Additionally, there is an optional environment variable/flag to help auto-start a local SurrealDB when the server cannot connect at startup:

- `GOMEM_SURREALDB_START_CMD` / `--surrealdb-start-cmd`

### Code-Specific Embedding Models (Optional)

For code indexing, you can use specialized code embedding models that are optimized for source code semantics. If not configured, the default embedder is used for code indexing as well.

**Recommended Code Embedding Models:**

| Provider | Model | Notes |
|----------|-------|-------|
| GGUF | `coderankembed.Q4_K_M.gguf` | CodeRankEmbed - optimized for code |
| Ollama | `jina/jina-embeddings-v2-base-code` | Jina Code Embeddings |
| OpenAI | `text-embedding-3-large` | Also works well for code |

**Configuration:**

```bash
# Use CodeRankEmbed for code, nomic-embed-text for general text
export GOMEM_GGUF_MODEL_PATH="/path/to/nomic-embed-text-v1.5.Q4_K_M.gguf"
export GOMEM_CODE_GGUF_MODEL_PATH="/path/to/coderankembed.Q4_K_M.gguf"

# Or via CLI flags
remembrances-mcp --gguf-model-path /path/to/nomic.gguf --code-gguf-model-path /path/to/coderank.gguf
```

### YAML Configuration

You can also configure the server using a YAML file. Use the `--config` flag to specify the path to the YAML configuration file.

The YAML file should contain the configuration options using the same keys as the CLI flags (with dashes replaced by underscores if needed, but matching the `mapstructure` tags). CLI flags and environment variables override YAML settings.

Example YAML configuration file (`config.yaml`):

```yaml
# Enable SSE transport
sse: true
sse-addr: ":4000"

# Database configuration
db-path: "./mydata.db"

# Embedder configuration
ollama-model: "llama2"

# Logging
log: "./server.log"
```

Example usage:

```bash
go run ./cmd/remembrances-mcp/main.go --config config.yaml
```

A sample configuration file with all options and default values is provided in `config.sample.yaml`.

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
#try to copy to project root if error, remove the binary in the project root first
cp dist/remembrances-mcp ./remembrances-mcp || (rm -f ./remembrances-mcp && cp dist/remembrances-mcp ./remembrances-mcp)
```

### build-and-copy

Build the project and copy the binary to path

interactive: true

```bash
make BUILD_TYPE=cuda build
cp ./build/libs/cuda/*.so ./build/
rm -f ~/bin/remembrances-mcp-beta
cp ./build/libs/cuda/*.so ~/bin/
cp ./build/remembrances-mcp ~/bin/remembrances-mcp-beta
rm -f *.log
```

### build-and-copy-embedded

Build the project and copy the binary to path

interactive: true

```bash
cp ./build/libs/cuda/*.so ./build/
make build-embedded-cuda
rm -f ~/bin/remembrances-mcp-beta
rm -f ~/bin/*.so
cp ./build/remembrances-mcp ~/bin/remembrances-mcp-beta
rm -f *.log
```

### changelog

Generate a changelog for the repo.

```
convco changelog > CHANGELOG.md
git add CHANGELOG.md
git commit -m "Update changelog"
git push
```


### dist

Create distribution packages without overwriting CUDA portable libraries. The key change is to **build libraries first** and then use `build-binary-only` so the Go binary is produced without recompiling (and replacing) the shared libraries.

```bash
echo "Cleaning previous dist-variants..."
rm -rf dist-variants/*

echo "Building Linux distribution variants with libs..."
echo "-----------------------------------"

echo "Building CUDA optimized for current CPU"
make BUILD_TYPE=cuda llama-cpp
# CUDA optimized for current CPU
make PORTABLE=0 build-libs-cuda
make BUILD_TYPE=cuda build
cd build
zip -9 remembrances-mcp-linux-x64-nvidia.zip remembrances-mcp *.so
mv remembrances-mcp-linux-x64-nvidia.zip ../dist-variants/
cd ..

echo "Building CUDA portable (AVX2-compatible for Intel/AMD)"
# CUDA portable (AVX2-compatible for Intel/AMD)
make PORTABLE=1 build-libs-cuda-portable
make BUILD_TYPE=cuda build
cd build
zip -9 remembrances-mcp-linux-x64-nvidia-portable.zip remembrances-mcp *.so
mv remembrances-mcp-linux-x64-nvidia-portable.zip ../dist-variants/
cd ..

echo "building CPU-only"
# CPU-only
make build-libs-cpu
make build-binary-only
cd build
zip -9 remembrances-mcp-linux-x64-cpu.zip remembrances-mcp *.so
mv remembrances-mcp-linux-x64-cpu.zip ../dist-variants/
cd ..

echo "Building embedded libraries variants only for CUDA"
echo "Building embedded CUDA libraries..."
make build-embedded-cuda
cd build
zip -9 remembrances-mcp-linux-x64-nvidia-embedded.zip remembrances-mcp
mv remembrances-mcp-linux-x64-nvidia-embedded.zip ../dist-variants/
cd ..

echo "Building embedded CUDA portable libraries..."
make build-embedded-cuda-portable
cd build
zip -9 remembrances-mcp-linux-x64-nvidia-embedded-portable.zip remembrances-mcp
mv remembrances-mcp-linux-x64-nvidia-embedded-portable.zip ../dist-variants/
cd ..

echo "-----------------------------------"
echo "Checking if memote osx is available..."
# if ssh connection with mac-mini-de-digio is available, build macos metal version
if ssh -o BatchMode=yes -o ConnectTimeout=5 mac-mini-de-digio 'echo 2>&1' && [ $? -eq 0 ]; then
    echo "Building macOS Metal variant on remote host..."
    ./scripts/build-osx-remote.sh
    ./scripts/copy-osx-build.sh
else
    echo "Skipping macOS Metal build - remote host not available."
fi 

# (Optional) Docker images remain the same
make docker-prepare-cpu && make docker-build-cpu && make docker-push-cpu
```

### build-osx

Build for macOS (arm64 metal)

interactive: true

```zsh
echo "Building macOS Metal binary..."
export PATH=$HOME/bin:$HOME/.local/bin:/usr/local/bin:/opt/homebrew/bin:$PATH
make BUILD_TYPE=metal build-binary-only
echo "Zipping macOS Metal binary..."
rm -f dist/remembrances-mcp-darwin-aarch64.zip
cp config.sample*.yaml build/
cd build
zip -9 ../dist/remembrances-mcp-darwin-aarch64.zip remembrances-mcp *.dylib config.sample*.yaml
```

### build-osx-embedded

Build for macOS (arm64 metal)

interactive: true

```zsh
echo "Building macOS Metal embedded libraries binary..."
export PATH=$HOME/bin:$HOME/.local/bin:/usr/local/bin:/opt/homebrew/bin:$PATH
make EMBEDDED_VARIANT=metal build-embedded
echo "Zipping macOS Metal embedded binary..."
rm -f dist/remembrances-mcp-darwin-aarch64-embedded.zip
cp config.sample*.yaml build/
cd build
zip -9 ../dist/remembrances-mcp-darwin-aarch64-embedded.zip remembrances-mcp config.sample*.yaml
```

### build-libs-osx

Build libs for macOS (arm64 metal)

interactive: true

```zsh
echo "Building macOS Metal libraries..."
export PATH=$HOME/bin:$HOME/.local/bin:/usr/local/bin:/opt/homebrew/bin:$PATH
make build-libs-metal
cp build/libs/metal/*.dylib build/
echo "Building SurrealDB embedded for macOS..."
make build-surrealdb-darwin-arm64
cp build/libs/surrealdb-aarch64-apple-darwin/*.dylib build/
```
