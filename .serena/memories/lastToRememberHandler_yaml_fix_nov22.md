# Fix for lastToRememberHandler YAML Response - 22 Nov 2025

## Problem Identified
The `lastToRememberHandler` tool was returning YAML wrapped in markdown formatting instead of pure YAML.

## Changes Made
**File**: `pkg/mcp_tools/remember_tools.go`
**Line**: 146 

### Before:
```go
return protocol.NewCallToolResult([]protocol.Content{
    &protocol.TextContent{
        Type: "text",
        Text: fmt.Sprintf("Recent memory context for user '%s':\n\n```yaml\n%s```", input.UserID, string(yamlBytes)),
    },
}, false), nil
```

### After:
```go
return protocol.NewCallToolResult([]protocol.Content{
    &protocol.TextContent{
        Type: "text",
        Text: string(yamlBytes),
    },
}, false), nil
```

## Result
The tool now returns clean YAML output without markdown code blocks or prefix text, as specified in the tool description.

## Testing
Tested with `mcp_remembrances_last_to_remember` and verified output is pure YAML:
```yaml
note: This information may be of interest to remember what you have been working on most recently or what is important to remember
recent_documents:
    - COMPLETE_FIX_SUMMARY_nov22.md#chunk5
retrieved_at: "2025-11-22T00:16:18+01:00"
to_remember: Siempre usar make BUILD_TYPE=cuda build para compilar la aplicación antes de probarla. El binario resultante se encuentra en build/remembrances-mcp.
user_id: remembrances-mcp
```

## Secondary Issue FIXED

**Tool**: `to_remember`  
**Previous Error**: When saving content with newlines, failed with:
```
failed to unmarshal result: invalid character '\n' in string literal
```

### Root Cause Analysis
The problem was NOT in the SurrealDB embedded library serialization, but in the `SaveFact` and `UpdateFact` implementations:

1. **CREATE operation**: Used `s.create()` method which passes data through Go's JSON marshal/unmarshal, causing issues with newline characters
2. **UPDATE operation**: Passed `time.Now().UTC()` as a Go `time.Time` object instead of using SurrealQL's `time::now()` function

### Solution
**File**: `internal/storage/surrealdb_facts.go`

Changed both CREATE and UPDATE operations to use `s.query()` with SurrealQL syntax:

**CREATE** - Before:
```go
data := map[string]interface{}{
    "user_id": userID,
    "key":     key,
    "value":   value,
}
if _, err := s.create(ctx, "kv_memories", data); err != nil {
    return fmt.Errorf("failed to save fact: %w", err)
}
```

**CREATE** - After:
```go
query := `
    CREATE kv_memories CONTENT {
        user_id: $user_id,
        key: $key,
        value: $value
    }
`
params := map[string]interface{}{
    "user_id": userID,
    "key":     key,
    "value":   value,
}
if _, err := s.query(ctx, query, params); err != nil {
    return fmt.Errorf("failed to save fact: %w", err)
}
```

**UPDATE** - Before:
```go
updateQuery := "UPDATE " + existingID + " SET value = $value, updated_at = $updated_at"
params := map[string]interface{}{
    "value":      value,
    "updated_at": time.Now().UTC(),
}
```

**UPDATE** - After:
```go
updateQuery := "UPDATE " + existingID + " SET value = $value, updated_at = time::now()"
params := map[string]interface{}{
    "value": value,
}
```

### Why This Works
- `s.query()` with parameterized SurrealQL properly handles string escaping for newlines and special characters
- `time::now()` is a SurrealDB function that generates the timestamp server-side, avoiding Go serialization issues
- This matches the pattern used successfully in `SaveDocument()` which already worked with multiline content

### Lessons Learned
When debugging serialization issues:
1. Look for differences between working and non-working implementations
2. Check if similar operations (like `SaveDocument`) already solve the problem
3. Prefer using database-native functions (`time::now()`) over passing language-specific types (`time.Time`)
4. Use parameterized queries (`s.query()`) rather than ORM-style methods (`s.create()`) when dealing with complex data

## Build Status
Successfully compiled with:
```bash
make BUILD_TYPE=cuda build
```

Binary location: `build/remembrances-mcp`
