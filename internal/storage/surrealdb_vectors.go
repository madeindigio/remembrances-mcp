package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/madeindigio/remembrances-mcp/v2/internal/storage/migrations"
)

// The vector memory surface, scoped per RM-2.
//
// Before this change these methods wrote vector_memories and read
// semantic_memories, a table no migration has ever defined, so an update or a
// delete was a no-op against an empty table. Everything here now addresses
// vector_memories, which is the table the schema, the HNSW index and the v13
// backfill know about.

// vectorContentHash fills content_hash on vector_memories. It must agree with
// migrations.ContentHash: the v14 uniqueness index depends on the two sides
// computing the same value.
var vectorContentHash = migrations.ContentHashSurrealExpr("$content")

// IndexVector stores a vector embedding with content and metadata in the
// caller's write scope.
func (s *SurrealDBStorage) IndexVector(ctx context.Context, userID, content string, embedding []float32, metadata map[string]interface{}) error {
	t, err := s.scope(ctx, userID)
	if err != nil {
		return err
	}

	// SurrealDB schema defines `metadata` as an object. Ensure we never send NULL.
	if metadata == nil {
		metadata = map[string]interface{}{}
	}

	params := t.params(map[string]interface{}{
		"content": content,
		// Pad or truncate to the configured vector width: SurrealDB 3.x reads
		// array<float, N> as exactly N elements.
		"embedding": normalizeEmbedding(embedding),
		"metadata":  metadata,
		"user_id":   t.Principal().LegacyUserID,
	})

	query := fmt.Sprintf(`
		CREATE vector_memories CONTENT {
			%suser_id: $user_id,
			content: $content,
			embedding: $embedding,
			metadata: $metadata,
			content_hash: %s
		}
	`, t.contentFields(), vectorContentHash)

	if _, err := s.query(ctx, query, params); err != nil {
		return fmt.Errorf("failed to index vector: %w", err)
	}

	if err := s.updateUserStat(ctx, t.Principal().LegacyUserID, "vector_count", 0); err != nil {
		// Log the error but don't fail the operation
		slog.Warn("failed to update vector_count stat", "user_id", userID, "error", err)
	}
	return nil
}

// SearchSimilar performs vector similarity search over the caller's visible
// scopes. The scope predicate is applied in SurrealQL, alongside the KNN
// operator, so a row outside the caller's scopes is never a candidate.
func (s *SurrealDBStorage) SearchSimilar(ctx context.Context, userID string, queryEmbedding []float32, limit int) ([]VectorResult, error) {
	t, err := s.scope(ctx, userID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, content, vector::similarity::cosine(embedding, $query_embedding) AS similarity,
		       metadata, created_at, updated_at
		FROM vector_memories
		%s
		ORDER BY similarity DESC
		LIMIT %d
	`, t.where(fmt.Sprintf("embedding <|%d,%d|> $query_embedding", limit, knnEF)), limit)

	result, err := s.query(ctx, query, t.params(map[string]interface{}{
		"query_embedding": normalizeEmbedding(queryEmbedding),
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to search similar vectors: %w", err)
	}

	return s.parseVectorResults(result)
}

// UpdateVector replaces a vector memory. As everywhere else the previous row
// is invalidated rather than removed, so the change feed carries both halves
// of the edit.
func (s *SurrealDBStorage) UpdateVector(ctx context.Context, id, userID, content string, embedding []float32, metadata map[string]interface{}) error {
	t, err := s.scope(ctx, userID)
	if err != nil {
		return err
	}
	if metadata == nil {
		metadata = map[string]interface{}{}
	}

	params := t.params(map[string]interface{}{
		"id":        id,
		"content":   content,
		"embedding": normalizeEmbedding(embedding),
		"metadata":  metadata,
		"user_id":   t.Principal().LegacyUserID,
	})

	invalidate := fmt.Sprintf("UPDATE vector_memories SET %s %s",
		t.invalidateSet(), t.where("id = type::record($id)"))
	if _, err := s.query(ctx, invalidate, params); err != nil {
		return fmt.Errorf("failed to invalidate the existing vector: %w", err)
	}

	create := fmt.Sprintf(`
		CREATE vector_memories CONTENT {
			%suser_id: $user_id,
			content: $content,
			embedding: $embedding,
			metadata: $metadata,
			content_hash: %s
		}
	`, t.contentFields(), vectorContentHash)
	if _, err := s.query(ctx, create, params); err != nil {
		return fmt.Errorf("failed to recreate vector: %w", err)
	}

	return nil
}

// DeleteVector invalidates a vector memory in the caller's scopes.
func (s *SurrealDBStorage) DeleteVector(ctx context.Context, id, userID string) error {
	t, err := s.scope(ctx, userID)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("UPDATE vector_memories SET %s %s",
		t.invalidateSet(), t.where("id = type::record($id)"))
	if _, err := s.query(ctx, query, t.params(map[string]interface{}{"id": id})); err != nil {
		return fmt.Errorf("failed to delete vector: %w", err)
	}

	if err := s.updateUserStat(ctx, t.Principal().LegacyUserID, "vector_count", 0); err != nil {
		// Log the error but don't fail the operation
		slog.Warn("failed to update vector_count stat", "user_id", userID, "error", err)
	}

	return nil
}

func (s *SurrealDBStorage) parseVectorResults(result *[]QueryResult) ([]VectorResult, error) {
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
				if uid := getString(itemMap, "user_id"); uid != "" {
					vectorResult.UserID = &uid
				}
				results = append(results, vectorResult)
			}
		}
	}

	return results, nil
}
