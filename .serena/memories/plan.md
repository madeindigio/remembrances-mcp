# Plan: Tool Description Optimization with `how_to_use` Tool

**Feature**: Reduce initial context token consumption by creating a help system  
**Status**: Not Started  
**Created**: November 29, 2025  
**Branch**: To be created from `main` after merging `feature/ast-embeddings-import`

---

## Problem Statement

Currently, all MCP tool descriptions are verbose and include detailed examples directly in the tool definition. This consumes a significant number of tokens when AI agents load the available tools. 

**Goal**: Create a new `how_to_use` tool that provides detailed help on-demand, while keeping default tool descriptions minimal.

---

## Phase Overview

| Phase | Title | Description | Status |
|-------|-------|-------------|--------|
| 1 | Text Files Structure | Create embedded text files for tool documentation | Not Started |
| 2 | Help Tool Implementation | Implement `how_to_use` tool with go:embed | Not Started |
| 3 | Tool Description Refactoring | Reduce existing tool descriptions to minimal | Not Started |
| 4 | Testing & Documentation | Test the new system and document usage | Not Started |

---

## PHASE 1: Text Files Structure

**Objective**: Create documentation files that will be loaded via `go:embed`

### Tasks

1. Create directory structure: `pkg/mcp_tools/docs/`
2. Create `overview.txt` - Brief overview of all tool categories
3. Create `kb_group.txt` - Knowledge Base tools detailed documentation
4. Create `memory_group.txt` - Memory tools (facts, vectors, graph) documentation
5. Create `code_group.txt` - Code indexing tools documentation
6. Create individual tool files in `pkg/mcp_tools/docs/tools/`:
   - One file per tool with name matching tool name (e.g., `remembrance_save_fact.txt`)

### Files to Create

```
pkg/mcp_tools/docs/
├── overview.txt              # Tool categories summary
├── kb_group.txt              # KB tools group documentation
├── memory_group.txt          # Memory tools group documentation  
├── code_group.txt            # Code tools group documentation
└── tools/
    ├── remembrance_save_fact.txt
    ├── remembrance_get_fact.txt
    ├── ... (one per tool)
    └── code_hybrid_search.txt
```

### Documentation Format

Each tool file should contain:
- Tool name
- Brief description (1-2 sentences)
- When to use
- Arguments with types and defaults
- Usage examples
- Related tools

### Dependencies

- None (first phase)

---

## PHASE 2: Help Tool Implementation

**Objective**: Implement the `how_to_use` MCP tool using go:embed

### Tasks

1. Create `pkg/mcp_tools/help_tool.go`
2. Implement go:embed to load all documentation files
3. Create HowToUseInput type with optional `topic` argument
4. Implement tool logic:
   - No argument → Return overview.txt (tool categories summary)
   - `topic="kb"` → Return kb_group.txt
   - `topic="memory"` → Return memory_group.txt
   - `topic="code"` → Return code_group.txt
   - `topic="<tool_name>"` → Return tools/<tool_name>.txt
5. Register tool in `tools.go`
6. Handle errors for unknown topics

### Input Schema

```go
type HowToUseInput struct {
    Topic string `json:"topic,omitempty" jsonschema:"description=Optional: tool name or group (kb/memory/code). If omitted returns overview."`
}
```

### Implementation Details

```go
//go:embed docs/*.txt docs/tools/*.txt
var docsFS embed.FS
```

### Dependencies

- Phase 1 (text files must exist)

---

## PHASE 3: Tool Description Refactoring

**Objective**: Reduce all existing tool descriptions to minimal versions

### Tasks

1. Create backup of current descriptions (in a memory or comment)
2. Refactor fact_tools.go descriptions:
   - Keep only: tool name + one-line description
   - Example: `"Save a key-value fact. Use 'how_to_use' tool for details."`
3. Refactor vector_tools.go descriptions
4. Refactor graph_tools.go descriptions
5. Refactor kb_tools.go descriptions
6. Refactor misc_tools.go descriptions
7. Refactor code_indexing_tools.go descriptions
8. Refactor code_search_tools.go descriptions
9. Refactor code_manipulation_tools.go descriptions

### Description Template

```go
tool, err := protocol.NewTool("remembrance_save_fact", 
    `Save a key-value fact for a user. Use how_to_use("remembrance_save_fact") for details.`,
    SaveFactInput{})
```

### Token Savings Estimate

| Tool File | Current Lines | Target Lines | Savings |
|-----------|---------------|--------------|---------|
| fact_tools.go | ~50 desc lines | ~8 lines | ~84% |
| vector_tools.go | ~60 desc lines | ~8 lines | ~87% |
| graph_tools.go | ~50 desc lines | ~8 lines | ~84% |
| kb_tools.go | ~60 desc lines | ~8 lines | ~87% |
| misc_tools.go | ~40 desc lines | ~4 lines | ~90% |
| code_indexing_tools.go | ~80 desc lines | ~14 lines | ~82% |
| code_search_tools.go | ~100 desc lines | ~12 lines | ~88% |
| code_manipulation_tools.go | ~60 desc lines | ~8 lines | ~87% |

**Total estimated reduction**: ~85% of description tokens

### Dependencies

- Phase 2 (help tool must work before removing descriptions)

---

## PHASE 4: Testing & Documentation

**Objective**: Verify the system works and document it

### Tasks

1. Build and verify compilation
2. Test `how_to_use` with no arguments
3. Test `how_to_use` with each group: kb, memory, code
4. Test `how_to_use` with individual tool names
5. Test error handling for unknown topics
6. Update README.md with new `how_to_use` tool documentation
7. Create docs/TOOL_HELP_SYSTEM.md explaining the architecture
8. Measure actual token savings before/after

### Test Cases

```
how_to_use()                          → overview
how_to_use("kb")                      → KB tools documentation
how_to_use("memory")                  → Memory tools documentation
how_to_use("code")                    → Code tools documentation
how_to_use("remembrance_save_fact")   → Specific tool docs
how_to_use("invalid_tool")            → Error message
```

### Dependencies

- Phase 3 (all refactoring complete)

---

## Success Criteria

1. ✅ All tool descriptions reduced to 1-2 lines
2. ✅ `how_to_use` tool returns detailed docs on demand
3. ✅ Build succeeds with no errors
4. ✅ All existing functionality preserved
5. ✅ Token consumption for initial tool loading reduced by ~80%

---

## Implementation Notes

- Use `embed.FS` for efficient file loading
- Keep documentation files as plain text for easy editing
- Consider YAML for structured tool documentation in future
- The `how_to_use` tool itself should have a minimal description

---

## Related Facts

- `howto_phase_1` - Phase 1 details
- `howto_phase_2` - Phase 2 details  
- `howto_phase_3` - Phase 3 details
- `howto_phase_4` - Phase 4 details
