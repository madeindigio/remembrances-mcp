# Plan: Events Storage System

**Feature**: Event logging and retrieval system with temporal queries and hybrid search  
**Status**: Not Started  
**Created**: November 30, 2025  
**Branch**: `feature/events-storage`

---

## Problem Statement

AI agents need a way to store and retrieve temporal events such as:
- Conversation logs and history
- Activity logs and audit trails
- Historical events and milestones
- System events and notifications

The system should support:
- Storing events with timestamps and semantic subjects
- Querying by project, subject, and time ranges
- Hybrid search combining text and vector similarity

---

## Phase Overview

| Phase | Title | Description | Status |
|-------|-------|-------------|--------|
| 1 | Storage Schema | Define event table schema in SurrealDB | Not Started |
| 2 | Storage Implementation | Implement event storage operations | Not Started |
| 3 | MCP Tools | Create save_event and search_events tools | Not Started |
| 4 | Documentation | Add how_to_use docs and update README | Not Started |
| 5 | Testing | Comprehensive testing of the feature | Not Started |

---

## Data Model

### Event Record

```go
type Event struct {
    ID          string                 `json:"id"`           // SurrealDB record ID
    UserID      string                 `json:"user_id"`      // Project/user identifier
    Subject     string                 `json:"subject"`      // Semantic subject identifier
    Content     string                 `json:"content"`      // Event content/message
    Embedding   []float32              `json:"embedding"`    // Vector embedding for semantic search
    Metadata    map[string]interface{} `json:"metadata"`     // Additional metadata
    CreatedAt   time.Time              `json:"created_at"`   // Event timestamp
}
```

### Subject Examples
- `conversation:session_123` - Conversation events
- `log:build` - Build logs
- `audit:user_action` - Audit trail events
- `milestone:release` - Project milestones
- `error:runtime` - Error events

---

## PHASE 1: Storage Schema

**Objective**: Define the event table schema in SurrealDB with appropriate indexes

### Tasks

1. Add `events` table definition in `surrealdb_schema.go`
2. Create fields:
   - `user_id` (string, indexed)
   - `subject` (string, indexed)
   - `content` (string, full-text indexed)
   - `embedding` (vector, MTREE indexed with dimension 768)
   - `metadata` (object)
   - `created_at` (datetime, indexed)
3. Add compound index on (user_id, subject, created_at)
4. Add schema migration for v8

### Files to Modify

- `internal/storage/surrealdb_schema.go` - Add events table definition
- `internal/storage/migrations/` - Add v8 migration

### Schema Definition

```surql
DEFINE TABLE events SCHEMAFULL;
DEFINE FIELD user_id ON events TYPE string;
DEFINE FIELD subject ON events TYPE string;
DEFINE FIELD content ON events TYPE string;
DEFINE FIELD embedding ON events TYPE array<float>;
DEFINE FIELD metadata ON events TYPE option<object>;
DEFINE FIELD created_at ON events TYPE datetime DEFAULT time::now();

DEFINE INDEX idx_events_user ON events FIELDS user_id;
DEFINE INDEX idx_events_subject ON events FIELDS subject;
DEFINE INDEX idx_events_created ON events FIELDS created_at;
DEFINE INDEX idx_events_user_subject ON events FIELDS user_id, subject;
DEFINE INDEX idx_events_embedding ON events FIELDS embedding MTREE DIMENSION 768;
DEFINE ANALYZER events_analyzer TOKENIZERS blank, class FILTERS lowercase, snowball(english);
DEFINE INDEX idx_events_content ON events FIELDS content SEARCH ANALYZER events_analyzer BM25;
```

---

## PHASE 2: Storage Implementation

**Objective**: Implement event storage operations in the storage layer

### Tasks

1. Create `internal/storage/surrealdb_events.go`
2. Implement `SaveEvent(ctx, userID, subject, content, embedding, metadata) (string, error)`
3. Implement `SearchEvents(ctx, params) ([]Event, error)` with hybrid search
4. Add temporal query support:
   - Date range (from/to)
   - Relative time offset (last X hours/days/months)
5. Implement hybrid ranking (text BM25 + vector similarity)

### Files to Create/Modify

- `internal/storage/surrealdb_events.go` (new)
- `internal/storage/storage.go` - Add interface methods

### Search Parameters

```go
type EventSearchParams struct {
    UserID      string     // Required: project identifier
    Subject     string     // Optional: filter by subject
    Query       string     // Optional: text/semantic query
    FromDate    *time.Time // Optional: start date
    ToDate      *time.Time // Optional: end date
    LastHours   *int       // Optional: last N hours
    LastDays    *int       // Optional: last N days
    LastMonths  *int       // Optional: last N months
    Limit       int        // Max results (default 50)
}
```

### Hybrid Search Query

```surql
SELECT *, 
    search::score(1) * 0.5 + vector::similarity::cosine(embedding, $query_embedding) * 0.5 AS relevance
FROM events
WHERE user_id = $user_id
    AND ($subject IS NONE OR subject = $subject)
    AND ($from_date IS NONE OR created_at >= $from_date)
    AND ($to_date IS NONE OR created_at <= $to_date)
    AND (content @1@ $query OR vector::similarity::cosine(embedding, $query_embedding) > 0.5)
ORDER BY relevance DESC
LIMIT $limit;
```

---

## PHASE 3: MCP Tools

**Objective**: Create the MCP tools for event storage and retrieval

### Tasks

1. Create `pkg/mcp_tools/event_tools.go`
2. Implement `save_event` tool:
   - Inputs: user_id, subject, content, metadata (optional)
   - Generates embedding from content
   - Returns event ID and timestamp
3. Implement `search_events` tool:
   - Inputs: user_id, subject (opt), query (opt), time filters (opt), limit (opt)
   - Performs hybrid search
   - Returns list of matching events
4. Register tools in `tools.go`
5. Add minimal descriptions with how_to_use references

### Tool Definitions

#### save_event

```go
type SaveEventInput struct {
    UserID   string                 `json:"user_id" jsonschema:"required,description=Project or user identifier"`
    Subject  string                 `json:"subject" jsonschema:"required,description=Semantic subject/category for the event (e.g. 'conversation:session_1' or 'log:build')"`
    Content  string                 `json:"content" jsonschema:"required,description=Event content or message"`
    Metadata map[string]interface{} `json:"metadata,omitempty" jsonschema:"description=Optional additional metadata"`
}
```

#### search_events

```go
type SearchEventsInput struct {
    UserID     string `json:"user_id" jsonschema:"required,description=Project or user identifier"`
    Subject    string `json:"subject,omitempty" jsonschema:"description=Filter by subject"`
    Query      string `json:"query,omitempty" jsonschema:"description=Text or semantic query"`
    FromDate   string `json:"from_date,omitempty" jsonschema:"description=Start date (RFC3339 format)"`
    ToDate     string `json:"to_date,omitempty" jsonschema:"description=End date (RFC3339 format)"`
    LastHours  int    `json:"last_hours,omitempty" jsonschema:"description=Get events from last N hours"`
    LastDays   int    `json:"last_days,omitempty" jsonschema:"description=Get events from last N days"`
    LastMonths int    `json:"last_months,omitempty" jsonschema:"description=Get events from last N months"`
    Limit      int    `json:"limit,omitempty" jsonschema:"description=Maximum results (default 50)"`
}
```

### Files to Create/Modify

- `pkg/mcp_tools/event_tools.go` (new)
- `pkg/mcp_tools/types.go` - Add input types
- `pkg/mcp_tools/tools.go` - Register tools

---

## PHASE 4: Documentation

**Objective**: Add documentation for the new events feature

### Tasks

1. Create `pkg/mcp_tools/docs/events_group.txt` - Group documentation
2. Create `pkg/mcp_tools/docs/tools/save_event.txt`
3. Create `pkg/mcp_tools/docs/tools/search_events.txt`
4. Update `pkg/mcp_tools/docs/overview.txt` - Add events category
5. Update `pkg/mcp_tools/help_tool.go` - Add "events" topic routing
6. Update `README.md` - Add events section

### Documentation Content

#### events_group.txt
```
=== EVENTS TOOLS GROUP ===

Temporal event storage for logs, conversations, and historical data.

TOOLS:
1. save_event - Store an event with timestamp and semantic subject
2. search_events - Query events with hybrid text+vector search

SUBJECT PATTERNS:
- conversation:session_id - Conversation logs
- log:category - Application logs
- audit:action - Audit trail
- milestone:name - Project milestones
- error:type - Error events

TIME QUERIES:
- Absolute: from_date/to_date in RFC3339 format
- Relative: last_hours, last_days, last_months

HYBRID SEARCH:
Combines BM25 text matching with vector similarity for best results.
```

---

## PHASE 5: Testing

**Objective**: Comprehensive testing of the events feature

### Tasks

1. Create unit tests in `pkg/mcp_tools/event_tools_test.go`
2. Test save_event with various subjects
3. Test search_events with:
   - Subject filter only
   - Text query
   - Date range filters
   - Relative time filters (last_hours, etc.)
   - Combined filters
4. Test hybrid search ranking
5. Verify temporal query accuracy
6. Build and verify compilation

### Test Cases

```go
// Save events
save_event(user_id="project1", subject="log:test", content="Test event 1")
save_event(user_id="project1", subject="log:test", content="Test event 2")
save_event(user_id="project1", subject="conversation:chat", content="User said hello")

// Search by subject
search_events(user_id="project1", subject="log:test") → 2 events

// Search with query
search_events(user_id="project1", query="hello") → 1 event

// Search with time filter
search_events(user_id="project1", last_hours=1) → recent events

// Combined search
search_events(user_id="project1", subject="log:test", query="test", last_days=7)
```

---

## Success Criteria

1. ✅ Events can be saved with user_id, subject, content, and optional metadata
2. ✅ Events are automatically timestamped
3. ✅ Events are embedded for semantic search
4. ✅ Search supports filtering by subject
5. ✅ Search supports text and semantic queries (hybrid)
6. ✅ Search supports absolute date ranges
7. ✅ Search supports relative time offsets
8. ✅ Documentation added to how_to_use system
9. ✅ All tests pass

---

## Related Facts

- `events_phase_1` - Schema definition phase
- `events_phase_2` - Storage implementation phase
- `events_phase_3` - MCP tools phase
- `events_phase_4` - Documentation phase
- `events_phase_5` - Testing phase
