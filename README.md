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

- `--sse` (default: true): Enable SSE transport
- `--rest-api-serve`: Enable REST API server
- `--knowledge-base`: Path to knowledge base directory
- `--db-path`: Path to embedded SurrealDB database (default: ./remembrances.db)
- `--surrealdb-url`: URL for remote SurrealDB instance
- `--surrealdb-user`: SurrealDB username (default: root)
- `--surrealdb-pass`: SurrealDB password (default: root)
- `--ollama-url`: Ollama server URL (default: http://localhost:11434)
- `--ollama-model`: Ollama model for embeddings
- `--openai-key`: OpenAI API key
- `--openai-url`: OpenAI base URL (default: https://api.openai.com/v1)
- `--openai-model`: OpenAI model for embeddings (default: text-embedding-3-large)

### Environment Variables

All flags can be set via environment variables prefixed with `GOMEM_` and dashes replaced by underscores. For example:

- `GOMEM_SSE`
- `GOMEM_REST_API_SERVE`
- `GOMEM_KNOWLEDGE_BASE`
- `GOMEM_DB_PATH`
- `GOMEM_SURREALDB_URL`
- `GOMEM_SURREALDB_USER`
- `GOMEM_SURREALDB_PASS`
- `GOMEM_OLLAMA_URL`
- `GOMEM_OLLAMA_MODEL`
- `GOMEM_OPENAI_KEY`
- `GOMEM_OPENAI_URL`
- `GOMEM_OPENAI_MODEL`

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