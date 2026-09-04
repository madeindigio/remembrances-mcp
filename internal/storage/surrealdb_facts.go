package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/madeindigio/remembrances-mcp/internal/storage/migrations"
)

// The key-value memory surface, scoped per RM-2.
//
// Every method resolves the principal first and refuses to run without one.
// The `userID` parameter is the legacy path: it is consulted only when the
// context carries no principal and require_principal is off, and it is then
// mapped onto the records the v13 backfill derived from that same label.
//
// Replacement is invalidation, not deletion: saving a new value for a key
// stamps valid_to on the old row and creates a new one. That is what lets the
// RM-9 feed tell a consumer "this fact changed" rather than silently losing
// the previous state.

// factContentHash is the SurrealQL that fills content_hash on kv_memories. It
// has to agree exactly with migrations.ContentHash, which the dedup index and
// the v13 backfill both rely on.
var factContentHash = migrations.ContentHashSurrealExpr("<string> $value")

// SaveFact saves a key-value fact in the caller's write scope, replacing any
// live fact with the same key.
func (s *SurrealDBStorage) SaveFact(ctx context.Context, userID, key string, value interface{}) error {
	t, err := s.scope(ctx, userID)
	if err != nil {
		return err
	}

	params := t.params(map[string]interface{}{
		"key":     key,
		"value":   value,
		"user_id": t.Principal().LegacyUserID,
	})

	invalidate := fmt.Sprintf("UPDATE kv_memories SET %s %s",
		t.invalidateSet(), t.where("key = $key"))
	if _, err := s.query(ctx, invalidate, params); err != nil {
		return fmt.Errorf("failed to invalidate the existing fact: %w", err)
	}

	create := fmt.Sprintf(`
		CREATE kv_memories CONTENT {
			%suser_id: $user_id,
			key: $key,
			value: $value,
			content_hash: %s
		}
	`, t.contentFields(), factContentHash)
	if _, err := s.query(ctx, create, params); err != nil {
		return fmt.Errorf("failed to save fact: %w", err)
	}

	// Recalculate statistics after mutation
	if err := s.updateUserStat(ctx, t.Principal().LegacyUserID, "key_value_count", 1); err != nil {
		slog.Warn("failed to update key_value_count stat", "user_id", userID, "error", err)
	}

	return nil
}

// GetFact retrieves the live value of a key in the caller's visible scopes.
func (s *SurrealDBStorage) GetFact(ctx context.Context, userID, key string) (interface{}, error) {
	t, err := s.scope(ctx, userID)
	if err != nil {
		return nil, err
	}

	// `value` is backtick escaped because SELECT VALUE is a SurrealQL modifier
	// and an unescaped leading `value` is parsed as that modifier. version is
	// projected as well as ordered by: 3.x rejects an ORDER BY over a column
	// the selection does not carry.
	query := fmt.Sprintf("SELECT `value`, version FROM kv_memories %s ORDER BY version DESC LIMIT 1",
		t.where("key = $key"))
	result, err := s.query(ctx, query, t.params(map[string]interface{}{"key": key}))
	if err != nil {
		return nil, fmt.Errorf("failed to get fact: %w", err)
	}

	if result == nil || len(*result) == 0 {
		return nil, nil
	}

	queryResult := (*result)[0]
	if queryResult.Status != "OK" {
		return nil, fmt.Errorf("query failed: %s", queryResult.Status)
	}

	if len(queryResult.Result) == 0 {
		return nil, nil
	}

	return queryResult.Result[0]["value"], nil
}

// UpdateFact replaces the value of an existing key. It differs from SaveFact
// only in refusing to create a fact that does not exist yet.
func (s *SurrealDBStorage) UpdateFact(ctx context.Context, userID, key string, value interface{}) error {
	existing, err := s.GetFact(ctx, userID, key)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("fact not found for key %s in the caller's scopes", key)
	}
	return s.SaveFact(ctx, userID, key, value)
}

// DeleteFact invalidates a fact. The row survives as a tombstone so the change
// feed can mirror the deletion; see migration v14 for why the uniqueness index
// tolerates it.
func (s *SurrealDBStorage) DeleteFact(ctx context.Context, userID, key string) error {
	t, err := s.scope(ctx, userID)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("UPDATE kv_memories SET %s %s", t.invalidateSet(), t.where("key = $key"))
	if _, err := s.query(ctx, query, t.params(map[string]interface{}{"key": key})); err != nil {
		return fmt.Errorf("failed to delete fact: %w", err)
	}

	if err := s.updateUserStat(ctx, t.Principal().LegacyUserID, "key_value_count", -1); err != nil {
		slog.Warn("failed to update key_value_count stat", "user_id", userID, "error", err)
	}

	return nil
}

// ListFacts retrieves every live fact visible to the caller.
func (s *SurrealDBStorage) ListFacts(ctx context.Context, userID string) (map[string]interface{}, error) {
	t, err := s.scope(ctx, userID)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT key, `value`, version FROM kv_memories %s ORDER BY version ASC", t.where())
	result, err := s.query(ctx, query, t.params(nil))
	if err != nil {
		return nil, fmt.Errorf("failed to list facts: %w", err)
	}

	facts := make(map[string]interface{})
	if result == nil || len(*result) == 0 {
		return facts, nil
	}

	queryResult := (*result)[0]
	if queryResult.Status != "OK" {
		return nil, fmt.Errorf("query failed: %s", queryResult.Status)
	}

	for _, item := range queryResult.Result {
		key, keyExists := item["key"]
		value, valueExists := item["value"]
		if !keyExists || !valueExists {
			continue
		}
		if keyStr, ok := key.(string); ok {
			facts[keyStr] = value
		}
	}

	return facts, nil
}
