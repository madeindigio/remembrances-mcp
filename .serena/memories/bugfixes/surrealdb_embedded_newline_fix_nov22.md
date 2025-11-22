# SurrealDB Embedded Newline Deserialization Bug - Complete Fix Report

**Date**: November 22, 2025  
**Branch**: bugfix/embeddings-embedded-surrealdb  
**Status**: ✅ Fixed and Tested

## Executive Summary

Fixed critical bug in SurrealDB embedded mode causing deserialization failures when storing or updating content containing newlines. The error manifested as `"invalid character '\n' in string literal"` during UPDATE and DELETE operations.

## Problem Description

### Symptoms
- `to_remember` tool failed when saving multiline content
- UPDATE operations on kv_memories, semantic_memories, and entities failed
- DELETE operations with record IDs failed when deleted records contained newlines
- Error: `failed to unmarshal result: invalid character '\n' in string literal`

### Root Cause

The SurrealDB embedded library (`libsurrealdb_embedded_rs.so`) has a response deserialization issue:

1. **UPDATE record_id** operations return the updated record in the response
2. **DELETE record_id** operations return the deleted record in the response
3. When these records contain newlines, the CBOR-to-JSON deserialization fails
4. The error occurs **during** `s.embeddedDB.Query()` execution, not after

### Investigation Path

Initial attempts that failed:
- ❌ Modifying UPDATE queries with RETURN BEFORE
- ❌ Using UPDATE with WHERE clauses
- ❌ Adding workaround to catch deserialization errors (masked problem but operation didn't execute)
- ❌ DELETE record_id with RETURN NONE (DELETE still returns records by default)

## Solution Implemented

### Strategy: DELETE FROM WHERE + CREATE Pattern

Changed all mutation operations from:
```sql
UPDATE record_id SET field = value
DELETE record_id
```

To:
```sql
DELETE FROM table WHERE condition  -- doesn't return records
CREATE table CONTENT {...}         -- recreates with new values
```

### Key Insight

- `DELETE FROM table WHERE condition` does NOT return deleted records by default
- `DELETE record_id` DOES return the deleted record
- `CREATE` operations work perfectly with newlines
- This pattern avoids deserialization entirely

## Files Modified

### 1. internal/storage/surrealdb_facts.go

**SaveFact**:
```go
// Old: UPDATE record_id SET value = $value
// New: DELETE FROM kv_memories WHERE user_id = X AND key = Y
//      CREATE kv_memories CONTENT {...}
```

**UpdateFact**:
```go
// Same DELETE + CREATE pattern
```

**DeleteFact**:
```go
// Removed RETURN BEFORE clause
// Changed from: DELETE FROM ... RETURN BEFORE
// To: DELETE FROM ... (no RETURN)
```

### 2. internal/storage/surrealdb_vectors.go

**UpdateVector**:
```go
// Old: UPDATE record_id SET content = $content, embedding = $embedding
// New: DELETE FROM semantic_memories WHERE id = $id
//      CREATE semantic_memories CONTENT {...}
```

**DeleteVector**:
```go
// Old: s.delete(ctx, id) -- which does DELETE record_id
// New: DELETE FROM semantic_memories WHERE id = $id
```

### 3. internal/storage/surrealdb_entities.go

**DeleteEntity**:
```go
// Old: s.delete(ctx, entityID)
// New: DELETE FROM entities WHERE id = $id
```

### 4. internal/storage/surrealdb_query_helper.go

**Removed**:
- Incorrect workaround that caught errors but didn't fix the operation
- Removed `strings` import (no longer needed)

### 5. internal/storage/surrealdb_schema.go

**Migration V8** (already implemented):
- Added FLEXIBLE TYPE to kv_memories.value field
- Supports dynamic content types (string, int, float, bool, object, array)

### 6. pkg/mcp_tools/remember_tools.go

**lastToRememberHandler**:
- Removed markdown wrapper from YAML responses
- Changed: `fmt.Sprintf("```yaml\n%s```")` → `string(yamlBytes)`

## Testing Results

### ✅ Facts (Key-Value)
- SaveFact: Works with multiline content
- UpdateFact: Works with multiline content
- DeleteFact: Works correctly
- GetFact: Returns multiline content intact

### ✅ Vectors (Semantic Memory)
- AddVector: Works with multiline content
- UpdateVector: Works with multiline content
- DeleteVector: Works correctly
- SearchVectors: Returns multiline results intact

### ✅ Entities (Knowledge Graph)
- CreateEntity: Works with properties containing newlines
- DeleteEntity: Works correctly
- GetEntity: Returns data intact

### ✅ Documents (Knowledge Base)
- SaveDocument: Already working (was using DELETE FROM WHERE)
- DeleteDocument: Already working

### ✅ MCP Tools
- `to_remember`: Successfully stores multiline content
- `last_to_remember`: Returns pure YAML (no markdown wrapper)
- `save_fact`: Works with multiline values
- `get_fact`: Retrieves multiline values correctly

## Build Instructions

```bash
make BUILD_TYPE=cuda build
```

Binary: `build/remembrances-mcp`

## Migration Information

**Schema Version**: V8  
**Migration Files**:
- `internal/storage/migrations/v8_flexible_kv_value.go`

**Migration SQL**:
```sql
REMOVE FIELD value ON kv_memories;
DEFINE FIELD value ON kv_memories 
  FLEXIBLE TYPE option<string|int|float|bool|object|array>;
```

## Impact Assessment

### Affected Operations
- All UPDATE operations on facts, vectors, and entities
- All DELETE operations using record IDs
- Approximately 6 functions modified

### Breaking Changes
- None (internal implementation only)

### Performance Impact
- DELETE + CREATE is slightly slower than UPDATE
- Negligible for typical usage patterns
- Benefit: Eliminates deserialization failures

## Verification Commands

Test multiline content storage:
```bash
# Via MCP tool
to_remember "Line 1\nLine 2\nLine 3"

# Check it worked
last_to_remember
```

## Related Issues

- Similar to metadata persistence bug fixed on Nov 21, 2025
- Both required FLEXIBLE TYPE to handle dynamic content
- Same deserialization issue pattern

## Conclusion

The DELETE FROM WHERE + CREATE pattern successfully resolves the SurrealDB embedded newline deserialization bug across all affected operations. All MCP tools now work correctly with multiline content.

**Status**: Production Ready ✅