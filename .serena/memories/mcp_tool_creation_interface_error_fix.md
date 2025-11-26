# MCP Tool Creation "unsupported type: interface" Error Fix

## Problem
When running the MCP server, we encountered the error:
```
failed to create tool name=remembrance_save_fact err="unsupported type: interface"
```

This was causing tool registration to fail with "tool remembrance_save_fact creation returned nil".

## Root Cause
The MCP protocol's `protocol.NewTool()` function doesn't support `interface{}` types in tool input struct definitions. Several input structs in `pkg/mcp_tools/tools.go` contained these unsupported types:

1. `SaveFactInput.Value` was `interface{}`
2. Multiple metadata/properties fields were `map[string]interface{}`

## Solution Applied
1. **Changed input struct types** to MCP-compatible types:
   - `interface{}` → `string` (for Value field)
   - `map[string]interface{}` → `map[string]string` (for metadata and properties)

2. **Added conversion helper**:
   ```go
   func stringMapToInterfaceMap(m map[string]string) map[string]interface{} {
       if m == nil {
           return nil
       }
       result := make(map[string]interface{}, len(m))
       for k, v := range m {
           result[k] = v
       }
       return result
   }
   ```

3. **Updated handlers** to convert between the input types and storage interface expectations:
   - `addVectorHandler`: `stringMapToInterfaceMap(input.Metadata)`
   - `updateVectorHandler`: `stringMapToInterfaceMap(input.Metadata)`
   - `createEntityHandler`: `stringMapToInterfaceMap(input.Properties)`
   - `createRelationshipHandler`: `stringMapToInterfaceMap(input.Properties)`
   - `addDocumentHandler`: `stringMapToInterfaceMap(input.Metadata)`

4. **Enhanced error handling** for tool creation:
   - Changed all `tool, _ := protocol.NewTool(...)` to `tool, err := protocol.NewTool(...)`
   - Added error logging with `slog.Error()` when tool creation fails
   - Return nil from factory functions on error so the registration helper reports which tool failed

## Result
- Tool creation should now succeed without "unsupported type: interface" errors
- If any tool creation still fails, the logs will show the specific tool name and underlying error
- The registration helper will report which specific tool returned nil

## Files Modified
- `~/www/MCP/remembrances-mcp/pkg/mcp_tools/tools.go`:
  - Updated all input struct type definitions
  - Added `stringMapToInterfaceMap` helper function
  - Updated all tool factory functions to handle errors
  - Updated all affected handlers to use the conversion helper

## Next Steps
- Test with `xc run-tests` to verify the fix
- Monitor logs for any remaining tool creation errors
- Consider refactoring the duplicated "failed to create tool" string literal into a constant if desired
