// Package storage provides code indexing storage operations for SurrealDB.
// This file contains chunk-related operations for code indexing.
package storage

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ===== CODE CHUNK OPERATIONS =====

// SaveCodeChunk saves a single code chunk with retry logic for transaction conflicts
func (s *SurrealDBStorage) SaveCodeChunk(ctx context.Context, chunk *CodeChunk) error {
	return s.withTxnRetry(ctx, func(ctx context.Context) error {
		return s.saveCodeChunkAttempt(ctx, chunk)
	})
}

// saveCodeChunkAttempt performs a single attempt to save a code chunk
func (s *SurrealDBStorage) saveCodeChunkAttempt(ctx context.Context, chunk *CodeChunk) error {
	t, err := s.scope(ctx, "")
	if err != nil {
		return err
	}

	params := t.params(map[string]interface{}{
		"symbol_id":    chunk.SymbolID,
		"project_id":   chunk.ProjectID,
		"file_path":    chunk.FilePath,
		"chunk_index":  chunk.ChunkIndex,
		"chunk_count":  chunk.ChunkCount,
		"content":      chunk.Content,
		"start_offset": chunk.StartOffset,
		"end_offset":   chunk.EndOffset,
		"embedding":    normalizeEmbedding(chunk.Embedding),
		"symbol_name":  chunk.SymbolName,
		"symbol_type":  chunk.SymbolType,
		"language":     chunk.Language,
	})

	// Check if the chunk is already stored in one of the caller's scopes.
	existsQuery := fmt.Sprintf("SELECT id FROM code_chunks %s",
		t.where("symbol_id = $symbol_id", "chunk_index = $chunk_index"))
	existsResult, err := s.query(ctx, existsQuery, params)
	if err != nil {
		return fmt.Errorf("failed to check existing chunk: %w", err)
	}

	isNewChunk := true
	if existsResult != nil && len(*existsResult) > 0 {
		queryResult := (*existsResult)[0]
		if queryResult.Status == "OK" && len(queryResult.Result) > 0 {
			isNewChunk = false
		}
	}

	if isNewChunk {
		query := fmt.Sprintf(`
			CREATE code_chunks CONTENT {
				%ssymbol_id: $symbol_id,
				project_id: $project_id,
				file_path: $file_path,
				chunk_index: $chunk_index,
				chunk_count: $chunk_count,
				content: $content,
				start_offset: $start_offset,
				end_offset: $end_offset,
				embedding: $embedding,
				symbol_name: $symbol_name,
				symbol_type: $symbol_type,
				language: $language
			}
		`, t.contentFields())
		if _, err := s.query(ctx, query, params); err != nil {
			return fmt.Errorf("failed to create chunk: %w", err)
		}
	} else {
		query := fmt.Sprintf(`
			UPDATE code_chunks SET
				%sproject_id = $project_id,
				file_path = $file_path,
				chunk_count = $chunk_count,
				content = $content,
				start_offset = $start_offset,
				end_offset = $end_offset,
				embedding = $embedding,
				symbol_name = $symbol_name,
				symbol_type = $symbol_type,
				language = $language
			%s
		`, t.updateSet(), t.where("symbol_id = $symbol_id", "chunk_index = $chunk_index"))
		if _, err := s.query(ctx, query, params); err != nil {
			return fmt.Errorf("failed to update chunk: %w", err)
		}
	}

	return nil
}

// SaveCodeChunks saves multiple code chunks in a batch
func (s *SurrealDBStorage) SaveCodeChunks(ctx context.Context, chunks []*CodeChunk) error {
	// Resolved up front so an empty batch with no principal still fails closed.
	if _, err := s.scope(ctx, ""); err != nil {
		return err
	}
	for _, chunk := range chunks {
		if err := s.SaveCodeChunk(ctx, chunk); err != nil {
			return fmt.Errorf("failed to save chunk: %w", err)
		}
	}
	return nil
}

// DeleteChunksBySymbol deletes all chunks for a symbol
func (s *SurrealDBStorage) DeleteChunksBySymbol(ctx context.Context, symbolID string) error {
	t, err := s.scope(ctx, "")
	if err != nil {
		return err
	}

	query := fmt.Sprintf("DELETE FROM code_chunks %s;", t.whereAny("symbol_id = $symbol_id"))
	params := t.params(map[string]interface{}{"symbol_id": symbolID})
	return s.withTxnRetry(ctx, func(ctx context.Context) error {
		_, err := s.query(ctx, query, params)
		return err
	})
}

// DeleteChunksByFile deletes all chunks for a file
func (s *SurrealDBStorage) DeleteChunksByFile(ctx context.Context, projectID, filePath string) error {
	t, err := s.scope(ctx, "")
	if err != nil {
		return err
	}

	query := fmt.Sprintf("DELETE FROM code_chunks %s;",
		t.whereAny("project_id = $project_id", "file_path = $file_path"))
	params := t.params(map[string]interface{}{
		"project_id": projectID,
		"file_path":  filePath,
	})

	return s.withTxnRetry(ctx, func(ctx context.Context) error {
		_, err := s.query(ctx, query, params)
		return err
	})
}

// withTxnRetry retries operations that can fail with SurrealDB read/write conflicts.
// It uses exponential backoff (20ms, 40ms, 80ms, 160ms, 320ms) for up to 5 attempts.
func (s *SurrealDBStorage) withTxnRetry(ctx context.Context, fn func(context.Context) error) error {
	const maxRetries = 5
	const baseDelay = 20 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := fn(ctx); err != nil {
			if strings.Contains(err.Error(), "read or write conflict") && attempt < maxRetries-1 {
				time.Sleep(baseDelay * time.Duration(1<<uint(attempt)))
				continue
			}
			return err
		}
		return nil
	}

	return fmt.Errorf("operation failed after %d attempts", maxRetries)
}

// GetChunksBySymbol retrieves all chunks for a symbol
func (s *SurrealDBStorage) GetChunksBySymbol(ctx context.Context, symbolID string) ([]CodeChunk, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM code_chunks %s ORDER BY chunk_index ASC;",
		t.where("symbol_id = $symbol_id"))
	result, err := s.query(ctx, query, t.params(map[string]interface{}{"symbol_id": symbolID}))
	if err != nil {
		return nil, fmt.Errorf("failed to get chunks: %w", err)
	}
	return decodeResult[CodeChunk](result)
}

// SearchChunksBySimilarity performs semantic search on code chunks
func (s *SurrealDBStorage) SearchChunksBySimilarity(ctx context.Context, projectID string, queryEmbedding []float32, limit int) ([]CodeChunkSearchResult, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT *, vector::similarity::cosine(embedding, $embedding) AS similarity
		FROM code_chunks
		%s
		ORDER BY similarity DESC
		LIMIT $limit;
	`, t.where("project_id = $project_id", "embedding != NONE"))

	params := t.params(map[string]interface{}{
		"project_id": projectID,
		"embedding":  normalizeEmbedding(queryEmbedding),
		"limit":      limit,
	})

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search chunks: %w", err)
	}

	// Decode results
	type searchResult struct {
		CodeChunk
		Similarity float64 `json:"similarity"`
	}

	results, err := decodeResult[searchResult](result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	// Convert to CodeChunkSearchResult
	searchResults := make([]CodeChunkSearchResult, len(results))
	for i, r := range results {
		chunk := r.CodeChunk
		searchResults[i] = CodeChunkSearchResult{
			Chunk:      &chunk,
			Similarity: r.Similarity,
		}
	}

	return searchResults, nil
}
