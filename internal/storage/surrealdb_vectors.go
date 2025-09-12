package storage

import (
	"context"
	"fmt"
	"log"

	"github.com/surrealdb/surrealdb.go"
)

// IndexVector stores a vector embedding with content and metadata
func (s *SurrealDBStorage) IndexVector(ctx context.Context, userID, content string, embedding []float32, metadata map[string]interface{}) error {
	// SurrealDB schema defines `metadata` as an object. Ensure we never send NULL.
	if metadata == nil {
		metadata = map[string]interface{}{}
	}

	// Normalize embedding length to the MTREE dimension (pad with zeros or truncate)
	if embedding == nil {
		embedding = make([]float32, defaultMtreeDim)
	} else if len(embedding) != defaultMtreeDim {
		norm := make([]float32, defaultMtreeDim)
		copy(norm, embedding)
		embedding = norm
	}

	// Convert embedding to []float64 as other methods do (SurrealDB JSON numeric consistency)
	emb64 := make([]float64, len(embedding))
	for i, v := range embedding {
		emb64[i] = float64(v)
	}

	query := `
		INSERT INTO vector_memories {
			user_id: $user_id,
			content: $content,
			embedding: $embedding,
			metadata: $metadata
		} RETURN id
	`

	params := map[string]interface{}{
		"user_id":   userID,
		"content":   content,
		"embedding": emb64,
		"metadata":  metadata,
	}

	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, params)
	if err != nil {
		return fmt.Errorf("failed to index vector: %w", err)
	}

	// Check if insert was successful
	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" {
			// Update user statistics on successful insert
			if err := s.updateUserStat(ctx, userID, "vector_count", 1); err != nil {
				// Log the error but don't fail the operation
				log.Printf("Warning: failed to update vector_count stat for user %s: %v", userID, err)
			}
			return nil
		}
	}

	return fmt.Errorf("failed to index vector")
}

// SearchSimilar performs vector similarity search
func (s *SurrealDBStorage) SearchSimilar(ctx context.Context, userID string, queryEmbedding []float32, limit int) ([]VectorResult, error) {
	query := fmt.Sprintf(`
		SELECT id, content, vector::similarity::cosine(embedding, $query_embedding) AS similarity, metadata, created_at, updated_at
		FROM vector_memories 
		WHERE user_id = $user_id AND embedding <|%d|> $query_embedding
		ORDER BY similarity DESC
	`, limit)

	params := map[string]interface{}{
		"user_id":         userID,
		"query_embedding": queryEmbedding,
	}

	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar vectors: %w", err)
	}

	return s.parseVectorResults(result)
}

// UpdateVector updates an existing vector memory
func (s *SurrealDBStorage) UpdateVector(ctx context.Context, id, userID, content string, embedding []float32, metadata map[string]interface{}) error {
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

	data := map[string]interface{}{
		"content":   content,
		"embedding": emb64,
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

	// Update user statistics
	if err := s.updateUserStat(ctx, userID, "vector_count", -1); err != nil {
		// Log the error but don't fail the operation
		log.Printf("Warning: failed to update vector_count stat for user %s: %v", userID, err)
	}

	return nil
}
