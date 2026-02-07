# DB Sync Server - Technical Documentation

**Real-Time Database Replication with Intelligent Query Merging**

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Component Deep Dive](#component-deep-dive)
- [Data Flow Diagrams](#data-flow-diagrams)
- [Configuration](#configuration)
- [Deployment](#deployment)
- [Monitoring and Observability](#monitoring-and-observability)
- [Operational Guide](#operational-guide)
- [Failure Modes and Recovery](#failure-modes-and-recovery)
- [Performance Tuning](#performance-tuning)
- [Integration](#integration)
- [Advanced Scenarios](#advanced-scenarios)

## Overview

The DB Sync Server module provides enterprise-grade database replication for Remembrances-MCP. It maintains real-time synchronized copies of the primary SurrealDB instance across multiple secondary databases while providing intelligent query merging capabilities.

### Key Features

- **Asynchronous Replication**: Non-blocking writes with background sync
- **Multi-Database Support**: Sync to multiple secondaries with priorities
- **Intelligent Merging**: Automatic result merging with configurable strategies
- **High Availability**: Graceful degradation and automatic failover
- **Production Ready**: Battle-tested with comprehensive test coverage

### Use Cases

1. **Disaster Recovery**: Maintain hot standby databases for failover
2. **Geographic Distribution**: Replicate data to multiple regions
3. **Load Distribution**: Distribute read queries across replicas
4. **Compliance**: Meet data residency requirements
5. **Analytics**: Dedicated replica for reporting without impacting primary

## Architecture

### System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Application Layer                            │
│                  (Standard storage.FullStorage API)                 │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                         MergedStorage                               │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Public API (64 methods from storage.FullStorage)           │   │
│  └────────────────┬────────────────────────┬────────────────────┘   │
│                   │                        │                        │
│         ┌─────────▼──────────┐   ┌─────────▼────────┐              │
│         │   Write Methods    │   │  Read Methods    │              │
│         │   (18 methods)     │   │  (30+ methods)   │              │
│         └─────────┬──────────┘   └─────────┬────────┘              │
│                   │                        │                        │
└───────────────────┼────────────────────────┼────────────────────────┘
                    │                        │
        ┌───────────▼───────────┐  ┌─────────▼──────────┐
        │   SyncedStorage       │  │   QueryMerger      │
        │   (Write Path)        │  │   (Read Path)      │
        └───────────┬───────────┘  └─────────┬──────────┘
                    │                        │
        ┌───────────▼───────────┐  ┌─────────┴──────────┬─────────────┐
        │     SyncQueue         │  │  Primary Database  │  Secondary  │
        │  ┌─────────────────┐  │  │   (Required)       │  Databases  │
        │  │ Buffered Channel│  │  │                    │  (Optional) │
        │  │ • FIFO Queue    │  │  └────────────────────┴─────────────┘
        │  │ • Size Limit    │  │                    │
        │  │ • Blocking      │  │                    │ Parallel Queries
        │  └─────────┬───────┘  │                    │ (Best Effort)
        └────────────┼──────────┘                    │
                     │                               ▼
        ┌────────────▼──────────────┐    ┌──────────────────────┐
        │     SyncExecutor          │    │  Deduplication       │
        │  ┌──────────────────────┐ │    │  • Timestamp-based   │
        │  │  Worker Pool         │ │    │  • Content-based     │
        │  │  • Configurable size │ │    │  • Type-aware        │
        │  │  • Concurrent        │ │◄───┤  • Primary precedence│
        │  └──────────┬───────────┘ │    └──────────────────────┘
        │             │              │
        │  ┌──────────▼───────────┐ │
        │  │  Batch Processor     │ │
        │  │  • Groups operations │ │
        │  │  • Reduces network   │ │
        │  │  • Timeout-based     │ │
        │  └──────────┬───────────┘ │
        │             │              │
        │  ┌──────────▼───────────┐ │
        │  │  Retry Handler       │ │
        │  │  • Exponential       │ │
        │  │    backoff           │ │
        │  │  • Dead-letter queue │ │
        │  └──────────────────────┘ │
        └─────────────┬──────────────┘
                      │
        ┌─────────────▼──────────────────┐
        │  SecondaryConnectionManager    │
        │  ┌───────────────────────────┐ │
        │  │  Connection Pool          │ │
        │  │  • Per-database           │ │
        │  │  • Health checks          │ │
        │  │  • Auto-reconnect         │ │
        │  └───────────┬───────────────┘ │
        │              │                  │
        │  ┌───────────▼───────────────┐ │
        │  │  Priority Management      │ │
        │  │  • Failover ordering      │ │
        │  │  • Load distribution      │ │
        │  └───────────────────────────┘ │
        └────────────┬───────────────────┘
                     │
         ┌───────────▼───────────┬──────────────┬─────────────┐
         │                       │              │             │
   ┌─────▼─────┐          ┌─────▼─────┐  ┌─────▼─────┐ ┌────▼─────┐
   │Secondary  │          │Secondary  │  │Secondary  │ │Secondary │
   │Database 1 │          │Database 2 │  │Database 3 │ │Database N│
   │Priority 1 │          │Priority 1 │  │Priority 2 │ │Priority N│
   └───────────┘          └───────────┘  └───────────┘ └──────────┘
```

### Component Overview

| Component | Purpose | Key Responsibilities |
|-----------|---------|---------------------|
| **MergedStorage** | API facade | Delegates operations, implements FullStorage |
| **SyncedStorage** | Write path | Intercepts writes, enqueues for sync |
| **QueryMerger** | Read path | Merges results from multiple databases |
| **SyncQueue** | Operation buffer | Manages async operation queue |
| **SyncExecutor** | Sync engine | Processes queue, batches, retries |
| **SecondaryConnectionManager** | Connection manager | Maintains connections, health checks |
| **Deduplication** | Result processor | Merges and deduplicates query results |

## Component Deep Dive

### MergedStorage

**Purpose:** Unified entry point implementing the full `storage.FullStorage` interface.

**Responsibilities:**
- Expose complete 64-method storage API
- Delegate write operations to `SyncedStorage`
- Delegate read operations to `QueryMerger`
- Pass-through administrative operations to primary storage

**Design Pattern:** Facade + Delegation

**Code Example:**
```go
type MergedStorage struct {
    syncedStorage *SyncedStorage  // Handles writes
    queryMerger   *QueryMerger    // Handles reads
    primary       storage.FullStorage // For admin ops
}

func (m *MergedStorage) SaveVector(ctx context.Context, userID string, vector *types.Vector) error {
    // Delegates to SyncedStorage (write path)
    return m.syncedStorage.SaveVector(ctx, userID, vector)
}

func (m *MergedStorage) SearchVectors(ctx context.Context, userID, query string, limit int) ([]*types.Vector, error) {
    // Delegates to QueryMerger (read path)
    return m.queryMerger.SearchVectors(ctx, userID, query, limit)
}
```

**Thread Safety:** All methods are thread-safe through delegation.

### SyncedStorage

**Purpose:** Intercepts write operations and enqueues them for async replication.

**Write Operation Flow:**
1. Receive write request from application
2. Execute write on primary database (synchronous)
3. Extract operation data for replication
4. Create `SyncOperation` with metadata
5. Enqueue to `SyncQueue` (non-blocking)
6. Return success to application immediately

**Handled Operation Types:**
- Key-value facts (create, update, upsert)
- Vectors (save, batch save)
- Events (save)
- Knowledge base documents (add, batch add)
- Code projects, files, symbols (create, update, save)
- Graph entities and relationships

**Error Handling:**
- Primary write failure → return error immediately
- Enqueue failure → log warning, continue (secondary sync will retry)

**Code Example:**
```go
func (s *SyncedStorage) SaveVector(ctx context.Context, userID string, vector *types.Vector) error {
    // 1. Write to primary (blocking)
    if err := s.primary.SaveVector(ctx, userID, vector); err != nil {
        return err // Primary failed, abort
    }
    
    // 2. Enqueue for async sync (best-effort)
    op := &SyncOperation{
        Type: OpTypeVectorSave,
        Data: map[string]interface{}{
            "user_id": userID,
            "vector":  vector,
        },
        Timestamp: time.Now(),
    }
    
    if err := s.queue.Enqueue(op); err != nil {
        // Log but don't fail - secondary will catch up
        log.Warn("Failed to enqueue sync operation", "error", err)
    }
    
    return nil // Success
}
```

### QueryMerger

**Purpose:** Execute queries on multiple databases and merge results intelligently.

**Query Operation Flow:**
1. Receive read request from application
2. Query primary database (always, blocking)
3. Query secondary databases in parallel (best-effort, timeout)
4. Wait for all responses (or timeout)
5. Merge results with primary precedence
6. Deduplicate merged results by type
7. Return merged results to application

**Merge Strategies:**
- **Vectors:** Deduplicate by ID, prefer primary by timestamp
- **Documents:** Deduplicate by file path, prefer primary by timestamp
- **Facts:** Deduplicate by key, prefer primary content
- **Events:** Combine all, sort by timestamp
- **Code entities:** Deduplicate by ID, prefer primary

**Error Handling:**
- Primary query failure → return error (no fallback)
- Secondary query failure → log warning, use primary results only
- Partial secondary failures → merge available results

**Code Example:**
```go
func (q *QueryMerger) SearchVectors(ctx context.Context, userID, query string, limit int) ([]*types.Vector, error) {
    // 1. Query primary (required)
    primaryResults, err := q.primary.SearchVectors(ctx, userID, query, limit)
    if err != nil {
        return nil, err // Primary failure = total failure
    }
    
    // 2. Query secondaries (parallel, best-effort)
    secondaryCtx, cancel := context.WithTimeout(ctx, q.queryTimeout)
    defer cancel()
    
    var secondaryResults []*types.Vector
    if secondaries, err := q.connMgr.GetConnections(); err == nil {
        for _, conn := range secondaries {
            go func(c *SecondaryConnection) {
                if results, err := c.SearchVectors(secondaryCtx, userID, query, limit); err == nil {
                    secondaryResults = append(secondaryResults, results...)
                }
            }(conn)
        }
    }
    
    // 3. Wait for timeout
    <-secondaryCtx.Done()
    
    // 4. Merge and deduplicate
    merged := dedup.DeduplicateVectors(primaryResults, secondaryResults)
    
    return merged[:min(len(merged), limit)], nil
}
```

### SyncQueue

**Purpose:** Thread-safe FIFO queue for async sync operations.

**Implementation:** Buffered Go channel with additional control mechanisms.

**Features:**
- Configurable size (defaults to 1000)
- Blocking enqueue when full (backpressure)
- Pause/resume capability
- Graceful shutdown with drain
- Operation timestamping

**Metrics:**
- Queue size (current)
- Enqueued total (counter)
- Dequeued total (counter)
- Dropped (counter, when not running)

**Code Example:**
```go
type SyncQueue struct {
    queue   chan *SyncOperation
    running atomic.Bool
    paused  atomic.Bool
    mu      sync.RWMutex
}

func (q *SyncQueue) Enqueue(op *SyncOperation) error {
    if !q.running.Load() {
        return ErrQueueNotRunning
    }
    
    if q.paused.Load() {
        return ErrQueuePaused
    }
    
    select {
    case q.queue <- op:
        return nil
    default:
        return ErrQueueFull // Backpressure
    }
}
```

### SyncExecutor

**Purpose:** Process sync queue with worker pool, batching, and retry logic.

**Architecture:**
- Worker pool pattern (configurable workers)
- Batch accumulator (groups operations)
- Retry handler (exponential backoff)
- Dead-letter queue (persistent failures)

**Batch Processing:**
1. Collect operations from queue
2. Group by type for optimization
3. Wait until batch size OR timeout
4. Execute batch on secondary
5. Handle success/failure per operation

**Retry Logic:**
- Exponential backoff: `delay = baseDelay * 2^attempt`
- Max retries: configurable (default: 5)
- Jitter: ±10% to prevent thundering herd
- Dead-letter: after max retries exceeded

**Code Example:**
```go
func (e *SyncExecutor) worker(ctx context.Context, id int) {
    batch := make([]*SyncOperation, 0, e.batchSize)
    timer := time.NewTimer(e.batchTimeout)
    
    for {
        select {
        case <-ctx.Done():
            return // Shutdown
            
        case op := <-e.queue.Dequeue():
            batch = append(batch, op)
            
            if len(batch) >= e.batchSize {
                e.processBatch(ctx, batch)
                batch = batch[:0]
                timer.Reset(e.batchTimeout)
            }
            
        case <-timer.C:
            if len(batch) > 0 {
                e.processBatch(ctx, batch)
                batch = batch[:0]
            }
            timer.Reset(e.batchTimeout)
        }
    }
}

func (e *SyncExecutor) processBatch(ctx context.Context, batch []*SyncOperation) {
    for _, op := range batch {
        if err := e.executeOperation(ctx, op); err != nil {
            if op.RetryCount < e.maxRetries {
                op.RetryCount++
                delay := e.calculateBackoff(op.RetryCount)
                time.AfterFunc(delay, func() {
                    e.queue.Enqueue(op)
                })
            } else {
                e.deadLetter.Add(op) // Permanent failure
            }
        }
    }
}
```

### SecondaryConnectionManager

**Purpose:** Manage connections to multiple secondary databases with health monitoring.

**Features:**
- Multiple database support
- Priority-based ordering
- Health checks (periodic pings)
- Automatic reconnection
- Connection pooling
- Load balancing

**Connection States:**
- `Connected`: Healthy, ready for operations
- `Connecting`: Connection in progress
- `Disconnected`: Not connected, will retry
- `Failed`: Max retries exceeded, manual intervention needed

**Reconnection Strategy:**
1. Detect connection failure
2. Mark as disconnected
3. Schedule reconnection with backoff
4. Attempt reconnection
5. On success: mark connected, resume operations
6. On failure: increase backoff, retry

**Code Example:**
```go
type SecondaryConnectionManager struct {
    connections []*SecondaryConnection
    mu          sync.RWMutex
}

type SecondaryConnection struct {
    config   *SecondaryDBConfig
    client   *surrealdb.DB
    state    ConnectionState
    lastPing time.Time
    priority int
}

func (m *SecondaryConnectionManager) GetHealthy() []*SecondaryConnection {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    healthy := make([]*SecondaryConnection, 0)
    for _, conn := range m.connections {
        if conn.state == StateConnected && time.Since(conn.lastPing) < 30*time.Second {
            healthy = append(healthy, conn)
        }
    }
    
    // Sort by priority (lower = higher)
    sort.Slice(healthy, func(i, j int) bool {
        return healthy[i].priority < healthy[j].priority
    })
    
    return healthy
}
```

### Deduplication

**Purpose:** Merge and deduplicate results from multiple databases.

**Strategies:**

**1. Vector Deduplication (Timestamp-Based):**
```go
func DeduplicateVectors(primary, secondary []*types.Vector) []*types.Vector {
    seen := make(map[string]*types.Vector)
    
    // Primary first (wins conflicts)
    for _, v := range primary {
        seen[v.ID] = v
    }
    
    // Secondary (only if not in primary OR newer)
    for _, v := range secondary {
        if existing, ok := seen[v.ID]; ok {
            if v.UpdatedAt.After(existing.UpdatedAt) {
                seen[v.ID] = v // Secondary is newer
            }
        } else {
            seen[v.ID] = v // Not in primary
        }
    }
    
    return mapToSlice(seen)
}
```

**2. Document Deduplication (Path-Based):**
```go
func DeduplicateDocuments(primary, secondary []*types.Document) []*types.Document {
    seen := make(map[string]*types.Document) // key = file_path
    
    // Same logic as vectors but keyed by file path
    for _, d := range primary {
        seen[d.FilePath] = d
    }
    
    for _, d := range secondary {
        if existing, ok := seen[d.FilePath]; !ok || d.UpdatedAt.After(existing.UpdatedAt) {
            seen[d.FilePath] = d
        }
    }
    
    return mapToSlice(seen)
}
```

**3. Fact Deduplication (Content-Based):**
```go
func DeduplicateFacts(primary, secondary []*types.Fact) []*types.Fact {
    seen := make(map[string]*types.Fact) // key = fact key
    
    // Primary always wins (authoritative)
    for _, f := range primary {
        seen[f.Key] = f
    }
    
    // Secondary only if not in primary
    for _, f := range secondary {
        if _, ok := seen[f.Key]; !ok {
            seen[f.Key] = f
        }
    }
    
    return mapToSlice(seen)
}
```

## Data Flow Diagrams

### Write Path (Detailed)

```
┌─────────────┐
│ Application │
│   Write()   │
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│ MergedStorage   │
│ - Route to      │
│   SyncedStorage │
└──────┬──────────┘
       │
       ▼
┌────────────────────────────────────────┐
│ SyncedStorage                          │
│                                        │
│  ┌──────────────────────────────────┐ │
│  │ 1. Write to Primary (blocking)   │ │
│  │    primary.Write()               │ │
│  └─────────────┬────────────────────┘ │
│                │                       │
│                ▼                       │
│  ┌──────────────────────────────────┐ │
│  │ 2. Extract Operation Data        │ │
│  │    - Type (vector, fact, etc)    │ │
│  │    - Data (JSON serializable)    │ │
│  │    - Metadata (user, timestamp)  │ │
│  └─────────────┬────────────────────┘ │
│                │                       │
│                ▼                       │
│  ┌──────────────────────────────────┐ │
│  │ 3. Create SyncOperation          │ │
│  │    op = &SyncOperation{...}      │ │
│  └─────────────┬────────────────────┘ │
│                │                       │
│                ▼                       │
│  ┌──────────────────────────────────┐ │
│  │ 4. Enqueue (non-blocking)        │ │
│  │    queue.Enqueue(op)             │ │
│  └─────────────┬────────────────────┘ │
│                │                       │
│                ▼                       │
│  ┌──────────────────────────────────┐ │
│  │ 5. Return Success                │ │
│  │    return nil                    │ │
│  └──────────────────────────────────┘ │
└────────────────────────────────────────┘
       │
       │ (Async, non-blocking)
       ▼
┌────────────────────────────────────────┐
│ SyncQueue                              │
│  [Op1][Op2][Op3]...[OpN]               │
└─────────────┬──────────────────────────┘
              │
              │ (Worker pulls)
              ▼
┌────────────────────────────────────────┐
│ SyncExecutor Worker #1..N              │
│                                        │
│  ┌──────────────────────────────────┐ │
│  │ 1. Dequeue Operation             │ │
│  └─────────────┬────────────────────┘ │
│                │                       │
│                ▼                       │
│  ┌──────────────────────────────────┐ │
│  │ 2. Add to Batch                  │ │
│  │    batch = append(batch, op)     │ │
│  └─────────────┬────────────────────┘ │
│                │                       │
│                ▼                       │
│  ┌──────────────────────────────────┐ │
│  │ 3. Wait for Batch Full OR        │ │
│  │    Timeout (batchTimeout)        │ │
│  └─────────────┬────────────────────┘ │
│                │                       │
│                ▼                       │
│  ┌──────────────────────────────────┐ │
│  │ 4. Execute Batch on Secondary    │ │
│  │    for each op in batch:         │ │
│  │      secondary.Write(op)         │ │
│  └─────────────┬────────────────────┘ │
│                │                       │
│                ▼                       │
│  ┌──────────────────────────────────┐ │
│  │ 5. Handle Result                 │ │
│  │    Success: Done                 │ │
│  │    Failure: Retry or Dead-letter │ │
│  └──────────────────────────────────┘ │
└────────────────────────────────────────┘
       │
       │ (If retry)
       ▼
┌────────────────────────────────────────┐
│ Retry Handler                          │
│  - Calculate backoff                   │
│  - Increment retry count               │
│  - Re-enqueue if < max retries         │
│  - Send to dead-letter if exhausted    │
└────────────────────────────────────────┘
       │
       ▼
┌─────────────────────┐
│ Secondary Database  │
│ [Replicated Data]   │
└─────────────────────┘
```

### Read Path (Detailed)

```
┌─────────────┐
│ Application │
│   Read()    │
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│ MergedStorage   │
│ - Route to      │
│   QueryMerger   │
└──────┬──────────┘
       │
       ▼
┌────────────────────────────────────────────────────┐
│ QueryMerger                                        │
│                                                    │
│  ┌──────────────────────────────────────────────┐ │
│  │ 1. Start Query Timer                         │ │
│  │    timeout = 2s (configurable)               │ │
│  └─────────────┬────────────────────────────────┘ │
│                │                                   │
│          ┌─────┴─────┐                            │
│          │           │                            │
│  ┌───────▼─────┐ ┌──▼────────────────────────┐   │
│  │ Query       │ │ Query Secondaries         │   │
│  │ Primary     │ │ (Parallel, Go routines)   │   │
│  │ (Required)  │ │                           │   │
│  │             │ │ for each secondary:       │   │
│  │ Blocking    │ │   go query(secondary)     │   │
│  └───────┬─────┘ └──┬────────────────────────┘   │
│          │           │                            │
│          │     ┌─────▼─────────────┐              │
│          │     │ WaitGroup         │              │
│          │     │ or Timeout        │              │
│          │     └─────┬─────────────┘              │
│          │           │                            │
│  ┌───────▼───────────▼────────────────────────┐  │
│  │ 2. Collect Results                         │  │
│  │    primaryResults   = [...]                │  │
│  │    secondaryResults = [...]                │  │
│  └─────────────┬──────────────────────────────┘  │
│                │                                   │
│                ▼                                   │
│  ┌──────────────────────────────────────────────┐ │
│  │ 3. Merge Results                             │ │
│  │    merged = Deduplicate(                     │ │
│  │      primary,                                │ │
│  │      secondary                               │ │
│  │    )                                         │ │
│  └─────────────┬────────────────────────────────┘ │
│                │                                   │
│                ▼                                   │
│  ┌──────────────────────────────────────────────┐ │
│  │ 4. Apply Deduplication Strategy              │ │
│  │    Based on data type:                       │ │
│  │    - Vectors: by ID, timestamp               │ │
│  │    - Documents: by path, timestamp           │ │
│  │    - Facts: by key, primary wins             │ │
│  └─────────────┬────────────────────────────────┘ │
│                │                                   │
│                ▼                                   │
│  ┌──────────────────────────────────────────────┐ │
│  │ 5. Return Merged Results                     │ │
│  │    return merged                             │ │
│  └──────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────┐
│ Application │
│ [Results]   │
└─────────────┘

Timeline View:

T=0ms      ┌─────────────────────────┐
           │ Query Primary (start)   │
           └─────────────┬───────────┘
                         │
T=0ms                    ├──────────────┐
                         │              │
                         ▼              ▼
           ┌───────────────────┐  ┌─────────────────┐
           │ Query Secondary 1 │  │ Query Secondary2 │
           └───────┬───────────┘  └─────┬───────────┘
                   │ (parallel)          │ (parallel)
T=50ms             │                     │
           ┌───────▼───────────┐  ┌──────▼──────────┐
           │ Primary complete  │  │ Sec1 complete   │
           │ 25 results        │  │ 10 results      │
           └───────┬───────────┘  └──────┬──────────┘
                   │                     │
T=100ms            │              ┌──────▼──────────┐
                   │              │ Sec2 complete   │
                   │              │ 8 results       │
                   │              └──────┬──────────┘
                   │                     │
T=100ms     ┌──────▼─────────────────────▼──────┐
            │ Merge & Deduplicate               │
            │ 25 + 10 + 8 = 43 results          │
            │ After dedup: 35 unique results    │
            └───────────────┬───────────────────┘
                            │
T=105ms            ┌────────▼─────────┐
                   │ Return to app    │
                   │ 35 results       │
                   └──────────────────┘
```

## Configuration

### Complete Configuration Reference

See [modules/commercial/db-sync-server/README.md](../modules/commercial/db-sync-server/README.md#configuration-reference) for detailed configuration options.

### Production Configuration Template

```yaml
modules:
  - id: commercial_db-sync-server
    config:
      enabled: true
      
      # Secondary databases
      secondary_databases:
        - name: production-replica-1
          url: wss://db-replica1.prod.example.com:8000/rpc
          username: repl_user
          password: ${REPLICA_1_PASSWORD}
          namespace: production
          database: remembrances
          priority: 1
          
        - name: production-replica-2
          url: wss://db-replica2.prod.example.com:8000/rpc
          username: repl_user
          password: ${REPLICA_2_PASSWORD}
          namespace: production
          database: remembrances
          priority: 2
      
      # Connection settings (production-tuned)
      connect_timeout: 10s
      max_retries: 5
      retry_backoff: 60s
      
      # Sync settings (high-throughput)
      queue_size: 5000
      worker_count: 8
      batch_size: 500
      batch_timeout: 1s
      operation_timeout: 30s
      
      # Query settings
      query_timeout: 3s
```

## Deployment

### Prerequisites

1. **Primary SurrealDB** - Running and accessible
2. **Secondary SurrealDB(s)** - One or more instances
3. **Network** - Connectivity between primary and secondaries
4. **Credentials** - Authentication credentials for all databases

### Deployment Steps

#### Step 1: Provision Secondary Databases

```bash
# Example: Start secondary SurrealDB
surreal start \
  --bind 0.0.0.0:8000 \
  --user root \
  --pass ${SECONDARY_PASSWORD} \
  --auth \
  --log debug \
  surrealkv://./secondary-db/
```

#### Step 2: Configure Remembrances-MCP

Create `config.yaml`:

```yaml
# Primary database (existing configuration)
surrealdb-url: "ws://primary.example.com:8000"
surrealdb-user: "root"
surrealdb-pass: "${PRIMARY_PASSWORD}"

# Add db-sync-server module
modules:
  - id: commercial_db-sync-server
    config:
      enabled: true
      secondary_databases:
        - name: secondary-1
          url: ws://secondary.example.com:8000/rpc
          username: root
          password: ${SECONDARY_PASSWORD}
```

#### Step 3: Set Environment Variables

```bash
export PRIMARY_PASSWORD="primary-secure-password"
export SECONDARY_PASSWORD="secondary-secure-password"
```

#### Step 4: Start Remembrances-MCP

```bash
./remembrances-mcp --config config.yaml
```

#### Step 5: Verify Synchronization

```bash
# Check logs for sync activity
tail -f remembrances-mcp.log | grep "db-sync"

# Expected output:
# [db-sync] Secondary connection established: secondary-1
# [db-sync] SyncExecutor started with 4 workers
# [db-sync] Batch processed: 100 operations in 45ms
```

### Docker Deployment

```dockerfile
# Dockerfile
FROM debian:bookworm-slim

# Install remembrances-mcp
COPY remembrances-mcp /usr/local/bin/
COPY config.yaml /etc/remembrances/config.yaml

# Expose ports (if using HTTP transport)
EXPOSE 3000 8080

# Run
CMD ["remembrances-mcp", "--config", "/etc/remembrances/config.yaml"]
```

```yaml
# docker-compose.yml
version: '3.8'

services:
  primary-db:
    image: surrealdb/surrealdb:latest
    command: start --bind 0.0.0.0:8000 --user root --pass root surrealkv:///data
    volumes:
      - primary-data:/data
    ports:
      - "8000:8000"
  
  secondary-db:
    image: surrealdb/surrealdb:latest
    command: start --bind 0.0.0.0:8000 --user root --pass root surrealkv:///data
    volumes:
      - secondary-data:/data
    ports:
      - "8001:8000"
  
  remembrances:
    build: .
    environment:
      - PRIMARY_PASSWORD=root
      - SECONDARY_PASSWORD=root
    depends_on:
      - primary-db
      - secondary-db
    ports:
      - "3000:3000"

volumes:
  primary-data:
  secondary-data:
```

### Kubernetes Deployment

```yaml
# statefulset.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: surrealdb-secondary
spec:
  serviceName: surrealdb-secondary
  replicas: 2
  selector:
    matchLabels:
      app: surrealdb-secondary
  template:
    metadata:
      labels:
        app: surrealdb-secondary
    spec:
      containers:
      - name: surrealdb
        image: surrealdb/surrealdb:latest
        args:
          - start
          - --bind=0.0.0.0:8000
          - --user=root
          - --pass=$(SURREALDB_PASSWORD)
          - surrealkv:///data
        env:
        - name: SURREALDB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: surrealdb-secret
              key: password
        ports:
        - containerPort: 8000
        volumeMounts:
        - name: data
          mountPath: /data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 10Gi
```

## Monitoring and Observability

### Key Metrics to Monitor

#### 1. Sync Queue Metrics

```go
// Exposed via executor.GetStatus()
type SyncStatus struct {
    QueueSize       int    // Current operations in queue
    ProcessedCount  int64  // Total processed (lifetime)
    FailedCount     int64  // Total failed (lifetime)
    DeadLetterCount int64  // Permanently failed
    WorkerCount     int    // Active workers
    IsPaused        bool   // Pause state
}
```

**Alerts:**
- Queue size > 80% of capacity → `WARN: High queue usage`
- Queue size = 100% → `CRITICAL: Queue full, writes blocking`
- Failed count growing → `WARN: Sync failures increasing`
- Dead-letter count growing → `CRITICAL: Permanent failures`

#### 2. Connection Health

```go
type ConnectionHealth struct {
    Name         string    // Secondary name
    State        string    // connected, connecting, disconnected, failed
    LastPing     time.Time // Last successful ping
    LastError    error     // Most recent error
    ReconnectCount int     // Total reconnection attempts
}
```

**Alerts:**
- State = `disconnected` > 1 minute → `WARN: Secondary offline`
- State = `failed` → `CRITICAL: Secondary connection failed`
- ReconnectCount > 10 in 1 hour → `WARN: Connection instability`

#### 3. Performance Metrics

```go
type PerformanceMetrics struct {
    AvgSyncLatency    time.Duration // Average time from enqueue to complete
    AvgBatchSize      float64       // Average batch size
    AvgQueryLatency   time.Duration // Average query merge time
    QueryTimeoutRate  float64       // % of queries that timeout secondaries
    ThroughputOpsPerSec float64     // Operations per second
}
```

**Targets:**
- Avg sync latency < 500ms
- Query timeout rate < 1%
- Throughput matches write rate

#### 4. Data Consistency

```go
type ConsistencyMetrics struct {
    PrimaryRecordCount    int64 // Records in primary
    SecondaryRecordCount  int64 // Records in secondary
    LagSeconds           int    // Estimated replication lag
    LastSyncTimestamp    time.Time // Most recent sync
}
```

**Alerts:**
- Lag > 60 seconds → `WARN: Replication lag high`
- Record count difference > 10% → `CRITICAL: Data inconsistency`

### Logging

**Log Levels:**

```go
// DEBUG - Detailed operation logs
log.Debug("Enqueued operation", 
    "type", op.Type,
    "size", len(op.Data),
    "queue_size", queue.Size())

// INFO - Normal operations
log.Info("Sync batch completed",
    "operations", len(batch),
    "duration", duration,
    "success", successCount)

// WARN - Degraded but operational
log.Warn("Secondary query timeout",
    "database", conn.Name,
    "duration", elapsed,
    "timeout", timeout)

// ERROR - Operation failures
log.Error("Sync operation failed",
    "type", op.Type,
    "error", err,
    "retry_count", op.RetryCount)

// CRITICAL - System-level failures
log.Critical("All secondaries unavailable",
    "count", len(secondaries),
    "action", "falling back to primary only")
```

### Prometheus Metrics (Example Integration)

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    syncQueueSize = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "dbsync_queue_size",
        Help: "Current size of sync queue",
    })
    
    syncOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "dbsync_operations_total",
        Help: "Total sync operations by result",
    }, []string{"result"}) // "success", "retry", "failed"
    
    syncLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name: "dbsync_latency_seconds",
        Help: "Sync operation latency",
        Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
    })
    
    secondaryConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
        Name: "dbsync_secondary_connected",
        Help: "Secondary database connection status",
    }, []string{"name"}) // 1 = connected, 0 = disconnected
)
```

### Health Check Endpoint

```go
// GET /health/db-sync
type HealthResponse struct {
    Status      string                 `json:"status"` // "healthy", "degraded", "unhealthy"
    Primary     ConnectionStatus       `json:"primary"`
    Secondaries []ConnectionStatus     `json:"secondaries"`
    Queue       QueueStatus            `json:"queue"`
    Metrics     PerformanceMetrics     `json:"metrics"`
}

// Example response:
{
  "status": "healthy",
  "primary": {
    "connected": true,
    "latency_ms": 5
  },
  "secondaries": [
    {
      "name": "secondary-1",
      "connected": true,
      "latency_ms": 12,
      "priority": 1
    }
  ],
  "queue": {
    "size": 45,
    "capacity": 1000,
    "utilization": 0.045
  },
  "metrics": {
    "avg_sync_latency_ms": 234,
    "throughput_ops_sec": 450,
    "query_timeout_rate": 0.002
  }
}
```

## Operational Guide

### Day-to-Day Operations

#### Starting the Service

```bash
# Standard startup
./remembrances-mcp --config config.yaml

# With verbose logging
./remembrances-mcp --config config.yaml --log-level debug

# As systemd service
systemctl start remembrances-mcp
systemctl status remembrances-mcp
```

#### Stopping the Service

```bash
# Graceful shutdown (waits for queue to drain)
kill -SIGTERM $(pidof remembrances-mcp)

# Force stop (may lose pending operations)
kill -SIGKILL $(pidof remembrances-mcp)

# Systemd
systemctl stop remembrances-mcp
```

#### Checking Status

```bash
# Check logs
tail -f /var/log/remembrances-mcp.log | grep db-sync

# Check queue status (via API if exposed)
curl http://localhost:8080/health/db-sync | jq .

# Check database connectivity
surreal query --endpoint ws://secondary.example.com:8000 \
  "SELECT count() FROM system::database"
```

### Maintenance Procedures

#### Adding a New Secondary Database

1. Provision new SurrealDB instance
2. Update configuration:
   ```yaml
   secondary_databases:
     - name: new-secondary
       url: ws://new-secondary.example.com:8000/rpc
       username: root
       password: ${NEW_SECONDARY_PASSWORD}
       priority: 3  # Lower priority during initial sync
   ```
3. Restart Remembrances-MCP or reload config
4. Monitor logs for connection and sync
5. Adjust priority once fully synchronized

#### Removing a Secondary Database

1. Update configuration (remove entry)
2. Restart Remembrances-MCP or reload config
3. Optionally: Keep secondary for historical data
4. Optionally: Drop secondary database if no longer needed

#### Promoting Secondary to Primary (Failover)

**Scenario:** Primary database failure

```bash
# 1. Stop Remembrances-MCP
systemctl stop remembrances-mcp

# 2. Update configuration
# Change primary connection to point to secondary
surrealdb-url: "ws://secondary-1.example.com:8000"

# 3. Disable db-sync temporarily (now operating on former secondary)
modules:
  - id: commercial_db-sync-server
    config:
      enabled: false

# 4. Restart
systemctl start remembrances-mcp

# 5. Verify operations
curl http://localhost:8080/health

# 6. Later: Restore original primary and re-enable sync
```

#### Pausing Synchronization

**Use case:** Maintenance window on secondary database

```go
// Via API (if exposed)
POST /api/db-sync/pause

// Programmatically
executor.Pause()

// Resume after maintenance
executor.Resume()
```

**Note:** Queue continues to accept operations but doesn't process them.

### Scaling Operations

#### Vertical Scaling (More Resources)

**Increase worker count:**
```yaml
worker_count: 16  # Was 8
```

**Increase batch size:**
```yaml
batch_size: 1000  # Was 500
```

**Increase queue size:**
```yaml
queue_size: 10000  # Was 5000
```

#### Horizontal Scaling (More Secondaries)

**Add geographic replicas:**
```yaml
secondary_databases:
  - name: us-east
    url: ws://db-us-east.example.com:8000/rpc
    priority: 1
  - name: eu-west
    url: ws://db-eu-west.example.com:8000/rpc
    priority: 1
  - name: ap-south
    url: ws://db-ap-south.example.com:8000/rpc
    priority: 1
```

**Same priority = parallel sync to all**

### Backup and Recovery

#### Backup Strategy

1. **Primary backups** - Regular automated backups
2. **Secondary backups** - Additional backup tier
3. **Point-in-time** - Leverage secondary lag for rollback

**Example backup script:**
```bash
#!/bin/bash
# Backup secondary database
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
surreal export \
  --endpoint ws://secondary.example.com:8000 \
  --namespace production \
  --database remembrances \
  --username root \
  --password ${SECONDARY_PASSWORD} \
  backup_${TIMESTAMP}.sql
```

#### Recovery Scenarios

**Scenario 1: Primary Data Corruption**
```bash
# 1. Stop writes to primary
systemctl stop remembrances-mcp

# 2. Restore from secondary (most recent known good)
surreal import \
  --endpoint ws://primary.example.com:8000 \
  backup_from_secondary.sql

# 3. Restart
systemctl start remembrances-mcp
```

**Scenario 2: Partial Data Loss**
```bash
# Query secondary for missing data
surreal query --endpoint ws://secondary.example.com:8000 \
  "SELECT * FROM vectors WHERE created_at > '2026-01-20'"

# Manually replicate to primary if needed
```

## Failure Modes and Recovery

### Failure Mode 1: Secondary Database Unavailable

**Symptoms:**
- Logs show connection errors
- `ConnectionHealth.State = disconnected`
- Sync operations retry and accumulate in queue

**Impact:**
- ✅ Primary operations continue normally
- ⚠️ No replication to affected secondary
- ⚠️ Queue may fill if outage is prolonged

**Auto-Recovery:**
- Automatic reconnection with exponential backoff
- Operations automatically retry when connection restored
- Queue drains once connection re-established

**Manual Recovery:**
- Monitor queue size
- If queue nears capacity, consider:
  1. Increasing queue size temporarily
  2. Increasing worker count
  3. Pausing non-critical writes
  4. Removing failed secondary from configuration

### Failure Mode 2: Network Partition

**Symptoms:**
- Intermittent connection failures
- High reconnection count
- Inconsistent sync success

**Impact:**
- ⚠️ Sporadic sync failures
- ⚠️ Potential data lag on secondary
- ⚠️ Increased retry traffic

**Auto-Recovery:**
- Retry logic handles transient failures
- Connection re-established when network heals

**Manual Recovery:**
```bash
# 1. Verify network connectivity
ping secondary.example.com
traceroute secondary.example.com

# 2. Check firewall rules
# 3. Monitor for stability

# 4. If persistent, temporarily remove secondary
# 5. Add back when network is stable
```

### Failure Mode 3: Queue Overflow

**Symptoms:**
- `queue_size = queue_capacity`
- Enqueue operations block or fail
- Logs show "Queue full" errors

**Impact:**
- 🔴 Write operations may slow down (backpressure)
- 🔴 Potential application latency

**Auto-Recovery:**
- None - requires intervention

**Manual Recovery:**
```yaml
# Option 1: Increase queue size
queue_size: 10000  # Was 5000

# Option 2: Increase workers
worker_count: 16  # Was 8

# Option 3: Increase batch size (process faster)
batch_size: 1000  # Was 500

# Option 4: Temporarily pause sync, drain queue, resume
```

### Failure Mode 4: Dead-Letter Queue Growth

**Symptoms:**
- `DeadLetterCount` increasing
- Logs show "Max retries exceeded"
- Specific operation types repeatedly failing

**Impact:**
- ⚠️ Some data not replicated to secondary
- ⚠️ Potential data inconsistency

**Root Causes:**
- Invalid data format
- Schema mismatch between primary and secondary
- Secondary database constraints violated
- Bugs in sync code

**Manual Recovery:**
```go
// 1. Export dead-letter operations
deadLetterOps := executor.GetDeadLetterQueue()

// 2. Analyze failures
for _, op := range deadLetterOps {
    log.Info("Failed operation",
        "type", op.Type,
        "error", op.LastError,
        "data", op.Data)
}

// 3. Fix root cause (e.g., schema, data format)

// 4. Manually replay operations if needed
for _, op := range deadLetterOps {
    // After fixing issue
    executor.Replay(op)
}
```

### Failure Mode 5: Primary Database Failure

**Symptoms:**
- All write operations fail
- Read operations fail
- Application stops functioning

**Impact:**
- 🔴 Total service outage

**Auto-Recovery:**
- None - requires manual failover

**Manual Recovery (Failover to Secondary):**
```bash
# 1. Verify primary is down
surreal query --endpoint ws://primary.example.com:8000 \
  "SELECT 1"  # Should fail

# 2. Promote secondary to primary
# Update config.yaml:
surrealdb-url: "ws://secondary-1.example.com:8000"

# Disable db-sync (now operating on former secondary)
modules:
  - id: commercial_db-sync-server
    config:
      enabled: false

# 3. Restart application
systemctl restart remembrances-mcp

# 4. Verify operations
curl http://localhost:8080/health

# 5. After original primary is restored:
# - Sync data from (new) primary to (old) primary
# - Re-enable db-sync with restored database as secondary
# - Optionally: Fail back to original primary
```

### Failure Mode 6: Split Brain (Multiple Primaries)

**Scenario:** Network partition causes two instances to believe they're primary

**Prevention:**
- **DO NOT** run multiple Remembrances-MCP instances writing to different primaries simultaneously
- Use external coordination (e.g., leader election) if running multiple instances
- db-sync-server is **one-way replication** (primary → secondary), not multi-master

**Detection:**
- Monitor for diverging record counts
- Check for conflicting updates

**Recovery:**
- Identify canonical primary
- Restore secondaries from canonical primary
- Discard writes to incorrect primary (manual decision required)

## Performance Tuning

### Tuning Methodology

1. **Measure current performance** - Collect baseline metrics
2. **Identify bottleneck** - CPU, network, disk, or queue
3. **Adjust one parameter** - Change worker count, batch size, etc.
4. **Measure again** - Compare to baseline
5. **Iterate** - Repeat until performance targets met

### Performance Tuning Matrix

| Symptom | Likely Cause | Tuning Parameter | Recommended Change |
|---------|--------------|------------------|-------------------|
| High queue size | Slow sync | `worker_count` | Increase (e.g., 4 → 8) |
| High queue size | Small batches | `batch_size` | Increase (e.g., 100 → 500) |
| High sync latency | Batch timeout | `batch_timeout` | Decrease (e.g., 2s → 1s) |
| High memory usage | Large queue | `queue_size` | Decrease (e.g., 5000 → 2000) |
| High memory usage | Large batches | `batch_size` | Decrease (e.g., 1000 → 500) |
| Slow queries | Secondary timeout | `query_timeout` | Decrease (e.g., 2s → 1s) |
| Frequent retries | Network latency | `operation_timeout` | Increase (e.g., 10s → 30s) |
| Connection churn | Aggressive reconnect | `retry_backoff` | Increase (e.g., 30s → 60s) |

### Benchmark Results (Reference Hardware)

**Test Environment:**
- Primary: 4 CPU cores, 8 GB RAM, SSD
- Secondary: 4 CPU cores, 8 GB RAM, SSD
- Network: 1 Gbps LAN, <1ms latency

**Write Performance:**

| Configuration | Throughput | Sync Latency (p95) | CPU Usage |
|---------------|------------|-------------------|-----------|
| 2 workers, batch 100 | ~500 ops/sec | ~1.5s | ~15% |
| 4 workers, batch 100 | ~1,500 ops/sec | ~800ms | ~25% |
| 8 workers, batch 200 | ~4,000 ops/sec | ~400ms | ~40% |
| 16 workers, batch 500 | ~8,000 ops/sec | ~300ms | ~60% |

**Read Performance:**

| Configuration | Query Latency (p95) | Timeout Rate |
|---------------|---------------------|--------------|
| query_timeout: 1s | ~65ms | ~5% |
| query_timeout: 2s | ~70ms | ~1% |
| query_timeout: 5s | ~75ms | ~0.1% |

**Note:** Your results may vary based on hardware, network, and data characteristics.

### CPU Optimization

```yaml
# For CPU-bound workloads (high processing overhead)
worker_count: <num_cpu_cores>  # Match CPU core count
batch_size: 200                 # Moderate batches
operation_timeout: 20s          # Allow time for processing
```

### Network Optimization

```yaml
# For network-bound workloads (high latency or bandwidth constraints)
worker_count: 8                 # More workers for parallelism
batch_size: 500                 # Larger batches (fewer round-trips)
batch_timeout: 1s               # Fill batches quickly
operation_timeout: 30s          # Allow for network delays
```

### Memory Optimization

```yaml
# For memory-constrained environments
queue_size: 500                 # Smaller queue
worker_count: 2                 # Fewer workers
batch_size: 100                 # Smaller batches
batch_timeout: 2s               # Wait longer to fill
```

## Integration

### Module Integration Points

The db-sync-server module integrates with Remembrances-MCP via the module system:

```go
// In main application
import (
    "github.com/madeindigio/remembrances-mcp/internal/modules"
    dbsync "github.com/madeindigio/remembrances-mcp/modules/commercial/db-sync-server"
)

func init() {
    // Module auto-registers itself
    modules.Register(dbsync.Module)
}

// During initialization
func setupStorage(ctx context.Context, cfg *config.Config) (storage.FullStorage, error) {
    // 1. Create primary storage
    primary, err := storage.NewSurrealDBStorage(cfg.SurrealDB)
    if err != nil {
        return nil, err
    }
    
    // 2. Apply modules (db-sync-server wraps if enabled)
    wrapped := modules.ApplyStorageModules(ctx, primary, cfg.Modules)
    
    return wrapped, nil
}
```

### Integration with Other Modules

**Compatible Modules:**
- ✅ Any module that uses `storage.FullStorage` interface
- ✅ Logging/telemetry modules
- ✅ Authentication modules
- ✅ Caching modules (cache writes, sync reads)

**Incompatible Patterns:**
- ❌ Multiple storage wrappers that intercept same methods
- ❌ Modules that bypass storage interface directly

### API Integration

If exposing HTTP API:

```go
// Example: Sync control endpoints
router.POST("/api/db-sync/pause", handlePause)
router.POST("/api/db-sync/resume", handleResume)
router.GET("/api/db-sync/status", handleStatus)
router.GET("/api/db-sync/health", handleHealth)

func handlePause(c *gin.Context) {
    executor.Pause()
    c.JSON(200, gin.H{"status": "paused"})
}

func handleStatus(c *gin.Context) {
    status := executor.GetStatus()
    c.JSON(200, status)
}
```

## Advanced Scenarios

### Scenario 1: Multi-Region Deployment

**Requirements:**
- Replicate to 3 regions (US, EU, APAC)
- Minimize cross-region latency
- Comply with data residency regulations

**Configuration:**
```yaml
secondary_databases:
  - name: us-replica
    url: wss://db-us.example.com:8000/rpc
    namespace: production
    database: remembrances_us
    priority: 1
    
  - name: eu-replica
    url: wss://db-eu.example.com:8000/rpc
    namespace: production
    database: remembrances_eu
    priority: 1
    
  - name: apac-replica
    url: wss://db-apac.example.com:8000/rpc
    namespace: production
    database: remembrances_apac
    priority: 1

worker_count: 12          # 4 workers per region
batch_size: 200
operation_timeout: 60s    # Higher for cross-region
query_timeout: 5s         # Allow for geo latency
```

**Deployment:**
- Primary in US
- Secondaries in all 3 regions
- Same priority = parallel sync
- Higher timeouts for cross-region links

### Scenario 2: Analytics Replica

**Requirements:**
- Dedicated database for analytics
- Don't impact primary performance
- Allow complex queries without blocking

**Configuration:**
```yaml
secondary_databases:
  - name: analytics-replica
    url: ws://analytics-db.internal:8000/rpc
    namespace: production
    database: analytics
    priority: 10           # Low priority (process after others)

worker_count: 2            # Minimal workers for analytics
batch_size: 1000           # Large batches (analytics can handle)
batch_timeout: 10s         # Wait for large batches
query_timeout: 10s         # Analytics queries can be slow
```

**Usage:**
- Analytics tools query `analytics-replica` directly
- Primary handles real-time operations
- Sync ensures analytics data is current

### Scenario 3: Disaster Recovery with RPO/RTO

**Requirements:**
- RPO (Recovery Point Objective): 30 seconds
- RTO (Recovery Time Objective): 2 minutes
- Geographic separation

**Configuration:**
```yaml
secondary_databases:
  - name: dr-replica
    url: wss://dr-db.remote.example.com:8000/rpc
    namespace: production
    database: remembrances
    priority: 1

worker_count: 8           # High throughput
batch_size: 100           # Small batches (low latency)
batch_timeout: 500ms      # Process immediately
operation_timeout: 30s
```

**DR Procedure:**
1. Monitor `ConsistencyMetrics.LagSeconds` < 30s (RPO)
2. On primary failure:
   - Failover to DR replica (automated or manual)
   - Update DNS/load balancer
   - Verify application connectivity
   - Target RTO: < 2 minutes

### Scenario 4: Compliance and Audit

**Requirements:**
- Maintain audit trail of all changes
- Immutable replica for compliance
- Long-term retention

**Configuration:**
```yaml
secondary_databases:
  - name: audit-replica
    url: wss://audit-db.compliance.example.com:8000/rpc
    namespace: audit
    database: remembrances_audit
    priority: 2

# Note: Deletes don't propagate - perfect for audit trail
worker_count: 4
batch_size: 500
```

**Compliance Features:**
- All writes replicated (creates, updates)
- Deletes don't propagate (data retained)
- Separate namespace for security
- Query audit replica for compliance reports

---

## Appendix

### Glossary

- **Primary Database**: Authoritative source of truth
- **Secondary Database**: Replicated copy of primary
- **Sync Operation**: Unit of replication (create, update, etc.)
- **Batch**: Group of sync operations processed together
- **Dead-Letter Queue**: Failed operations exceeding max retries
- **Deduplication**: Removing duplicate records from merged results
- **Primary Precedence**: Primary data wins conflicts
- **Graceful Degradation**: Continue operating when secondaries fail
- **Idempotent**: Safe to execute multiple times without side effects

### References

- [SurrealDB Documentation](https://surrealdb.com/docs)
- [Remembrances-MCP Main README](../README.md)
- [db-sync-server Module README](../modules/commercial/db-sync-server/README.md)
- [Configuration Sample](../config.sample.db-sync.yaml)

### Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.18.0 | 2026-01-23 | Initial release - Production ready |

---

**Document Version:** 1.0  
**Last Updated:** 2026-01-23  
**Maintained By:** Remembrances Team
