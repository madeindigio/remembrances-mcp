# Fix: SurrealDB External RecordID Deserialization Issue

**Date**: December 10, 2025  
**Status**: ✅ Resolved  
**Severity**: Critical - Blocking code indexing with external SurrealDB

## Problem Description

When using external SurrealDB (via WebSocket connection), the code indexing system was failing with JSON deserialization errors:

```
json: cannot unmarshal object into Go struct field CodeProject.id of type string
```

The error occurred because Record IDs were being returned as nested objects instead of strings:

```json
"id": {"Table": "code_projects", "ID": "ik7glid45nqg9w1q18wc"}
```

## Root Cause

The issue was a difference in how SurrealDB returns Record IDs depending on the connection type:

1. **Embedded SurrealDB**: Returns Record IDs that get deserialized as `map[string]interface{}` with lowercase keys `{"id": "xxx", "tb": "table"}`

2. **External SurrealDB (WebSocket)**: Returns Record IDs as `models.RecordID` struct instances with fields `Table` and `ID`

The existing `normalizeSurrealDBDatetimes()` function only handled the map format, not the struct format.

## Solution

Modified `internal/storage/surrealdb_helpers.go` to detect and convert `models.RecordID` structs:

```go
// Detect models.RecordID by type name
if strings.Contains(dataType, "RecordID") {
    // Use reflection to access Table and ID fields
    val := reflect.ValueOf(data)
    if val.Kind() == reflect.Struct {
        // Extract fields and convert to "table:id" format
        tableStr := fmt.Sprintf("%v", tableField.Interface())
        idStr := fmt.Sprintf("%v", idField.Interface())
        result := tableStr + ":" + idStr
        return result
    }
}
```

### Key Changes

1. **Type Detection**: Check for `"RecordID"` in the type name
2. **Reflection Access**: Use Go reflection to access struct fields directly
3. **Format Conversion**: Convert to standard `"table:id"` string format before JSON marshaling

### Files Modified

- `internal/storage/surrealdb_helpers.go`: Added RecordID struct detection and conversion logic
- `internal/config/config.go`: Temporarily set log level to DEBUG for troubleshooting (should revert to INFO)

## Verification

After the fix:
- ✅ Code indexing works with external SurrealDB
- ✅ Projects list correctly with properly formatted IDs
- ✅ No deserialization errors in logs
- ✅ Compatible with both embedded and external SurrealDB

## Future Considerations

1. Revert log level back to INFO in production
2. Consider adding unit tests for RecordID normalization
3. Document SurrealDB external setup requirements

## Related Issues

- Initial implementation worked only with embedded SurrealDB
- External SurrealDB support required for distributed deployments
- Affects all code indexing operations (projects, files, symbols)
