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

## Secondary Issue Discovered
Found a bug in `SaveFact` with multiline strings - SurrealDB embedded mode has issues with newline characters in JSON serialization. This causes `to_remember` tool to fail when saving content with line breaks.

Error: "failed to unmarshal result: invalid character '\n' in string literal"
Location: `internal/storage/surrealdb_helpers.go:191`

This is a separate issue that requires investigation of how SurrealDB embedded handles string escaping in JSON marshaling/unmarshaling.

## Build Status
Successfully compiled with:
```bash
make BUILD_TYPE=cuda build
```

Binary location: `build/remembrances-mcp`
