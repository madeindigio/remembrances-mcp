# Plan: Code File Refactoring

**Feature**: Split large source files into smaller, more maintainable modules  
**Status**: 📋 PLANNED  
**Created**: December 1, 2025  
**Priority**: Medium

---

## Problem Statement

Several source files in the project have grown too large, making them difficult to navigate, understand, and maintain. The following files exceed 450+ lines and should be split into smaller, focused modules:

| File | Current Lines | Concern |
|------|--------------|---------|
| `pkg/mcp_tools/code_search_tools.go` | 951 | Too many tools in one file |
| `internal/storage/surrealdb_code.go` | 801 | Mixed concerns (projects, files, symbols, jobs, chunks) |
| `pkg/mcp_tools/code_indexing_tools.go` | 717 | Project, file, and watcher tools mixed |
| `internal/indexer/indexer.go` | 638 | Core indexing, embeddings, progress, chunking mixed |
| `pkg/mcp_tools/code_manipulation_tools.go` | 583 | Types and handlers mixed |
| `internal/storage/surrealdb_schema.go` | 477 | Schema management and migration definitions mixed |

**Target**: No file should exceed 300-400 lines. Each file should have a single, clear responsibility.

---

## Phase Overview

| Phase | Title | Files Affected | Status |
|-------|-------|----------------|--------|
| 1 | Split code_search_tools.go | 951 → 3 files | ☐ Not Started |
| 2 | Split surrealdb_code.go | 801 → 6 files | ☐ Not Started |
| 3 | Split code_indexing_tools.go | 717 → 3 files | ☐ Not Started |
| 4 | Split indexer.go | 638 → 5 files | ☐ Not Started |
| 5 | Split code_manipulation_tools.go | 583 → 2 files | ☐ Not Started |
| 6 | Split surrealdb_schema.go | 477 → 2 files | ☐ Not Started |
| 7 | Review and Consolidation | All | ☐ Not Started |

---

## Reference Facts

Each phase is stored as a fact in remembrances:
- `refactoring_phase_1` through `refactoring_phase_7`

Use `remembrance_get_fact(user_id="remembrances-mcp", key="refactoring_phase_N")` to retrieve details.

---

## PHASE 1: Split code_search_tools.go (951 lines)

**Current File**: `pkg/mcp_tools/code_search_tools.go`

**Problem**: 6 different tools with input types, tool definitions, and handlers all in one file

### New File Structure

| New File | Lines | Contents |
|----------|-------|----------|
| `code_search_tools_types.go` | ~80 | Input type structs (CodeGetSymbolsOverviewInput, CodeFindSymbolInput, etc.) |
| `code_search_tools_handlers.go` | ~600 | Handler methods with complex logic |
| `code_search_tools.go` | ~250 | Tool definitions and helper functions |

### Files to Create
- `pkg/mcp_tools/code_search_tools_types.go`
- `pkg/mcp_tools/code_search_tools_handlers.go`

---

## PHASE 2: Split surrealdb_code.go (801 lines)

**Current File**: `internal/storage/surrealdb_code.go`

**Problem**: Projects, files, symbols, jobs, and chunks all mixed together

### New File Structure

| New File | Lines | Contents |
|----------|-------|----------|
| `surrealdb_code_types.go` | ~100 | CodeProject, CodeFile, CodeSymbol, CodeChunk structs |
| `surrealdb_code_projects.go` | ~150 | CreateCodeProject, GetCodeProject, ListCodeProjects, etc. |
| `surrealdb_code_files.go` | ~100 | SaveCodeFile, GetCodeFile, ListCodeFiles, DeleteCodeFile |
| `surrealdb_code_symbols.go` | ~250 | Symbol CRUD and search operations |
| `surrealdb_code_jobs.go` | ~100 | Indexing job operations and project stats |
| `surrealdb_code_chunks.go` | ~100 | Chunk save, delete, get, and search operations |

### Files to Create
- `internal/storage/surrealdb_code_types.go`
- `internal/storage/surrealdb_code_projects.go`
- `internal/storage/surrealdb_code_files.go`
- `internal/storage/surrealdb_code_symbols.go`
- `internal/storage/surrealdb_code_jobs.go`
- `internal/storage/surrealdb_code_chunks.go`

---

## PHASE 3: Split code_indexing_tools.go (717 lines)

**Current File**: `pkg/mcp_tools/code_indexing_tools.go`

**Problem**: Project tools, file tools, and watcher tools all mixed

### New File Structure

| New File | Lines | Contents |
|----------|-------|----------|
| `code_indexing_tools_types.go` | ~70 | All input structs for indexing tools |
| `code_indexing_tools.go` | ~350 | Core project and file operations |
| `code_watch_tools.go` | ~200 | Watcher-specific tools (activate, deactivate, status) |

### Files to Create
- `pkg/mcp_tools/code_indexing_tools_types.go`
- `pkg/mcp_tools/code_watch_tools.go`

---

## PHASE 4: Split indexer.go (638 lines)

**Current File**: `internal/indexer/indexer.go`

**Problem**: Core indexing, embedding generation, progress tracking, and chunking logic mixed

### New File Structure

| New File | Lines | Contents |
|----------|-------|----------|
| `indexer_types.go` | ~70 | IndexerConfig, IndexingProgress, DefaultIndexerConfig |
| `indexer.go` | ~300 | Core Indexer struct, IndexProject, processFiles, processFile |
| `indexer_embeddings.go` | ~120 | generateEmbeddings, prepareSymbolText |
| `indexer_progress.go` | ~80 | Progress tracking functions |
| `indexer_chunks.go` | ~100 | Large symbol chunking operations |

### Files to Create
- `internal/indexer/indexer_types.go`
- `internal/indexer/indexer_embeddings.go`
- `internal/indexer/indexer_progress.go`
- `internal/indexer/indexer_chunks.go`

---

## PHASE 5: Split code_manipulation_tools.go (583 lines)

**Current File**: `pkg/mcp_tools/code_manipulation_tools.go`

**Problem**: Types mixed with handlers (less critical than other phases)

### New File Structure

| New File | Lines | Contents |
|----------|-------|----------|
| `code_manipulation_tools_types.go` | ~60 | Input structs and symbolInfo helper |
| `code_manipulation_tools.go` | ~520 | Tool manager, definitions, and handlers |

### Files to Create
- `pkg/mcp_tools/code_manipulation_tools_types.go`

---

## PHASE 6: Split surrealdb_schema.go (477 lines)

**Current File**: `internal/storage/surrealdb_schema.go`

**Problem**: Schema management logic and embedded migration definitions mixed

### New File Structure

| New File | Lines | Contents |
|----------|-------|----------|
| `surrealdb_schema.go` | ~200 | Schema management functions (InitializeSchema, migrations runner) |
| `surrealdb_schema_embedded.go` | ~280 | applyMigrationEmbedded and all V1-V12 migration statements |

### Files to Create
- `internal/storage/surrealdb_schema_embedded.go`

---

## PHASE 7: Review and Consolidation

### Tasks

1. **Build Verification**
   ```bash
   make build  # or xc build
   ```

2. **Test Execution**
   - Run existing test suite
   - Verify no regressions

3. **Documentation Updates**
   - Update `AGENTS.md` with new file structure
   - Add file header comments explaining purpose

4. **Code Style**
   ```bash
   go fmt ./...
   go vet ./...
   ```

5. **Final Review**
   - Verify no file exceeds 400 lines
   - Ensure logical grouping is maintained
   - Check all imports are correct

---

## Expected Results

### Before Refactoring
- 6 large files averaging ~600 lines each
- Difficult navigation and maintenance
- Mixed concerns in single files

### After Refactoring
- 21 focused files averaging ~180 lines each
- Clear separation of concerns
- Easy to navigate and maintain

| Category | Before | After |
|----------|--------|-------|
| Total Large Files | 6 | 0 |
| Average File Size | ~600 lines | ~180 lines |
| Max File Size | 951 lines | ~350 lines |
| Total New Files Created | - | 15 |

---

## Implementation Notes

### Go Package Considerations
- All new files within the same directory belong to the same package
- No import changes needed for internal refactoring
- Receiver types (e.g., `*SurrealDBStorage`, `*CodeSearchToolManager`) work across files

### Naming Conventions
- Type files: `*_types.go`
- Handler files: `*_handlers.go`
- Feature-specific files: `*_feature.go` (e.g., `indexer_embeddings.go`)

### Testing Strategy
- Run full build after each phase
- Run existing tests after each phase
- No new tests needed for pure refactoring

---

## Priority Execution Order

1. **High Priority** (Most benefit, most complex):
   - Phase 1: code_search_tools.go
   - Phase 2: surrealdb_code.go

2. **Medium Priority**:
   - Phase 3: code_indexing_tools.go
   - Phase 4: indexer.go

3. **Low Priority** (Smaller impact):
   - Phase 5: code_manipulation_tools.go
   - Phase 6: surrealdb_schema.go

4. **Final**:
   - Phase 7: Consolidation

---

## Previous Plans (Completed)

### Code Project File Monitoring System (December 1, 2025)
✅ Completed - Added file monitoring and automatic reindexing for code projects.

### Dual Code Embeddings System (November 30, 2025)
✅ Completed - Added support for specialized embedding models for code indexing.
