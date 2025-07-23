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
		config.Namespace = "remembrances"
	}
	if config.Database == "" {
		config.Database = "memories"
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
	config := &ConnectionConfig{
		URL:       os.Getenv("SURREALDB_URL"),
		Username:  os.Getenv("SURREALDB_USER"),
		Password:  os.Getenv("SURREALDB_PASS"),
		DBPath:    dbPath,
		Namespace: "remembrances",
		Database:  "memories",
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
	_, err := surrealdb.Query[map[string]interface{}](s.db, "SELECT 1", nil)
	return err
}

// InitializeSchema creates all required tables and indexes
func (s *SurrealDBStorage) InitializeSchema(ctx context.Context) error {
	log.Println("Initializing SurrealDB schema...")

	schemas := []string{
		// Key-Value memories table
		`DEFINE TABLE kv_memories SCHEMAFULL;`,
		`DEFINE FIELD user_id ON kv_memories TYPE string;`,
		`DEFINE FIELD key ON kv_memories TYPE string;`,
		`DEFINE FIELD value ON kv_memories TYPE any;`,
		`DEFINE FIELD created_at ON kv_memories TYPE datetime VALUE time::now();`,
		`DEFINE FIELD updated_at ON kv_memories TYPE datetime VALUE time::now() ON UPDATE time::now();`,
		`DEFINE INDEX idx_kv_user_key ON kv_memories FIELDS user_id, key UNIQUE;`,

		// Vector memories table for semantic search
		`DEFINE TABLE vector_memories SCHEMAFULL;`,
		`DEFINE FIELD user_id ON vector_memories TYPE string;`,
		`DEFINE FIELD content ON vector_memories TYPE string;`,
		`DEFINE FIELD embedding ON vector_memories TYPE array<float>;`,
		`DEFINE FIELD metadata ON vector_memories TYPE object;`,
		`DEFINE FIELD created_at ON vector_memories TYPE datetime VALUE time::now();`,
		`DEFINE FIELD updated_at ON vector_memories TYPE datetime VALUE time::now() ON UPDATE time::now();`,
		`DEFINE INDEX idx_vector_user ON vector_memories FIELDS user_id;`,
		`DEFINE INDEX idx_embedding ON vector_memories FIELDS embedding MTREE DIMENSION 768 DIST COSINE;`,

		// Knowledge base table for markdown documents
		`DEFINE TABLE knowledge_base SCHEMAFULL;`,
		`DEFINE FIELD file_path ON knowledge_base TYPE string;`,
		`DEFINE FIELD content ON knowledge_base TYPE string;`,
		`DEFINE FIELD embedding ON knowledge_base TYPE array<float>;`,
		`DEFINE FIELD metadata ON knowledge_base TYPE object;`,
		`DEFINE FIELD created_at ON knowledge_base TYPE datetime VALUE time::now();`,
		`DEFINE FIELD updated_at ON knowledge_base TYPE datetime VALUE time::now() ON UPDATE time::now();`,
		`DEFINE INDEX idx_kb_path ON knowledge_base FIELDS file_path UNIQUE;`,
		`DEFINE INDEX idx_kb_embedding ON knowledge_base FIELDS embedding MTREE DIMENSION 768 DIST COSINE;`,

		// Entities table for graph nodes
		`DEFINE TABLE entities SCHEMAFULL;`,
		`DEFINE FIELD name ON entities TYPE string;`,
		`DEFINE FIELD type ON entities TYPE string;`,
		`DEFINE FIELD properties ON entities TYPE object;`,
		`DEFINE FIELD created_at ON entities TYPE datetime VALUE time::now();`,
		`DEFINE FIELD updated_at ON entities TYPE datetime VALUE time::now() ON UPDATE time::now();`,
		`DEFINE INDEX idx_entity_name ON entities FIELDS name;`,
		`DEFINE INDEX idx_entity_type ON entities FIELDS type;`,

		// Relationships table for graph edges
		`DEFINE TABLE wrote SCHEMALESS;`,
		`DEFINE FIELD in ON wrote TYPE record(entities);`,
		`DEFINE FIELD out ON wrote TYPE record(entities);`,
		`DEFINE FIELD timestamp ON wrote TYPE datetime VALUE time::now();`,
		`DEFINE FIELD properties ON wrote TYPE object;`,

		// Additional relationship types
		`DEFINE TABLE mentioned_in SCHEMALESS;`,
		`DEFINE FIELD in ON mentioned_in TYPE record(entities);`,
		`DEFINE FIELD out ON mentioned_in TYPE record(entities);`,
		`DEFINE FIELD timestamp ON mentioned_in TYPE datetime VALUE time::now();`,
		`DEFINE FIELD properties ON mentioned_in TYPE object;`,

		`DEFINE TABLE related_to SCHEMALESS;`,
		`DEFINE FIELD in ON related_to TYPE record(entities);`,
		`DEFINE FIELD out ON related_to TYPE record(entities);`,
		`DEFINE FIELD timestamp ON related_to TYPE datetime VALUE time::now();`,
		`DEFINE FIELD properties ON related_to TYPE object;`,
	}

	for _, schema := range schemas {
		_, err := surrealdb.Query[map[string]interface{}](s.db, schema, nil)
		if err != nil {
			return fmt.Errorf("failed to execute schema statement '%s': %w", schema, err)
		}
	}

	log.Println("Schema initialization completed")
	return nil
}

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

	result, err := surrealdb.Select[map[string]interface{}](s.db, recordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fact: %w", err)
	}

	// Handle empty result
	if result == nil {
		return nil, nil
	}

	// Extract value from result
	return (*result)["value"], nil
}

// UpdateFact updates an existing key-value fact
func (s *SurrealDBStorage) UpdateFact(ctx context.Context, userID, key string, value interface{}) error {
	recordID := fmt.Sprintf("kv_memories:['%s:%s']", userID, key)

	data := map[string]interface{}{
		"value": value,
	}

	_, err := surrealdb.Update[map[string]interface{}](s.db, recordID, data)
	if err != nil {
		return fmt.Errorf("failed to update fact: %w", err)
	}

	return nil
}

// DeleteFact deletes a key-value fact
func (s *SurrealDBStorage) DeleteFact(ctx context.Context, userID, key string) error {
	recordID := fmt.Sprintf("kv_memories:['%s:%s']", userID, key)

	_, err := surrealdb.Delete[map[string]interface{}](s.db, recordID)
	if err != nil {
		return fmt.Errorf("failed to delete fact: %w", err)
	}

	return nil
}

// ListFacts retrieves all facts for a user
func (s *SurrealDBStorage) ListFacts(ctx context.Context, userID string) (map[string]interface{}, error) {
	query := "SELECT key, value FROM kv_memories WHERE user_id = $user_id"
	params := map[string]interface{}{
		"user_id": userID,
	}

	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list facts: %w", err)
	}

	facts := make(map[string]interface{})

	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" && queryResult.Result != nil {
			resultSlice := queryResult.Result
			for _, item := range resultSlice {
				if key, hasKey := item["key"].(string); hasKey {
					if value, hasValue := item["value"]; hasValue {
						facts[key] = value
					}
				}
			}
		}
	}

	return facts, nil
}

// IndexVector stores a vector embedding with content and metadata
func (s *SurrealDBStorage) IndexVector(ctx context.Context, userID, content string, embedding []float32, metadata map[string]interface{}) (string, error) {
	data := map[string]interface{}{
		"user_id":   userID,
		"content":   content,
		"embedding": embedding,
		"metadata":  metadata,
	}

	result, err := surrealdb.Create[map[string]interface{}](s.db, "vector_memories", data)
	if err != nil {
		return "", fmt.Errorf("failed to index vector: %w", err)
	}

	// Extract ID from result
	if result != nil {
		if id, ok := (*result)["id"].(string); ok {
			return id, nil
		}
	}

	return "", fmt.Errorf("failed to extract ID from result")
}

// SearchSimilar performs vector similarity search
func (s *SurrealDBStorage) SearchSimilar(ctx context.Context, userID string, queryEmbedding []float32, limit int) ([]VectorResult, error) {
	query := `
		SELECT id, content, vector::similarity::cosine(embedding, $query_embedding) AS similarity, metadata, created_at, updated_at
		FROM vector_memories 
		WHERE user_id = $user_id AND embedding <|$limit|> $query_embedding
		ORDER BY similarity DESC
	`

	params := map[string]interface{}{
		"user_id":         userID,
		"query_embedding": queryEmbedding,
		"limit":           limit,
	}

	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar vectors: %w", err)
	}

	return s.parseVectorResults(result)
}

// UpdateVector updates an existing vector memory
func (s *SurrealDBStorage) UpdateVector(ctx context.Context, id, userID, content string, embedding []float32, metadata map[string]interface{}) error {
	data := map[string]interface{}{
		"content":   content,
		"embedding": embedding,
		"metadata":  metadata,
	}

	_, err := surrealdb.Update[map[string]interface{}](s.db, id, data)
	if err != nil {
		return fmt.Errorf("failed to update vector: %w", err)
	}

	return nil
}

// DeleteVector deletes a vector memory
func (s *SurrealDBStorage) DeleteVector(ctx context.Context, id, userID string) error {
	_, err := surrealdb.Delete[map[string]interface{}](s.db, id)
	if err != nil {
		return fmt.Errorf("failed to delete vector: %w", err)
	}

	return nil
}

// CreateEntity creates a new entity in the graph
func (s *SurrealDBStorage) CreateEntity(ctx context.Context, entityType, name string, properties map[string]interface{}) (string, error) {
	data := map[string]interface{}{
		"type":       entityType,
		"name":       name,
		"properties": properties,
	}

	result, err := surrealdb.Create[map[string]interface{}](s.db, "entities", data)
	if err != nil {
		return "", fmt.Errorf("failed to create entity: %w", err)
	}

	// Extract ID from result
	if result != nil {
		if id, ok := (*result)["id"].(string); ok {
			return id, nil
		}
	}

	return "", fmt.Errorf("failed to extract ID from result")
}

// CreateRelationship creates a relationship between two entities
func (s *SurrealDBStorage) CreateRelationship(ctx context.Context, fromEntity, toEntity, relationshipType string, properties map[string]interface{}) error {
	query := fmt.Sprintf("RELATE $from->%s->$to CONTENT $content", relationshipType)

	content := properties
	if content == nil {
		content = make(map[string]interface{})
	}
	content["timestamp"] = time.Now()

	params := map[string]interface{}{
		"from":    fromEntity,
		"to":      toEntity,
		"content": content,
	}

	_, err := surrealdb.Query[map[string]interface{}](s.db, query, params)
	if err != nil {
		return fmt.Errorf("failed to create relationship: %w", err)
	}

	return nil
}

// TraverseGraph traverses the graph starting from an entity
func (s *SurrealDBStorage) TraverseGraph(ctx context.Context, startEntity, relationshipType string, depth int) ([]GraphResult, error) {
	var query string

	if relationshipType == "" {
		// Traverse all relationship types
		query = fmt.Sprintf("SELECT ->* FROM %s", startEntity)
	} else {
		// Traverse specific relationship type
		query = fmt.Sprintf("SELECT ->%s->entities FROM %s", relationshipType, startEntity)
	}

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

// GetEntity retrieves an entity by ID
func (s *SurrealDBStorage) GetEntity(ctx context.Context, entityID string) (*Entity, error) {
	result, err := surrealdb.Select[map[string]interface{}](s.db, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	if result == nil {
		return nil, nil
	}

	resultMap := *result
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

	return nil
}

// SaveDocument saves a knowledge base document
func (s *SurrealDBStorage) SaveDocument(ctx context.Context, filePath, content string, embedding []float32, metadata map[string]interface{}) error {
	data := map[string]interface{}{
		"file_path": filePath,
		"content":   content,
		"embedding": embedding,
		"metadata":  metadata,
	}

	recordID := fmt.Sprintf("knowledge_base:['%s']", strings.ReplaceAll(filePath, "/", "_"))
	_, err := surrealdb.Create[map[string]interface{}](s.db, recordID, data)
	if err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}

	return nil
}

// SearchDocuments performs similarity search on knowledge base documents
func (s *SurrealDBStorage) SearchDocuments(ctx context.Context, queryEmbedding []float32, limit int) ([]DocumentResult, error) {
	query := `
		SELECT id, file_path, content, embedding, metadata, created_at, updated_at,
		       vector::similarity::cosine(embedding, $query_embedding) AS similarity
		FROM knowledge_base 
		WHERE embedding <|$limit|> $query_embedding
		ORDER BY similarity DESC
	`

	params := map[string]interface{}{
		"query_embedding": queryEmbedding,
		"limit":           limit,
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

	return nil
}

// GetDocument retrieves a knowledge base document by file path
func (s *SurrealDBStorage) GetDocument(ctx context.Context, filePath string) (*Document, error) {
	recordID := fmt.Sprintf("knowledge_base:['%s']", strings.ReplaceAll(filePath, "/", "_"))

	result, err := surrealdb.Select[map[string]interface{}](s.db, recordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if result == nil {
		return nil, nil
	}

	resultMap := *result
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

	// Count key-value memories
	kvQuery := "SELECT count() FROM kv_memories WHERE user_id = $user_id GROUP ALL"
	kvResult, err := surrealdb.Query[[]map[string]interface{}](s.db, kvQuery, map[string]interface{}{"user_id": userID})
	if err == nil {
		if count := s.extractCount(kvResult); count >= 0 {
			stats.KeyValueCount = count
		}
	}

	// Count vector memories
	vectorQuery := "SELECT count() FROM vector_memories WHERE user_id = $user_id GROUP ALL"
	vectorResult, err := surrealdb.Query[[]map[string]interface{}](s.db, vectorQuery, map[string]interface{}{"user_id": userID})
	if err == nil {
		if count := s.extractCount(vectorResult); count >= 0 {
			stats.VectorCount = count
		}
	}

	// Count entities
	entityQuery := "SELECT count() FROM entities GROUP ALL"
	entityResult, err := surrealdb.Query[[]map[string]interface{}](s.db, entityQuery, nil)
	if err == nil {
		if count := s.extractCount(entityResult); count >= 0 {
			stats.EntityCount = count
		}
	}

	// Count relationships
	relQuery := "SELECT count() FROM wrote GROUP ALL"
	relResult, err := surrealdb.Query[[]map[string]interface{}](s.db, relQuery, nil)
	if err == nil {
		if count := s.extractCount(relResult); count >= 0 {
			stats.RelationshipCount = count
		}
	}

	// Count knowledge base documents
	kbQuery := "SELECT count() FROM knowledge_base GROUP ALL"
	kbResult, err := surrealdb.Query[[]map[string]interface{}](s.db, kbQuery, nil)
	if err == nil {
		if count := s.extractCount(kbResult); count >= 0 {
			stats.DocumentCount = count
		}
	}

	return stats, nil
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
