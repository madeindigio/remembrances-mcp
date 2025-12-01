# Plan: Code Project File Monitoring System

**Feature**: Automatic file monitoring and reindexing for code projects  
**Status**: 🚧 IN PLANNING  
**Created**: December 1, 2025  
**Branch**: `feature/code-project-monitoring`

---

## Problem Statement

Currently, the code indexing feature allows starting indexing or reindexing a project via explicit tool calls. However, there's no automatic detection of file changes within an indexed code project. This means:

- When source files are modified, the index becomes stale
- Users must manually trigger reindexing after file changes
- New files added to a project are not automatically indexed
- Deleted files may still appear in search results

The system should support:
- Activating file monitoring for a specific code project
- Detecting file modifications, additions, and deletions
- Automatically triggering reindexing for changed files
- Ensuring only ONE project can be monitored at a time (resource constraint)
- Similar implementation pattern to Knowledge Base watcher (`internal/kb/kb.go`)

---

## Phase Overview

| Phase | Title | Description | Status |
|-------|-------|-------------|--------|
| 1 | Model Extension | Add WatcherEnabled field to CodeProject | 🔲 Not Started |
| 2 | CodeWatcher Implementation | Create file watcher based on kb.Watcher pattern | 🔲 Not Started |
| 3 | Outdated File Detection | Implement hash-based file change detection | 🔲 Not Started |
| 4 | Single Watcher Management | WatcherManager for exclusive project monitoring | 🔲 Not Started |
| 5 | Tool: code_activate_project_watch | Activate monitoring for a project | 🔲 Not Started |
| 6 | Tool: code_deactivate_project_watch | Deactivate monitoring | 🔲 Not Started |
| 7 | Tool: code_get_watch_status | Query monitoring status | 🔲 Not Started |
| 8 | main.go Integration | Startup/shutdown wiring | 🔲 Not Started |
| 9 | Documentation & Testing | Docs and Python tests | 🔲 Not Started |

---

## Reference Facts

Each phase is stored as a fact in remembrances:
- `code_monitoring_phase_1` through `code_monitoring_phase_9`

Use `remembrance_get_fact(user_id="remembrances-mcp", key="code_monitoring_phase_N")` to retrieve details.

---

## PHASE 1: Model Extension

**Objective**: Extend CodeProject model to track monitoring state

### Tasks

1. Add `WatcherEnabled` field to `CodeProject` struct
2. Add `UpdateProjectWatcher(ctx, projectID, enabled bool)` method to storage interface
3. Implement method in `SurrealDBStorage`
4. Ensure schema migration if needed

### Files to Modify

- `internal/storage/storage.go` - Add interface method
- `internal/storage/surrealdb_code.go` - Add field and implement method
- `internal/storage/surrealdb_schema.go` - Add field definition if needed

### CodeProject Struct Extension

```go
type CodeProject struct {
    // ... existing fields ...
    WatcherEnabled bool `json:"watcher_enabled"` // NEW
}
```

---

## PHASE 2: CodeWatcher Implementation

**Objective**: Create file watcher for code projects based on kb.Watcher pattern

### Reference: internal/kb/kb.go

The Knowledge Base watcher uses:
- `fsnotify.NewWatcher()` for filesystem events
- Debounce mechanism to batch rapid changes
- Initial scan to process existing files
- Event loop for Create/Write/Remove/Rename events

### Tasks

1. Create `internal/indexer/code_watcher.go`
2. Implement `CodeWatcher` struct
3. Implement `StartCodeWatcher(ctx, projectID, storage, indexer)` function
4. Implement event loop with debounce
5. Integrate with `Indexer.ReindexFile` for processing changes

### CodeWatcher Struct

```go
type CodeWatcher struct {
    projectID    string
    rootPath     string
    indexer      *Indexer
    storage      storage.FullStorage
    watcher      *fsnotify.Watcher
    cancel       context.CancelFunc
    once         sync.Once
}
```

### Key Methods

- `StartCodeWatcher(ctx, project *storage.CodeProject, indexer *Indexer, storage storage.FullStorage) (*CodeWatcher, error)`
- `(*CodeWatcher).Stop()`
- `(*CodeWatcher).initialScan(ctx)`
- `(*CodeWatcher).run(ctx)`
- `(*CodeWatcher).processFile(ctx, filePath string)`
- `(*CodeWatcher).isCodeFile(filePath string) bool`

---

## PHASE 3: Outdated File Detection

**Objective**: Detect files that need reindexing at watcher activation

### Detection Criteria

1. **Modified files**: File hash differs from stored `CodeFile.FileHash`
2. **New files**: Code files in project that don't exist in `code_files` table
3. **Deleted files**: Entries in `code_files` with no corresponding file on disk

### Tasks

1. Add `(*CodeWatcher).scanOutdatedFiles(ctx) ([]OutdatedFile, error)` method
2. Implement hash comparison logic using `FileScanner.calculateHash`
3. Queue outdated files for reindexing during `initialScan`
4. Handle deleted files by removing from index

### OutdatedFile Struct

```go
type OutdatedFile struct {
    FilePath string
    Reason   string // "modified", "new", "deleted"
}
```

---

## PHASE 4: Single Active Watcher Management

**Objective**: Ensure only ONE code project can have active monitoring

### Rationale

- File watchers consume system resources (file handles, memory)
- Multiple project watchers could overwhelm the system
- Resource constraint is intentional to maintain stability

### Tasks

1. Create `internal/indexer/watcher_manager.go`
2. Implement `WatcherManager` struct with singleton pattern
3. Track currently active watcher
4. Handle graceful deactivation before new activation

### WatcherManager Struct

```go
type WatcherManager struct {
    mu            sync.RWMutex
    activeWatcher *CodeWatcher
    activeProject string
    indexer       *Indexer
    storage       storage.FullStorage
}
```

### Key Methods

- `NewWatcherManager(indexer, storage) *WatcherManager`
- `(*WatcherManager).ActivateProject(ctx, projectID string) error`
- `(*WatcherManager).DeactivateProject(ctx, projectID string) error`
- `(*WatcherManager).DeactivateCurrent(ctx) error`
- `(*WatcherManager).GetActiveProject() string`
- `(*WatcherManager).IsProjectActive(projectID string) bool`
- `(*WatcherManager).Stop() error`

---

## PHASE 5: MCP Tool - code_activate_project_watch

**Objective**: Create tool to activate monitoring for a code project

### Tool Definition

```yaml
name: code_activate_project_watch
description: Activate file monitoring for a code project. Automatically deactivates any previously watched project.
input:
  project_id: string (required) - The project ID to start monitoring
output:
  success: boolean
  message: string
  previous_project: string (if another project was being watched)
  outdated_files_found: int (files queued for reindexing)
```

### Handler Logic

1. Validate project exists and is indexed
2. Check current active project
3. If different project active, deactivate it first
4. Activate new project monitoring
5. Run initial scan for outdated files
6. Update `CodeProject.WatcherEnabled = true`
7. Return success with previous project info

### Files to Modify

- `pkg/mcp_tools/code_indexing_tools.go` - Add tool and handler
- `pkg/mcp_tools/types.go` - Add input struct

---

## PHASE 6: MCP Tool - code_deactivate_project_watch

**Objective**: Create tool to explicitly stop monitoring

### Tool Definition

```yaml
name: code_deactivate_project_watch
description: Stop file monitoring for a code project
input:
  project_id: string (optional) - If omitted, deactivates current watched project
output:
  success: boolean
  message: string
  deactivated_project: string
```

### Handler Logic

1. If project_id provided, verify it's the active project
2. Stop the watcher
3. Update `CodeProject.WatcherEnabled = false`
4. Return confirmation

---

## PHASE 7: MCP Tool - code_get_watch_status

**Objective**: Query current monitoring status

### Tool Definition

```yaml
name: code_get_watch_status
description: Get the current file monitoring status
input:
  project_id: string (optional) - Query specific project status
output:
  active_project: string (currently monitored project ID, or null)
  project_status: object (if project_id provided)
    - project_id: string
    - watcher_enabled: boolean
    - is_currently_active: boolean
```

### Additional Enhancement

Update `code_list_projects` output to include `watcher_enabled` field for each project.

---

## PHASE 8: Integration with main.go

**Objective**: Wire WatcherManager into application lifecycle

### Tasks

1. Create `WatcherManager` in `main.go`
2. Pass to `CodeToolManager`
3. On startup: if any project has `WatcherEnabled=true`, auto-activate
4. On shutdown: gracefully stop any active watcher
5. Add config flag `--disable-code-watch` to prevent auto-activation

### Startup Flow

```go
// In main.go
watcherManager := indexer.NewWatcherManager(idx, storage)

// Find project with watcher enabled
projects, _ := storage.ListCodeProjects(ctx)
for _, p := range projects {
    if p.WatcherEnabled {
        watcherManager.ActivateProject(ctx, p.ProjectID)
        break // Only one can be active
    }
}
```

### Shutdown Flow

```go
// In main.go shutdown handler
watcherManager.Stop()
```

---

## PHASE 9: Documentation & Testing

**Objective**: Complete documentation and test coverage

### Documentation Tasks

1. Update `docs/CODE_INDEXING.md` with monitoring section
2. Add new tool documentation to help system
3. Update `README.md` with monitoring feature overview

### Test File: tests/test_code_monitoring.py

```python
# Test cases
- test_activate_project_watch()
- test_deactivate_project_watch()
- test_single_watcher_constraint()
- test_outdated_file_detection()
- test_watcher_auto_restart()
- test_file_change_triggers_reindex()
```

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        main.go                                   │
├─────────────────────────────────────────────────────────────────┤
│  WatcherManager ──────────────────────────────────────────────┐ │
│      │                                                         │ │
│      ▼                                                         │ │
│  ┌─────────────────────┐                                       │ │
│  │   CodeWatcher       │◄── fsnotify events                    │ │
│  │   (one per project) │                                       │ │
│  └──────────┬──────────┘                                       │ │
│             │                                                   │ │
│             ▼                                                   │ │
│  ┌─────────────────────┐    ┌─────────────────────┐            │ │
│  │      Indexer        │    │  CodeToolManager    │            │ │
│  │  .ReindexFile()     │    │  (MCP tools)        │            │ │
│  └─────────────────────┘    └─────────────────────┘            │ │
└─────────────────────────────────────────────────────────────────┘

MCP Tools:
┌──────────────────────────────────────┐
│ code_activate_project_watch          │──┐
├──────────────────────────────────────┤  │
│ code_deactivate_project_watch        │──┼──► WatcherManager
├──────────────────────────────────────┤  │
│ code_get_watch_status                │──┘
└──────────────────────────────────────┘
```

---

## File Change Flow

```
File Modified on Disk
        │
        ▼
   fsnotify event
        │
        ▼
   CodeWatcher.run()
        │
        ▼
   debounce (500ms)
        │
        ▼
   isCodeFile() check
        │
        ▼
   Indexer.ReindexFile()
        │
        ▼
   Storage updated
```

---

## Success Criteria

1. ☐ `code_activate_project_watch` tool functional
2. ☐ `code_deactivate_project_watch` tool functional
3. ☐ `code_get_watch_status` tool functional
4. ☐ Only ONE project can be monitored at a time
5. ☐ File changes trigger automatic reindexing
6. ☐ Outdated files detected and queued on activation
7. ☐ Graceful shutdown of watcher
8. ☐ Auto-restart of watcher on server restart (if enabled)
9. ☐ Documentation updated
10. ☐ Tests passing

---

## Related Facts (stored in remembrances)

- `code_monitoring_phase_1` - Model Extension details
- `code_monitoring_phase_2` - CodeWatcher Implementation details
- `code_monitoring_phase_3` - Outdated File Detection details
- `code_monitoring_phase_4` - Single Watcher Management details
- `code_monitoring_phase_5` - code_activate_project_watch tool details
- `code_monitoring_phase_6` - code_deactivate_project_watch tool details
- `code_monitoring_phase_7` - code_get_watch_status tool details
- `code_monitoring_phase_8` - main.go Integration details
- `code_monitoring_phase_9` - Documentation & Testing details

---

## Previous Plan (Completed)

The previous plan "Dual Code Embeddings System" was completed on November 30, 2025. That feature added support for specialized embedding models for code indexing (CodeRankEmbed, Jina-code-embeddings, etc.).
