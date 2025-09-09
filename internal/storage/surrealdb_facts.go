package storage

import (
	"context"
	"fmt"

	"github.com/surrealdb/surrealdb.go"
)

// SaveFact saves a key-value fact for a user
func (s *SurrealDBStorage) SaveFact(ctx context.Context, userID, key string, value interface{}) error {
	data := map[string]interface{}{
		"user_id": userID,
		"key":     key,
		"value":   value,
	}

	recordID := fmt.Sprintf("kv_memories:['%s:%s']", userID, key)
	_, err := surrealdb.Create[map[string]interface{}](s.db, recordID, data)
	if err != nil {
		return fmt.Errorf("failed to save fact: %w", err)
	}

	return nil
}

// GetFact retrieves a key-value fact for a user
func (s *SurrealDBStorage) GetFact(ctx context.Context, userID, key string) (interface{}, error) {
	recordID := fmt.Sprintf("kv_memories:['%s:%s']", userID, key)
	query := "SELECT * FROM " + recordID
	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, nil)
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
	data := map[string]interface{}{
		"value": value,
	}

	recordID := fmt.Sprintf("kv_memories:['%s:%s']", userID, key)
	_, err := surrealdb.Update[map[string]interface{}](s.db, recordID, data)
	if err != nil {
		return fmt.Errorf("failed to update fact: %w", err)
	}

	return nil
}

// DeleteFact deletes a key-value fact for a user
func (s *SurrealDBStorage) DeleteFact(ctx context.Context, userID, key string) error {
	recordID := fmt.Sprintf("kv_memories:['%s:%s']", userID, key)
	_, err := surrealdb.Delete[interface{}](s.db, recordID)
	if err != nil {
		return fmt.Errorf("failed to delete fact: %w", err)
	}

	return nil
}

// ListFacts retrieves all key-value facts for a user
func (s *SurrealDBStorage) ListFacts(ctx context.Context, userID string) (map[string]interface{}, error) {
	query := "SELECT * FROM kv_memories WHERE user_id = $user_id"
	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, map[string]interface{}{
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
