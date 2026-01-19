---
title: "Code Indexing Tools"
linkTitle: "Code Tools"
weight: 12
description: >
  Source code indexing and semantic search
---

Code tools enable indexing entire projects for semantic search, intelligent navigation, and code manipulation using AST analysis with Tree-sitter.

## Tool Categories

### Indexing Tools

Manage code project indexing:

- `code_index_project`: Start indexing a project (async execution)
- `code_index_status`: Check indexing job progress
- `code_list_projects`: List all indexed projects
- `code_delete_project`: Remove a project and its data
- `code_reindex_file`: Update index for a specific file
- `code_get_project_stats`: Get project statistics
- `code_get_file_symbols`: List symbols in a specific file

### Search Tools

Find code using different methods:

- `code_get_symbols_overview`: Get high-level file structure (use first!)
- `code_find_symbol`: Find by name or path pattern
- `code_search_symbols_semantic`: Natural language search
- `code_search_pattern`: Text/regex pattern search
- `code_find_references`: Find symbol usages
- `code_hybrid_search`: Combined semantic + pattern search

### Manipulation Tools

Modify source code:

- `code_replace_symbol`: Replace a symbol's entire body
- `code_insert_after_symbol`: Add code after a symbol
- `code_insert_before_symbol`: Add code before a symbol
- `code_delete_symbol`: Remove a symbol from file

## Recommended Prompts

### For Indexing Projects

```
Index the project at /path/to/project with name "my-project"
```

```
Index only Go and TypeScript files from the project at /src/backend
```

```
What's the indexing status of project "api-service"?
```

```
Show me statistics for project "frontend-app"
```

### For Searching Code (Semantic Search)

```
Find code that handles user authentication and session management
```

```
Search for functions related to form data validation
```

```
Where's the code that processes credit card payments?
```

```
Find implementations of error handling for database operations
```

### For Finding Specific Symbols

```
Find the definition of the UserService class
```

```
Search for the authenticate method in the auth module
```

```
Show me all symbols in file src/services/payment.go
```

```
Find where the validateToken function is used
```

### For Hybrid Search

```
Search for "authentication" using hybrid search (semantic + text pattern)
```

```
Find code related to cache that contains the word "redis"
```

### For Code Manipulation

```
Replace the login method with this new implementation: [code]
```

```
Insert this new function after the validateUser method
```

```
Delete the obsolete oldAuthentication method
```

## Typical Workflow

### 1. Index Project

```
Index the backend project at /home/user/projects/api
```

Wait for indexing to complete:

```
Check indexing status for project "api"
```

### 2. Explore Structure

First get an overview of the file:

```
Show me the structure of file src/auth/handler.go
```

Then search for specific code:

```
Find functions related to JWT in project api
```

### 3. Search and Navigate

Semantic search by concept:

```
Search for code that implements email validation
```

Or search by specific name:

```
Find the processPayment function
```

Find usages:

```
Where is the sendEmail function called?
```

### 4. Modify and Update

Make changes:

```
Replace the validateInput method with: [new code]
```

Update the index:

```
Re-index file src/validation/input.go
```

## Supported Languages

The system supports 14+ languages without additional configuration:

| Language | Extensions | Extracted Symbols |
|----------|------------|-------------------|
| Go | `.go` | functions, methods, structs, interfaces |
| TypeScript | `.ts`, `.tsx` | classes, functions, methods, interfaces |
| JavaScript | `.js`, `.jsx`, `.mjs` | classes, functions, methods |
| Python | `.py` | classes, functions, methods |
| Java | `.java` | classes, methods, interfaces |
| C# | `.cs` | classes, methods, interfaces |
| Rust | `.rs` | functions, structs, traits |
| C/C++ | `.c`, `.h`, `.cpp`, `.hpp` | functions, structs, classes |
| Ruby | `.rb` | classes, modules, methods |
| PHP | `.php` | classes, functions, methods |
| Swift | `.swift` | classes, structs, protocols |
| Kotlin | `.kt` | classes, functions, interfaces |

## Best Practices

### Project Organization

- Index related codebases as separate projects
- Use descriptive project names for easy identification
- Include only the languages you actively work with

### Efficient Searches

**For conceptual search:**
```
Search for functions that handle data caching
Find code related to form validation
```

**For specific symbols:**
```
Find the UserRepository class
Search for the findById method
```

**For code exploration:**
```
Show me the structure of auth/service.go
List all symbols in payment/processor.ts
```

### Keeping Index Updated

- The indexer detects changes automatically if watching is enabled
- For manual updates, use `code_reindex_file` after significant changes
- Re-index the entire project if you've made major structural changes

## Specialized Embedding Models

For optimal code search results, configure a specialized model:

**Recommended models:**
- GGUF: `coderankembed.Q4_K_M.gguf` - CodeRankEmbed optimized for code
- Ollama: `jina/jina-embeddings-v2-base-code` - Jina Code Embeddings
- OpenAI: `text-embedding-3-large` - Also works well for code

**Configuration:**

```bash
# General model + code-specific model
export GOMEM_GGUF_MODEL_PATH="/path/to/nomic-embed-text-v1.5.Q4_K_M.gguf"
export GOMEM_CODE_GGUF_MODEL_PATH="/path/to/coderankembed.Q4_K_M.gguf"
```

See the [Configuration](/en/docs/configuration/) section for more details.

## See More

For detailed documentation of each tool:

```
how_to_use("code")
how_to_use("code_index_project")
how_to_use("code_search_symbols_semantic")
how_to_use("code_find_symbol")
how_to_use("code_hybrid_search")
```

Also check:
- [Code Indexing](/en/docs/code-indexing/) - Complete user guide
