## Mem0 Project Analysis for Remembrances-MCP Implementation

This document summarizes the key architectural and implementation details from the `mem0` Python project, intended to guide the development of the `remembrances-mcp` Go project.

### 1. Overall Architecture

The `mem0` project is a sophisticated memory system for AI agents, structured in a modular and configurable way. It supports multiple memory types and backend stores. The key components are:

- **Client (`mem0/client/`)**: Provides a user-facing API (`MemoryClient`) to interact with the memory system.
- **Core Memory Logic (`mem0/memory/`)**: Implements the main `Memory` class, which orchestrates operations across different memory types. It handles memory creation, retrieval, and updates.
- **Configuration (`mem0/configs/`)**: Pydantic models define the configuration for all components, including LLMs, embedders, vector stores, and graph stores. This allows for a flexible, config-driven setup.
- **Embeddings (`mem0/embeddings/`)**: A factory-based approach supports various embedding providers (OpenAI, Ollama, HuggingFace, etc.). This is analogous to `remembrances-mcp/pkg/embedder`.
- **Vector Stores (`mem0/vector_stores/`)**: An extensive collection of vector database backends (Chroma, FAISS, Pinecone, Qdrant, etc.). `remembrances-mcp` uses SurrealDB for this purpose.
- **Graph Memory (`mem0/graphs/`)**: Implements graph-based memory using backends like Neo4j, Memgraph, and Neptune. This is a key area of interest.

### 2. Tool-Based Interface

`mem0` exposes its functionality through a set of tools, particularly for graph operations. These tools are defined in `mem0/graphs/tools.py` and are used by an LLM to manipulate the graph memory. The core idea is to let the LLM decide which tool to use based on the user's input.

#### Key Graph Memory Tools to Implement in Go:

1.  **`add_memory` / `add_memory_structured`**:
    -   **Purpose**: Adds new information to the memory. It takes a list of facts or structured data.
    -   **Go Implementation**: This would correspond to a `mem_add` tool in `remembrances-mcp`. The tool would take text, extract entities and relationships, and store them as nodes and edges in SurrealDB.

2.  **`update_memory` / `update_memory_structured`**:
    -   **Purpose**: Updates existing information in the graph. It identifies existing nodes and modifies their relationships or properties based on new information.
    -   **Go Implementation**: A `mem_update` tool. It would need to search for existing memories (nodes/edges) and apply changes. This requires a search/retrieval mechanism first.

3.  **`delete_memory` / `delete_memory_structured`**:
    -   **Purpose**: Removes information from the graph that is outdated or incorrect.
    -   **Go Implementation**: A `mem_delete` tool. It would take identifiers for memories and remove the corresponding nodes or edges from SurrealDB.

4.  **`extract_entities` / `extract_entities_structured`**:
    -   **Purpose**: A utility tool to extract entities (nodes) from a piece of text.
    -   **Go Implementation**: This might be an internal function used by other tools rather than a user-facing tool. It would use an LLM to perform the extraction.

5.  **`extract_relations` / `relations_structured`**:
    -   **Purpose**: Extracts relationships (edges) between entities from a text.
    -   **Go Implementation**: Similar to entity extraction, this would likely be an internal function that calls an LLM with a specific prompt to get structured relationship data.

6.  **`noop` / `noop_structured`**:
    -   **Purpose**: A "no operation" tool, used when the new information is irrelevant or already present in the memory.
    -   **Go Implementation**: This is a control flow tool for the agent, indicating that no database operation is needed.

### 3. Memory Storage in SurrealDB

While `mem0` uses various databases, `remembrances-mcp` uses SurrealDB for all three layers of memory:

-   **Key-Value**: Simple `CREATE` and `SELECT` on tables for direct data storage.
-   **Vector/RAG**: SurrealDB's `vector` type and `MTREE` indexes on an `embedding` field. Search is done using the `vector::similarity::cosine` function.
-   **Graph**: SurrealDB's graph capabilities, using `RELATE` to create edges between records (nodes).

### 4. Implementation Plan for `remembrances-mcp`

Based on the `mem0` analysis, the following steps should be taken to implement the memory tools in Go:

1.  **Define Tool Signatures**: In `pkg/mcp_tools/tools.go`, define the new `mem_...` tools (`mem_add`, `mem_search`, `mem_update`, `mem_delete`, `mem_get_graph`). Each tool needs a name, a description, and an input schema.

2.  **Implement Tool Handlers**: For each tool, create a handler function. These handlers will:
    -   Unmarshal the JSON arguments from the tool call request.
    -   Interact with the `Storage` interface (`internal/storage/storage.go`) to perform the required SurrealDB operations.
    -   Marshal the result back into a JSON response.

3.  **Extend the Storage Interface**: Add new methods to the `Storage` interface in `internal/storage/storage.go` and implement them in `internal/storage/surrealdb.go`. These methods will encapsulate the SurrealDB queries for:
    -   Adding/updating/deleting nodes (memories).
    -   Adding/updating/deleting edges (relationships).
    -   Performing vector similarity searches.
    -   Querying the graph structure.

4.  **Use LLM for Extraction**: For tools like `mem_add`, the handler will need to call an LLM to extract structured entities and relationships from the input text before storing them in the database. This requires creating specific prompts, similar to those in `mem0/graphs/utils.py`.

This analysis provides a clear roadmap for building a powerful, multi-layered memory system in `remembrances-mcp`, leveraging the proven patterns from the `mem0` project and adapting them to the Go and SurrealDB technology stack.