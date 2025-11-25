# SurrealDB CBOR Unmarshal Fix

## Problem Identified
The test failures were caused by CBOR unmarshaling errors in SurrealDB operations:
- Error: "cbor: cannot unmarshal array into Go value of type map[string]interface {}"
- Occurring in `GetFact` and `DeleteFact` operations

## Root Cause
The SurrealDB Go client's `surrealdb.Select()` and `surrealdb.Delete()` methods return arrays instead of single objects, but the code was expecting single map objects, causing CBOR unmarshaling errors.

## Fix Applied
Updated two methods in `~/www/MCP/remembrances-mcp/internal/storage/surrealdb.go`:

### 1. GetFact Method
**Before:** Used `surrealdb.Select[map[string]interface{}]`
**After:** Uses `surrealdb.Query[[]map[string]interface{}]` with proper array handling

### 2. DeleteFact Method  
**Before:** Used `surrealdb.Delete[map[string]interface{}]`
**After:** Uses `surrealdb.Query[[]map[string]interface{}]` with DELETE statement

## Implementation Details
Both methods now:
1. Use `surrealdb.Query` instead of direct operations
2. Expect array results with proper QueryResult structure
3. Handle empty results gracefully
4. Extract data from `queryResult.Result[0]` for records

## Testing Required
- Build the project: `go build -o dist/remembrances-mcp ./cmd/remembrances-mcp`
- Run tests: `./tests/test_stdio.sh`
- Verify no more CBOR unmarshal errors occur

## Impact
This fix should resolve the test timeout issues and CBOR errors reported in the terminal output, allowing the MCP tools to work properly with SurrealDB operations.
