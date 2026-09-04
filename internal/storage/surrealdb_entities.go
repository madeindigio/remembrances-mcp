package storage

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/madeindigio/remembrances-mcp/internal/storage/migrations"
)

// The graph surface, scoped per RM-3.
//
// Nothing here had any notion of ownership: GetEntity resolved a name across
// the whole datastore, ListEntityIDs returned every id in it, and TraverseGraph
// did not traverse at all, it selected every row of `entities` and called the
// result a traversal. Each method now runs inside the caller's visible scopes,
// and the traversal is a real bounded walk whose every hop carries the same
// predicate, so an edge cannot be used to step out of a scope.

// defaultTraversalDepth bounds a traversal whose caller asked for none.
const defaultTraversalDepth = 2

// maxTraversalDepth bounds every traversal. A caller-supplied depth is clamped
// to it, because the walk issues one query per hop per edge table.
const maxTraversalDepth = 5

// CreateEntity creates a new entity in the caller's write scope.
func (s *SurrealDBStorage) CreateEntity(ctx context.Context, entityType, name string, properties map[string]interface{}) error {
	t, err := s.scope(ctx, "")
	if err != nil {
		return err
	}
	if properties == nil {
		properties = map[string]interface{}{}
	}

	query := fmt.Sprintf(`
		CREATE entities CONTENT {
			%suser_id: $user_id,
			entity_type: $entity_type,
			name: $name,
			properties: $properties
		}
	`, t.contentFields())

	params := t.params(map[string]interface{}{
		"entity_type": entityType,
		"name":        name,
		"properties":  properties,
		"user_id":     t.Principal().LegacyUserID,
	})

	if _, err := s.query(ctx, query, params); err != nil {
		return fmt.Errorf("failed to create entity: %w", err)
	}

	if err := s.updateUserStat(ctx, t.Principal().LegacyUserID, "entity_count", 1); err != nil {
		slog.Warn("failed to update entity_count stat", "error", err)
	}
	return nil
}

// resolveEntityID resolves an entity name or record id to a record id that
// lives in one of the caller's visible scopes. An entity the caller cannot see
// is reported as not found rather than as a permission error: telling an
// unrelated tenant that a name exists is itself a leak.
func (s *SurrealDBStorage) resolveEntityID(ctx context.Context, t *tenancy, entityNameOrID string) (string, error) {
	condition := "name = $entity"
	if strings.Contains(entityNameOrID, ":") {
		condition = "(id = type::record($entity) OR name = $entity)"
	}

	query := fmt.Sprintf("SELECT id FROM entities %s LIMIT 1", t.where(condition))
	result, err := s.query(ctx, query, t.params(map[string]interface{}{"entity": entityNameOrID}))
	if err != nil {
		return "", fmt.Errorf("failed to query entity %q: %w", entityNameOrID, err)
	}
	if result == nil || len(*result) == 0 {
		return "", fmt.Errorf("entity not found: %s", entityNameOrID)
	}

	queryResult := (*result)[0]
	if queryResult.Status != "OK" || len(queryResult.Result) == 0 {
		return "", fmt.Errorf("entity not found: %s", entityNameOrID)
	}

	entityID := extractRecordID(queryResult.Result[0]["id"])
	if entityID == "" {
		return "", fmt.Errorf("entity ID not found for: %s", entityNameOrID)
	}
	return entityID, nil
}

// CreateRelationship creates an edge between two entities the caller can see.
// The edge is stamped with the caller's org and write scope, so a later
// traversal filters it exactly like a node.
func (s *SurrealDBStorage) CreateRelationship(ctx context.Context, fromEntity, toEntity, relationshipType string, properties map[string]interface{}) error {
	t, err := s.scope(ctx, "")
	if err != nil {
		return err
	}

	table, err := edgeTableName(relationshipType)
	if err != nil {
		return err
	}

	fromEntityID, err := s.resolveEntityID(ctx, t, fromEntity)
	if err != nil {
		return fmt.Errorf("failed to resolve from entity '%s': %w", fromEntity, err)
	}
	toEntityID, err := s.resolveEntityID(ctx, t, toEntity)
	if err != nil {
		return fmt.Errorf("failed to resolve to entity '%s': %w", toEntity, err)
	}

	if properties == nil {
		properties = map[string]interface{}{}
	}

	// The edge tables outside the canonical three are created on demand. The
	// name has already been validated as a plain identifier.
	if _, err := s.query(ctx, fmt.Sprintf("DEFINE TABLE IF NOT EXISTS %s SCHEMALESS", table), nil); err != nil {
		slog.Debug("edge table definition returned an error, continuing", "table", table, "error", err)
	}

	// in and out are the canonical edge columns the v1 schema declares as
	// required records; from_entity and to_entity mirror them because `in` and
	// `out` are SurrealQL keywords and reading them back in a WHERE clause
	// would need escaping at every call site.
	query := fmt.Sprintf(`
		CREATE %s CONTENT {
			%suser_id: $user_id,
			in: type::record($from),
			out: type::record($to),
			from_entity: type::record($from),
			to_entity: type::record($to),
			relationship_type: $relationship_type,
			properties: $properties
		}
	`, table, t.contentFields())

	params := t.params(map[string]interface{}{
		"from":              fromEntityID,
		"to":                toEntityID,
		"relationship_type": relationshipType,
		"properties":        properties,
		"user_id":           t.Principal().LegacyUserID,
	})

	if _, err := s.query(ctx, query, params); err != nil {
		return fmt.Errorf("failed to create relationship: %w", err)
	}

	if err := s.updateUserStat(ctx, t.Principal().LegacyUserID, "relationship_count", 1); err != nil {
		slog.Warn("failed to update relationship_count stat", "error", err)
	}
	return nil
}

// edgeTableName validates a relationship type as a table name. The type comes
// from a tool argument and is interpolated into SurrealQL, so anything that is
// not a plain identifier is refused rather than escaped.
func edgeTableName(relationshipType string) (string, error) {
	name := strings.TrimSpace(relationshipType)
	if name == "" {
		return "", fmt.Errorf("relationship type is required")
	}
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return "", fmt.Errorf("invalid relationship type %q: expected letters, digits and underscore", relationshipType)
		}
	}
	return name, nil
}

// TraverseGraph walks the graph from an entity, staying inside the caller's
// visible scopes.
//
// Both halves of a hop are filtered: the edge row must be visible, and so must
// the entity it points at. An edge that leaves the caller's scopes therefore
// terminates the walk instead of widening it, which is the property RM-3 calls
// "refinement never crosses scopes".
//
// An empty relationshipType walks the three canonical edge tables.
func (s *SurrealDBStorage) TraverseGraph(ctx context.Context, startEntity, relationshipType string, depth int) ([]GraphResult, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}

	if depth <= 0 {
		depth = defaultTraversalDepth
	}
	if depth > maxTraversalDepth {
		depth = maxTraversalDepth
	}

	tables := migrations.EdgeTables
	if relationshipType != "" {
		table, err := edgeTableName(relationshipType)
		if err != nil {
			return nil, err
		}
		tables = []string{table}
	}

	startID, err := s.resolveEntityID(ctx, t, startEntity)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve start entity '%s': %w", startEntity, err)
	}

	start, err := s.entityByID(ctx, t, startID)
	if err != nil {
		return nil, err
	}
	if start == nil {
		return nil, fmt.Errorf("entity not found: %s", startEntity)
	}

	visited := map[string]struct{}{startID: {}}
	frontier := []frontierNode{{entity: start, path: []string{startID}}}
	var results []GraphResult

	for level := 1; level <= depth && len(frontier) > 0; level++ {
		var next []frontierNode
		for _, node := range frontier {
			for _, table := range tables {
				edges, err := s.outgoingEdges(ctx, t, table, node.entity.ID)
				if err != nil {
					return nil, err
				}
				for _, edge := range edges {
					if _, seen := visited[edge.To]; seen {
						continue
					}
					// The target must itself be visible. This is the step that
					// stops a cross-scope edge.
					target, err := s.entityByID(ctx, t, edge.To)
					if err != nil {
						return nil, err
					}
					if target == nil {
						continue
					}
					visited[edge.To] = struct{}{}

					path := append(append([]string(nil), node.path...), target.ID)
					rel := edge
					results = append(results, GraphResult{
						Entity:       target,
						Relationship: &rel,
						Path:         path,
						Depth:        level,
					})
					next = append(next, frontierNode{entity: target, path: path})
				}
			}
		}
		frontier = next
	}

	return results, nil
}

// frontierNode is one entry of the breadth-first frontier.
type frontierNode struct {
	entity *Entity
	path   []string
}

// outgoingEdges returns the visible edges leaving an entity in one edge table.
// A table that does not exist yet is not an error: relationship tables are
// created on first write.
func (s *SurrealDBStorage) outgoingEdges(ctx context.Context, t *tenancy, table, fromID string) ([]Relationship, error) {
	query := fmt.Sprintf(
		"SELECT id, from_entity, to_entity, relationship_type, properties, user_id, timestamp FROM %s %s",
		table, t.where("from_entity = type::record($from)"))

	result, err := s.query(ctx, query, t.params(map[string]interface{}{"from": fromID}))
	if err != nil {
		if isMissingTable(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read the %s edges of %s: %w", table, fromID, err)
	}
	if result == nil || len(*result) == 0 {
		return nil, nil
	}
	queryResult := (*result)[0]
	if queryResult.Status != "OK" {
		return nil, nil
	}

	edges := make([]Relationship, 0, len(queryResult.Result))
	for _, row := range queryResult.Result {
		rel := Relationship{
			ID:         getString(row, "id"),
			From:       extractRecordID(row["from_entity"]),
			To:         extractRecordID(row["to_entity"]),
			Type:       getString(row, "relationship_type"),
			Properties: getMap(row, "properties"),
			Timestamp:  getTime(row, "timestamp"),
		}
		if uid := getString(row, "user_id"); uid != "" {
			rel.UserID = &uid
		}
		if rel.To == "" {
			continue
		}
		edges = append(edges, rel)
	}
	return edges, nil
}

// isMissingTable reports whether the error is the engine complaining about a
// table that was never defined.
func isMissingTable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "does not exist") || strings.Contains(msg, "not found")
}

// entityByID loads one visible entity, or nil when it is outside the caller's
// scopes or does not exist.
func (s *SurrealDBStorage) entityByID(ctx context.Context, t *tenancy, entityID string) (*Entity, error) {
	query := fmt.Sprintf("SELECT * FROM entities %s LIMIT 1", t.where("id = type::record($id)"))
	result, err := s.query(ctx, query, t.params(map[string]interface{}{"id": entityID}))
	if err != nil {
		return nil, fmt.Errorf("failed to get entity %s: %w", entityID, err)
	}
	if result == nil || len(*result) == 0 {
		return nil, nil
	}
	queryResult := (*result)[0]
	if queryResult.Status != "OK" || len(queryResult.Result) == 0 {
		return nil, nil
	}
	return entityFromRow(queryResult.Result[0]), nil
}

// GetEntity retrieves a visible entity by record id or by name.
func (s *SurrealDBStorage) GetEntity(ctx context.Context, entityID string) (*Entity, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}

	condition := "name = $entity"
	if strings.Contains(entityID, ":") {
		condition = "(id = type::record($entity) OR name = $entity)"
	}

	query := fmt.Sprintf("SELECT * FROM entities %s LIMIT 1", t.where(condition))
	result, err := s.query(ctx, query, t.params(map[string]interface{}{"entity": entityID}))
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}
	if result == nil || len(*result) == 0 {
		return nil, nil
	}
	queryResult := (*result)[0]
	if queryResult.Status != "OK" || len(queryResult.Result) == 0 {
		return nil, nil
	}
	return entityFromRow(queryResult.Result[0]), nil
}

// DeleteEntity invalidates an entity the caller can see.
func (s *SurrealDBStorage) DeleteEntity(ctx context.Context, entityID string) error {
	t, err := s.scope(ctx, "")
	if err != nil {
		return err
	}

	query := fmt.Sprintf("UPDATE entities SET %s %s", t.invalidateSet(),
		t.where("(id = type::record($entity) OR name = $entity)"))
	if _, err := s.query(ctx, query, t.params(map[string]interface{}{"entity": entityID})); err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	if err := s.updateUserStat(ctx, t.Principal().LegacyUserID, "entity_count", -1); err != nil {
		slog.Warn("failed to update entity_count stat", "error", err)
	}
	return nil
}

// entityFromRow decodes one entities row. entity_type is the canonical column:
// it is what the write path sends. The remote schema used to declare a `type`
// column that no write ever populated, so every read of it returned an empty
// string.
func entityFromRow(row map[string]interface{}) *Entity {
	e := &Entity{
		ID:         getString(row, "id"),
		Type:       getString(row, "entity_type"),
		Name:       getString(row, "name"),
		Properties: getMap(row, "properties"),
		CreatedAt:  getTime(row, "created_at"),
		UpdatedAt:  getTime(row, "updated_at"),
	}
	if uid := getString(row, "user_id"); uid != "" {
		e.UserID = &uid
	}
	return e
}
