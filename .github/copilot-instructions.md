<!--
Guidance for AI coding agents working on Remembrances-MCP.
Keep this file short and focused: reference concrete files, patterns, and commands
that are discoverable in the repository so an agent can be productive immediately.
-->

# Remembrances‑Mcp — instructions

This project is a Go-based MCP (Model Context Protocol) server that implements a 3-layer memory system (key-value, vector/RAG, graph) backed by SurrealDB and exposes functionality as MCP tools.

## Work methodology

Always use Serena tools for editing or finding code in this project. Check Serena onboarding performed in the work session, and list and read memories before start the work for is we have previous knowledge about the task that you are working on.

Search in serper web search or brave search if faults the first, for more info and fetch pages that you consider useful for the task, if needed. Also use Context7 for finding the API's use and how work the libraries to use.

When you start a new task, first use remembrances tools for getting the knowledge base, facts, entities and relationships related to the task, and read them for understand the context.

When you finish the task use remembrances tools for storing the knowledge base, facts, entities and relationships related to the task, for future use.

When you are testing the tools connected, check the errors of the remembrances tools and try to fix them, then build the binary with `xc build` and wait to the user restart the server manually for continuing.

## Architecture Overview

- Entry point: `cmd/remembrances-mcp/main.go` — sets up config, logging, transport (stdio or SSE), storage and registers MCP tools.
- Configuration: `internal/config/config.go` — all CLI flags are defined here and map to environment variables prefixed with `GOMEM_` (dashes -> underscores).
- Storage contract: `internal/storage/storage.go` defines `Storage`/`StorageWithStats` interfaces. Implementation lives in `internal/storage/surrealdb.go`.
- Tool surface: `pkg/mcp_tools/tools.go` — all MCP tools are declared via `protocol.NewTool(...)` and registered in `ToolManager.RegisterTools`. Handlers accept `context` and a `*protocol.CallToolRequest` and usually `json.Unmarshal(request.RawArguments, &InputStruct{})`.
- Embeddings: `pkg/embedder/embedder.go` defines the `Embedder` interface (EmbedDocuments, EmbedQuery, Dimension). Concrete embedder implementations are wired from config in `cmd/.../main.go` (Ollama or OpenAI via env/flags).

Important project-specific details

- SurrealDB usage

  - The project supports embedded (surrealkv://) and remote SurrealDB. See `internal/storage/surrealdb.go` for Connect/Use patterns.
  - Schema initialization (tables, fields, MTREE indexes) is performed in `InitializeSchema` and assumes embedding dimension 768 for MTREE indexes (`DEFINE INDEX ... MTREE DIMENSION 768 DIST COSINE`).
  - Default namespace/database: `test` unless overridden by flags or `GOMEM_SURREALDB_NAMESPACE` / `GOMEM_SURREALDB_DATABASE`.

- MCP tool patterns
  - Tool names are prefixed by domain: e.g. `mem_...` for memory ops, `kb_...` for knowledge-base ops (see `pkg/mcp_tools/tools.go`).
  - Handlers return `protocol.NewCallToolResult([]protocol.Content{...}, false)` and commonly include JSON-marshaled results for readability.
  - When adding a new tool: add the Input struct, a `tool()` factory returning `protocol.NewTool(name, desc, Input{})`, and a handler with the `json.Unmarshal(request.RawArguments, &input)` pattern, then register it in `RegisterTools`.

Developer workflows & commands (verified from repo)

- Build the binary:

  xc build

Files and locations to inspect when making changes

- Add new MCP tools: `pkg/mcp_tools/tools.go`
- Update storage behavior/schema: `internal/storage/surrealdb.go` and `internal/storage/storage.go`
- Change CLI/flags/logging: `internal/config/config.go` and `cmd/.../main.go`
- Embedder contract and wiring: `pkg/embedder/embedder.go` and embedder factory code called from `main.go`.

If something is unclear or you need examples for a specific change (new tool, storage migration, embedder wiring), ask for the exact file to modify and I will add a focused example patch.
