package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/surrealdb/surrealdb.go"
)

// SurrealDBStorage implements the Storage interface using SurrealDB
type SurrealDBStorage struct {
	db     *surrealdb.DB
	config *ConnectionConfig
}

// NewSurrealDBStorage creates a new SurrealDB storage instance
func NewSurrealDBStorage(config *ConnectionConfig) *SurrealDBStorage {
	if config.Namespace == "" {
		config.Namespace = "test"
	}
	if config.Database == "" {
		config.Database = "test"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &SurrealDBStorage{
		config: config,
	}
}

// NewSurrealDBStorageFromEnv creates a SurrealDB storage instance from environment variables
func NewSurrealDBStorageFromEnv(dbPath string) *SurrealDBStorage {
	namespace := os.Getenv("SURREALDB_NAMESPACE")
	if namespace == "" {
		namespace = "test"
	}

	database := os.Getenv("SURREALDB_DATABASE")
	if database == "" {
		database = "test"
	}

	config := &ConnectionConfig{
		URL:       os.Getenv("SURREALDB_URL"),
		Username:  os.Getenv("SURREALDB_USER"),
		Password:  os.Getenv("SURREALDB_PASS"),
		DBPath:    dbPath,
		Namespace: namespace,
		Database:  database,
		Timeout:   30 * time.Second,
	}

	return NewSurrealDBStorage(config)
}

// Connect establishes connection to SurrealDB (embedded or remote)
func (s *SurrealDBStorage) Connect(ctx context.Context) error {
	var err error

	// Determine connection type: remote vs embedded
	if s.config.URL != "" {
		// Connect to remote SurrealDB instance
		log.Printf("Connecting to remote SurrealDB at %s", s.config.URL)
		s.db, err = surrealdb.New(s.config.URL)
		if err != nil {
			return fmt.Errorf("failed to connect to remote SurrealDB: %w", err)
		}

		// Authenticate if credentials provided
		if s.config.Username != "" && s.config.Password != "" {
			_, err = s.db.SignIn(map[string]interface{}{
				"user": s.config.Username,
				"pass": s.config.Password,
			})
			if err != nil {
				return fmt.Errorf("failed to authenticate with SurrealDB: %w", err)
			}
		}
	} else {
		// Use embedded SurrealDB
		dbURL := fmt.Sprintf("rocksdb://%s", s.config.DBPath)
		log.Printf("Connecting to embedded SurrealDB at %s", dbURL)
		s.db, err = surrealdb.New(dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to embedded SurrealDB: %w", err)
		}
	}

	// Use the configured namespace and database
	err = s.db.Use(s.config.Namespace, s.config.Database)
	if err != nil {
		return fmt.Errorf("failed to use namespace/database: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *SurrealDBStorage) Close() error {
	if s.db != nil {
		s.db.Close()
	}
	return nil
}

// Ping checks if the database connection is alive
func (s *SurrealDBStorage) Ping(ctx context.Context) error {
	if s.db == nil {
		return fmt.Errorf("database connection not established")
	}

	// Simple query to test connection
	_, err := surrealdb.Query[[]map[string]interface{}](s.db, "SELECT 1", nil)
	return err
}

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

	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, params)
	if err != nil {
		return fmt.Errorf("failed to create entity: %w", err)
	}

	// Check if insert was successful
	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" {
			// Update global entity statistics (entities are global, not user-scoped)
			if err := s.updateUserStat(ctx, "global", "entity_count", 1); err != nil {
				// Log the error but don't fail the operation
				log.Printf("Warning: failed to update entity_count stat: %v", err)
			}
			return nil
		}
	}

	return fmt.Errorf("failed to create entity")
}

// resolveEntityID resolves an entity name to its SurrealDB record ID
func (s *SurrealDBStorage) resolveEntityID(ctx context.Context, entityNameOrID string) (string, error) {
	// First, try if it's already a valid record ID by checking if it has the format table:id
	if strings.Contains(entityNameOrID, ":") {
		// Test if it's a valid record ID by querying it directly (same as GetEntity)
		query := "SELECT * FROM " + entityNameOrID
		result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, nil)
		if err == nil && result != nil && len(*result) > 0 {
			queryResult := (*result)[0]
			if queryResult.Status == "OK" && queryResult.Result != nil && len(queryResult.Result) > 0 {
				// It's a valid record ID, return it
				return entityNameOrID, nil
			}
		}
	}

	// If not a record ID or direct lookup failed, try to find by name (same as GetEntity)
	query := "SELECT * FROM entities WHERE name = $name"
	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, map[string]interface{}{"name": entityNameOrID})
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
	// First, resolve entity names to their actual record IDs
	fromEntityID, err := s.resolveEntityID(ctx, fromEntity)
	if err != nil {
		return fmt.Errorf("failed to resolve from entity '%s': %w", fromEntity, err)
	}

	toEntityID, err := s.resolveEntityID(ctx, toEntity)
	if err != nil {
		return fmt.Errorf("failed to resolve to entity '%s': %w", toEntity, err)
	}

	// Use a simple table name based on relationship type
	tableName := relationshipType

	// Create the table if it doesn't exist (SCHEMALESS)
	createTableQuery := fmt.Sprintf("DEFINE TABLE %s SCHEMALESS", tableName)
	_, err = surrealdb.Query[[]map[string]interface{}](s.db, createTableQuery, nil)
	if err != nil {
		// Table might already exist, continue
	}

	// Insert the relationship
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

	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, params)
	if err != nil {
		return fmt.Errorf("failed to create relationship: %w", err)
	}

	// Check if insert was successful
	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" {
			// Update global relationship statistics
			if err := s.updateUserStat(ctx, "global", "relationship_count", 1); err != nil {
				// Log the error but don't fail the operation
				log.Printf("Warning: failed to update relationship_count stat: %v", err)
			}
			return nil
		}
	}

	return fmt.Errorf("failed to create relationship")
}

// TraverseGraph traverses the graph starting from an entity
func (s *SurrealDBStorage) TraverseGraph(ctx context.Context, startEntity, relationshipType string, depth int) ([]GraphResult, error) {
	// For now, just return all entities since graph traversal syntax is complex
	query := "SELECT id, name, type, properties FROM entities"

	params := map[string]interface{}{
		"start_entity": startEntity,
		"depth":        depth,
	}

	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to traverse graph: %w", err)
	}

	return s.parseGraphResults(result)
}

// GetEntity retrieves an entity by ID or name
func (s *SurrealDBStorage) GetEntity(ctx context.Context, entityID string) (*Entity, error) {
	// Try to get by ID first
	query := "SELECT * FROM " + entityID
	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, nil)
	if err != nil {
		// If ID lookup fails, try to find by name
		query = "SELECT * FROM entities WHERE name = $name"
		result, err = surrealdb.Query[[]map[string]interface{}](s.db, query, map[string]interface{}{"name": entityID})
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
	_, err := surrealdb.Delete[map[string]interface{}](s.db, entityID)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	// Update global entity statistics
	if err := s.updateUserStat(ctx, "global", "entity_count", -1); err != nil {
		// Log the error but don't fail the operation
		log.Printf("Warning: failed to update entity_count stat: %v", err)
	}

	return nil
}

// SaveDocument saves a knowledge base document
func (s *SurrealDBStorage) SaveDocument(ctx context.Context, filePath, content string, embedding []float32, metadata map[string]interface{}) error {
	// Ensure metadata is an object and convert embedding
	if metadata == nil {
		metadata = map[string]interface{}{}
	}

	if embedding == nil {
		embedding = make([]float32, defaultMtreeDim)
	} else if len(embedding) != defaultMtreeDim {
		norm := make([]float32, defaultMtreeDim)
		copy(norm, embedding)
		embedding = norm
	}

	emb64 := make([]float64, len(embedding))
	for i, v := range embedding {
		emb64[i] = float64(v)
	}

	// First, check if the document already exists to determine if this is a new document
	existsQuery := "SELECT id FROM knowledge_base WHERE file_path = $file_path"
	existsResult, err := surrealdb.Query[[]map[string]interface{}](s.db, existsQuery, map[string]interface{}{
		"file_path": filePath,
	})

	var isNewDocument bool
	if err != nil || existsResult == nil || len(*existsResult) == 0 {
		isNewDocument = true
	} else {
		queryResult := (*existsResult)[0]
		isNewDocument = queryResult.Status != "OK" || len(queryResult.Result) == 0
	}

	query := `
		UPSERT knowledge_base SET
			file_path = $file_path,
			content = $content,
			embedding = $embedding,
			metadata = $metadata
		WHERE file_path = $file_path
	`

	params := map[string]interface{}{
		"file_path": filePath,
		"content":   content,
		"embedding": emb64,
		"metadata":  metadata,
	}

	_, err = surrealdb.Query[[]map[string]interface{}](s.db, query, params)
	if err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}

	// Update global document statistics only if this is a new document
	if isNewDocument {
		if err := s.updateUserStat(ctx, "global", "document_count", 1); err != nil {
			// Log the error but don't fail the operation
			log.Printf("Warning: failed to update document_count stat: %v", err)
		}
	}

	return nil
}

// SearchDocuments performs similarity search on knowledge base documents
func (s *SurrealDBStorage) SearchDocuments(ctx context.Context, queryEmbedding []float32, limit int) ([]DocumentResult, error) {
	query := fmt.Sprintf(`
		SELECT id, file_path, content, embedding, metadata, created_at, updated_at,
		       vector::similarity::cosine(embedding, $query_embedding) AS similarity
		FROM knowledge_base 
		WHERE embedding <|%d|> $query_embedding
		ORDER BY similarity DESC
	`, limit)

	params := map[string]interface{}{
		"query_embedding": queryEmbedding,
	}

	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	return s.parseDocumentResults(result)
}

// DeleteDocument deletes a knowledge base document
func (s *SurrealDBStorage) DeleteDocument(ctx context.Context, filePath string) error {
	recordID := fmt.Sprintf("knowledge_base:['%s']", strings.ReplaceAll(filePath, "/", "_"))

	_, err := surrealdb.Delete[map[string]interface{}](s.db, recordID)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	// Update global document statistics
	if err := s.updateUserStat(ctx, "global", "document_count", -1); err != nil {
		// Log the error but don't fail the operation
		log.Printf("Warning: failed to update document_count stat: %v", err)
	}

	return nil
}

// GetDocument retrieves a knowledge base document by file path
func (s *SurrealDBStorage) GetDocument(ctx context.Context, filePath string) (*Document, error) {
	recordID := fmt.Sprintf("knowledge_base:['%s']", strings.ReplaceAll(filePath, "/", "_"))

	query := "SELECT * FROM " + recordID
	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if result == nil || len(*result) == 0 {
		return nil, nil
	}

	queryResult := (*result)[0]
	if queryResult.Status != "OK" || queryResult.Result == nil || len(queryResult.Result) == 0 {
		return nil, nil
	}

	resultMap := queryResult.Result[0]
	// Convert embedding from interface{} to []float32
	var embedding []float32
	if embeddingSlice, ok := resultMap["embedding"].([]interface{}); ok {
		embedding = make([]float32, len(embeddingSlice))
		for i, v := range embeddingSlice {
			if f, ok := v.(float64); ok {
				embedding[i] = float32(f)
			} else if f, ok := v.(float32); ok {
				embedding[i] = f
			}
		}
	}

	document := &Document{
		ID:        getString(resultMap, "id"),
		FilePath:  getString(resultMap, "file_path"),
		Content:   getString(resultMap, "content"),
		Embedding: embedding,
		Metadata:  getMap(resultMap, "metadata"),
		CreatedAt: getTime(resultMap, "created_at"),
		UpdatedAt: getTime(resultMap, "updated_at"),
	}
	return document, nil
}

// HybridSearch performs a combined search across vector, graph, and key-value stores
func (s *SurrealDBStorage) HybridSearch(ctx context.Context, userID string, queryEmbedding []float32, entities []string, limit int) (*HybridSearchResult, error) {
	start := time.Now()

	// Perform vector search
	vectorResults, err := s.SearchSimilar(ctx, userID, queryEmbedding, limit)
	if err != nil {
		return nil, fmt.Errorf("failed vector search: %w", err)
	}

	// Perform graph traversal for each entity
	var graphResults []GraphResult
	for _, entity := range entities {
		results, err := s.TraverseGraph(ctx, entity, "", 2) // depth 2
		if err != nil {
			log.Printf("Warning: failed to traverse graph for entity %s: %v", entity, err)
			continue
		}
		graphResults = append(graphResults, results...)
	}

	// Get relevant facts
	facts, err := s.ListFacts(ctx, userID)
	if err != nil {
		log.Printf("Warning: failed to get facts: %v", err)
		facts = make(map[string]interface{})
	}

	result := &HybridSearchResult{
		VectorResults: vectorResults,
		GraphResults:  graphResults,
		Facts:         facts,
		TotalResults:  len(vectorResults) + len(graphResults) + len(facts),
		QueryTime:     time.Since(start),
	}

	return result, nil
}

// GetStats returns statistics about stored memories
func (s *SurrealDBStorage) GetStats(ctx context.Context, userID string) (*MemoryStats, error) {
	stats := &MemoryStats{}

	// Get user-specific stats from user_stats table
	userQuery := "SELECT * FROM user_stats WHERE user_id = $user_id"
	userResult, err := surrealdb.Query[[]map[string]interface{}](s.db, userQuery, map[string]interface{}{"user_id": userID})
	if err == nil && userResult != nil && len(*userResult) > 0 {
		queryResult := (*userResult)[0]
		if queryResult.Status == "OK" && len(queryResult.Result) > 0 {
			userStatsData := queryResult.Result[0]
			// Extract user-specific statistics
			if kvCount, ok := userStatsData["key_value_count"]; ok {
				if count, ok := kvCount.(float64); ok {
					stats.KeyValueCount = int(count)
				}
			}
			if vectorCount, ok := userStatsData["vector_count"]; ok {
				if count, ok := vectorCount.(float64); ok {
					stats.VectorCount = int(count)
				}
			}
		}
	}

	// Get global stats from global user_stats entry
	globalQuery := "SELECT * FROM user_stats WHERE user_id = 'global'"
	globalResult, err := surrealdb.Query[[]map[string]interface{}](s.db, globalQuery, nil)
	if err == nil && globalResult != nil && len(*globalResult) > 0 {
		queryResult := (*globalResult)[0]
		if queryResult.Status == "OK" && len(queryResult.Result) > 0 {
			globalStatsData := queryResult.Result[0]
			// Extract global statistics
			if entityCount, ok := globalStatsData["entity_count"]; ok {
				if count, ok := entityCount.(float64); ok {
					stats.EntityCount = int(count)
				}
			}
			if relCount, ok := globalStatsData["relationship_count"]; ok {
				if count, ok := relCount.(float64); ok {
					stats.RelationshipCount = int(count)
				}
			}
			if docCount, ok := globalStatsData["document_count"]; ok {
				if count, ok := docCount.(float64); ok {
					stats.DocumentCount = int(count)
				}
			}
		}
	}

	// Note: TotalSize is not implemented in the new stats system
	// It would require additional tracking or calculations

	return stats, nil
}

// updateUserStat atomically updates a specific statistic for a user.
// It uses an upsert approach to ensure consistency and handle new users.
func (s *SurrealDBStorage) updateUserStat(ctx context.Context, userID, statField string, delta int) error {
	// First, try to get the existing record
	selectQuery := "SELECT * FROM user_stats WHERE user_id = $user_id"
	selectResult, err := surrealdb.Query[[]map[string]interface{}](s.db, selectQuery, map[string]interface{}{
		"user_id": userID,
	})

	var currentValue int
	recordExists := false

	if err == nil && selectResult != nil && len(*selectResult) > 0 {
		queryResult := (*selectResult)[0]
		if queryResult.Status == "OK" && len(queryResult.Result) > 0 {
			recordExists = true
			if val, ok := queryResult.Result[0][statField]; ok {
				if floatVal, ok := val.(float64); ok {
					currentValue = int(floatVal)
				}
			}
		}
	}

	newValue := currentValue + delta

	if recordExists {
		// Update existing record
		updateQuery := fmt.Sprintf("UPDATE user_stats SET %s = $new_value, updated_at = time::now() WHERE user_id = $user_id", statField)
		_, err = surrealdb.Query[[]map[string]interface{}](s.db, updateQuery, map[string]interface{}{
			"user_id":   userID,
			"new_value": newValue,
		})
	} else {
		// Create new record
		createData := map[string]interface{}{
			"user_id":            userID,
			statField:            newValue,
			"key_value_count":    0,
			"vector_count":       0,
			"entity_count":       0,
			"relationship_count": 0,
			"document_count":     0,
		}
		createData[statField] = newValue

		_, err = surrealdb.Create[map[string]interface{}](s.db, "user_stats", createData)
	}

	if err != nil {
		return fmt.Errorf("failed to update user stat %s for user %s: %w", statField, userID, err)
	}

	return nil
}

// Helper function to extract count from query result
func (s *SurrealDBStorage) extractCount(result *[]surrealdb.QueryResult[[]map[string]interface{}]) int {
	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" && queryResult.Result != nil {
			resultSlice := queryResult.Result
			if len(resultSlice) > 0 {
				if count, ok := resultSlice[0]["count"].(float64); ok {
					return int(count)
				}
			}
		}
	}
	return -1
}

// parseVectorResults converts raw query results to VectorResult structs
func (s *SurrealDBStorage) parseVectorResults(result *[]surrealdb.QueryResult[[]map[string]interface{}]) ([]VectorResult, error) {
	var results []VectorResult

	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" && queryResult.Result != nil {
			resultSlice := queryResult.Result
			for _, itemMap := range resultSlice {
				vectorResult := VectorResult{
					ID:         getString(itemMap, "id"),
					Content:    getString(itemMap, "content"),
					Similarity: getFloat64(itemMap, "similarity"),
					Metadata:   getMap(itemMap, "metadata"),
					CreatedAt:  getTime(itemMap, "created_at"),
					UpdatedAt:  getTime(itemMap, "updated_at"),
				}
				results = append(results, vectorResult)
			}
		}
	}

	return results, nil
}

// parseGraphResults converts raw query results to GraphResult structs
func (s *SurrealDBStorage) parseGraphResults(result *[]surrealdb.QueryResult[[]map[string]interface{}]) ([]GraphResult, error) {
	var results []GraphResult

	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" && queryResult.Result != nil {
			resultSlice := queryResult.Result
			for _, itemMap := range resultSlice {
				// Parse entity information
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
					Path:   []string{entity.ID}, // Simplified path for now
					Depth:  1,                   // Simplified depth for now
				}
				results = append(results, graphResult)
			}
		}
	}

	return results, nil
}

// parseDocumentResults converts raw query results to DocumentResult structs
func (s *SurrealDBStorage) parseDocumentResults(result *[]surrealdb.QueryResult[[]map[string]interface{}]) ([]DocumentResult, error) {
	var results []DocumentResult

	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" && queryResult.Result != nil {
			resultSlice := queryResult.Result
			for _, itemMap := range resultSlice {
				// Convert embedding from interface{} to []float32
				var embedding []float32
				if embeddingSlice, ok := itemMap["embedding"].([]interface{}); ok {
					embedding = make([]float32, len(embeddingSlice))
					for i, v := range embeddingSlice {
						if f, ok := v.(float64); ok {
							embedding[i] = float32(f)
						} else if f, ok := v.(float32); ok {
							embedding[i] = f
						}
					}
				}

				document := &Document{
					ID:        getString(itemMap, "id"),
					FilePath:  getString(itemMap, "file_path"),
					Content:   getString(itemMap, "content"),
					Embedding: embedding,
					Metadata:  getMap(itemMap, "metadata"),
					CreatedAt: getTime(itemMap, "created_at"),
					UpdatedAt: getTime(itemMap, "updated_at"),
				}

				similarity := getFloat64(itemMap, "similarity")

				documentResult := DocumentResult{
					Document:   document,
					Similarity: similarity,
					Score:      similarity, // Use similarity as score for now
				}
				results = append(results, documentResult)
			}
		}
	}

	return results, nil
}

// Helper functions for type conversion
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// extractRecordID extracts a SurrealDB record ID from various formats
func extractRecordID(id interface{}) string {
	if id == nil {
		return ""
	}

	// If it's already a string, return it
	if str, ok := id.(string); ok {
		return str
	}

	// If it's a map/struct with Table and ID fields (SurrealDB record format)
	if idMap, ok := id.(map[string]interface{}); ok {
		if table, hasTable := idMap["Table"]; hasTable {
			if tableStr, ok := table.(string); ok {
				if recordID, hasID := idMap["ID"]; hasID {
					if idStr, ok := recordID.(string); ok {
						return tableStr + ":" + idStr
					}
				}
			}
		}
	}

	// Handle models.RecordID type (check by string representation)
	idStr := fmt.Sprintf("%v", id)

	// Check if it looks like a SurrealDB record ID format: {table id}
	if strings.HasPrefix(idStr, "{") && strings.Contains(idStr, " ") && strings.HasSuffix(idStr, "}") {
		// Parse {table id} format
		inner := idStr[1 : len(idStr)-1] // Remove { }
		parts := strings.SplitN(inner, " ", 2)
		if len(parts) == 2 {
			table := parts[0]
			recordID := parts[1]
			return table + ":" + recordID
		}
	}

	// Try to convert to string as fallback
	return idStr
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	if val, ok := m[key].(float32); ok {
		return float64(val)
	}
	return 0
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if val, ok := m[key].(map[string]interface{}); ok {
		return val
	}
	return make(map[string]interface{})
}

func getTime(m map[string]interface{}, key string) time.Time {
	if val, ok := m[key].(string); ok {
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return t
		}
	}
	return time.Time{}
}

// convertEmbeddingToFloat64 normalizes an input []float32 embedding to defaultMtreeDim length
// (padding with zeros or truncating) and returns a []float64 representation suitable for JSON/CBOR.
func convertEmbeddingToFloat64(embedding []float32) []float64 {
	if embedding == nil {
		embedding = make([]float32, defaultMtreeDim)
	} else if len(embedding) != defaultMtreeDim {
		norm := make([]float32, defaultMtreeDim)
		copy(norm, embedding)
		embedding = norm
	}

	emb64 := make([]float64, len(embedding))
	for i, v := range embedding {
		emb64[i] = float64(v)
	}
	return emb64
}
