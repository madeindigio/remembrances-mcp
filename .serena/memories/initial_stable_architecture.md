# 🏗️ Architecture Overview (Post-Refactoring)

### 📁 Project Structure

**Entry Point**

- `cmd/remembrances-mcp/main.go` — Application entry: config loading, storage/embedder wiring, tool registration, transport selection

**Configuration**

- `internal/config/config.go` — CLI flags and environment variables (prefix: `GOMEM_`)

### 🗄️ Storage Layer (Refactored into Specialized Files)

- `internal/storage/storage.go` — Interface definitions for storage capabilities, including stats extensions
- `internal/storage/surrealdb.go` — SurrealDB-backed implementation, connection lifecycle, and orchestration of concrete operations
- `internal/storage/surrealdb_helpers.go` — Shared query helpers, result decoding, and utility functions used across storage modules
- `internal/storage/surrealdb_schema.go` — Schema initialization, migrations, and index management
- `internal/storage/surrealdb_documents.go` — Knowledge base document storage, retrieval, and search helpers
- `internal/storage/surrealdb_entities.go` — Entity and relationship CRUD plus traversal utilities
- `internal/storage/surrealdb_facts.go` — Key-value fact persistence with user scoping
- `internal/storage/surrealdb_vectors.go` — Vector embedding persistence and similarity queries
- `internal/storage/surrealdb_hybrid.go` — Hybrid search operations that blend facts, vectors, and graph data
- `internal/storage/surrealdb_stats.go` — Aggregation routines for user and system statistics
- `internal/storage/migrations/` — Versioned migration definitions coordinating schema evolution
- `internal/storage/embedding_test.go`, `internal/storage/surrealdb_test.go` — Storage-focused regression and integration tests

### 🔧 MCP Tools Layer (Organized by Function)

- `pkg/mcp_tools/tools.go` (135 lines) — Core registry and tool manager
- `pkg/mcp_tools/types.go` (117 lines) — Input/output type definitions
- `pkg/mcp_tools/fact_tools.go` (173 lines) — Key-value memory operations
- `pkg/mcp_tools/vector_tools.go` (188 lines) — Semantic search and RAG operations
- `pkg/mcp_tools/graph_tools.go` (180 lines) — Entity and relationship management
- `pkg/mcp_tools/kb_tools.go` (191 lines) — Document search and management
- `pkg/mcp_tools/misc_tools.go` (101 lines) — Statistics and hybrid search

### 🧠 Embeddings Support

- `pkg/embedder/embedder.go` — Interface definition (`EmbedDocuments`, `EmbedQuery`, `Dimension`)
- `pkg/embedder/factory.go` — Factory for Ollama/OpenAI implementation selection
- `pkg/embedder/ollama.go` — Local Ollama implementation
- `pkg/embedder/openai.go` — OpenAI cloud implementation

## ⚡ Key Technical Details

### SurrealDB Configuration

- **Supported modes**: Embedded (`surrealkv://`) and remote SurrealDB
- **Schema initialization**: Automated via `InitializeSchema` in `surrealdb_schema.go`
- **Vector indexes**: MTREE dimension **768** (must match embedder dimension)
- **Default namespace/database**: `test` (override with `GOMEM_SURREALDB_NAMESPACE`/`GOMEM_SURREALDB_DATABASE`)

### MCP Tool Conventions

- **Tool naming**: Domain-prefixed (`remembrance_*` for memory ops, `kb_*` for knowledge base ops)
- **Handler pattern**: `protocol.NewCallToolResult([]protocol.Content{...}, false)`
- **Input processing**: JSON unmarshal to typed structs in `types.go`
- **Error handling**: Return errors up the stack, converted to `fmt.Errorf` at handler level

### Environment Variables

- **Prefix**: `GOMEM_` (CLI flags with dashes become underscores)
- **Required**: Either `GOMEM_OLLAMA_MODEL` or `GOMEM_OPENAI_KEY` must be set
- **Example**: `GOMEM_DB_PATH=./data.db GOMEM_OPENAI_KEY=sk-xxx`

## 🧪 Development Patterns

### Adding New MCP Tools

1. **Add Input struct** to `pkg/mcp_tools/types.go`
2. **Create tool factory and handler** in appropriate `*_tools.go` file
3. **Register tool** in `pkg/mcp_tools/tools.go` `RegisterTools` method

### Storage Modifications

- **Schema changes**: Update `internal/storage/surrealdb_schema.go`
- **New operations**: Add to appropriate specialized file (`surrealdb_facts.go`, `surrealdb_vectors.go`, etc.)
- **Interface changes**: Update `internal/storage/storage.go`

### Code Style

- **Language**: Go with standard project layout, use `go fmt` and `go vet`
- **Logging**: Structured logging with `log/slog` (configured in `config.go`)
- **Tests**: Keep in `tests/` folder as Python files, use table-driven Go tests where appropriate
- **Error handling**: Return errors up the stack, handle at appropriate level


### System Status: All Tools Operational ✅

- Vector Memory: Semantic search with 768-dimension embeddings
- Graph Database: Entity creation, retrieval, and traversal
- Knowledge Base: Document storage with semantic search
- Fact Storage: Key-value memory with user scoping
- Hybrid Search: Combined multi-layer query capabilities

## 🔍 Files to Inspect for Most Tasks

**Configuration & Entry**: `cmd/.../main.go`, `internal/config/config.go`
**Tool Development**: `pkg/mcp_tools/tools.go`, `pkg/mcp_tools/types.go`, appropriate `*_tools.go` files
**Storage Changes**: `internal/storage/storage.go`, `internal/storage/surrealdb*.go` files
**Embedder Work**: `pkg/embedder/embedder.go`, `pkg/embedder/factory.go`
