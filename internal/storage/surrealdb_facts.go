package storage

import (
	"context"
	"fmt"
	"log/slog"
)

// SaveFact saves a key-value fact for a user
func (s *SurrealDBStorage) SaveFact(ctx context.Context, userID, key string, value interface{}) error {
	existingID, err := s.findFactRecordID(ctx, userID, key)
	if err != nil {
		return fmt.Errorf("failed to check existing fact: %w", err)
	}

	if existingID != "" {
		// Fact exists - use DELETE FROM WHERE strategy to avoid response deserialization
		// DELETE FROM doesn't return the deleted records by default
		deleteQuery := `DELETE FROM kv_memories WHERE user_id = $user_id AND key = $key`
		params := map[string]interface{}{
			"user_id": userID,
			"key":     key,
		}
		if _, err := s.query(ctx, deleteQuery, params); err != nil {
			return fmt.Errorf("failed to delete existing fact: %w", err)
		}
	}

	// Create (or recreate) the fact using query syntax
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

	// Recalculate statistics after mutation
	if err := s.updateUserStat(ctx, userID, "key_value_count", 1); err != nil {
		slog.Warn("failed to update key_value_count stat", "user_id", userID, "error", err)
	}

	return nil
}

// GetFact retrieves a key-value fact for a user
func (s *SurrealDBStorage) GetFact(ctx context.Context, userID, key string) (interface{}, error) {
	query := "SELECT * FROM kv_memories WHERE user_id = $user_id AND key = $key LIMIT 1"
	params := map[string]interface{}{
		"user_id": userID,
		"key":     key,
	}
	result, err := s.query(ctx, query, params)
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

	if queryResult.Result == nil || len(queryResult.Result) == 0 {
		return nil, nil
	}

	factData := queryResult.Result[0]
	return factData["value"], nil
}

// UpdateFact updates a key-value fact for a user
func (s *SurrealDBStorage) UpdateFact(ctx context.Context, userID, key string, value interface{}) error {
	recordID, err := s.findFactRecordID(ctx, userID, key)
	if err != nil {
		return fmt.Errorf("failed to check existing fact: %w", err)
	}
	if recordID == "" {
		return fmt.Errorf("fact not found for user %s and key %s", userID, key)
	}

	// Use DELETE FROM WHERE + CREATE strategy to avoid response deserialization issues
	deleteQuery := `DELETE FROM kv_memories WHERE user_id = $user_id AND key = $key`
	params := map[string]interface{}{
		"user_id": userID,
		"key":     key,
	}
	if _, err := s.query(ctx, deleteQuery, params); err != nil {
		return fmt.Errorf("failed to delete existing fact: %w", err)
	}

	// Recreate with new value
	query := `
		CREATE kv_memories CONTENT {
			user_id: $user_id,
			key: $key,
			value: $value
		}
	`
	params = map[string]interface{}{
		"user_id": userID,
		"key":     key,
		"value":   value,
	}
	if _, err := s.query(ctx, query, params); err != nil {
		return fmt.Errorf("failed to recreate fact: %w", err)
	}

	return nil
}

// DeleteFact deletes a key-value fact for a user
func (s *SurrealDBStorage) DeleteFact(ctx context.Context, userID, key string) error {
	params := map[string]interface{}{
		"user_id": userID,
		"key":     key,
	}

	// Use DELETE FROM WHERE without RETURN to avoid deserialization issues
	deleteQuery := "DELETE FROM kv_memories WHERE user_id = $user_id AND key = $key"
	_, err := s.query(ctx, deleteQuery, params)
	if err != nil {
		return fmt.Errorf("failed to delete fact: %w", err)
	}

	if err := s.updateUserStat(ctx, userID, "key_value_count", -1); err != nil {
		slog.Warn("failed to update key_value_count stat", "user_id", userID, "error", err)
	}

	return nil
}

// ListFacts retrieves all key-value facts for a user
func (s *SurrealDBStorage) ListFacts(ctx context.Context, userID string) (map[string]interface{}, error) {
	query := "SELECT * FROM kv_memories WHERE user_id = $user_id"
	result, err := s.query(ctx, query, map[string]interface{}{
		"user_id": userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list facts: %w", err)
	}

	if result == nil || len(*result) == 0 {
		return map[string]interface{}{}, nil
	}

	queryResult := (*result)[0]
	if queryResult.Status != "OK" {
		return nil, fmt.Errorf("query failed: %s", queryResult.Status)
	}

	facts := make(map[string]interface{})
	if queryResult.Result != nil {
		for _, item := range queryResult.Result {
			key, keyExists := item["key"]
			value, valueExists := item["value"]
			if keyExists && valueExists {
				if keyStr, ok := key.(string); ok {
					facts[keyStr] = value
				}
			}
		}
	}

	return facts, nil
}

// findFactRecordID returns the SurrealDB record ID for a fact or an empty string if it does not exist.
func (s *SurrealDBStorage) findFactRecordID(ctx context.Context, userID, key string) (string, error) {
	query := "SELECT id FROM kv_memories WHERE user_id = $user_id AND key = $key LIMIT 1"
	params := map[string]interface{}{
		"user_id": userID,
		"key":     key,
	}

	result, err := s.query(ctx, query, params)
	if err != nil {
		return "", err
	}
	if result == nil || len(*result) == 0 {
		return "", nil
	}

	queryResult := (*result)[0]
	if queryResult.Status != "OK" || queryResult.Result == nil || len(queryResult.Result) == 0 {
		return "", nil
	}

	return extractRecordID(queryResult.Result[0]["id"]), nil
}
