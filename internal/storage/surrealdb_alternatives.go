package storage

import (
	"context"
	"fmt"
	"sort"
)

// ListUserIDs returns distinct user_ids for the given table.
func (s *SurrealDBStorage) ListUserIDs(ctx context.Context, table string) ([]string, error) {
	values := []string{}
	if table == "" {
		return values, fmt.Errorf("table name is required")
	}

	query := fmt.Sprintf("SELECT array::distinct(user_id) AS user_ids FROM %s WHERE user_id IS NOT NULL", table)
	result, err := s.query(ctx, query, nil)
	if err != nil {
		return values, fmt.Errorf("failed to list user ids for %s: %w", table, err)
	}

	values = extractStringArray(result, "user_ids")
	sort.Strings(values)
	return values, nil
}

// ListFactKeys returns distinct fact keys for a user.
func (s *SurrealDBStorage) ListFactKeys(ctx context.Context, userID string) ([]string, error) {
	query := "SELECT array::distinct(key) AS keys FROM kv_memories WHERE user_id = $user_id"
	result, err := s.query(ctx, query, map[string]interface{}{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("failed to list fact keys: %w", err)
	}

	keys := extractStringArray(result, "keys")
	sort.Strings(keys)
	return keys, nil
}

// ListEntityIDs returns distinct entity IDs.
func (s *SurrealDBStorage) ListEntityIDs(ctx context.Context) ([]string, error) {
	result, err := s.query(ctx, "SELECT array::distinct(id) AS ids FROM entities", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list entity ids: %w", err)
	}

	ids := extractStringArray(result, "ids")
	sort.Strings(ids)
	return ids, nil
}

// ListDocumentPaths returns distinct document file paths.
func (s *SurrealDBStorage) ListDocumentPaths(ctx context.Context) ([]string, error) {
	result, err := s.query(ctx, "SELECT array::distinct(file_path) AS paths FROM knowledge_base", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list document paths: %w", err)
	}

	paths := extractStringArray(result, "paths")
	sort.Strings(paths)
	return paths, nil
}

// extractStringArray pulls a []string from a query result field, normalizing any value type.
func extractStringArray(results *[]QueryResult, field string) []string {
	values := []string{}
	if results == nil || len(*results) == 0 {
		return values
	}

	qr := (*results)[0]
	if qr.Status != "OK" || qr.Result == nil || len(qr.Result) == 0 {
		return values
	}

	raw, ok := qr.Result[0][field]
	if !ok || raw == nil {
		return values
	}

	switch v := raw.(type) {
	case []interface{}:
		for _, item := range v {
			if item != nil {
				values = append(values, fmt.Sprint(item))
			}
		}
	case []string:
		values = append(values, v...)
	default:
		values = append(values, fmt.Sprint(v))
	}

	return values
}
