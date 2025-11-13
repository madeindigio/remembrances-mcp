package storage

import (
	"context"
	"fmt"
	"log"

)

// SaveDocument saves a knowledge base document
func (s *SurrealDBStorage) SaveDocument(ctx context.Context, filePath, content string, embedding []float32, metadata map[string]interface{}) error {
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

	existsQuery := "SELECT id FROM knowledge_base WHERE file_path = $file_path"
	existsResult, err := s.query(ctx, existsQuery, map[string]interface{}{
		"file_path": filePath,
	})

	isNewDocument := true
	if err != nil {
		return fmt.Errorf("failed to check existing document: %w", err)
	}

	if existsResult != nil && len(*existsResult) > 0 {
		queryResult := (*existsResult)[0]
		if queryResult.Status == "OK" && len(queryResult.Result) > 0 {
			isNewDocument = false
		}
	}

	params := map[string]interface{}{
		"file_path": filePath,
		"content":   content,
		"embedding": emb64,
		"metadata":  metadata,
	}

	if isNewDocument {
		query := `
            CREATE knowledge_base CONTENT {
                file_path: $file_path,
                content: $content,
                embedding: $embedding,
                metadata: $metadata
            }
        `
		if _, err := s.query(ctx, query, params); err != nil {
			return fmt.Errorf("failed to create document: %w", err)
		}
	} else {
		query := `
            UPDATE knowledge_base
            SET content = $content,
                embedding = $embedding,
                metadata = $metadata,
                updated_at = time::now()
            WHERE file_path = $file_path
        `
		if _, err := s.query(ctx, query, params); err != nil {
			return fmt.Errorf("failed to update document: %w", err)
		}
	}

	if isNewDocument {
		if err := s.updateUserStat(ctx, "global", "document_count", 1); err != nil {
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

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	return s.parseDocumentResults(result)
}

// DeleteDocument deletes a knowledge base document
func (s *SurrealDBStorage) DeleteDocument(ctx context.Context, filePath string) error {
	query := "DELETE FROM knowledge_base WHERE file_path = $file_path"
	params := map[string]interface{}{
		"file_path": filePath,
	}

	_, err := s.query(ctx, query, params)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if err := s.updateUserStat(ctx, "global", "document_count", -1); err != nil {
		log.Printf("Warning: failed to update document_count stat: %v", err)
	}

	return nil
}

// GetDocument retrieves a knowledge base document by file path
func (s *SurrealDBStorage) GetDocument(ctx context.Context, filePath string) (*Document, error) {
	query := "SELECT * FROM knowledge_base WHERE file_path = $file_path"
	params := map[string]interface{}{
		"file_path": filePath,
	}

	result, err := s.query(ctx, query, params)
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

func (s *SurrealDBStorage) parseDocumentResults(result *[]QueryResult) ([]DocumentResult, error) {
	var results []DocumentResult

	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" && queryResult.Result != nil {
			resultSlice := queryResult.Result
			for _, itemMap := range resultSlice {
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
					Score:      similarity,
				}
				results = append(results, documentResult)
			}
		}
	}

	return results, nil
}
