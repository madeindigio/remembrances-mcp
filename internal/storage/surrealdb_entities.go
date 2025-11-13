package storage

import (
	"context"
	"fmt"
	"log"
	"strings"

)

// CreateEntity creates a new entity in the graph
func (s *SurrealDBStorage) CreateEntity(ctx context.Context, entityType, name string, properties map[string]interface{}) error {
	if properties == nil {
		properties = map[string]interface{}{}
	}

	query := `
        INSERT INTO entities {
            type: $type,
            name: $name,
            properties: $properties
        } RETURN id
    `

	params := map[string]interface{}{
		"type":       entityType,
		"name":       name,
		"properties": properties,
	}

	result, err := s.query(ctx, query, params)
	if err != nil {
		return fmt.Errorf("failed to create entity: %w", err)
	}

	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" {
			if err := s.updateUserStat(ctx, "global", "entity_count", 1); err != nil {
				log.Printf("Warning: failed to update entity_count stat: %v", err)
			}
			return nil
		}
	}

	return fmt.Errorf("failed to create entity")
}

// resolveEntityID resolves an entity name to its SurrealDB record ID
func (s *SurrealDBStorage) resolveEntityID(ctx context.Context, entityNameOrID string) (string, error) {
	if strings.Contains(entityNameOrID, ":") {
		query := "SELECT * FROM " + entityNameOrID
		result, err := s.query(ctx, query, nil)
		if err == nil && result != nil && len(*result) > 0 {
			queryResult := (*result)[0]
			if queryResult.Status == "OK" && queryResult.Result != nil && len(queryResult.Result) > 0 {
				return entityNameOrID, nil
			}
		}
	}

	query := "SELECT * FROM entities WHERE name = $name"
	result, err := s.query(ctx, query, map[string]interface{}{"name": entityNameOrID})
	if err != nil {
		return "", fmt.Errorf("failed to query entity by name: %w", err)
	}

	if result == nil || len(*result) == 0 {
		return "", fmt.Errorf("entity not found: %s", entityNameOrID)
	}

	queryResult := (*result)[0]
	if queryResult.Status != "OK" || queryResult.Result == nil || len(queryResult.Result) == 0 {
		return "", fmt.Errorf("entity not found: %s", entityNameOrID)
	}

	resultMap := queryResult.Result[0]
	entityID := extractRecordID(resultMap["id"])
	if entityID == "" {
		return "", fmt.Errorf("entity ID not found for name: %s", entityNameOrID)
	}

	return entityID, nil
}

// CreateRelationship creates a relationship between two entities
func (s *SurrealDBStorage) CreateRelationship(ctx context.Context, fromEntity, toEntity, relationshipType string, properties map[string]interface{}) error {
	fromEntityID, err := s.resolveEntityID(ctx, fromEntity)
	if err != nil {
		return fmt.Errorf("failed to resolve from entity '%s': %w", fromEntity, err)
	}

	toEntityID, err := s.resolveEntityID(ctx, toEntity)
	if err != nil {
		return fmt.Errorf("failed to resolve to entity '%s': %w", toEntity, err)
	}

	tableName := relationshipType

	createTableQuery := fmt.Sprintf("DEFINE TABLE %s SCHEMALESS", tableName)
	_, err = s.query(ctx, createTableQuery, nil)
	if err != nil {
		// Table might already exist; SurrealDB returns an error we can ignore here.
	}

	query := fmt.Sprintf(`
        INSERT INTO %s {
            from_entity: $from,
            to_entity: $to,
            relationship_type: $relationshipType,
            properties: $properties
        }
    `, tableName)

	params := map[string]interface{}{
		"from":             fromEntityID,
		"to":               toEntityID,
		"relationshipType": relationshipType,
		"properties":       properties,
	}

	result, err := s.query(ctx, query, params)
	if err != nil {
		return fmt.Errorf("failed to create relationship: %w", err)
	}

	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" {
			if err := s.updateUserStat(ctx, "global", "relationship_count", 1); err != nil {
				log.Printf("Warning: failed to update relationship_count stat: %v", err)
			}
			return nil
		}
	}

	return fmt.Errorf("failed to create relationship")
}

// TraverseGraph traverses the graph starting from an entity
func (s *SurrealDBStorage) TraverseGraph(ctx context.Context, startEntity, relationshipType string, depth int) ([]GraphResult, error) {
	query := "SELECT id, name, type, properties FROM entities"

	params := map[string]interface{}{
		"start_entity": startEntity,
		"depth":        depth,
	}

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to traverse graph: %w", err)
	}

	return s.parseGraphResults(result)
}

// GetEntity retrieves an entity by ID or name
func (s *SurrealDBStorage) GetEntity(ctx context.Context, entityID string) (*Entity, error) {
	query := "SELECT * FROM " + entityID
	result, err := s.query(ctx, query, nil)
	if err != nil {
		query = "SELECT * FROM entities WHERE name = $name"
		result, err = s.query(ctx, query, map[string]interface{}{"name": entityID})
		if err != nil {
			return nil, fmt.Errorf("failed to get entity: %w", err)
		}
	}

	if result == nil || len(*result) == 0 {
		return nil, nil
	}

	queryResult := (*result)[0]
	if queryResult.Status != "OK" || queryResult.Result == nil || len(queryResult.Result) == 0 {
		return nil, nil
	}

	resultMap := queryResult.Result[0]
	entity := &Entity{
		ID:         getString(resultMap, "id"),
		Type:       getString(resultMap, "type"),
		Name:       getString(resultMap, "name"),
		Properties: getMap(resultMap, "properties"),
		CreatedAt:  getTime(resultMap, "created_at"),
		UpdatedAt:  getTime(resultMap, "updated_at"),
	}
	return entity, nil
}

// DeleteEntity deletes an entity and its relationships
func (s *SurrealDBStorage) DeleteEntity(ctx context.Context, entityID string) error {
	_, err := s.delete(ctx, entityID)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	if err := s.updateUserStat(ctx, "global", "entity_count", -1); err != nil {
		log.Printf("Warning: failed to update entity_count stat: %v", err)
	}

	return nil
}

func (s *SurrealDBStorage) parseGraphResults(result *[]QueryResult) ([]GraphResult, error) {
	var results []GraphResult

	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" && queryResult.Result != nil {
			resultSlice := queryResult.Result
			for _, itemMap := range resultSlice {
				entity := &Entity{
					ID:         getString(itemMap, "id"),
					Type:       getString(itemMap, "type"),
					Name:       getString(itemMap, "name"),
					Properties: getMap(itemMap, "properties"),
					CreatedAt:  getTime(itemMap, "created_at"),
					UpdatedAt:  getTime(itemMap, "updated_at"),
				}

				graphResult := GraphResult{
					Entity: entity,
					Path:   []string{entity.ID},
					Depth:  1,
				}
				results = append(results, graphResult)
			}
		}
	}

	return results, nil
}
