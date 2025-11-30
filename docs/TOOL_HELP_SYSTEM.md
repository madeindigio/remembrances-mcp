# Tool Help System (how_to_use)

The `how_to_use` tool provides on-demand documentation for remembrances-mcp tools, significantly reducing initial context token consumption while maintaining full functionality.

## Overview

Instead of loading detailed descriptions for all 37+ tools at startup (consuming ~15,000+ tokens), the system now provides:
- **Minimal descriptions** (1-2 lines) for each tool in the initial context
- **Full documentation** available on-demand via `how_to_use()`

This results in approximately **85% reduction** in initial token consumption.

## Usage

### Get Complete Overview
```
how_to_use()
```
Returns a high-level overview of all tool categories (memory, knowledge base, code indexing).

### Get Group Documentation
```
how_to_use("memory")     # Memory tools (facts, vectors, graph)
how_to_use("kb")         # Knowledge base tools
how_to_use("code")       # Code indexing and search tools
```

### Get Specific Tool Documentation
```
how_to_use("remembrance_save_fact")
how_to_use("kb_add_document")
how_to_use("index_repository")
how_to_use("search_code")
```

## Tool Categories

### Memory Tools (`memory`)
- **Fact Tools**: Key-value storage (`remembrance_save_fact`, `remembrance_get_fact`, etc.)
- **Vector Tools**: Semantic search (`remembrance_add_vector`, `remembrance_search_vectors`, etc.)
- **Graph Tools**: Entity/relationship management (`remembrance_create_entity`, `remembrance_traverse_graph`, etc.)

### Knowledge Base Tools (`kb`)
- Document storage and retrieval (`kb_add_document`, `kb_get_document`, etc.)
- Semantic document search (`kb_search_documents`)

### Code Tools (`code`)
- **Indexing**: Repository/file indexing (`index_repository`, `index_directory`, etc.)
- **Search**: Code and symbol search (`search_code`, `search_symbols`, etc.)
- **Analysis**: Code analysis tools (`get_file_summary`, `find_references`, etc.)

## Documentation Structure

The documentation is embedded in the binary using Go's `//go:embed` directive:

```
pkg/mcp_tools/docs/
├── overview.txt           # Main overview
├── memory_group.txt       # Memory tools group docs
├── kb_group.txt          # KB tools group docs
├── code_group.txt        # Code tools group docs
└── tools/                # Individual tool docs
    ├── remembrance_save_fact.txt
    ├── kb_add_document.txt
    ├── index_repository.txt
    └── ... (37 total files)
```

## Implementation Details

### File: `pkg/mcp_tools/help_tool.go`

- Uses `embed.FS` to include documentation at compile time
- Routes requests based on topic:
  - Empty/overview → `docs/overview.txt`
  - Group names → `docs/{group}_group.txt`
  - Tool names → `docs/tools/{tool_name}.txt`
- Handles unknown topics gracefully with helpful error message

### Tool Description Pattern

All tools now follow this pattern:
```go
Description: "Brief one-line description. Use how_to_use(\"tool_name\") for details."
```

## Token Savings Estimate

| Metric | Before | After | Savings |
|--------|--------|-------|---------|
| Initial descriptions | ~15,000 tokens | ~2,500 tokens | ~83% |
| Per-tool documentation | N/A | ~200-500 tokens on demand | Loaded only when needed |

## Testing

Unit tests are available in `pkg/mcp_tools/help_tool_test.go`:
- Tests for overview loading
- Tests for group documentation
- Tests for individual tool documentation
- Tests for embedded file verification
- Tests for error handling

Run tests:
```bash
go test -v ./pkg/mcp_tools/...
```

Note: Tests require native libraries (llama.cpp, surrealdb) to be linked. In environments without these libraries, compilation verification can be done with:
```bash
go build ./pkg/mcp_tools/...
```

## Adding New Tool Documentation

1. Create `pkg/mcp_tools/docs/tools/{tool_name}.txt`
2. Follow the standard format:
   ```
   === {TOOL_NAME} ===
   
   PURPOSE:
   Brief description of what the tool does.
   
   ARGUMENTS:
   - arg1 (type, required/optional): Description
   - arg2 (type, required/optional): Description
   
   EXAMPLE:
   {tool_name}(arg1="value", arg2="value")
   
   RELATED TOOLS:
   - related_tool_1
   - related_tool_2
   ```

3. Update the minimal description in the tool's factory function to reference `how_to_use()`
