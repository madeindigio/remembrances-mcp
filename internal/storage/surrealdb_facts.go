package storage

import (
	"context"
	"fmt"
	"log"
	"time"

)

// SaveFact saves a key-value fact for a user
func (s *SurrealDBStorage) SaveFact(ctx context.Context, userID, key string, value interface{}) error {
	existingID, err := s.findFactRecordID(ctx, userID, key)
	if err != nil {
		return fmt.Errorf("failed to check existing fact: %w", err)
	}

	if existingID != "" {
		// Fact exists, update value and timestamp
		updateQuery := "UPDATE " + existingID + " SET value = $value, updated_at = $updated_at"
		params := map[string]interface{}{
			"value":      value,
			"updated_at": time.Now().UTC(),
		}
		if _, err := s.query(ctx, updateQuery, params); err != nil {
			return fmt.Errorf("failed to update fact: %w", err)
		}
	} else {
		data := map[string]interface{}{
			"user_id": userID,
			"key":     key,
			"value":   value,
		}
		if _, err := s.create(ctx, "kv_memories", data); err != nil {
			return fmt.Errorf("failed to save fact: %w", err)
		}
	}

	// Recalculate statistics after mutation
	if err := s.updateUserStat(ctx, userID, "key_value_count", 1); err != nil {
		log.Printf("Warning: failed to update key_value_count stat for user %s: %v", userID, err)
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

	query := "UPDATE " + recordID + " SET value = $value, updated_at = $updated_at"
	params := map[string]interface{}{
		"value":      value,
		"updated_at": time.Now().UTC(),
	}
	if _, err := s.query(ctx, query, params); err != nil {
		return fmt.Errorf("failed to update fact: %w", err)
	}

	return nil
}

// DeleteFact deletes a key-value fact for a user
func (s *SurrealDBStorage) DeleteFact(ctx context.Context, userID, key string) error {
	params := map[string]interface{}{
		"user_id": userID,
		"key":     key,
	}

	deleteQuery := "DELETE FROM kv_memories WHERE user_id = $user_id AND key = $key RETURN BEFORE"
	result, err := s.query(ctx, deleteQuery, params)
	if err != nil {
		return fmt.Errorf("failed to delete fact: %w", err)
	}

	deletedCount := 0
	if result != nil && len(*result) > 0 {
		qr := (*result)[0]
		if qr.Status == "OK" {
			deletedCount = len(qr.Result)
		}
	}

	if deletedCount == 0 {
		return nil
	}

	if err := s.updateUserStat(ctx, userID, "key_value_count", -1); err != nil {
		log.Printf("Warning: failed to update key_value_count stat for user %s: %v", userID, err)
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
