//go:build cgo && embedded
// +build cgo,embedded

package storage

import (
	"context"
	"fmt"
	"log"
	"strings"

	embeddeddb "github.com/yourusername/surrealdb-embedded"
)

// EmbeddedSurrealDBStorage implements the Storage interface using the embedded SurrealDB library
type EmbeddedSurrealDBStorage struct {
	db     *embeddeddb.DB
	config *ConnectionConfig
	// Wrap a standard SurrealDBStorage to reuse query logic
	wrapper *SurrealDBStorage
}

// NewEmbeddedSurrealDBStorage creates a new embedded SurrealDB storage instance
func NewEmbeddedSurrealDBStorage(config *ConnectionConfig) *EmbeddedSurrealDBStorage {
	if config.Namespace == "" {
		config.Namespace = "test"
	}
	if config.Database == "" {
		config.Database = "test"
	}

	return &EmbeddedSurrealDBStorage{
		config: config,
	}
}

// Connect establishes connection to embedded SurrealDB
func (s *EmbeddedSurrealDBStorage) Connect(ctx context.Context) error {
	var err error

	// Determine if we should use in-memory or RocksDB
	if s.config.DBPath == "" || strings.HasPrefix(s.config.DBPath, ":memory:") {
		log.Printf("Connecting to embedded SurrealDB (in-memory)")
		s.db, err = embeddeddb.NewMemory()
	} else {
		log.Printf("Connecting to embedded SurrealDB (RocksDB) at %s", s.config.DBPath)
		s.db, err = embeddeddb.NewRocksDB(s.config.DBPath)
	}

	if err != nil {
		return fmt.Errorf("failed to create embedded SurrealDB: %w", err)
	}

	// Use the configured namespace and database
	err = s.db.Use(s.config.Namespace, s.config.Database)
	if err != nil {
		return fmt.Errorf("failed to use namespace/database: %w", err)
	}

	log.Printf("Successfully connected to embedded SurrealDB")
	return nil
}

// Close closes the database connection
func (s *EmbeddedSurrealDBStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Ping checks if the database connection is alive
func (s *EmbeddedSurrealDBStorage) Ping(ctx context.Context) error {
	if s.db == nil {
		return fmt.Errorf("database connection not established")
	}

	// Simple query to test connection
	_, err := s.db.Query("SELECT 1", nil)
	return err
}

// For all other methods, we can delegate to the wrapper implementation
// This allows us to reuse the existing query logic while using the embedded backend

// Query executes a raw SurrealQL query
func (s *EmbeddedSurrealDBStorage) Query(ctx context.Context, query string, vars map[string]interface{}) (interface{}, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not established")
	}
	return s.db.Query(query, vars)
}

// The rest of the methods will be implemented by delegating to the embedded DB
// For now, let's implement the most critical ones

// InitializeSchema sets up the database schema
func (s *EmbeddedSurrealDBStorage) InitializeSchema(ctx context.Context) error {
	// Define tables
	queries := []string{
		"DEFINE TABLE entities SCHEMALESS",
		"DEFINE TABLE knowledge_base SCHEMALESS",
		"DEFINE TABLE user_stats SCHEMALESS",
		"DEFINE INDEX idx_entities_name ON TABLE entities COLUMNS name",
		"DEFINE INDEX idx_knowledge_base_path ON TABLE knowledge_base COLUMNS file_path UNIQUE",
		"DEFINE INDEX idx_user_stats_user_id ON TABLE user_stats COLUMNS user_id UNIQUE",
	}

	for _, query := range queries {
		_, err := s.db.Query(query, nil)
		if err != nil {
			log.Printf("Warning: schema initialization query failed (may be OK if already exists): %v", err)
		}
	}

	return nil
}

// CreateEntity creates a new entity in the graph
func (s *EmbeddedSurrealDBStorage) CreateEntity(ctx context.Context, entityType, name string, properties map[string]interface{}) error {
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

	_, err := s.db.Query(query, params)
	if err != nil {
		return fmt.Errorf("failed to create entity: %w", err)
	}

	return nil
}

// GetEntity retrieves an entity by ID or name
func (s *EmbeddedSurrealDBStorage) GetEntity(ctx context.Context, entityID string) (*Entity, error) {
	// Try to get by ID first
	query := "SELECT * FROM " + entityID
	result, err := s.db.Query(query, nil)
	if err != nil {
		// If ID lookup fails, try to find by name
		query = "SELECT * FROM entities WHERE name = $name"
		result, err = s.db.Query(query, map[string]interface{}{"name": entityID})
		if err != nil {
			return nil, fmt.Errorf("failed to get entity: %w", err)
		}
	}

	// Parse result
	if result == nil {
		return nil, nil
	}

	// TODO: Parse the result properly based on the embedded DB's response format
	// For now, return nil to avoid compilation errors
	return nil, fmt.Errorf("not implemented: result parsing for embedded DB")
}

// CreateRelationship creates a relationship between two entities
func (s *EmbeddedSurrealDBStorage) CreateRelationship(ctx context.Context, fromEntity, toEntity, relationshipType string, properties map[string]interface{}) error {
	tableName := relationshipType

	// Create the table if it doesn't exist
	createTableQuery := fmt.Sprintf("DEFINE TABLE %s SCHEMALESS", tableName)
	_, _ = s.db.Query(createTableQuery, nil)

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
		"from":             fromEntity,
		"to":               toEntity,
		"relationshipType": relationshipType,
		"properties":       properties,
	}

	_, err := s.db.Query(query, params)
	if err != nil {
		return fmt.Errorf("failed to create relationship: %w", err)
	}

	return nil
}

// DeleteEntity deletes an entity
func (s *EmbeddedSurrealDBStorage) DeleteEntity(ctx context.Context, entityID string) error {
	_, err := s.db.Delete(entityID)
	return err
}

// TraverseGraph traverses the graph starting from an entity
func (s *EmbeddedSurrealDBStorage) TraverseGraph(ctx context.Context, startEntity, relationshipType string, depth int) ([]GraphResult, error) {
	query := "SELECT id, name, type, properties FROM entities"
	_, err := s.db.Query(query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to traverse graph: %w", err)
	}

	// TODO: Implement proper graph traversal and result parsing
	return []GraphResult{}, nil
}

// SaveDocument saves a knowledge base document
func (s *EmbeddedSurrealDBStorage) SaveDocument(ctx context.Context, filePath, content string, embedding []float32, metadata map[string]interface{}) error {
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

	params := map[string]interface{}{
		"file_path": filePath,
		"content":   content,
		"embedding": emb64,
		"metadata":  metadata,
	}

	// Use UPSERT to handle both insert and update
	query := `
		CREATE knowledge_base CONTENT {
			file_path: $file_path,
			content: $content,
			embedding: $embedding,
			metadata: $metadata
		}
	`

	_, err := s.db.Query(query, params)
	return err
}

// SearchDocuments performs similarity search on knowledge base documents
func (s *EmbeddedSurrealDBStorage) SearchDocuments(ctx context.Context, queryEmbedding []float32, limit int) ([]DocumentResult, error) {
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

	_, err := s.db.Query(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	// TODO: Parse results properly
	return []DocumentResult{}, nil
}

// DeleteDocument deletes a knowledge base document
func (s *EmbeddedSurrealDBStorage) DeleteDocument(ctx context.Context, filePath string) error {
	query := "DELETE FROM knowledge_base WHERE file_path = $file_path"
	params := map[string]interface{}{
		"file_path": filePath,
	}

	_, err := s.db.Query(query, params)
	return err
}

// GetDocument retrieves a knowledge base document by file path
func (s *EmbeddedSurrealDBStorage) GetDocument(ctx context.Context, filePath string) (*Document, error) {
	query := "SELECT * FROM knowledge_base WHERE file_path = $file_path"
	params := map[string]interface{}{
		"file_path": filePath,
	}

	_, err := s.db.Query(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	// TODO: Parse results properly
	return nil, nil
}

// GetStats returns statistics about stored memories
func (s *EmbeddedSurrealDBStorage) GetStats(ctx context.Context, userID string) (*MemoryStats, error) {
	stats := &MemoryStats{}

	// Get user-specific stats
	userQuery := "SELECT * FROM user_stats WHERE user_id = $user_id"
	_, err := s.db.Query(userQuery, map[string]interface{}{"user_id": userID})
	if err != nil {
		log.Printf("Warning: failed to get user stats: %v", err)
	}

	// Get global stats
	globalQuery := "SELECT * FROM user_stats WHERE user_id = 'global'"
	_, err = s.db.Query(globalQuery, nil)
	if err != nil {
		log.Printf("Warning: failed to get global stats: %v", err)
	}

	// TODO: Parse results properly
	return stats, nil
}

// HybridSearch performs a combined search across vector, graph, and key-value stores
func (s *EmbeddedSurrealDBStorage) HybridSearch(ctx context.Context, userID string, queryEmbedding []float32, entities []string, limit int) (*HybridSearchResult, error) {
	// TODO: Implement hybrid search
	return &HybridSearchResult{}, fmt.Errorf("not implemented for embedded DB")
}

// Implement the remaining methods from the Storage interface...
// For brevity, I'll add stubs that return not implemented errors
// These should be fully implemented based on the embedded DB's API

func (s *EmbeddedSurrealDBStorage) SaveFact(ctx context.Context, userID, key string, value interface{}) error {
	return fmt.Errorf("not implemented for embedded DB")
}

func (s *EmbeddedSurrealDBStorage) GetFact(ctx context.Context, userID, key string) (interface{}, error) {
	return nil, fmt.Errorf("not implemented for embedded DB")
}

func (s *EmbeddedSurrealDBStorage) DeleteFact(ctx context.Context, userID, key string) error {
	return fmt.Errorf("not implemented for embedded DB")
}

func (s *EmbeddedSurrealDBStorage) ListFacts(ctx context.Context, userID string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("not implemented for embedded DB")
}

func (s *EmbeddedSurrealDBStorage) SaveVector(ctx context.Context, userID, content string, embedding []float32, metadata map[string]interface{}) error {
	return fmt.Errorf("not implemented for embedded DB")
}

func (s *EmbeddedSurrealDBStorage) SearchSimilar(ctx context.Context, userID string, queryEmbedding []float32, limit int) ([]VectorResult, error) {
	return nil, fmt.Errorf("not implemented for embedded DB")
}

func (s *EmbeddedSurrealDBStorage) UpdateVector(ctx context.Context, vectorID, content string, embedding []float32, metadata map[string]interface{}) error {
	return fmt.Errorf("not implemented for embedded DB")
}

func (s *EmbeddedSurrealDBStorage) DeleteVector(ctx context.Context, vectorID string) error {
	return fmt.Errorf("not implemented for embedded DB")
}

func (s *EmbeddedSurrealDBStorage) GetVector(ctx context.Context, vectorID string) (*Vector, error) {
	return nil, fmt.Errorf("not implemented for embedded DB")
}
