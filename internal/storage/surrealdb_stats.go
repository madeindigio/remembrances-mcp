package storage

import (
	"context"
	"fmt"
	"strconv"
)

// GetStats returns statistics about stored memories by counting current records.
func (s *SurrealDBStorage) GetStats(ctx context.Context, userID string) (*MemoryStats, error) {
	stats := &MemoryStats{}
	scoped := userID != "" && userID != "global"

	var params map[string]interface{}
	if scoped {
		params = map[string]interface{}{"user_id": userID}
	}

	if scoped {
		stats.KeyValueCount = s.getCount(ctx, "SELECT count() AS count FROM kv_memories WHERE user_id = $user_id", params)
	} else {
		stats.KeyValueCount = s.getCount(ctx, "SELECT count() AS count FROM kv_memories", nil)
	}

	if scoped {
		stats.VectorCount = s.getCount(ctx, "SELECT count() AS count FROM vector_memories WHERE user_id = $user_id", params)
	} else {
		stats.VectorCount = s.getCount(ctx, "SELECT count() AS count FROM vector_memories", nil)
	}

	stats.EntityCount = s.getCount(ctx, "SELECT count() AS count FROM entities", nil)

	relTables, _ := s.getRelationshipTables(ctx)
	relationshipCount := 0
	for _, tbl := range relTables {
		count := s.getCount(ctx, "SELECT count() AS count FROM "+tbl, nil)
		relationshipCount += count
	}
	stats.RelationshipCount = relationshipCount

	stats.DocumentCount = s.getCount(ctx, "SELECT count() AS count FROM knowledge_base", nil)

	var totalSize int64
	if scoped {
		q := "SELECT content FROM vector_memories WHERE user_id = $user_id"
		res, _ := s.query(ctx, q, params)
		if res != nil && len(*res) > 0 {
			qr := (*res)[0]
			if qr.Status == "OK" && len(qr.Result) > 0 {
				for _, row := range qr.Result {
					if c, ok := row["content"].(string); ok {
						totalSize += int64(len(c))
					}
				}
			}
		}

		q = "SELECT value FROM kv_memories WHERE user_id = $user_id"
		res, _ = s.query(ctx, q, params)
		if res != nil && len(*res) > 0 {
			qr := (*res)[0]
			if qr.Status == "OK" && len(qr.Result) > 0 {
				for _, row := range qr.Result {
					if v, ok := row["value"].(string); ok {
						totalSize += int64(len(v))
					}
				}
			}
		}

		q = "SELECT content FROM knowledge_base WHERE user_id = $user_id"
		res, _ = s.query(ctx, q, params)
		if res != nil && len(*res) > 0 {
			qr := (*res)[0]
			if qr.Status == "OK" && len(qr.Result) > 0 {
				for _, row := range qr.Result {
					if c, ok := row["content"].(string); ok {
						totalSize += int64(len(c))
					}
				}
			}
		}
	} else {
		q := "SELECT name FROM entities"
		res, _ := s.query(ctx, q, nil)
		if res != nil && len(*res) > 0 {
			qr := (*res)[0]
			if qr.Status == "OK" && len(qr.Result) > 0 {
				for _, row := range qr.Result {
					if n, ok := row["name"].(string); ok {
						totalSize += int64(len(n))
					}
				}
			}
		}

		for _, tbl := range relTables {
			q = "SELECT relationship_type FROM " + tbl
			res, _ = s.query(ctx, q, nil)
			if res != nil && len(*res) > 0 {
				qr := (*res)[0]
				if qr.Status == "OK" && len(qr.Result) > 0 {
					for _, row := range qr.Result {
						if t, ok := row["relationship_type"].(string); ok {
							totalSize += int64(len(t))
						}
					}
				}
			}
		}

		q = "SELECT content FROM knowledge_base"
		res, _ = s.query(ctx, q, nil)
		if res != nil && len(*res) > 0 {
			qr := (*res)[0]
			if qr.Status == "OK" && len(qr.Result) > 0 {
				for _, row := range qr.Result {
					if c, ok := row["content"].(string); ok {
						totalSize += int64(len(c))
					}
				}
			}
		}
	}

	stats.TotalSize = totalSize
	return stats, nil
}

// updateUserStat atomically updates a specific statistic for a user.
func (s *SurrealDBStorage) updateUserStat(ctx context.Context, userID, statField string, delta int) error {
	var countQuery string
	var params map[string]interface{}
	var newValue int
	switch statField {
	case "vector_count":
		countQuery = "SELECT count() AS count FROM vector_memories WHERE user_id = $user_id"
		params = map[string]interface{}{"user_id": userID}
		newValue = s.getCount(ctx, countQuery, params)
	case "entity_count":
		countQuery = "SELECT count() AS count FROM entities"
		params = map[string]interface{}{}
		newValue = s.getCount(ctx, countQuery, params)
	case "relationship_count":
		relTables, _ := s.getRelationshipTables(ctx)
		for _, tbl := range relTables {
			q := "SELECT count() AS count FROM " + tbl
			newValue += s.getCount(ctx, q, map[string]interface{}{})
		}
	case "document_count":
		countQuery = "SELECT count() AS count FROM knowledge_base"
		params = map[string]interface{}{}
		newValue = s.getCount(ctx, countQuery, params)
	case "key_value_count":
		countQuery = "SELECT count() AS count FROM kv_memories WHERE user_id = $user_id"
		params = map[string]interface{}{"user_id": userID}
		newValue = s.getCount(ctx, countQuery, params)
	default:
		return fmt.Errorf("invalid stat field: %s", statField)
	}

	upsertQuery := "UPDATE user_stats SET " + statField + " = $new_value, updated_at = time::now() WHERE user_id = $user_id;"
	upsertParams := map[string]interface{}{
		"user_id":   userID,
		"new_value": newValue,
	}
	if _, err := s.query(ctx, upsertQuery, upsertParams); err != nil {
		createData := map[string]interface{}{
			"user_id":            userID,
			"key_value_count":    0,
			"vector_count":       0,
			"entity_count":       0,
			"relationship_count": 0,
			"document_count":     0,
		}
		createData[statField] = newValue
		if _, err := s.create(ctx, "user_stats", createData); err != nil {
			return fmt.Errorf("failed to create user stat %s for user %s: %w", statField, userID, err)
		}
	}
	return nil
}

// getCount is a helper to run a count query and extract the count
func (s *SurrealDBStorage) getCount(ctx context.Context, query string, params map[string]interface{}) int {
	countResult, err := s.query(ctx, query, params)
	if err != nil || countResult == nil || len(*countResult) == 0 {
		return 0
	}
	queryResult := (*countResult)[0]
	if queryResult.Status == "OK" && len(queryResult.Result) > 0 {
		total := 0
		for _, row := range queryResult.Result {
			val, ok := row["count"]
			if !ok {
				continue
			}
			switch v := val.(type) {
			case float64:
				total += int(v)
			case float32:
				total += int(v)
			case int:
				total += v
			case int64:
				total += int(v)
			case uint64:
				total += int(v)
			case string:
				if parsed, err := strconv.Atoi(v); err == nil {
					total += parsed
				}
			default:
				if parsed, err := strconv.Atoi(fmt.Sprint(v)); err == nil {
					total += parsed
				}
			}
		}
		return total
	}
	return 0
}

// getRelationshipTables returns all relationship tables (excluding system tables)
func (s *SurrealDBStorage) getRelationshipTables(ctx context.Context) ([]string, error) {
	tables := []string{}
	result, err := s.query(ctx, "SHOW TABLES", nil)
	if err != nil || result == nil || len(*result) == 0 {
		return tables, nil
	}
	queryResult := (*result)[0]
	if queryResult.Status == "OK" && len(queryResult.Result) > 0 {
		for _, row := range queryResult.Result {
			if tbl, ok := row["name"].(string); ok {
				if tbl != "entities" && tbl != "vector_memories" && tbl != "kv_memories" && tbl != "knowledge_base" && tbl != "user_stats" && tbl != "schema_version" {
					tables = append(tables, tbl)
				}
			}
		}
	}
	return tables, nil
}
