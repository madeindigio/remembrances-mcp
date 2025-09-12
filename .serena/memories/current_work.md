# Plan to Fix `remembrance_get_stats`

This document outlines the plan to fix the `remembrance_get_stats` tool to provide accurate, user-scoped memory statistics. The current implementation suffers from several issues, including returning global counts for some stats instead of user-specific ones and being inefficient.

The proposed solution is to create a dedicated `user_stats` table to store aggregated statistics for each user. This table will be updated atomically with each memory operation (create, delete), ensuring that the stats are always up-to-date and can be retrieved with a single, efficient query.

### 1. Update the Database Schema

A new table, `user_stats`, will be added to the SurrealDB schema to store the memory counts for each user.

**File to Modify:** `internal/storage/surrealdb_schema.go`

**Action:** Add the following table definition within the `InitializeSchema` function.

```go
// In InitializeSchema function
// ... (other table definitions)

// Define user_stats table
slog.Info("Defining table: user_stats")
if _, err := db.Query(db, "DEFINE TABLE user_stats SCHEMALESS", nil); err != nil {
    return fmt.Errorf("failed to define user_stats table: %w", err)
}
```

### 2. Create Helper Functions for Atomic Updates

A helper function will be created to handle atomic increments and decrements of the statistics in the `user_stats` table. This ensures that even with concurrent operations, the counts remain accurate.

**File to Modify:** `internal/storage/surrealdb.go`

**Action:** Add the following helper function to the `SurrealDBStorage` struct. This function will handle both incrementing and decrementing stat counters.

```go
// updateUserStat atomically updates a specific statistic for a user.
// It uses a transaction to ensure consistency.
func (s *SurrealDBStorage) updateUserStat(ctx context.Context, userID, statField string, delta int) error {
    // Use a transaction for atomicity
    tx, err := s.db.Begin()
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Check if the user stats record exists
    query := "SELECT * FROM user_stats WHERE user_id = $user_id"
    params := map[string]interface{}{"user_id": userID}
    
    var results []map[string]interface{}
    err = tx.Select(query, params, &results)

    if err != nil || len(results) == 0 {
        // Record doesn't exist, create it
        createParams := map[string]interface{}{
            "user_id": userID,
            statField: delta,
        }
        _, err = tx.Create("user_stats", createParams)
    } else {
        // Record exists, update it
        updateQuery := fmt.Sprintf("UPDATE user_stats SET %s += %d WHERE user_id = $user_id", statField, delta)
        _, err = tx.Query(updateQuery, params)
    }

    if err != nil {
        return fmt.Errorf("failed to update user stat: %w", err)
    }

    return tx.Commit()
}
```

### 3. Integrate Statistics Updates into All Memory Operations

The `updateUserStat` function will be called from every function that creates or deletes a memory item.

**Files to Modify and Actions:**

*   **`internal/storage/surrealdb_facts.go`**:
    *   In `SaveFact`, call `updateUserStat(ctx, userID, "key_value_count", 1)`.
    *   In `DeleteFact`, call `updateUserStat(ctx, userID, "key_value_count", -1)`.

*   **`internal/storage/surrealdb_vectors.go`**:
    *   In `IndexVector`, call `updateUserStat(ctx, userID, "vector_count", 1)`.
    *   In `DeleteVector`, call `updateUserStat(ctx, userID, "vector_count", -1)`.

*   **`internal/storage/surrealdb.go`**:
    *   In `CreateEntity`, call `updateUserStat(ctx, "global", "entity_count", 1)` (assuming entities are global for now, or adapt if they become user-scoped).
    *   In `DeleteEntity`, call `updateUserStat(ctx, "global", "entity_count", -1)`.
    *   In `CreateRelationship`, call `updateUserStat(ctx, "global", "relationship_count", 1)`.
    *   In `SaveDocument`, call `updateUserStat(ctx, "global", "document_count", 1)`.
    *   In `DeleteDocument`, call `updateUserStat(ctx, "global", "document_count", -1)`.

### 4. Refactor `GetStats` to Use the New Table

The `GetStats` function will be simplified to query the `user_stats` table directly, making it much more efficient.

**File to Modify:** `internal/storage/surrealdb.go`

**Action:** Replace the current `GetStats` implementation with the following:

```go
// GetStats returns statistics about stored memories for a user.
func (s *SurrealDBStorage) GetStats(ctx context.Context, userID string) (*MemoryStats, error) {
    query := "SELECT * FROM user_stats WHERE user_id = $user_id"
    params := map[string]interface{}{"user_id": userID}

    var stats []MemoryStats
    err := s.db.Select(query, params, &stats)
    if err != nil {
        return nil, fmt.Errorf("failed to get stats for user %s: %w", userID, err)
    }

    if len(stats) == 0 {
        // Return empty stats if no record found
        return &MemoryStats{}, nil
    }

    return &stats[0], nil
}
```

This plan will ensure that the `remembrance_get_stats` tool provides accurate, user-specific, and performant memory statistics.
