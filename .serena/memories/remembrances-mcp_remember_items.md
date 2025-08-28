Title: remembrances-mcp — canonical "what to remember" reference

Purpose
- A single, machine- and human-readable reference containing the important project- and user-level facts that future "remembrance" tools should use to decide what to store, how to index it, and how to return it.
- Not a dump of all files, but a structured schema + concrete items derived from collected Serena memories.

High-level project facts (short)
- Project: remembrances-mcp (Go)
- Purpose: MCP server providing a 3-layer remembrance system: key-value facts, vector/RAG memories, and knowledge graph; SurrealDB is the storage backend.
- Entry points: `cmd/remembrances-mcp/main.go` (start, config), `pkg/mcp_tools/tools.go` (tools/handlers), `internal/storage/surrealdb.go` (storage impl), `pkg/embedder/*` (embedder contract/impl).
- Tech stack: Go modules, SurrealDB (embedded or remote), embedders: Ollama/OpenAI (pluggable).

Important config & runtime facts
- Env prefix: `GOMEM_` (flags -> viper -> env). Key env vars: `GOMEM_OPENAI_KEY`, `GOMEM_DB_PATH`, `GOMEM_SURREALDB_URL`, `GOMEM_SURREALDB_NAMESPACE`, `GOMEM_SURREALDB_DATABASE`, `GOMEM_OLLAMA_URL`, `GOMEM_OLLAMA_MODEL`.
- Default SurrealDB namespace/db: `test` unless overridden.
- MTREE embedding dimension used in schema: 768 (must match Embedder.Dimension()).
- Recommended dev run example (non-secret): `GOMEM_OPENAI_KEY=... GOMEM_DB_PATH=./data.db go run ./cmd/remembrances-mcp/main.go --knowledge-base ./kb --rest-api-serve`.

Storage & schema facts
- Tables created: `kv_memories`, `vector_memories`, `knowledge_base`, `entities`, relationship tables like `wrote`, `mentioned_in`, `related_to`.
- Vector index: MTREE index on embedding field; dimension must match embedder.
- Schema is versioned (schema_version table) and migrations should be idempotent.
- CBOR result shapes: SurrealDB client returns arrays for SELECT/DELETE; handlers must handle []map[string]interface{}.

Embedder contract (programmatic)
- Methods: `EmbedDocuments(ctx, []string) ([][]float32, error)`, `EmbedQuery(ctx, string) ([]float32, error)`, `Dimension() int`.
- Embeddings are float32 vectors, dimension must match DB index.

MCP tools — canonical list & purpose
- renaming: all mem_* -> remembrance_* (e.g., `remembrance_save_fact`, `remembrance_search_vectors`, `remembrance_create_entity`, `kb_add_document`, `remembrance_hybrid_search`, `remembrance_get_stats`).
- Handler pattern: unmarshal `request.RawArguments` into typed input struct (avoid interface{}), call embedder/storage, return protocol content.
- Input type constraints: avoid `interface{}`/map[string]interface{} in tool input structs (use map[string]string or string and convert internally).

Known issues & gotchas (to remember)
- SSE transport: session management problems in go-mcp; HTTP transport implemented as workaround.
- Tool creation errors: protocol.NewTool rejects interface{} fields; input structs were adapted.
- CBOR unmarshal errors when expecting single object but Surreal returns arrays — handlers updated to use Query and handle arrays.
- MTREE dimension mismatch risk — keep embedder.Dimension() and schema in sync.

Suggested programmatic memory schema for future tools
1) Project-level facts (single record)
   - id: project:remembrances-mcp
   - fields: {name, purpose, entry_points:[paths], tech_stack:[strings], default_env:{...}, docs:[paths]}
2) Tool definitions (records)
   - id: tool:<tool_name>
   - fields: {name,desc,input_schema_json,handler_file,path,examples:[...],tags:[project,remembrance,kb]}
3) Storage/schema facts (records)
   - id: schema:surrealdb
   - fields: {tables:[{name,fields,indexes}], mtree_dim:int, schema_version:int, migrations:[{v,desc,file_or_stmt}]}
4) Embedder facts
   - id: embedder:<name>
   - fields: {type,dimension,endpoint,config_tips}
5) Known issues & troubleshooting notes (time-stamped)
   - id: issue:<short_key>
   - fields: {title,description,files_affected,fix_or_workaround,first_seen}
6) Dev/run commands (short list, not secrets)
   - id: dev:commands
   - fields: {build,run,test,debug}
7) User-level preferences (opt-in only, no secrets)
   - id: user:<user_id>:prefs
   - fields: {preferred_embedder,default_transport,dev_env_vars,safe_to_store:boolean}

Privacy & security rules to remember
- Never store secrets (API keys, passwords, tokens) in persistent memories.
- For user-level memories, require explicit opt-in and provide facilities to delete/update facts (`remembrance_delete_fact`).
- Store only derived or metadata for user credentials (e.g., "uses OpenAI" boolean), never raw keys.

Concrete items to add now (short)
- Project overview (summary above)
- Tool inventory (brief list of canonical tool names and purpose)
- Schema summary (tables and MTREE dim + migration/version note)
- Known issues list (SSE, CBOR, tool input types)
- Suggested memory record shapes (JSON examples) for tools to use when creating memories

Minimal JSON examples for tools (templates)
- Fact (key-value): {"type":"fact","user_id":"<id>","key":"timezone","value":"Europe/Madrid","source":"user:cli"}
- Vector memory: {"type":"vector","id":"uuid","embedding":[...float32...],"text":"...","metadata":{"source":"kb","doc_id":"..."}}
- Entity: {"type":"entity","id":"entity:Person:alice","labels":["Person"],"properties":{}} 
- Relationship: {"from":"entity:Person:alice","to":"entity:Project:remembrances-mcp","type":"works_on","properties":{}}

Next steps (recommended)
1. Persist this summary into Serena (done) so future remembrance tools can reference it.
2. Optionally split the summary into targeted memories (project_overview, tools_index, schema_summary, issues), if you prefer finer granularity.
3. Implement programmatic schemas in code (storage + tools) so tools can create records matching the templates above.
4. Add tests that validate embedder.Dimension() vs DB MTREE index and verify handlers handle SurrealDB array results.

Notes
- This memory intentionally avoids storing any secret keys or user PII.
- If you want me to split this into multiple memory files or add one memory per major section (project, tools, schema, issues), say which split you want.
