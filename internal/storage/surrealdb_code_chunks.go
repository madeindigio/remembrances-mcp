// Package storage provides code indexing storage operations for SurrealDB.
// This file contains chunk-related operations for code indexing.
package storage

import (
	"context"
	"fmt"
)

// ===== CODE CHUNK OPERATIONS =====

// SaveCodeChunk saves a single code chunk
func (s *SurrealDBStorage) SaveCodeChunk(ctx context.Context, chunk *CodeChunk) error {
	query := `
		UPSERT code_chunks SET
			symbol_id = $symbol_id,
			project_id = $project_id,
			file_path = $file_path,
			chunk_index = $chunk_index,
			chunk_count = $chunk_count,
			content = $content,
			start_offset = $start_offset,
			end_offset = $end_offset,
			embedding = $embedding,
			symbol_name = $symbol_name,
			symbol_type = $symbol_type,
			language = $language
		WHERE symbol_id = $symbol_id AND chunk_index = $chunk_index;
	`

	params := map[string]interface{}{
		"symbol_id":    chunk.SymbolID,
		"project_id":   chunk.ProjectID,
		"file_path":    chunk.FilePath,
		"chunk_index":  chunk.ChunkIndex,
		"chunk_count":  chunk.ChunkCount,
		"content":      chunk.Content,
		"start_offset": chunk.StartOffset,
		"end_offset":   chunk.EndOffset,
		"embedding":    chunk.Embedding,
		"symbol_name":  chunk.SymbolName,
		"symbol_type":  chunk.SymbolType,
		"language":     chunk.Language,
	}

	_, err := s.query(ctx, query, params)
	return err
}

// SaveCodeChunks saves multiple code chunks in a batch
func (s *SurrealDBStorage) SaveCodeChunks(ctx context.Context, chunks []*CodeChunk) error {
	for _, chunk := range chunks {
		if err := s.SaveCodeChunk(ctx, chunk); err != nil {
			return fmt.Errorf("failed to save chunk: %w", err)
		}
	}
	return nil
}

// DeleteChunksBySymbol deletes all chunks for a symbol
func (s *SurrealDBStorage) DeleteChunksBySymbol(ctx context.Context, symbolID string) error {
	query := `DELETE FROM code_chunks WHERE symbol_id = $symbol_id;`
	_, err := s.query(ctx, query, map[string]interface{}{"symbol_id": symbolID})
	return err
}

// DeleteChunksByFile deletes all chunks for a file
func (s *SurrealDBStorage) DeleteChunksByFile(ctx context.Context, projectID, filePath string) error {
	query := `DELETE FROM code_chunks WHERE project_id = $project_id AND file_path = $file_path;`
	_, err := s.query(ctx, query, map[string]interface{}{
		"project_id": projectID,
		"file_path":  filePath,
	})
	return err
}

// GetChunksBySymbol retrieves all chunks for a symbol
func (s *SurrealDBStorage) GetChunksBySymbol(ctx context.Context, symbolID string) ([]CodeChunk, error) {
	query := `SELECT * FROM code_chunks WHERE symbol_id = $symbol_id ORDER BY chunk_index ASC;`
	result, err := s.query(ctx, query, map[string]interface{}{"symbol_id": symbolID})
	if err != nil {
		return nil, fmt.Errorf("failed to get chunks: %w", err)
	}
	return decodeResult[CodeChunk](result)
}

// SearchChunksBySimilarity performs semantic search on code chunks
func (s *SurrealDBStorage) SearchChunksBySimilarity(ctx context.Context, projectID string, queryEmbedding []float32, limit int) ([]CodeChunkSearchResult, error) {
	query := `
		SELECT *, vector::similarity::cosine(embedding, $embedding) AS similarity 
		FROM code_chunks 
		WHERE project_id = $project_id 
		AND embedding != NONE
		ORDER BY similarity DESC
		LIMIT $limit;
	`

	params := map[string]interface{}{
		"project_id": projectID,
		"embedding":  queryEmbedding,
		"limit":      limit,
	}

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
