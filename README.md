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
rm -f ~/bin/remembrances-mcp
cp ./build/remembrances-mcp ~/bin/
rm *.log
```

### starts-surrealdb

Starts the SurrealDB instance

interactive: true
```bash
surreal start --user root --pass root surrealkv://~/www/MCP/remembrances-mcp/surreal_data
```

### run-tests

Runs the test suite

interactive: true

```bash
./tests/run_all.sh
```

### run-tests-all

Runs the full Go unit tests plus the MCP integration tests executed above.

interactive: true

```bash
go test ./...
python3 tests/test_user_stats.py
python3 tests/test_kb_simple.py
python3 tests/test_kb_comprehensive.py
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