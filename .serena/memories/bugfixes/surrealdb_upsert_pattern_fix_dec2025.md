# SurrealDB UPSERT Pattern Fix - December 2025

**Date**: December 2, 2025
**Status**: ✅ Fixed

## Problem

The `UPSERT ... SET ... WHERE` pattern in SurrealDB does not work as expected. When used with a WHERE clause, it does not properly update existing records - instead it may create duplicates or fail silently.

## Affected Files (Fixed)

1. `internal/storage/surrealdb_code_projects.go` - CreateCodeProject
2. `internal/storage/surrealdb_code_files.go` - SaveCodeFile
3. `internal/storage/surrealdb_code_symbols.go` - SaveCodeSymbol
4. `internal/storage/surrealdb_code_chunks.go` - SaveCodeChunk

## Incorrect Pattern (DO NOT USE)

```go
query := `
    UPSERT table_name SET
        field1 = $field1,
        field2 = $field2
    WHERE key_field = $key_field;
`
_, err := s.query(ctx, query, params)
```

## Correct Pattern (USE THIS)

The working pattern is SELECT + CREATE/UPDATE, as used in `SaveDocument` and `SaveFact`:

```go
// 1. Check if record exists
existsQuery := "SELECT id FROM table_name WHERE key_field = $key_field"
existsResult, err := s.query(ctx, existsQuery, map[string]interface{}{
    "key_field": value,
})

isNew := true
if existsResult != nil && len(*existsResult) > 0 {
    queryResult := (*existsResult)[0]
    if queryResult.Status == "OK" && len(queryResult.Result) > 0 {
        isNew = false
    }
}

// 2. CREATE or UPDATE based on existence
if isNew {
    query := `CREATE table_name CONTENT { ... }`
    // ...
} else {
    query := `UPDATE table_name SET ... WHERE key_field = $key_field`
    // ...
}
```

## Files That Already Use Correct Pattern

- `internal/storage/surrealdb_documents.go` - SaveDocument ✅
- `internal/storage/surrealdb_facts.go` - SaveFact (uses DELETE + CREATE) ✅
- `internal/storage/surrealdb_events.go` - SaveEvent (INSERT only, events are immutable) ✅

## Special Case: Schema Version

The `UPSERT schema_version:current SET version = $version` works because it uses a **fixed record ID** (`:current`), not a WHERE clause.

## Root Cause

SurrealDB's UPSERT with WHERE clause behaves differently than expected:
- The WHERE clause filters which records to update
- If no records match, UPSERT does NOT create a new record
- This is different from traditional SQL UPSERT/MERGE behavior

## Testing

After applying these fixes, the `indexing_status` field properly updates from `in_progress` to `completed` when code indexing finishes.
