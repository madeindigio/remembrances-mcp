# Remembrances-MCP

Remembrances-MCP is a Go-based MCP server that provides long-term memory capabilities to AI agents. It supports multiple memory layers (key-value, vector/RAG, graph database) using SurrealDB, and can manage knowledge bases via Markdown files.

## Features

- MCP server for AI agent memory
- SurrealDB support (embedded or external)
- Knowledge base management with Markdown files
- Embedding generation via Ollama (local) or OpenAI API
- REST API and SSE transport options

## Usage

Run the server with CLI flags or environment variables:

```bash
go run ./cmd/remembrances-mcp/main.go [flags]
```

### CLI Flags

- `--sse` (default: false): Enable SSE transport
- `--sse-addr` (default: :4001): Address to bind SSE transport (host:port). Can also be set via `GOMEM_SSE_ADDR`.
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

- `--surrealdb-start-cmd`: Optional command to start an external SurrealDB instance when an initial connection cannot be established. Can also be set via `GOMEM_SURREALDB_START_CMD`.

### Environment Variables

All flags can be set via environment variables prefixed with `GOMEM_` and dashes replaced by underscores. For example:

- `GOMEM_SSE`
- `GOMEM_SSE_ADDR` (e.g. `:4001` or `0.0.0.0:4001`)
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

Additionally, there is an optional environment variable/flag to help auto-start a local SurrealDB when the server cannot connect at startup:

- `GOMEM_SURREALDB_START_CMD` / `--surrealdb-start-cmd`

Example usage (start command provided via env):

```bash
export GOMEM_SURREALDB_START_CMD="surreal start --user root --pass root surrealkv:///path/to/surreal_data"
go run ./cmd/remembrances-mcp/main.go --knowledge-base ./kb

# Start SSE transport on a custom address via CLI flag
go run ./cmd/remembrances-mcp/main.go --sse --sse-addr=":4001"

# Or via environment variable
GOMEM_SSE=true GOMEM_SSE_ADDR=":4001" go run ./cmd/remembrances-mcp/main.go --sse
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