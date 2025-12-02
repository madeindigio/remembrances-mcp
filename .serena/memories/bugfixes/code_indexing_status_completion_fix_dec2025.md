# Code Indexing Status Completion Fix - December 2025

## Problem
The `indexing_status` field in `code_projects` table was not being updated from `in_progress` to `completed` when code indexing finished successfully.

## Symptoms
- `code_list_projects` showed `indexing_status: "in_progress"` even after indexing completed
- `code_index_status` showed job as `completed` with all files processed
- `code_get_project_stats` showed all files and symbols indexed but status stuck at `in_progress`

## Root Cause Analysis
In `internal/indexer/indexer.go`, after processing files successfully:
1. The code set `project.IndexingStatus = treesitter.IndexingStatusCompleted`
2. Called `idx.storage.CreateCodeProject(ctx, project)` which uses `INSERT ... ON DUPLICATE KEY UPDATE`
3. The `ON DUPLICATE KEY UPDATE` was returning `result_count=0` indicating no rows affected

The `UpdateProjectStatus` function was only called on failure (line 111), never on success.

## Solution
Added explicit call to `UpdateProjectStatus` after successful indexing completion in `internal/indexer/indexer.go`:

```go
// Update project with final stats
now := time.Now()
project.LastIndexedAt = &now
project.IndexingStatus = treesitter.IndexingStatusCompleted
project.LanguageStats = scanResult.GetLanguageStats()

if err := idx.storage.CreateCodeProject(ctx, project); err != nil {
    log.Printf("Warning: failed to update project stats: %v", err)
}

// Explicitly update the project status to completed
// This ensures the status is updated even if CreateCodeProject's upsert doesn't update it
if err := idx.storage.UpdateProjectStatus(ctx, projectID, treesitter.IndexingStatusCompleted); err != nil {
    log.Printf("Warning: failed to update project status to completed: %v", err)
}
```

## Files Modified
- `internal/indexer/indexer.go` - Added explicit `UpdateProjectStatus` call after successful indexing

## Verification
After fix:
- `code_list_projects` correctly shows `indexing_status: "completed"`
- `code_get_project_stats` shows `indexing_status: "completed"`
- All 334 files and 1576+ symbols indexed correctly

## Related Issues
- Previous fix in `surrealdb_upsert_pattern_fix_dec2025.md` addressed similar upsert issues
- The `INSERT ON DUPLICATE KEY UPDATE` pattern in SurrealDB may not always report affected rows correctly

## Key Insight
When using SurrealDB's `INSERT ... ON DUPLICATE KEY UPDATE`, always verify the update happened or use explicit UPDATE queries for critical status changes.
