---
title: "Code Indexing"
linkTitle: "Code Indexing"
weight: 5
description: >
  Index your codebase for semantic search and intelligent navigation
---

The Code Indexing feature enables AI agents to understand, search, and navigate your codebase using semantic search. Instead of simple text matching, your AI assistant can find code by meaning – search for "user authentication" and find the relevant login functions, session handlers, and security modules.

## What is Code Indexing?

Code Indexing analyzes your source code using [tree-sitter](https://tree-sitter.github.io/tree-sitter/) for accurate parsing across multiple programming languages. It extracts meaningful symbols (classes, functions, methods, interfaces) and creates vector embeddings for semantic search.

This means you can:
- **Search code by meaning**: Find "error handling for database operations" and get relevant try-catch blocks, error handlers, and logging code
- **Navigate large codebases**: Quickly find implementations, references, and related code
- **Get intelligent context**: Help AI agents understand your code structure and patterns

## Supported Languages

Remembrances supports **14+ programming languages** out of the box:

| Language | Extensions | Symbols Extracted |
|----------|------------|-------------------|
| Go | `.go` | functions, methods, structs, interfaces, constants |
| TypeScript | `.ts`, `.tsx` | classes, functions, methods, interfaces, types |
| JavaScript | `.js`, `.jsx`, `.mjs` | classes, functions, methods, exports |
| Python | `.py` | classes, functions, methods, decorators |
| Java | `.java` | classes, methods, interfaces, enums |
| C# | `.cs` | classes, methods, interfaces, structs |
| Rust | `.rs` | functions, structs, traits, impls, enums |
| C/C++ | `.c`, `.h`, `.cpp`, `.hpp` | functions, structs, classes |
| Ruby | `.rb` | classes, modules, methods |
| PHP | `.php` | classes, functions, methods, interfaces |
| Swift | `.swift` | classes, structs, protocols, functions |
| Kotlin | `.kt` | classes, functions, interfaces |
| Scala | `.scala` | classes, objects, traits, functions |
| Bash | `.sh`, `.bash` | functions |

## How to Use Code Indexing

### 1. Index a Project

Use the `code_index_project` tool to start indexing your codebase:

```
code_index_project({
  "project_path": "/path/to/your/project",
  "project_name": "My Project",
  "languages": ["go", "typescript", "python"]
})
```

The indexing runs in the background. For large projects, you can check progress with `code_index_status`.

### 2. Search for Code

**Semantic Search** – Find code by describing what you're looking for:

```
code_semantic_search({
  "project_id": "my-project",
  "query": "user authentication and session management",
  "limit": 10
})
```

This returns relevant code snippets ranked by semantic similarity, even if they don't contain the exact words in your query.

**Find Symbols** – Search by function, class, or method name:

```
code_find_symbol({
  "project_id": "my-project",
  "name_path_pattern": "UserService/authenticate",
  "include_body": true
})
```

### 3. Navigate Code Structure

**Get File Overview** – See all symbols in a file:

```
code_get_file_symbols({
  "project_id": "my-project",
  "relative_path": "src/services/auth.go"
})
```

**Find References** – Find where a symbol is used:

```
code_find_references({
  "project_id": "my-project",
  "symbol_name": "validateToken"
})
```

**Get Call Hierarchy** – See what calls a function and what it calls:

```
code_get_call_hierarchy({
  "project_id": "my-project",
  "symbol_name": "processPayment"
})
```

### 4. Manage Projects

**List Projects** – See all indexed projects:

```
code_list_projects()
```

**Get Project Statistics** – See indexing details:

```
code_get_project_stats({
  "project_id": "my-project"
})
```

**Re-index a File** – Update after changes:

```
code_reindex_file({
  "project_id": "my-project",
  "relative_path": "src/services/auth.go"
})
```

## Code Manipulation

Beyond search and navigation, Remembrances provides tools for modifying code:

**Get Symbol Body** – Retrieve the full implementation:

```
code_get_symbol_body({
  "project_id": "my-project",
  "symbol_name": "validateToken"
})
```

**Replace Symbol Body** – Update an implementation:

```
code_replace_symbol_body({
  "project_id": "my-project",
  "symbol_name": "validateToken",
  "new_body": "func validateToken(token string) bool {\n  // New implementation\n}"
})
```

**Insert Symbol** – Add new code at a specific location:

```
code_insert_symbol({
  "project_id": "my-project",
  "file_path": "src/services/auth.go",
  "position": "after:validateToken",
  "code": "func refreshToken(token string) (string, error) {\n  // Implementation\n}"
})
```

## Best Practices

### Project Organization

- Index related codebases as separate projects
- Use descriptive project names for easy identification
- Include only the languages you actively work with

### Efficient Searching

- Use semantic search for conceptual queries ("error handling", "data validation")
- Use symbol search for specific functions or classes you know by name
- Combine searches to narrow down results

### Keeping Index Updated

- The indexer detects file changes and re-indexes automatically when watching is enabled
- For manual updates, use `code_reindex_file` after significant changes
- Delete and re-index a project if you've made major structural changes

## Specialized Code Embedding Models

For optimal code search results, you can configure a dedicated embedding model specialized for code. See the [Configuration](/docs/configuration/) page for details on setting up code-specific embedding models like CodeRankEmbed or Jina Code Embeddings.
