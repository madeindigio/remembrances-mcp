Project: remembrances-mcp

Purpose:
- A Go-based MCP (Model Connector Protocol) server implementing a 3-layer memory system: key-value (facts), vector/RAG (semantic memories & knowledge base), and graph (entities/relationships). It uses SurrealDB as the storage backend and exposes functionality as MCP tools.

Tech stack:
- Go (modules)
- SurrealDB (embedded via rocksdb:// or remote)
- Embedding providers: Ollama or OpenAI (pluggable via pkg/embedder)
- MCP protocol: github.com/ThinkInAIXYZ/go-mcp

Important entry points & files:
- `cmd/remembrances-mcp/main.go` — program entry, config load, storage + embedder wiring, tools registration, transport selection (stdio or SSE)
- `internal/config/config.go` — CLI flags and viper env mapping (prefix GOMEM_)
- `pkg/mcp_tools/tools.go` — all tool definitions and handlers; register in ToolManager.RegisterTools
- `internal/storage/surrealdb.go` & `internal/storage/storage.go` — storage interface and SurrealDB implementation; schema initialization lives here
- `pkg/embedder/embedder.go` and implementations `pkg/embedder/ollama.go`, `pkg/embedder/openai.go` — embedder contract and wiring

SurrealDB specifics (important to remember):
- InitializeSchema creates tables: `kv_memories`, `vector_memories`, `knowledge_base`, `entities`, plus relationship tables like `wrote`, `mentioned_in`, `related_to`.
- Vector MTREE indexes assumed dimension 768 in schema statements (embedding dimension must match Embedder.Dimension()).
- Default namespace/database: `test` unless overridden by env/flags (GOMEM_SURREALDB_NAMESPACE / GOMEM_SURREALDB_DATABASE).

How tools interact with storage/embeddings:
- Handlers parse JSON args into input structs, call embedder.EmbedQuery or EmbedDocuments when needed, then call storage methods (IndexVector, SearchSimilar, SaveFact, CreateEntity, etc.). Responses are returned via protocol.NewCallToolResult.

Runtime notes:
- Config mapping: CLI flags -> viper -> env variables with prefix `GOMEM_` (dashes become underscores).
- Example run: `GOMEM_OPENAI_KEY=sk-xxx GOMEM_DB_PATH=./data.db go run ./cmd/remembrances-mcp/main.go --knowledge-base ./kb --rest-api-serve`

Files to inspect first for most tasks: `cmd/.../main.go`, `pkg/mcp_tools/tools.go`, `internal/storage/surrealdb.go`, `internal/config/config.go`, `pkg/embedder/*`.