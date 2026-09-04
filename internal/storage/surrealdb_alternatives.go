package storage

import (
	"context"
	"fmt"
	"sort"

	"github.com/madeindigio/remembrances-mcp/internal/storage/migrations"
)

// ListUserIDs returns the distinct legacy user_id labels visible to the caller
// in the given table.
//
// It is a legacy listing: user_id is the pre-v13 ownership label, and the
// answer is now bounded by the caller's org and scopes, so it can no longer be
// used to enumerate the tenants of a shared datastore.
func (s *SurrealDBStorage) ListUserIDs(ctx context.Context, table string) ([]string, error) {
	values := []string{}
	t, err := s.scope(ctx, "")
	if err != nil {
		return values, err
	}
	if !isMemoryTable(table) {
		return values, fmt.Errorf("unknown memory table %q", table)
	}

	query := fmt.Sprintf("SELECT array::distinct(user_id) AS user_ids FROM %s %s",
		table, t.where("user_id != NONE"))
	result, err := s.query(ctx, query, t.params(nil))
	if err != nil {
		return values, fmt.Errorf("failed to list user ids for %s: %w", table, err)
	}

	values = extractStringArray(result, "user_ids")
	sort.Strings(values)
	return values, nil
}

// ListFactKeys returns the distinct keys of the live facts visible to the
// caller.
func (s *SurrealDBStorage) ListFactKeys(ctx context.Context, userID string) ([]string, error) {
	t, err := s.scope(ctx, userID)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT array::distinct(key) AS keys FROM kv_memories %s", t.where())
	result, err := s.query(ctx, query, t.params(nil))
	if err != nil {
		return nil, fmt.Errorf("failed to list fact keys: %w", err)
	}

	keys := extractStringArray(result, "keys")
	sort.Strings(keys)
	return keys, nil
}

// ListEntityIDs returns the ids of the live entities visible to the caller.
func (s *SurrealDBStorage) ListEntityIDs(ctx context.Context) ([]string, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}

	result, err := s.query(ctx,
		fmt.Sprintf("SELECT array::distinct(id) AS ids FROM entities %s", t.where()), t.params(nil))
	if err != nil {
		return nil, fmt.Errorf("failed to list entity ids: %w", err)
	}

	ids := extractStringArray(result, "ids")
	sort.Strings(ids)
	return ids, nil
}

// ListDocumentPaths returns the paths of the live documents visible to the
// caller.
func (s *SurrealDBStorage) ListDocumentPaths(ctx context.Context) ([]string, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}

	result, err := s.query(ctx,
		fmt.Sprintf("SELECT array::distinct(file_path) AS paths FROM knowledge_base %s", t.where()), t.params(nil))
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

// isMemoryTable reports whether the name is one of the tables that carry the
// v13 tenant columns. Table names reach SurrealQL by interpolation, so the
// allowed set is a list rather than an escape.
func isMemoryTable(table string) bool {
	for _, t := range migrations.MemoryTables {
		if t == table {
			return true
		}
	}
	return false
}
