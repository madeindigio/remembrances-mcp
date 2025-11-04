//go:build !cgo || !embedded
// +build !cgo !embedded

package storage

import (
	"context"
	"fmt"
)

// EmbeddedSurrealDBStorage is a stub implementation when CGO is not available
type EmbeddedSurrealDBStorage struct {
	config *ConnectionConfig
}

// NewEmbeddedSurrealDBStorage creates a stub that returns an error
func NewEmbeddedSurrealDBStorage(config *ConnectionConfig) *EmbeddedSurrealDBStorage {
	return &EmbeddedSurrealDBStorage{
		config: config,
	}
}

// Connect returns an error indicating embedded DB is not available
func (s *EmbeddedSurrealDBStorage) Connect(ctx context.Context) error {
	return fmt.Errorf("embedded SurrealDB support not available in this build (CGO required, rebuild with -tags embedded)")
}

// Close is a no-op for the stub
func (s *EmbeddedSurrealDBStorage) Close() error {
	return nil
}

// Ping returns an error
func (s *EmbeddedSurrealDBStorage) Ping(ctx context.Context) error {
	return fmt.Errorf("embedded SurrealDB not available")
}

// All other methods return not available errors
func (s *EmbeddedSurrealDBStorage) InitializeSchema(ctx context.Context) error {
	return fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) CreateEntity(ctx context.Context, entityType, name string, properties map[string]interface{}) error {
	return fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) GetEntity(ctx context.Context, entityID string) (*Entity, error) {
	return nil, fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) DeleteEntity(ctx context.Context, entityID string) error {
	return fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) CreateRelationship(ctx context.Context, fromEntity, toEntity, relationshipType string, properties map[string]interface{}) error {
	return fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) TraverseGraph(ctx context.Context, startEntity, relationshipType string, depth int) ([]GraphResult, error) {
	return nil, fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) SaveFact(ctx context.Context, userID, key string, value interface{}) error {
	return fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) GetFact(ctx context.Context, userID, key string) (interface{}, error) {
	return nil, fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) DeleteFact(ctx context.Context, userID, key string) error {
	return fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) ListFacts(ctx context.Context, userID string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) SaveVector(ctx context.Context, userID, content string, embedding []float32, metadata map[string]interface{}) error {
	return fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) SearchSimilar(ctx context.Context, userID string, queryEmbedding []float32, limit int) ([]VectorResult, error) {
	return nil, fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) UpdateVector(ctx context.Context, vectorID, content string, embedding []float32, metadata map[string]interface{}) error {
	return fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) DeleteVector(ctx context.Context, vectorID string) error {
	return fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) GetVector(ctx context.Context, vectorID string) (*Vector, error) {
	return nil, fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) SaveDocument(ctx context.Context, filePath, content string, embedding []float32, metadata map[string]interface{}) error {
	return fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) SearchDocuments(ctx context.Context, queryEmbedding []float32, limit int) ([]DocumentResult, error) {
	return nil, fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) DeleteDocument(ctx context.Context, filePath string) error {
	return fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) GetDocument(ctx context.Context, filePath string) (*Document, error) {
	return nil, fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) HybridSearch(ctx context.Context, userID string, queryEmbedding []float32, entities []string, limit int) (*HybridSearchResult, error) {
	return nil, fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) GetStats(ctx context.Context, userID string) (*MemoryStats, error) {
	return nil, fmt.Errorf("embedded SurrealDB not available")
}

func (s *EmbeddedSurrealDBStorage) Query(ctx context.Context, query string, vars map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("embedded SurrealDB not available")
}
