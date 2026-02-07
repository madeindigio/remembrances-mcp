# db-sync-server Commercial Module - Development Plan

## Executive Summary

The **db-sync-server** module is a commercial module for Remembrances-MCP that enables:

1. **Secondary Database Replication**: Configure a secondary SurrealDB instance that receives replicated data from the primary database in real-time
2. **Merged Queries**: All read operations query both databases and return merged/deduplicated results
3. **Primary Precedence**: When data exists in both databases, primary data always takes precedence
4. **One-Way Sync**: Data flows only from primary → secondary; deletes are NOT propagated
5. **Resilient Sync**: If secondary is unreachable, sync pauses and resumes when connection is restored

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        MergedStorage (FullStorage)                       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   ┌──────────────────────┐           ┌──────────────────────┐           │
│   │   SyncedStorage      │           │   QueryMerger        │           │
│   │   (Write Path)       │           │   (Read Path)        │           │
│   └──────────┬───────────┘           └──────────┬───────────┘           │
│              │                                   │                       │
│              │  ┌─────────────────┐             │                       │
│              └──► SyncQueue       │             │                       │
│                 │ (Background)    │             │                       │
│                 └────────┬────────┘             │                       │
│                          │                      │                       │
│                          ▼                      ▼                       │
│   ┌──────────────────────┐           ┌──────────────────────┐           │
│   │  SyncExecutor        │           │   Secondary          │           │
│   │  (Writes to 2nd)     ├──────────►│   Connection Mgr     │           │
│   └──────────────────────┘           └──────────────────────┘           │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
                    │                              │
                    ▼                              ▼
         ┌─────────────────┐            ┌─────────────────┐
         │ Primary SurrealDB │            │ Secondary SurrealDB │
         │ (embedded/remote) │            │ (remote only)        │
         └─────────────────┘            └─────────────────┘
```

## Key Design Decisions

### 1. Storage Wrapper Pattern
The module implements `StorageWrapperProvider` interface to wrap the primary storage without modifying core code.

### 2. Async Sync with Queue
Writes complete immediately on primary; sync to secondary happens asynchronously via a buffered queue to avoid blocking the main request path.

### 3. UPSERT Strategy
All syncs use UPSERT (or DELETE+CREATE pattern) to ensure idempotency on retry.

### 4. Graceful Degradation
- If secondary is down during write: queue operation, resume sync when reconnected
- If secondary is down during read: return primary-only results (log warning)

### 5. Deduplication Strategy
Results are deduplicated by unique key per table type:
- `kv_memories`: (user_id, key)
- `vector_memories`: id
- `events`: id
- `entities`: id
- `relationships`: id
- `knowledge_base`: file_path
- `code_*`: project_id + primary key

## Development Phases

| Phase | Name | Description | Fact Reference |
|-------|------|-------------|----------------|
| 1 | Module Scaffold | Base module structure & config | `db-sync-server/phase-1-module-scaffold` |
| 2 | Secondary Connection | Connection manager with auto-reconnect | `db-sync-server/phase-2-secondary-connection` |
| 3 | Sync Queue | Background queue system with pause/resume | `db-sync-server/phase-3-sync-queue` |
| 4 | Storage Wrapper | Intercept writes for sync, skip deletes | `db-sync-server/phase-4-storage-wrapper` |
| 5 | Sync Executor | Execute syncs to secondary DB | `db-sync-server/phase-5-sync-executor` |
| 6 | Query Merger | Merge results from both sources | `db-sync-server/phase-6-query-merger` |
| 7 | Merged Storage | Complete FullStorage implementation | `db-sync-server/phase-7-merged-storage` |
| 8 | Module Integration | Integration with ModuleManager | `db-sync-server/phase-8-module-integration` |
| 9 | Testing | Unit, integration, E2E tests | `db-sync-server/phase-9-testing` |
| 10 | Documentation | Docs, examples, CHANGELOG | `db-sync-server/phase-10-documentation` |

## File Structure

```
modules/commercial/db-sync-server/
├── README.md              # Module documentation
├── db_sync.go             # Main module, registration, lifecycle
├── config.go              # Configuration types
├── connection.go          # SecondaryConnectionManager
├── sync_queue.go          # SyncQueue implementation
├── sync_types.go          # SyncOperation types
├── sync_executor.go       # SyncExecutor
├── synced_storage.go      # SyncedStorage (write wrapper)
├── query_merger.go        # QueryMerger (read merger)
├── dedup.go               # Deduplication utilities
├── merged_storage.go      # MergedStorage (final wrapper)
├── db_sync_test.go        # Unit tests
└── integration_test.go    # Integration tests
```

## Configuration Schema

```yaml
modules:
  commercial_db-sync-server:
    enabled: true
    config:
      # Secondary database connection
      secondary_url: "ws://secondary-db:8000/rpc"
      secondary_user: "root"
      secondary_pass: "secretpassword"
      secondary_namespace: "remembrances"
      secondary_database: "production"
      
      # Sync settings
      sync_enabled: true
      sync_retry_interval: "5s"    # How often to retry failed syncs
      sync_batch_size: 100         # Max operations per batch
      sync_max_retries: 3          # Max retries before dead-letter
      sync_workers: 4              # Number of sync worker goroutines
      
      # Health check settings
      health_check_interval: "30s" # Secondary health check interval
      reconnect_backoff: "10s"     # Backoff between reconnect attempts
```

## Key Interfaces

### New Interface: StorageWrapperProvider

```go
// StorageWrapperProvider allows a module to wrap the primary storage
type StorageWrapperProvider interface {
    Module
    WrapStorage(primary storage.FullStorage) storage.FullStorage
}
```

### SyncOperation Structure

```go
type SyncOperation struct {
    ID            string                 `json:"id"`
    Table         string                 `json:"table"`
    OperationType OperationType          `json:"operation_type"` // create, update, upsert
    RecordID      string                 `json:"record_id"`
    Data          map[string]interface{} `json:"data"`
    Timestamp     time.Time              `json:"timestamp"`
    RetryCount    int                    `json:"retry_count"`
    LastError     string                 `json:"last_error,omitempty"`
}

type OperationType string
const (
    OpCreate OperationType = "create"
    OpUpdate OperationType = "update"
    OpUpsert OperationType = "upsert"
)
```

## Sync Flow

1. **Write Request** arrives at MergedStorage
2. MergedStorage delegates to SyncedStorage
3. SyncedStorage writes to **Primary DB**
4. On success, SyncedStorage creates SyncOperation and enqueues
5. SyncQueue worker picks up operation (if secondary connected)
6. SyncExecutor writes to **Secondary DB** using UPSERT
7. On failure: retry with backoff, eventually dead-letter

## Query Flow

1. **Read Request** arrives at MergedStorage
2. MergedStorage uses QueryMerger
3. QueryMerger queries **Primary DB** (required)
4. QueryMerger queries **Secondary DB** (optional, may fail)
5. Results are **merged** with deduplication
6. **Primary data takes precedence** on conflicts
7. Merged results returned to caller

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Secondary down causes request failures | Graceful degradation: primary-only reads, queued writes |
| Sync lag causes stale data | Primary precedence ensures fresh data always shown |
| Queue overflow on prolonged outage | Bounded queue with overflow to disk/dead-letter |
| Data inconsistency | UPSERT idempotency, no deletes to secondary |
| Performance impact on reads | Parallel queries, timeouts on secondary |

## Estimated Effort

| Phase | Estimated Hours |
|-------|----------------|
| Phase 1 | 4h |
| Phase 2 | 6h |
| Phase 3 | 8h |
| Phase 4 | 6h |
| Phase 5 | 8h |
| Phase 6 | 10h |
| Phase 7 | 8h |
| Phase 8 | 4h |
| Phase 9 | 12h |
| Phase 10 | 6h |
| **Total** | **~72h** |

## How to Access Phase Details

Each phase has detailed implementation notes saved as facts. To retrieve:

```
get_fact(user_id="remembrances-mcp", key="db-sync-server/phase-N-name")
```

Where N is 1-10 and name is the phase slug (e.g., `phase-1-module-scaffold`).

## Related Existing Code

- **Module system**: `pkg/modules/` - Module interfaces and manager
- **Storage interface**: `internal/storage/storage.go` - FullStorage interface
- **SurrealDB implementation**: `internal/storage/surrealdb*.go` - Reference for sync operations
- **WebUI module example**: `modules/commercial/webui/` - Commercial module pattern
- **Config system**: `internal/config/config.go` - Config loading pattern

---

*Plan created: January 2026*
*Author: Development Team*
*Status: Planning Complete - Ready for Implementation*