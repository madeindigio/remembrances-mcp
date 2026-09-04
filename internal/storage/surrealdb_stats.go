package storage

import (
	"context"
	"fmt"
	"strconv"

	"github.com/madeindigio/remembrances-mcp/internal/storage/migrations"
)

// Statistics, scoped per RM-2.
//
// Counts are as leaky as rows: "this datastore holds 4,812 documents" tells a
// tenant about every other tenant. Every count here therefore runs behind the
// same scope predicate as the query that returns the rows themselves, and the
// legacy "global" user id no longer widens anything.

// GetStats returns statistics about the memories visible to the caller.
func (s *SurrealDBStorage) GetStats(ctx context.Context, userID string) (*MemoryStats, error) {
	t, err := s.scope(ctx, userID)
	if err != nil {
		return nil, err
	}

	params := t.params(nil)
	where := t.where()
	stats := &MemoryStats{}

	stats.KeyValueCount = s.getCount(ctx, fmt.Sprintf("SELECT count() AS count FROM kv_memories %s GROUP ALL", where), params)
	stats.VectorCount = s.getCount(ctx, fmt.Sprintf("SELECT count() AS count FROM vector_memories %s GROUP ALL", where), params)
	stats.EntityCount = s.getCount(ctx, fmt.Sprintf("SELECT count() AS count FROM entities %s GROUP ALL", where), params)
	stats.EventCount = s.getCount(ctx, fmt.Sprintf("SELECT count() AS count FROM events %s GROUP ALL", where), params)

	relationshipCount := 0
	for _, tbl := range migrations.EdgeTables {
		relationshipCount += s.getCount(ctx, fmt.Sprintf("SELECT count() AS count FROM %s %s GROUP ALL", tbl, where), params)
	}
	stats.RelationshipCount = relationshipCount

	// Count distinct documents, not individual chunks.
	stats.DocumentCount = s.getDistinctDocumentCount(ctx, t)

	stats.TotalSize = s.contentBytes(ctx, t)
	return stats, nil
}

// contentBytes sums the stored text of the caller's visible rows. It is an
// approximation of the tenant's footprint, not a storage engine measurement.
func (s *SurrealDBStorage) contentBytes(ctx context.Context, t *tenancy) int64 {
	var total int64
	params := t.params(nil)
	where := t.where()

	for _, q := range []struct{ query, field string }{
		{fmt.Sprintf("SELECT content FROM vector_memories %s", where), "content"},
		{fmt.Sprintf("SELECT `value` FROM kv_memories %s", where), "value"},
		{fmt.Sprintf("SELECT content FROM knowledge_base %s", where), "content"},
		{fmt.Sprintf("SELECT content FROM events %s", where), "content"},
	} {
		res, err := s.query(ctx, q.query, params)
		if err != nil || res == nil || len(*res) == 0 {
			continue
		}
		qr := (*res)[0]
		if qr.Status != "OK" {
			continue
		}
		for _, row := range qr.Result {
			if v, ok := row[q.field].(string); ok {
				total += int64(len(v))
			}
		}
	}
	return total
}

// updateUserStat recomputes one statistic for a legacy user label and stores
// it in user_stats.
//
// user_stats is a legacy denormalised cache keyed by the pre-v13 user_id
// label; it holds no memory content, so it carries no tenant columns. The
// counts written into it are computed behind the caller's scope predicate, so
// a tenant can only ever refresh a number derived from rows it can already
// read.
func (s *SurrealDBStorage) updateUserStat(ctx context.Context, userID, statField string, delta int) error {
	t, err := s.scope(ctx, userID)
	if err != nil {
		return err
	}

	params := t.params(nil)
	where := t.where()

	var newValue int
	switch statField {
	case "vector_count":
		newValue = s.getCount(ctx, fmt.Sprintf("SELECT count() AS count FROM vector_memories %s GROUP ALL", where), params)
	case "entity_count":
		newValue = s.getCount(ctx, fmt.Sprintf("SELECT count() AS count FROM entities %s GROUP ALL", where), params)
	case "relationship_count":
		for _, tbl := range migrations.EdgeTables {
			newValue += s.getCount(ctx, fmt.Sprintf("SELECT count() AS count FROM %s %s GROUP ALL", tbl, where), params)
		}
	case "document_count":
		// Count distinct documents, not individual chunks
		newValue = s.getDistinctDocumentCount(ctx, t)
	case "key_value_count":
		newValue = s.getCount(ctx, fmt.Sprintf("SELECT count() AS count FROM kv_memories %s GROUP ALL", where), params)
	default:
		return fmt.Errorf("invalid stat field: %s", statField)
	}

	label := userID
	if label == "" {
		label = t.Principal().LegacyUserID
	}

	upsertQuery := "UPDATE user_stats SET " + statField + " = $new_value, updated_at = time::now() WHERE user_id = $user_id;"
	upsertParams := map[string]interface{}{
		"user_id":   label,
		"new_value": newValue,
	}
	if _, err := s.query(ctx, upsertQuery, upsertParams); err != nil {
		createData := map[string]interface{}{
			"user_id":            label,
			"key_value_count":    0,
			"vector_count":       0,
			"entity_count":       0,
			"relationship_count": 0,
			"document_count":     0,
		}
		createData[statField] = newValue
		if _, err := s.create(ctx, "user_stats", createData); err != nil {
			return fmt.Errorf("failed to create user stat %s for user %s: %w", statField, label, err)
		}
	}
	return nil
}

// getDistinctDocumentCount returns the number of unique documents visible to
// the caller. Chunked documents share the same source_file value while
// non-chunked documents only have file_path set, so the coalesce operator (??)
// picks the canonical identifier before the grouping.
func (s *SurrealDBStorage) getDistinctDocumentCount(ctx context.Context, t *tenancy) int {
	query := fmt.Sprintf(
		"SELECT count() AS count FROM (SELECT (source_file ?? file_path) AS doc_id FROM knowledge_base %s GROUP BY doc_id) GROUP ALL",
		t.where())
	return s.getCount(ctx, query, t.params(nil))
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

// CountByUserID returns counts grouped by the legacy user_id label for a
// table, restricted to the caller's visible scopes. It is used by MCP tools to
// suggest alternative user_ids when a query returns zero results, which is
// exactly the kind of hint that must not reach across a tenant boundary.
func (s *SurrealDBStorage) CountByUserID(ctx context.Context, tableName string) (map[string]int, error) {
	counts := make(map[string]int)

	t, err := s.scope(ctx, "")
	if err != nil {
		return counts, err
	}
	if !isMemoryTable(tableName) {
		return counts, fmt.Errorf("unknown memory table %q", tableName)
	}

	query := fmt.Sprintf("SELECT user_id, count() AS count FROM %s %s GROUP BY user_id ORDER BY count DESC",
		tableName, t.where("user_id != NONE"))
	result, err := s.query(ctx, query, t.params(nil))
	if err != nil {
		return counts, fmt.Errorf("failed to count user_ids for table %s: %w", tableName, err)
	}

	if result == nil || len(*result) == 0 {
		return counts, nil
	}

	qr := (*result)[0]
	if qr.Status != "OK" || len(qr.Result) == 0 {
		return counts, nil
	}

	for _, row := range qr.Result {
		uidVal, ok := row["user_id"]
		if !ok || uidVal == nil {
			continue
		}
		uid := fmt.Sprint(uidVal)
		cntVal, ok := row["count"]
		if !ok {
			continue
		}
		switch v := cntVal.(type) {
		case float64:
			counts[uid] = int(v)
		case float32:
			counts[uid] = int(v)
		case int:
			counts[uid] = v
		case int64:
			counts[uid] = int(v)
		case uint64:
			counts[uid] = int(v)
		case string:
			if parsed, err := strconv.Atoi(v); err == nil {
				counts[uid] = parsed
			}
		default:
			if parsed, err := strconv.Atoi(fmt.Sprint(v)); err == nil {
				counts[uid] = parsed
			}
		}
	}

	return counts, nil
}
