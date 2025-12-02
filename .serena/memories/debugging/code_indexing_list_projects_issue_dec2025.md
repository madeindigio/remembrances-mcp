# Code Indexing: List Projects Issue - December 2025

## Problem Summary
The `code_list_projects` tool returns empty results even though code indexing completes successfully (334 files, 7+ symbols found).

## Root Cause Analysis

### Issue 1: Missing `watcher_enabled` field in schema
- The field `watcher_enabled` was used in `CreateCodeProject` but not defined in the v9_code_indexing migration
- SurrealDB SCHEMAFULL tables reject fields not in the schema
- **Fix**: Created migration v12 to add `DEFINE FIELD watcher_enabled ON code_projects TYPE bool DEFAULT false;`

### Issue 2: DELETE + CREATE timing problem
When updating an existing project:
1. First CREATE (status=in_progress): `result_count=1` ✅ WORKS
2. DELETE existing project
3. Second CREATE (status=completed): `result_count=0` ❌ FAILS

The CREATE immediately after DELETE returns empty results in SurrealDB embedded mode.

### Issue 3: UPDATE doesn't persist changes
Tried UPDATE instead of DELETE+CREATE:
- UPDATE query executes successfully
- `updated_at` field changes (proves UPDATE ran)
- But `indexing_status` remains `in_progress` instead of `completed`
- This appears to be a SurrealDB embedded mode bug with UPDATE

## Key Log Evidence

```
# First CREATE works:
time=2025-12-02T19:58:44.063 level=INFO msg="[DEBUG] CREATE code_projects result: status=OK, result_count=1"

# After DELETE, second CREATE returns empty:
time=2025-12-02T19:58:46.510 level=INFO msg="[DEBUG] CREATE code_projects result: status=OK, result_count=0"
time=2025-12-02T19:58:46.510 level=INFO msg="[WARN] CREATE code_projects returned empty result"
```

## Working Pattern Reference
`SaveFact` in `surrealdb_facts.go` uses DELETE + CREATE pattern successfully:
```go
if existingID != "" {
    deleteQuery := `DELETE FROM kv_memories WHERE user_id = $user_id AND key = $key`
    // DELETE
}
// CREATE
query := `CREATE kv_memories CONTENT {...}`
```

## Files Modified
1. `/internal/storage/migrations/v12_code_projects_watcher.go` - New migration for watcher_enabled field
2. `/internal/storage/surrealdb_schema.go` - Added case 12 for migration, updated targetVersion to 12
3. `/internal/storage/surrealdb_code_projects.go` - Multiple attempts to fix CreateCodeProject

## Current State (needs testing)
Reverted to DELETE + CREATE pattern with enhanced logging:
- DELETE existing project by project_id
- CREATE new project with all fields including preserved watcher_enabled

## Questions to Investigate
1. Why does CREATE after DELETE return empty in embedded mode?
2. Is there a timing/flush issue with SurrealDB embedded?
3. Does the same issue occur with remote SurrealDB?

## Related Code Locations
- `internal/storage/surrealdb_code_projects.go` - CreateCodeProject, GetCodeProject, ListCodeProjects
- `internal/storage/surrealdb_query_helper.go` - query, queryEmbedded methods
- `pkg/mcp_tools/code_indexing_tools.go` - codeListProjectsHandler (line 246)
- `internal/indexer/indexer.go` - calls CreateCodeProject at lines 91 and 121

## Stats Tool Works
Despite list_projects issues, `code_get_project_stats` returns correct data:
- files_count: 334
- symbols_count: 1589
- Proper breakdown by language and symbol type
