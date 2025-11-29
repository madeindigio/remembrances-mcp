# Code Indexing API Reference

Complete reference for all Code Indexing MCP tools. Each tool includes input parameters, return format, and usage examples.

## Table of Contents

- [Indexing Tools](#indexing-tools)
  - [code_index_project](#code_index_project)
  - [code_index_status](#code_index_status)
  - [code_list_projects](#code_list_projects)
  - [code_delete_project](#code_delete_project)
  - [code_reindex_file](#code_reindex_file)
- [Navigation Tools](#navigation-tools)
  - [code_get_project_stats](#code_get_project_stats)
  - [code_get_file_symbols](#code_get_file_symbols)
  - [code_get_symbols_overview](#code_get_symbols_overview)
- [Search Tools](#search-tools)
  - [code_find_symbol](#code_find_symbol)
  - [code_search_symbols_semantic](#code_search_symbols_semantic)
  - [code_search_pattern](#code_search_pattern)
  - [code_find_references](#code_find_references)
  - [code_hybrid_search](#code_hybrid_search)
- [Manipulation Tools](#manipulation-tools)
  - [code_replace_symbol](#code_replace_symbol)
  - [code_insert_after_symbol](#code_insert_after_symbol)
  - [code_insert_before_symbol](#code_insert_before_symbol)
  - [code_delete_symbol](#code_delete_symbol)

---

## Indexing Tools

### code_index_project

Start indexing a code project for semantic search capabilities.

**Description**: Scans the specified directory for source code files, parses them using tree-sitter to extract symbols (classes, functions, methods, etc.), generates embeddings for semantic search, and stores everything in the database. The indexing runs asynchronously in the background.

**Input Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_path` | string | ✅ | Absolute path to the project directory to index |
| `project_name` | string | ❌ | Human-readable name for the project. If omitted, uses the directory name |
| `languages` | string[] | ❌ | List of programming languages to index. If omitted, indexes all supported languages |

**Example Request**:
```json
{
  "project_path": "/home/user/projects/my-app",
  "project_name": "My Application",
  "languages": ["go", "typescript", "python"]
}
```

**Example Response**:
```json
{
  "message": "Indexing started for project: /home/user/projects/my-app",
  "job_id": "job_1701234567890",
  "status": "pending",
  "created_at": "2025-11-29T10:30:00Z"
}
```

---

### code_index_status

Check the status of an indexing job or list all active jobs.

**Description**: Returns information about indexing job progress, including files processed, symbols found, and any errors encountered.

**Input Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `job_id` | string | ❌ | The job ID returned by code_index_project. If omitted, lists all active jobs |

**Example Request (specific job)**:
```json
{
  "job_id": "job_1701234567890"
}
```

**Example Response (specific job)**:
```json
{
  "job_id": "job_1701234567890",
  "project_id": "home_user_projects_my-app",
  "project_path": "/home/user/projects/my-app",
  "status": "in_progress",
  "progress": 0.45,
  "files_total": 200,
  "files_indexed": 90,
  "symbols_found": 1250,
  "started_at": "2025-11-29T10:30:00Z"
}
```

**Example Response (all jobs)**:
```json
{
  "active_jobs": [
    {
      "job_id": "job_1701234567890",
      "project_path": "/home/user/projects/my-app",
      "status": "in_progress",
      "progress": 0.45,
      "files_indexed": 90,
      "files_total": 200
    }
  ],
  "count": 1
}
```

---

### code_list_projects

List all indexed code projects.

**Description**: Returns a list of all projects that have been indexed for code search, including their names, paths, and indexing status.

**Input Parameters**: None

**Example Response**:
```json
{
  "projects": [
    {
      "id": "code_projects:abc123",
      "project_id": "home_user_projects_my-app",
      "name": "My Application",
      "root_path": "/home/user/projects/my-app",
      "indexing_status": "completed",
      "language_stats": {
        "go": 150,
        "typescript": 80
      },
      "last_indexed_at": "2025-11-29T10:35:00Z"
    }
  ],
  "count": 1
}
```

---

### code_delete_project

Delete an indexed project and all its data.

**Description**: Removes a project and all its indexed files and symbols from the database. This cannot be undone.

**Input Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | ✅ | The project ID to delete |

**Example Request**:
```json
{
  "project_id": "home_user_projects_my-app"
}
```

**Example Response**:
```json
{
  "message": "Project home_user_projects_my-app deleted successfully",
  "project_id": "home_user_projects_my-app"
}
```

---

### code_reindex_file

Re-index a single file in a project.

**Description**: Updates the index for a specific file that may have changed. Useful for keeping the index up to date after file modifications.

**Input Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | ✅ | The project ID containing the file |
| `file_path` | string | ✅ | Relative path to the file within the project |

**Example Request**:
```json
{
  "project_id": "home_user_projects_my-app",
  "file_path": "src/services/user.go"
}
```

**Example Response**:
```json
{
  "message": "File src/services/user.go reindexed successfully",
  "project_id": "home_user_projects_my-app",
  "file_path": "src/services/user.go"
}
```

---

## Navigation Tools

### code_get_project_stats

Get detailed statistics for an indexed code project.

**Description**: Returns comprehensive statistics about an indexed project including file counts, symbol counts, language breakdown, and symbol type distribution.

**Input Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | ✅ | The project ID to get statistics for |

**Example Request**:
```json
{
  "project_id": "home_user_projects_my-app"
}
```

**Example Response**:
```json
{
  "project_id": "home_user_projects_my-app",
  "name": "My Application",
  "root_path": "/home/user/projects/my-app",
  "indexing_status": "completed",
  "language_stats": {
    "go": 150,
    "typescript": 80
  },
  "last_indexed_at": "2025-11-29T10:35:00Z",
  "files_count": 230,
  "symbols_count": 3500,
  "symbols_by_type": [
    {"symbol_type": "function", "count": 1200},
    {"symbol_type": "method", "count": 1500},
    {"symbol_type": "class", "count": 200},
    {"symbol_type": "interface", "count": 100}
  ],
  "files_by_language": [
    {"language": "go", "count": 150},
    {"language": "typescript", "count": 80}
  ]
}
```

---

### code_get_file_symbols

Get all symbols from a specific file with hierarchical structure.

**Description**: Returns all code symbols (classes, functions, methods, etc.) found in a specific file, organized in a hierarchical tree structure showing parent-child relationships.

**Input Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | ✅ | The project ID containing the file |
| `relative_path` | string | ✅ | Relative path to the file within the project |
| `include_body` | boolean | ❌ | Whether to include the source code body of each symbol |

**Example Request**:
```json
{
  "project_id": "home_user_projects_my-app",
  "relative_path": "src/services/user_service.go",
  "include_body": false
}
```

**Example Response**:
```json
{
  "project_id": "home_user_projects_my-app",
  "relative_path": "src/services/user_service.go",
  "symbols": [
    {
      "id": "code_symbols:abc123",
      "name": "UserService",
      "name_path": "UserService",
      "symbol_type": "struct",
      "start_line": 10,
      "end_line": 15,
      "signature": "type UserService struct",
      "children": [
        {
          "id": "code_symbols:abc124",
          "name": "Authenticate",
          "name_path": "UserService/Authenticate",
          "symbol_type": "method",
          "start_line": 20,
          "end_line": 45,
          "signature": "func (s *UserService) Authenticate(email, password string) (*User, error)"
        }
      ]
    }
  ],
  "total_count": 5
}
```

---

### code_get_symbols_overview

Get a high-level overview of code symbols in a file.

**Description**: Lists top-level symbols (classes, functions, interfaces, etc.) in a specific file, showing their names, types, line numbers, and signatures. Does not include source code bodies.

**Input Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | ✅ | The project ID to search in |
| `relative_path` | string | ✅ | Relative path to the file within the project |
| `max_results` | integer | ❌ | Maximum number of symbols to return. Default is 100 |

**Example Request**:
```json
{
  "project_id": "home_user_projects_my-app",
  "relative_path": "src/services/user.ts",
  "max_results": 50
}
```

**Example Response**:
```json
{
  "file_path": "src/services/user.ts",
  "symbols": [
    {
      "name": "UserService",
      "type": "class",
      "name_path": "UserService",
      "start_line": 5,
      "end_line": 120,
      "signature": "export class UserService"
    },
    {
      "name": "createUser",
      "type": "function",
      "name_path": "createUser",
      "start_line": 125,
      "end_line": 150,
      "signature": "export function createUser(data: UserInput): Promise<User>"
    }
  ],
  "count": 2
}
```

---

## Search Tools

### code_find_symbol

Find symbols by name path pattern with optional filtering.

**Description**: Retrieves information on symbols based on name path patterns. Supports exact matches, suffix matches, and simple name matches.

**Input Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | ✅ | The project ID to search in |
| `name_path_pattern` | string | ✅ | Symbol name or path pattern |
| `relative_path` | string | ❌ | Restrict search to this file or directory |
| `depth` | integer | ❌ | Include children up to this depth level |
| `include_body` | boolean | ❌ | Include source code in results |
| `include_kinds` | string[] | ❌ | Filter by symbol types |
| `exclude_kinds` | string[] | ❌ | Exclude these symbol types |
| `substring_matching` | boolean | ❌ | Enable partial name matching |

**Name Path Patterns**:
- `method` - Matches any symbol named "method"
- `Class/method` - Matches symbols with this suffix
- `/Class/method` - Exact match of the full name path

**Example Request**:
```json
{
  "project_id": "home_user_projects_my-app",
  "name_path_pattern": "UserService/Authenticate",
  "include_body": true,
  "depth": 1
}
```

**Example Response**:
```json
{
  "symbols": [
    {
      "id": "code_symbols:abc123",
      "name": "Authenticate",
      "name_path": "UserService/Authenticate",
      "symbol_type": "method",
      "file_path": "src/services/user.go",
      "language": "go",
      "start_line": 20,
      "end_line": 45,
      "signature": "func (s *UserService) Authenticate(email, password string) (*User, error)",
      "source_code": "func (s *UserService) Authenticate(email, password string) (*User, error) {\n\t// ...\n}",
      "children": []
    }
  ],
  "count": 1
}
```

---

### code_search_symbols_semantic

Search for code symbols using natural language semantic similarity.

**Description**: Performs semantic search on code symbols using vector embeddings. Finds code by meaning rather than exact text matches.

**Input Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | ✅ | The project ID to search in |
| `query` | string | ✅ | Natural language query describing what you're looking for |
| `limit` | integer | ❌ | Maximum number of results. Default is 10 |
| `languages` | string[] | ❌ | Filter by programming languages |
| `symbol_types` | string[] | ❌ | Filter by symbol types |

**Example Request**:
```json
{
  "project_id": "home_user_projects_my-app",
  "query": "user authentication and password validation",
  "limit": 10,
  "languages": ["go"],
  "symbol_types": ["function", "method"]
}
```

**Example Response**:
```json
{
  "query": "user authentication and password validation",
  "results": [
    {
      "name": "Authenticate",
      "name_path": "UserService/Authenticate",
      "symbol_type": "method",
      "file_path": "src/services/user.go",
      "language": "go",
      "start_line": 20,
      "end_line": 45,
      "similarity": 0.89,
      "signature": "func (s *UserService) Authenticate(email, password string) (*User, error)"
    },
    {
      "name": "ValidatePassword",
      "name_path": "ValidatePassword",
      "symbol_type": "function",
      "file_path": "src/utils/auth.go",
      "language": "go",
      "start_line": 10,
      "end_line": 25,
      "similarity": 0.82,
      "signature": "func ValidatePassword(password string) error"
    }
  ],
  "count": 2
}
```

---

### code_search_pattern

Search for text patterns in code with optional regex support.

**Description**: Searches for text patterns in source code across the indexed project. Supports both literal text and regular expressions.

**Input Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | ✅ | The project ID to search in |
| `pattern` | string | ✅ | Text pattern or regex to search for |
| `is_regex` | boolean | ❌ | Treat pattern as regular expression |
| `languages` | string[] | ❌ | Filter by programming languages |
| `symbol_types` | string[] | ❌ | Filter by symbol types |
| `case_sensitive` | boolean | ❌ | Enable case-sensitive matching. Default is false |
| `limit` | integer | ❌ | Maximum number of results. Default is 50 |

**Example Request**:
```json
{
  "project_id": "home_user_projects_my-app",
  "pattern": "TODO|FIXME|HACK",
  "is_regex": true,
  "limit": 20
}
```

**Example Response**:
```json
{
  "pattern": "TODO|FIXME|HACK",
  "results": [
    {
      "name": "processOrder",
      "symbol_type": "function",
      "file_path": "src/services/order.go",
      "start_line": 45,
      "match_line": 52,
      "match_text": "// TODO: Add validation"
    }
  ],
  "count": 1
}
```

---

### code_find_references

Find references to a symbol throughout the codebase.

**Description**: Searches for usages of a specific symbol across all indexed files.

**Input Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | ✅ | The project ID to search in |
| `symbol_id` | string | ❌ | ID of the symbol to find references for |
| `symbol_name` | string | ❌ | Name of the symbol (alternative to symbol_id) |
| `include_kinds` | string[] | ❌ | Filter referencing symbols by type |
| `limit` | integer | ❌ | Maximum number of references. Default is 50 |

**Example Request**:
```json
{
  "project_id": "home_user_projects_my-app",
  "symbol_name": "UserService",
  "limit": 20
}
```

**Example Response**:
```json
{
  "target_symbol": "UserService",
  "references": [
    {
      "name": "main",
      "type": "function",
      "name_path": "main",
      "file_path": "cmd/server/main.go",
      "language": "go",
      "start_line": 10,
      "end_line": 30,
      "reference_lines": [15, 22]
    }
  ],
  "count": 1
}
```

---

### code_hybrid_search

Perform advanced hybrid search combining semantic similarity with structural filters.

**Description**: Combines vector similarity search with filters for language, symbol type, and file paths. Optionally searches in code chunks for better coverage of large symbols.

**Input Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | ✅ | The project ID to search in |
| `query` | string | ✅ | Natural language query for semantic search |
| `languages` | string[] | ❌ | Filter by programming languages |
| `symbol_types` | string[] | ❌ | Filter by symbol types |
| `path_pattern` | string | ❌ | Filter by file path pattern (e.g., "src/auth/**") |
| `include_chunks` | boolean | ❌ | Search in code chunks for better large-symbol coverage |
| `limit` | integer | ❌ | Maximum number of results. Default is 20 |

**Example Request**:
```json
{
  "project_id": "home_user_projects_my-app",
  "query": "user authentication and session management",
  "languages": ["go", "typescript"],
  "symbol_types": ["function", "method"],
  "path_pattern": "src/auth/**",
  "include_chunks": true,
  "limit": 20
}
```

**Example Response**:
```json
{
  "query": "user authentication and session management",
  "results": [
    {
      "source": "symbol",
      "name": "Authenticate",
      "name_path": "UserService/Authenticate",
      "symbol_type": "method",
      "file_path": "src/auth/user.go",
      "language": "go",
      "start_line": 20,
      "end_line": 45,
      "similarity": 0.91,
      "preview": "func (s *UserService) Authenticate(email, password string) (*User, error)"
    },
    {
      "source": "chunk",
      "name": "SessionManager",
      "symbol_type": "class",
      "file_path": "src/auth/session.ts",
      "language": "typescript",
      "start_line": 100,
      "end_line": 200,
      "similarity": 0.85,
      "chunk_index": 2,
      "preview": "async validateSession(token: string): Promise<Session> {\n  // Validate JWT..."
    }
  ],
  "count": 2,
  "filters": {
    "languages": ["go", "typescript"],
    "symbol_types": ["function", "method"],
    "path_pattern": "src/auth/**",
    "include_chunks": true
  }
}
```

---

## Manipulation Tools

### code_replace_symbol

Replace the body of a symbol with new code.

**Description**: Replaces the entire source code of a symbol with new content. The symbol must exist and be found by its name path.

**Input Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | ✅ | The project ID containing the symbol |
| `file_path` | string | ✅ | Relative path to the file |
| `name_path` | string | ✅ | Name path of the symbol to replace |
| `new_body` | string | ✅ | The new source code for the symbol |

**Example Request**:
```json
{
  "project_id": "home_user_projects_my-app",
  "file_path": "src/utils/helper.go",
  "name_path": "formatDate",
  "new_body": "func formatDate(t time.Time) string {\n\treturn t.Format(\"2006-01-02\")\n}"
}
```

**Example Response**:
```json
{
  "success": true,
  "message": "Symbol formatDate replaced successfully",
  "file_path": "src/utils/helper.go",
  "name_path": "formatDate"
}
```

---

### code_insert_after_symbol

Insert code after a symbol.

**Description**: Inserts new code immediately after the end of a symbol's definition. Useful for adding new functions or methods.

**Input Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | ✅ | The project ID |
| `file_path` | string | ✅ | Relative path to the file |
| `name_path` | string | ✅ | Name path of the symbol to insert after |
| `content` | string | ✅ | The code to insert |

**Example Request**:
```json
{
  "project_id": "home_user_projects_my-app",
  "file_path": "src/services/user.go",
  "name_path": "UserService/Authenticate",
  "content": "\n\nfunc (s *UserService) Logout(userID string) error {\n\treturn s.sessionStore.Delete(userID)\n}"
}
```

**Example Response**:
```json
{
  "success": true,
  "message": "Content inserted after symbol UserService/Authenticate",
  "file_path": "src/services/user.go"
}
```

---

### code_insert_before_symbol

Insert code before a symbol.

**Description**: Inserts new code immediately before a symbol's definition. Useful for adding imports or comments.

**Input Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | ✅ | The project ID |
| `file_path` | string | ✅ | Relative path to the file |
| `name_path` | string | ✅ | Name path of the symbol to insert before |
| `content` | string | ✅ | The code to insert |

**Example Request**:
```json
{
  "project_id": "home_user_projects_my-app",
  "file_path": "src/services/user.go",
  "name_path": "UserService",
  "content": "// UserService handles user authentication and management\n"
}
```

**Example Response**:
```json
{
  "success": true,
  "message": "Content inserted before symbol UserService",
  "file_path": "src/services/user.go"
}
```

---

### code_delete_symbol

Delete a symbol from a file.

**Description**: Removes a symbol and its source code from a file. Use with caution as this modifies the actual source file.

**Input Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_id` | string | ✅ | The project ID |
| `file_path` | string | ✅ | Relative path to the file |
| `name_path` | string | ✅ | Name path of the symbol to delete |

**Example Request**:
```json
{
  "project_id": "home_user_projects_my-app",
  "file_path": "src/utils/deprecated.go",
  "name_path": "oldFunction"
}
```

**Example Response**:
```json
{
  "success": true,
  "message": "Symbol oldFunction deleted successfully",
  "file_path": "src/utils/deprecated.go"
}
```

---

## Error Handling

All tools return errors in this format:

```json
{
  "error": "Error message describing what went wrong"
}
```

Common error codes:
- `project_id is required` - Missing required parameter
- `project not found` - The specified project doesn't exist
- `symbol not found` - Could not find the specified symbol
- `storage does not support code operations` - Database not properly configured
- `failed to parse file` - File has syntax errors

## See Also

- [CODE_INDEXING.md](CODE_INDEXING.md) - User guide and concepts
- [TREE_SITTER_LANGUAGES.md](TREE_SITTER_LANGUAGES.md) - Language support details
