MCP tools summary (defined in `pkg/mcp_tools/tools.go`)

Overview:
- Tools are grouped by domain: mem_... for memory ops and kb_... for knowledge-base ops. Each tool is created with protocol.NewTool(name, desc, InputStruct{}). Handlers unmarshal `request.RawArguments` into the Input struct and call storage/embedder.

Registered tools (name -> purpose):
- mem_save_fact: Save a key-value fact for a user (SaveFactInput)
- mem_get_fact: Retrieve a key-value fact (GetFactInput)
- mem_list_facts: List facts for a user (ListFactsInput)
- mem_delete_fact: Delete a fact (DeleteFactInput)

- mem_add_memory: Add semantic memory (embedding generated via EmbedQuery then IndexVector)
- mem_search_memories: Search similar memories using embedder + SearchSimilar
- mem_update_memory: Update memory content + embedding
- mem_delete_memory: Delete memory by ID

- mem_create_entity / mem_get_entity / mem_create_relationship / mem_traverse_graph: Graph CRUD and traversal (CreateEntity, CreateRelationship, TraverseGraph, GetEntity)

- kb_add_document / kb_search_documents / kb_get_document / kb_delete_document: Knowledge base document storage and similarity search (SaveDocument, SearchDocuments, GetDocument, DeleteDocument)

- mem_hybrid_search: Hybrid search combining vector, graph, and key-value (calls HybridSearch)
- mem_get_stats: Get memory stats for a user (GetStats)

Handler patterns & behaviors to remember:
- Input structs defined near top of file; default limits set (e.g., limit=10 when zero)
- Embeddings: handlers call tm.embedder.EmbedQuery for queries and documents, and then call storage.* methods with embeddings
- Storage errors are proxied back as handler errors; successful responses are wrapped with protocol.TextContent JSON strings for readability

Where to add new tools:
- Add Input struct + tool factory function (tool()) + handler in `pkg/mcp_tools/tools.go` and register in `RegisterTools`.

Testing hints:
- Tool handlers are side-effecting (DB). For unit tests, mock `storage.StorageWithStats` and `embedder.Embedder` implementations.

