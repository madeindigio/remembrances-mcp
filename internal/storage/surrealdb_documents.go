package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/madeindigio/remembrances-mcp/v2/internal/storage/migrations"
)

// The knowledge base surface, scoped per RM-3.
//
// These methods had no tenant filter at all: SearchDocuments searched every
// document in the datastore and knowledge_base.file_path was globally unique,
// so two tenants could not even hold a file of the same name. Uniqueness is
// now (org_id, scope, file_path, valid_to) and every read carries the scope
// predicate.

// documentContentHash fills content_hash on knowledge_base rows.
var documentContentHash = migrations.ContentHashSurrealExpr("$content")

// SaveDocument saves a knowledge base document in the caller's write scope,
// replacing any live document with the same path.
func (s *SurrealDBStorage) SaveDocument(ctx context.Context, filePath, content string, embedding []float32, metadata map[string]interface{}) error {
	t, err := s.scope(ctx, "")
	if err != nil {
		return err
	}
	if metadata == nil {
		metadata = map[string]interface{}{}
	}

	params := t.params(map[string]interface{}{
		"file_path": filePath,
		"content":   content,
		"embedding": normalizeEmbedding(embedding),
		"metadata":  metadata,
		"user_id":   t.Principal().LegacyUserID,
	})

	invalidate := fmt.Sprintf("UPDATE knowledge_base SET %s %s",
		t.invalidateSet(), t.where("file_path = $file_path"))
	if _, err := s.query(ctx, invalidate, params); err != nil {
		return fmt.Errorf("failed to invalidate the existing document: %w", err)
	}

	create := fmt.Sprintf(`
		CREATE knowledge_base CONTENT {
			%suser_id: $user_id,
			file_path: $file_path,
			content: $content,
			embedding: $embedding,
			metadata: $metadata,
			content_hash: %s
		}
	`, t.contentFields(), documentContentHash)
	if _, err := s.query(ctx, create, params); err != nil {
		return fmt.Errorf("failed to create document: %w", err)
	}

	if err := s.updateUserStat(ctx, t.Principal().LegacyUserID, "document_count", 1); err != nil {
		slog.Warn("failed to update document_count stat", "error", err)
	}

	return nil
}

// SearchDocuments performs similarity search over the documents the caller may
// read. Passing no scope means the caller's visible scopes, never all scopes.
func (s *SurrealDBStorage) SearchDocuments(ctx context.Context, queryEmbedding []float32, limit int) ([]DocumentResult, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, file_path, content, embedding, metadata, created_at, updated_at,
		       vector::similarity::cosine(embedding, $query_embedding) AS similarity
		FROM knowledge_base
		%s
		ORDER BY similarity DESC
		LIMIT %d
	`, t.where(fmt.Sprintf("embedding <|%d,%d|> $query_embedding", limit, knnEF)), limit)

	result, err := s.query(ctx, query, t.params(map[string]interface{}{
		"query_embedding": normalizeEmbedding(queryEmbedding),
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	return s.parseDocumentResults(result)
}

// DeleteDocument invalidates a document and all of its chunks.
func (s *SurrealDBStorage) DeleteDocument(ctx context.Context, filePath string) error {
	t, err := s.scope(ctx, "")
	if err != nil {
		return err
	}

	query := fmt.Sprintf("UPDATE knowledge_base SET %s %s", t.invalidateSet(),
		t.where("(source_file = $file_path OR file_path = $file_path)"))
	if _, err := s.query(ctx, query, t.params(map[string]interface{}{"file_path": filePath})); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if err := s.updateUserStat(ctx, t.Principal().LegacyUserID, "document_count", -1); err != nil {
		slog.Warn("failed to update document_count stat", "error", err)
	}

	return nil
}

// GetDocument retrieves a live knowledge base document by file path, from the
// caller's visible scopes.
func (s *SurrealDBStorage) GetDocument(ctx context.Context, filePath string) (*Document, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}

	// Try to find by source_file first (for chunked documents), then by
	// file_path. Ordering by chunk_index returns the first chunk.
	query := fmt.Sprintf("SELECT * FROM knowledge_base %s ORDER BY chunk_index ASC LIMIT 1",
		t.where("(source_file = $file_path OR file_path = $file_path)"))
	result, err := s.query(ctx, query, t.params(map[string]interface{}{"file_path": filePath}))
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if result == nil || len(*result) == 0 {
		return nil, nil
	}

	queryResult := (*result)[0]
	if queryResult.Status != "OK" || len(queryResult.Result) == 0 {
		return nil, nil
	}

	return documentFromRow(queryResult.Result[0]), nil
}

// documentFromRow decodes one knowledge_base row.
func documentFromRow(row map[string]interface{}) *Document {
	doc := &Document{
		ID:        getString(row, "id"),
		FilePath:  getString(row, "file_path"),
		Content:   getString(row, "content"),
		Embedding: embeddingFromRow(row["embedding"]),
		Metadata:  getMap(row, "metadata"),
		CreatedAt: getTime(row, "created_at"),
		UpdatedAt: getTime(row, "updated_at"),
	}
	if uid := getString(row, "user_id"); uid != "" {
		doc.UserID = &uid
	}
	return doc
}

// embeddingFromRow converts a decoded SurrealDB float array back to []float32.
func embeddingFromRow(raw interface{}) []float32 {
	slice, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	out := make([]float32, len(slice))
	for i, v := range slice {
		switch f := v.(type) {
		case float64:
			out[i] = float32(f)
		case float32:
			out[i] = f
		}
	}
	return out
}

func (s *SurrealDBStorage) parseDocumentResults(result *[]QueryResult) ([]DocumentResult, error) {
	var results []DocumentResult

	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" && queryResult.Result != nil {
			for _, itemMap := range queryResult.Result {
				similarity := getFloat64(itemMap, "similarity")
				results = append(results, DocumentResult{
					Document:   documentFromRow(itemMap),
					Similarity: similarity,
					Score:      similarity,
				})
			}
		}
	}

	return results, nil
}

// SaveDocumentChunks saves a document as individual chunks with embeddings, in
// the caller's write scope.
func (s *SurrealDBStorage) SaveDocumentChunks(ctx context.Context, filePath string, chunks []string, embeddings [][]float32, metadata map[string]interface{}) error {
	// The principal is resolved before the arguments are validated, so that a
	// call with no identity fails closed whatever else is wrong with it.
	t, err := s.scope(ctx, "")
	if err != nil {
		return err
	}
	if len(chunks) == 0 {
		return fmt.Errorf("no chunks provided")
	}
	if len(chunks) != len(embeddings) {
		return fmt.Errorf("chunks and embeddings length mismatch: %d chunks vs %d embeddings", len(chunks), len(embeddings))
	}
	if metadata == nil {
		metadata = map[string]interface{}{}
	}

	// Invalidate the previous generation of chunks first, so the unique
	// (org_id, scope, file_path, valid_to) key is free for the new ones.
	invalidate := fmt.Sprintf("UPDATE knowledge_base SET %s %s", t.invalidateSet(),
		t.where("(source_file = $file_path OR file_path = $file_path)"))
	if err := s.withTxnRetry(ctx, func(ctx context.Context) error {
		_, err := s.query(ctx, invalidate, t.params(map[string]interface{}{"file_path": filePath}))
		return err
	}); err != nil {
		return fmt.Errorf("failed to invalidate existing chunks: %w", err)
	}

	chunkCount := len(chunks)
	create := fmt.Sprintf(`
		CREATE knowledge_base CONTENT {
			%suser_id: $user_id,
			file_path: $file_path,
			content: $content,
			embedding: $embedding,
			metadata: $metadata,
			chunk_index: $chunk_index,
			chunk_count: $chunk_count,
			source_file: $source_file,
			content_hash: %s
		}
	`, t.contentFields(), documentContentHash)

	for i, chunk := range chunks {
		// Chunk metadata is the document metadata plus the chunk coordinates.
		chunkMetadata := make(map[string]interface{}, len(metadata)+2)
		for k, v := range metadata {
			chunkMetadata[k] = v
		}
		chunkMetadata["chunk_index"] = i
		chunkMetadata["chunk_count"] = chunkCount

		params := t.params(map[string]interface{}{
			// A unique file_path per chunk keeps the uniqueness index usable.
			"file_path":   fmt.Sprintf("%s#chunk%d", filePath, i),
			"content":     chunk,
			"embedding":   normalizeEmbedding(embeddings[i]),
			"metadata":    chunkMetadata,
			"chunk_index": i,
			"chunk_count": chunkCount,
			"source_file": filePath,
			"user_id":     t.Principal().LegacyUserID,
		})

		if _, err := s.query(ctx, create, params); err != nil {
			return fmt.Errorf("failed to create chunk %d: %w", i, err)
		}
	}

	// Update document count stat (count by source_file, not by chunks)
	if err := s.updateUserStat(ctx, t.Principal().LegacyUserID, "document_count", 1); err != nil {
		slog.Warn("failed to update document_count stat", "error", err)
	}

	return nil
}
