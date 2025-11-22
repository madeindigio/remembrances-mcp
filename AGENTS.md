This repository is an implementation of a memory system for AI agents, designed to enhance their ability to retain and recall information over time. The system is built to support various types of agents, including conversational agents, virtual assistants, and autonomous systems.

## Features

- MCP server support, stdio and http api
- API REST endpoints for memory management
- Integration with embeddings models using LangChain (OpenAI and ollama implemented)
- Plain text knowledge base and vector database support (Surrealdb implemented)
- Knowledge graph and memory graph support
- gguf models support for local embeddings generation and surrealdb embedded database support and external surrealdb database
- Support for llama.cpp with gpu optimizations

## Deveplopment

To build the project, execute the following command:

```bash
make BUILD_TYPE=cuda build 
```

## 🛠️ Work Methodology

### Essential First Steps

1. **Check Serena onboarding**: Use `mcp_serena_check_onboarding_performed`
2. **Read memories**: Use `mcp_serena_list_memories` and read relevant ones for context
3. **Search knowledge base**: Use `mcp_remembrances_kb_search_documents` for project information
4. **Check the plan**: Review `.serena/memories/plan.md` for current tasks

### Development Workflow

- **Always use Serena tools** for code editing and navigation
- Use always english as language for all code, comments, and documentation although the user may write in other languages
- **Build command**: `xc build` (rebuild after code changes)
- **Testing**: Create tests in `tests/` folder (Python files, not in root)
- **Server restart**: Wait for user to manually restart MCP server after fixes
- **Knowledge storage**: Store findings using remembrances tools for future reference

### External Research

- Use web search (serper/brave) for additional information when needed
- Use Context7 for API documentation and library usage patterns

## 🏗️ Architecture Overview (Post-Refactoring)

### 📁 Project Structure

**Entry Point**

- `cmd/remembrances-mcp/main.go` — Application entry: config loading, storage/embedder wiring, tool registration, transport selection

**Configuration**

- `internal/config/config.go` — CLI flags and environment variables (prefix: `GOMEM_`)

### 🗄️ Storage Layer (Refactored into Specialized Files)

- `internal/storage/storage.go` — Interface definitions (`Storage`/`StorageWithStats`)
- `internal/storage/surrealdb.go` (630 lines) — Core implementation, connection management, entity/graph ops
- `internal/storage/surrealdb_schema.go` (444 lines) — Schema management, migrations, table/field/index management
- `internal/storage/surrealdb_facts.go` (112 lines) — Key-value fact operations
- `internal/storage/surrealdb_vectors.go` (127 lines) — Vector/embedding operations

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

## 📚 Recently Completed Major Work

### Refactoring Achievements (September 2025)

- **51% reduction** in main storage file size (1287 → 630 lines)
- **87% reduction** in main tools file size (1027 → 135 lines)
- **Improved maintainability** through logical separation of concerns
- **Enhanced modularity** for easier debugging and development
- **Preserved compatibility** - all builds and tests successful

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

## 🚨 Important Constraints & Warnings

1. **MTREE Dimension**: Embedder.Dimension() must return 768 to match schema indexes
2. **Config Validation**: Either OllamaModel or OpenAIKey required - intentional but blocks local dev without setup
3. **SurrealDB Syntax**: Some queries use custom parameterization syntax specific to SurrealDB Go client
4. **Tool Testing**: Mock storage and embedder in tests to avoid external API calls
5. **Schema Changes**: Be careful with migrations - embedded SurrealDB data persistence matters
