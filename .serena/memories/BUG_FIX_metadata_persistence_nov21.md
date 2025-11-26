# Bug Fix: Metadata Persistence in SurrealDB Embedded - November 21, 2025

## Executive Summary

**Status**: ✅ **RESOLVED**

Fixed a critical bug where nested `metadata` maps were being stored as empty objects `{}` in the SurrealDB embedded database, causing data loss and forcing unnecessary reprocessing of documents.

**Root Cause**: Schema definition used `TYPE object` instead of `FLEXIBLE TYPE object`, preventing SurrealDB from storing dynamic nested fields.

**Solution**: Added Migration V7 to redefine metadata fields with `FLEXIBLE` keyword, allowing dynamic object structures.

**Impact**: 
- ✅ Metadata now persists correctly with all nested fields
- ✅ Documents no longer reprocessed on every restart
- ✅ Knowledge base properly tracks file modifications
- ✅ All storage layers (vector_memories, knowledge_base, entities) fixed

---

## Bug Analysis Process

### Investigation Timeline

1. **Initial Symptoms** (documented in `surrealdb_embedded_fixes_nov21.md`):
   - Every restart reprocessed entire knowledge base
   - Metadata appeared to be lost between sessions
   - Database files existed but queries returned empty metadata

2. **BUG_ANALYSIS.md Findings** (from surrealdb-embedded library):
   - Comprehensive testing proved the **library works correctly**
   - Bug confirmed to be in **application layer**, not library
   - Test suite showed nested maps serialize/deserialize perfectly

3. **Direct Testing**:
   - `test-simple-create`: Library direct usage ✅ PASSED
   - `test-direct-save`: Storage layer usage ❌ FAILED (metadata empty)
   - Confirmed bug in `remembrances-mcp` application

4. **Schema Investigation**:
   - Created test comparing `SCHEMAFULL` vs `SCHEMALESS`
   - Found: `TYPE object` → metadata empty `{}`
   - Found: `SCHEMALESS` → metadata works perfectly ✅

### Root Cause Identified

**File**: `internal/storage/surrealdb_schema.go`
**Lines**: 212, 222, 232 (in Migration V1)

```go
// WRONG - Does not allow dynamic fields
`DEFINE FIELD metadata ON knowledge_base TYPE object DEFAULT {};`
`DEFINE FIELD metadata ON vector_memories TYPE object DEFAULT {};`
`DEFINE FIELD properties ON entities TYPE object DEFAULT {};`
```

**Why it fails**:
- SurrealDB's `TYPE object` requires pre-defined schema for all nested fields
- Dynamic fields like `last_modified`, `source`, `chunk_index` are rejected
- Fields are silently discarded, resulting in empty `{}`

**Why Migration V5 didn't fix it**:
```go
case 5:
    // V5: Flexible metadata/properties (already in V1 for embedded)
    log.Println("Migration V5: flexible metadata already present in embedded schema")
```
- Migration V5 **incorrectly claimed** flexible metadata was already present
- No actual schema modification was performed
- Bug persisted from initial schema creation

---

## The Fix: Migration V7

### Changes Made

#### 1. Updated Target Version
**File**: `internal/storage/surrealdb_schema.go:33`
```go
targetVersion := 7 // v7: fix metadata to allow flexible fields
```

#### 2. Fixed Initial Schema (Migration V1)
**Lines**: 212, 222, 232
```go
// CORRECT - Allows dynamic nested fields
`DEFINE FIELD metadata ON vector_memories FLEXIBLE TYPE object DEFAULT {};`
`DEFINE FIELD metadata ON knowledge_base FLEXIBLE TYPE object DEFAULT {};`
`DEFINE FIELD properties ON entities FLEXIBLE TYPE object DEFAULT {};`
```

#### 3. Added Migration V7 for Existing Databases
**Lines**: 265-277
```go
case 7:
    // V7: Fix metadata/properties fields to be FLEXIBLE
    // This allows dynamic nested fields in metadata objects
    statements = []string{
        // Remove old field definitions
        `REMOVE FIELD metadata ON vector_memories;`,
        `REMOVE FIELD metadata ON knowledge_base;`,
        `REMOVE FIELD properties ON entities;`,
        // Redefine with FLEXIBLE
        `DEFINE FIELD metadata ON vector_memories FLEXIBLE TYPE object DEFAULT {};`,
        `DEFINE FIELD metadata ON knowledge_base FLEXIBLE TYPE object DEFAULT {};`,
        `DEFINE FIELD properties ON entities FLEXIBLE TYPE object DEFAULT {};`,
    }
    log.Println("Migration V7: Fixed metadata/properties fields to be FLEXIBLE (allows dynamic nested fields)")
```

#### 4. Remote Mode Handler
**Lines**: 177-179
```go
case 7:
    // V7 is embedded-only, skip for remote
    log.Println("Migration V7: metadata fix is embedded-only, skipping for remote")
    return nil
```

#### 5. Removed Debug Logging
Cleaned up temporary debug statements from:
- `internal/storage/surrealdb_query_helper.go` (lines 52-68)
- `internal/storage/surrealdb_documents.go` (lines 153, 286, 307-310)
- `internal/storage/surrealdb_stats.go` (lines 134, 185)

---

## Test Results

### Before Fix
```
Input metadata: map[chunk_count:2 chunk_index:0 last_modified:2025-11-21T23:27:23+01:00 source:test]

[DEBUG queryEmbedded] BEFORE Query - params metadata: map[chunk_count:2 chunk_index:0 ...]
[DEBUG queryEmbedded] AFTER Query - raw results[0] metadata: map[]  ← EMPTY!

Retrieved Metadata: map[]  ← DATA LOST!
```

### After Fix
```
Input metadata: map[chunk_count:2 chunk_index:0 last_modified:2025-11-21T23:31:06+01:00 source:test]

CREATE results:
{
  "metadata": {
    "chunk_count": 2,
    "chunk_index": 0,
    "last_modified": "2025-11-21T23:31:06+01:00",
    "source": "test"
  }
}

✅ SUCCESS! Document persisted: test-file.md#chunk0
   Metadata: map[chunk_count:2 chunk_index:0 last_modified:2025-11-21T23:31:06+01:00 source:test]
```

### Test Suite Results

| Test | Status | Evidence |
|------|--------|----------|
| `test-simple-create` (library direct) | ✅ PASS | Data persists correctly |
| `test-schemaless` | ✅ PASS | Metadata with 4 fields preserved |
| `test-schemafull-before-fix` | ❌ FAIL | Metadata empty `{}` |
| `test-schemafull-after-fix` | ✅ PASS | All metadata fields preserved |
| `test-direct-save` (full stack) | ✅ PASS | Metadata persists across restarts |

---

## Migration Process

### For Fresh Installations
- Migration V7 runs automatically during `InitializeSchema()`
- Initial schema (V1) now includes `FLEXIBLE` keyword
- No action needed

### For Existing Databases
1. On first startup after update:
   ```
   Running schema migrations from version 6 to 7
   Applying migration to version 7
   Migration V7: Fixed metadata/properties fields to be FLEXIBLE (allows dynamic nested fields)
   Schema initialization completed
   ```

2. Migration automatically:
   - Removes old `TYPE object` field definitions
   - Recreates fields with `FLEXIBLE TYPE object`
   - Preserves existing data (SurrealDB handles gracefully)

3. Verification:
   ```bash
   # Check migration applied
   SELECT * FROM schema_version:current;
   # Should show: {"version": 7}
   
   # Test metadata persistence
   go run cmd/test-direct-save/main.go
   # Should show: ✅ SUCCESS! with full metadata
   ```

---

## Related Files Modified

### Core Fix
1. `internal/storage/surrealdb_schema.go`
   - Line 33: Updated target version to 7
   - Lines 177-179: Added V7 handler for remote mode
   - Lines 212, 222, 232: Added `FLEXIBLE` to V1 schema
   - Lines 265-277: Added V7 migration logic

### Cleanup
2. `internal/storage/surrealdb_query_helper.go`
   - Removed debug logging (lines 52-68)

3. `internal/storage/surrealdb_documents.go`
   - Removed debug logging and unused variable
   - Fixed compilation warning

4. `internal/storage/surrealdb_stats.go`
   - Removed debug logging

---

## Technical Details

### SurrealDB Schema Keywords

| Keyword | Behavior | Use Case |
|---------|----------|----------|
| `TYPE object` | Strict schema - all fields must be pre-defined | Fixed structure objects |
| `FLEXIBLE TYPE object` | Dynamic schema - allows any fields | User metadata, dynamic properties |
| `TYPE option<object>` | Nullable flexible object | Optional metadata |

### Why FLEXIBLE is Required

SurrealDB's default `TYPE object` enforces strict field validation:

```surrealql
-- This FAILS with TYPE object
CREATE knowledge_base CONTENT {
    metadata: {
        source: "test",           -- Not in schema → REJECTED
        last_modified: "2025..."  -- Not in schema → REJECTED
    }
};
-- Result: metadata = {}

-- This WORKS with FLEXIBLE TYPE object
CREATE knowledge_base CONTENT {
    metadata: {
        source: "test",           -- Dynamic field → ACCEPTED
        last_modified: "2025..."  -- Dynamic field → ACCEPTED
        any_field: "any_value"    -- Dynamic field → ACCEPTED
    }
};
-- Result: metadata = {source: "test", last_modified: "2025...", any_field: "any_value"}
```

### Backward Compatibility

Migration V7 is **non-destructive**:
- `REMOVE FIELD` does not delete data, only schema definition
- `DEFINE FIELD` redefines schema without touching data
- Existing metadata in documents is preserved
- New documents work immediately after migration

---

## Lessons Learned

1. **Always test schema changes with real data**
   - The migration V5 comment was misleading
   - No actual verification was performed

2. **SurrealDB's TYPE object is strict by default**
   - Unlike JSON or MongoDB, requires explicit FLEXIBLE
   - Silent failures can occur (empty objects instead of errors)

3. **Debug logging is essential during diagnosis**
   - Logging at each layer helped identify exact failure point
   - `[DEBUG queryEmbedded]` logs were critical

4. **Cross-reference external bug analyses**
   - BUG_ANALYSIS.md from surrealdb-embedded saved significant time
   - Confirmed library was not the issue

5. **Test at multiple abstraction levels**
   - Library level: `test-simple-create`
   - Storage layer: `test-direct-save`
   - Helped pinpoint exact layer of failure

---

## References

- **Related Documentation**:
  - `.serena/memories/surrealdb_embedded_fixes_nov21.md` - Initial investigation
  - `~/www/MCP/Remembrances/surrealdb-embedded/BUG_ANALYSIS.md` - Library testing
  - `internal/storage/surrealdb_schema.go` - Schema definitions

- **Test Programs**:
  - `cmd/test-simple-create/main.go` - Direct library test
  - `cmd/test-direct-save/main.go` - Full stack persistence test
  - `cmd/test-all-tables/main.go` - Database inspection

- **Key Commits**:
  - Migration V7 implementation
  - Schema V1 FLEXIBLE keyword addition
  - Debug logging cleanup

---

## Validation Checklist

- [x] Migration V7 runs successfully
- [x] Fresh installations use correct schema
- [x] Existing databases migrate without data loss
- [x] Metadata persists with all nested fields
- [x] Documents not reprocessed on restart
- [x] All test programs pass
- [x] Debug logging removed
- [x] Code compiles without warnings
- [x] Documentation updated

---

## Date

November 21, 2025

## Status

✅ **RESOLVED AND DEPLOYED**