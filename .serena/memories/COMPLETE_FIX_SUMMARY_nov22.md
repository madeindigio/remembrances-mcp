# Complete Fix Summary: Metadata Persistence and Timestamp Issues

**Date:** November 22, 2025  
**Status:** ✅ Both issues resolved

## Overview

Two related issues were identified and fixed in the knowledge base system:

1. **Metadata not persisting** - Documents had empty metadata after storage
2. **Continuous reprocessing** - Files were reprocessed even when unchanged

## Issue 1: Metadata Not Persisting

### Problem
Documents saved to the knowledge base had empty metadata (`map[]`), despite being provided during save operations.

### Root Cause
The SurrealDB schema defined metadata fields as `TYPE object`, which enforces strict structure and silently discards dynamic nested fields.

### Solution: Migration V7
- **File:** `internal/storage/migrations/v7_flexible_metadata_fix.go`
- **Change:** Updated schema to use `FLEXIBLE TYPE object` for all metadata/properties fields
- **Tables affected:**
  - `knowledge_base.metadata`
  - `vector_memories.metadata`
  - `entities.properties`

### Migration Details
```go
DEFINE FIELD metadata ON knowledge_base FLEXIBLE TYPE object DEFAULT {};
DEFINE FIELD metadata ON vector_memories FLEXIBLE TYPE object DEFAULT {};
DEFINE FIELD properties ON entities FLEXIBLE TYPE object DEFAULT {};
```

### Verification
Created test (`cmd/test-save-with-metadata`) confirming nested metadata now persists correctly:
```json
{
  "source": "test",
  "last_modified": "2025-11-21T23:00:00Z",
  "nested": {
    "level1": "value1",
    "level2": {
      "deep": "value2"
    }
  },
  "array": ["item1", "item2"]
}
```
✅ All nested structures preserved

## Issue 2: Continuous Reprocessing

### Problem  
After fixing metadata persistence, files were being reprocessed on every watcher cycle, even though timestamps appeared identical in logs:
```
file_mtime=2025-11-21T23:53:22+01:00
db_mtime=2025-11-21T23:53:22+01:00
```

### Root Cause
Nanosecond precision mismatch:
- **Filesystem**: `2025-11-21T23:53:22.123456789+01:00` (with nanoseconds)
- **Database**: `2025-11-21T23:53:22.000000000+01:00` (seconds only)
- **Comparison**: `fileModTime.After(lastModTime)` always true if any nanoseconds present

### Solution: Timestamp Truncation
- **File:** `internal/kb/kb.go`
- **Function:** `processFile()`
- **Change:** Truncate both timestamps to second precision before comparison

```go
fileModTimeTrunc := fileModTime.Truncate(time.Second)
lastModTimeTrunc := lastModTime.Truncate(time.Second)
if !fileModTimeTrunc.After(lastModTimeTrunc) {
    slog.Debug("kb file not modified since last processing, skipping", ...)
    return
}
```

### Verification
Test confirmed the fix:
- **Before truncation**: `After? = true` ❌
- **After truncation**: `After? = false` ✅

## Combined Impact

### Before Fixes
- ❌ Metadata lost (empty `map[]`)
- ❌ All 39 files reprocessed continuously
- ❌ Excessive CPU/GPU usage
- ❌ 5+ minute startup times
- ❌ Log spam with reprocessing messages

### After Fixes
- ✅ Metadata preserved with full nesting
- ✅ Only modified files reprocessed
- ✅ Minimal resource usage
- ✅ ~30 second startup
- ✅ Clean logs (DEBUG level for skips)

## Files Modified

1. `internal/storage/migrations/v7_flexible_metadata_fix.go` (new)
2. `internal/storage/surrealdb_schema.go` (migration registration)
3. `internal/kb/kb.go` (timestamp comparison fix)

## Testing Tools Created

1. `cmd/check-db-state/` - Verify database contents and metadata
2. `cmd/test-save-with-metadata/` - Test metadata persistence
3. `cmd/test-migration-v7/` - Verify migration execution

## Build & Deploy

```bash
# Build with CUDA support
make BUILD_TYPE=cuda build

# Run server
./build/remembrances-mcp --config ./config.sample.gguf.yaml
```

## Current Status

- ✅ Schema at version 7
- ✅ All metadata fields using FLEXIBLE TYPE
- ✅ Timestamp comparison working correctly
- ✅ All tests passing
- ✅ Documentation complete

## Force Reprocessing (if needed)

If old documents still have empty metadata after migration, force reprocessing:

```bash
# Touch all markdown files to update their modification time
find .serena/memories -name "*.md" -type f -exec touch {} \;

# Restart server - watcher will detect "modified" files and reprocess
./build/remembrances-mcp --config ./config.sample.gguf.yaml
```

After reprocessing, all documents will have proper metadata including:
- `source`: "watcher"
- `total_size`: file size in bytes
- `last_modified`: ISO 8601 timestamp
- `chunk_count`: number of chunks
- `chunk_index`: chunk position

## Monitoring

Watch logs during operation:
```bash
# Should see minimal reprocessing
tail -f remembrances-mcp.log | grep -E "(kb file|modified|skipping)"
```

Expected behavior:
- Initial scan: processes all files once
- Subsequent cycles: DEBUG "skipping" messages only
- Only when file changes: "kb file modified, reprocessing"
