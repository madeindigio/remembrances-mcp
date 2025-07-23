// Package storage handles the interaction with the underlying database.
package storage

import (
	"context"
	"time"
)

// Storage provides a unified interface for all database operations
// supporting the three-level memory architecture: key-value, vector, and graph
type Storage interface {
	// Connection management
	Connect(ctx context.Context) error
	Close() error
	Ping(ctx context.Context) error
	InitializeSchema(ctx context.Context) error

	// Key-Value operations for discrete facts and preferences
	SaveFact(ctx context.Context, userID, key string, value interface{}) error
	GetFact(ctx context.Context, userID, key string) (interface{}, error)
	UpdateFact(ctx context.Context, userID, key string, value interface{}) error
	DeleteFact(ctx context.Context, userID, key string) error
	ListFacts(ctx context.Context, userID string) (map[string]interface{}, error)

	// Vector operations for semantic/RAG storage
	IndexVector(ctx context.Context, userID, content string, embedding []float32, metadata map[string]interface{}) (string, error)
	SearchSimilar(ctx context.Context, userID string, queryEmbedding []float32, limit int) ([]VectorResult, error)
	UpdateVector(ctx context.Context, id, userID, content string, embedding []float32, metadata map[string]interface{}) error
	DeleteVector(ctx context.Context, id, userID string) error

	// Graph operations for entities and relationships
	CreateEntity(ctx context.Context, entityType, name string, properties map[string]interface{}) (string, error)
	CreateRelationship(ctx context.Context, fromEntity, toEntity, relationshipType string, properties map[string]interface{}) error
	TraverseGraph(ctx context.Context, startEntity, relationshipType string, depth int) ([]GraphResult, error)
	GetEntity(ctx context.Context, entityID string) (*Entity, error)
	DeleteEntity(ctx context.Context, entityID string) error

	// Knowledge base operations for markdown documents
	SaveDocument(ctx context.Context, filePath, content string, embedding []float32, metadata map[string]interface{}) error
	SearchDocuments(ctx context.Context, queryEmbedding []float32, limit int) ([]DocumentResult, error)
	DeleteDocument(ctx context.Context, filePath string) error
	GetDocument(ctx context.Context, filePath string) (*Document, error)

	// Hybrid search combining vector, key-value, and graph queries
	HybridSearch(ctx context.Context, userID string, queryEmbedding []float32, entities []string, limit int) (*HybridSearchResult, error)
}

// VectorResult represents a result from vector similarity search
type VectorResult struct {
	ID         string                 `json:"id"`
	Content    string                 `json:"content"`
	Similarity float64                `json:"similarity"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// Entity represents a graph node
type Entity struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// Relationship represents a graph edge
type Relationship struct {
	ID         string                 `json:"id"`
	From       string                 `json:"from"`
	To         string                 `json:"to"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Timestamp  time.Time              `json:"timestamp"`
}

// GraphResult represents a result from graph traversal
type GraphResult struct {
	Entity       *Entity       `json:"entity"`
	Relationship *Relationship `json:"relationship,omitempty"`
	Path         []string      `json:"path"`
	Depth        int           `json:"depth"`
}

// Document represents a knowledge base document
type Document struct {
	ID        string                 `json:"id"`
	FilePath  string                 `json:"file_path"`
	Content   string                 `json:"content"`
	Embedding []float32              `json:"embedding"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// DocumentResult represents a result from document search
type DocumentResult struct {
	Document   *Document `json:"document"`
	Similarity float64   `json:"similarity"`
	Score      float64   `json:"score"`
}

// HybridSearchResult combines results from multiple search types
type HybridSearchResult struct {
	VectorResults []VectorResult         `json:"vector_results"`
	GraphResults  []GraphResult          `json:"graph_results"`
	Facts         map[string]interface{} `json:"facts"`
	TotalResults  int                    `json:"total_results"`
	QueryTime     time.Duration          `json:"query_time"`
}

// ConnectionConfig holds database connection configuration
type ConnectionConfig struct {
	// Remote connection settings
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`

	// Embedded database settings
	DBPath string `json:"db_path"`

	// General settings
	Namespace string        `json:"namespace"`
	Database  string        `json:"database"`
	Timeout   time.Duration `json:"timeout"`
}

// MemoryStats provides statistics about stored memories
type MemoryStats struct {
	KeyValueCount     int   `json:"key_value_count"`
	VectorCount       int   `json:"vector_count"`
	EntityCount       int   `json:"entity_count"`
	RelationshipCount int   `json:"relationship_count"`
	DocumentCount     int   `json:"document_count"`
	TotalSize         int64 `json:"total_size_bytes"`
}

// GetStats returns statistics about stored memories
type StatsProvider interface {
	GetStats(ctx context.Context, userID string) (*MemoryStats, error)
}

// Combined interface that includes stats
type StorageWithStats interface {
	Storage
	StatsProvider
}
