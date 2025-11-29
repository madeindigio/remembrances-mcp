# Code Indexing System

The Code Indexing System provides powerful semantic code search and navigation capabilities for AI agents. It uses tree-sitter for accurate parsing across 14+ programming languages and vector embeddings for semantic similarity search.

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [How It Works](#how-it-works)
- [Supported Languages](#supported-languages)
- [MCP Tools Overview](#mcp-tools-overview)
- [Configuration](#configuration)
- [Performance Considerations](#performance-considerations)
- [Troubleshooting](#troubleshooting)

## Features

- **Multi-Language Support**: Parse and index 14+ programming languages using tree-sitter
- **Semantic Search**: Find code by meaning, not just text matching
- **Symbol Extraction**: Automatically extract classes, functions, methods, interfaces, and more
- **Hierarchical Structure**: Maintain parent-child relationships between symbols
- **Incremental Indexing**: Only re-index files that have changed
- **Chunk-Based Search**: Large symbols are chunked for better semantic coverage
- **Hybrid Search**: Combine semantic similarity with structural filters

## Quick Start

### 1. Index a Project

Use the `code_index_project` tool to start indexing:

```json
{
  "project_path": "/path/to/your/project",
  "project_name": "My Project",
  "languages": ["go", "typescript", "python"]
}
```

Indexing runs asynchronously. Check status with `code_index_status`.

### 2. Search for Code

**Semantic Search** - Find code by meaning:
```json
{
  "project_id": "my-project",
  "query": "user authentication and session management",
  "limit": 10
}
```

**Find Symbol** - Find by name or path:
```json
{
  "project_id": "my-project",
  "name_path_pattern": "UserService/authenticate",
  "include_body": true
}
```

### 3. Navigate Code

**Get File Overview**:
```json
{
  "project_id": "my-project",
  "relative_path": "src/services/user.go"
}
```

**Get Project Stats**:
```json
{
  "project_id": "my-project"
}
```

## How It Works

### Indexing Pipeline

1. **File Discovery**: Scans project directory for supported source files
2. **Parsing**: Uses tree-sitter to build AST for each file
3. **Symbol Extraction**: Walks AST to identify functions, classes, methods, etc.
4. **Embedding Generation**: Creates vector embeddings for semantic search
5. **Chunking**: Large symbols (>1500 chars) are split for better coverage
6. **Storage**: Saves symbols to SurrealDB with vector indexes

### Symbol Types Extracted

| Type | Description | Languages |
|------|-------------|-----------|
| `class` | Class definitions | Go, TS, JS, Java, Python, Ruby, C#, etc. |
| `function` | Standalone functions | All languages |
| `method` | Class/struct methods | All languages |
| `interface` | Interface definitions | Go, TS, Java, C# |
| `struct` | Struct definitions | Go, Rust, C, C++ |
| `enum` | Enumeration types | Most languages |
| `constant` | Constants/const declarations | All languages |
| `variable` | Variable declarations | All languages |
| `property` | Class properties/fields | TS, JS, Java, Python |
| `type` | Type aliases | Go, TS, Rust |

### Name Path Convention

Symbols are identified by their "name path" - a hierarchical path within a file:

- `MyClass` - Top-level class
- `MyClass/myMethod` - Method within a class
- `MyClass/myMethod/innerFunc` - Nested function
- `/MyClass/myMethod` - Absolute path (exact match)

## Supported Languages

| Language | Extensions | Full Support |
|----------|------------|--------------|
| Go | `.go` | ✅ |
| TypeScript | `.ts`, `.mts`, `.cts` | ✅ |
| JavaScript | `.js`, `.mjs`, `.cjs`, `.jsx` | ✅ |
| TSX | `.tsx` | ✅ |
| Python | `.py`, `.pyw`, `.pyi` | ✅ |
| Java | `.java` | ✅ |
| Kotlin | `.kt`, `.kts` | ✅ |
| Rust | `.rs` | ✅ |
| PHP | `.php`, `.phtml` | ✅ |
| Swift | `.swift` | ✅ |
| C | `.c` | ✅ |
| C++ | `.cpp`, `.cc`, `.cxx`, `.hpp` | ✅ |
| C# | `.cs` | ✅ |
| Ruby | `.rb`, `.rake` | ✅ |
| Scala | `.scala`, `.sc` | ✅ |
| Bash | `.sh`, `.bash`, `.zsh` | Partial |

## MCP Tools Overview

### Indexing Tools

| Tool | Description |
|------|-------------|
| `code_index_project` | Start indexing a project directory |
| `code_index_status` | Check indexing job status |
| `code_list_projects` | List all indexed projects |
| `code_delete_project` | Remove a project from the index |
| `code_reindex_file` | Re-index a single file |

### Navigation Tools

| Tool | Description |
|------|-------------|
| `code_get_project_stats` | Get project statistics |
| `code_get_file_symbols` | Get hierarchical file structure |
| `code_get_symbols_overview` | Get top-level symbols in a file |

### Search Tools

| Tool | Description |
|------|-------------|
| `code_find_symbol` | Find symbol by name path pattern |
| `code_search_symbols_semantic` | Semantic similarity search |
| `code_search_pattern` | Text/regex pattern search |
| `code_find_references` | Find symbol references |
| `code_hybrid_search` | Combined semantic + filters |

### Manipulation Tools

| Tool | Description |
|------|-------------|
| `code_replace_symbol` | Replace symbol body |
| `code_insert_after_symbol` | Insert code after symbol |
| `code_insert_before_symbol` | Insert code before symbol |
| `code_delete_symbol` | Delete a symbol |

For detailed API documentation, see [CODE_INDEXING_API.md](CODE_INDEXING_API.md).

## Configuration

### Environment Variables

Code indexing uses the same configuration as the main application:

```bash
# Required: Embedding model configuration
GOMEM_OLLAMA_MODEL=nomic-embed-text
# or
GOMEM_OPENAI_KEY=sk-xxx

# Database configuration
GOMEM_DB_PATH=./data.db
GOMEM_SURREALDB_NAMESPACE=test
GOMEM_SURREALDB_DATABASE=test
```

### Indexer Settings

The indexer uses these default settings:

| Setting | Default | Description |
|---------|---------|-------------|
| Concurrency | 4 | Parallel file processing workers |
| Batch Size | 10 | Embeddings generated per batch |
| Max Source Length | 10000 | Maximum source code stored per symbol |
| Chunk Threshold | 1500 | Symbols larger than this are chunked |
| Chunk Overlap | 200 | Character overlap between chunks |

## Performance Considerations

### Initial Indexing

- **Large projects**: May take several minutes for projects with 1000+ files
- **Embedding generation**: The bottleneck is typically embedding API calls
- **Memory**: Tree-sitter parsing is memory-efficient

### Incremental Updates

- **File hashing**: Only changed files are re-indexed
- **Symbol tracking**: Old symbols are cleaned up before new ones are added
- **Chunk updates**: Chunks are regenerated when symbols change

### Search Performance

- **Vector indexes**: MTREE indexes on embeddings for fast similarity search
- **Limit results**: Always use reasonable limits (10-50) for semantic search
- **Filter first**: Use language/type filters to reduce search space

### Best Practices

1. **Index only what you need**: Use language filters during indexing
2. **Use hybrid search**: Combine semantic search with structural filters
3. **Leverage hierarchical structure**: Use `code_get_file_symbols` for navigation
4. **Re-index after major changes**: Use `code_reindex_file` for single files

## Troubleshooting

### Common Issues

**"Storage does not support code operations"**
- Ensure you're using a storage implementation that supports code indexing
- Check that the database has been migrated to version 10+

**"Failed to parse file"**
- The file may have syntax errors that tree-sitter cannot handle
- Check if the language is supported

**"Project not found"**
- Verify the project_id matches an indexed project
- Use `code_list_projects` to see available projects

**Slow indexing**
- Reduce concurrency if hitting API rate limits
- Consider indexing only specific languages

**Empty search results**
- Verify embeddings were generated (check logs during indexing)
- Try broader search terms
- Ensure the project has completed indexing

### Logs

Enable debug logging to see detailed information:

```bash
GOMEM_LOG_LEVEL=debug ./remembrances-mcp
```

### Database Schema

The code indexing uses these tables:

- `code_projects` - Project metadata and status
- `code_files` - Indexed files with hashes
- `code_symbols` - Extracted symbols with embeddings
- `code_chunks` - Chunked content for large symbols
- `code_indexing_jobs` - Async job tracking

## See Also

- [CODE_INDEXING_API.md](CODE_INDEXING_API.md) - Complete API reference
- [TREE_SITTER_LANGUAGES.md](TREE_SITTER_LANGUAGES.md) - Language support details
