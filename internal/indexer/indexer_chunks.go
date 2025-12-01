// Package indexer provides chunking functionality for large symbols.
package indexer

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/embedder"
	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

const (
	// ChunkThreshold is the minimum source code length to trigger chunking
	ChunkThreshold = 1500

	// ChunkSize is the maximum size of each chunk
	ChunkSize = 1500

	// ChunkOverlap is the overlap between consecutive chunks
	ChunkOverlap = 200
)

// processLargeSymbols creates chunks for symbols larger than the threshold
func (idx *Indexer) processLargeSymbols(ctx context.Context, projectID, filePath string, symbols []*treesitter.CodeSymbol) error {
	// First, delete existing chunks for this file
	if err := idx.storage.DeleteChunksByFile(ctx, projectID, filePath); err != nil {
		log.Printf("Warning: failed to delete existing chunks: %v", err)
	}

	var allChunks []*storage.CodeChunk

	for _, sym := range symbols {
		if sym.SourceCode == "" || len(sym.SourceCode) < ChunkThreshold {
			continue
		}

		// Create chunks for this symbol
		chunks := idx.createSymbolChunks(sym, projectID, filePath)
		allChunks = append(allChunks, chunks...)
	}

	if len(allChunks) == 0 {
		return nil
	}

	// Generate embeddings for chunks
	if err := idx.generateChunkEmbeddings(ctx, allChunks); err != nil {
		return fmt.Errorf("failed to generate chunk embeddings: %w", err)
	}

	// Save chunks
	if err := idx.storage.SaveCodeChunks(ctx, allChunks); err != nil {
		return fmt.Errorf("failed to save chunks: %w", err)
	}

	log.Printf("Created %d chunks for large symbols in %s", len(allChunks), filePath)
	return nil
}

// createSymbolChunks splits a large symbol into chunks
func (idx *Indexer) createSymbolChunks(sym *treesitter.CodeSymbol, projectID, filePath string) []*storage.CodeChunk {
	sourceCode := sym.SourceCode
	chunks := embedder.ChunkText(sourceCode, ChunkSize, ChunkOverlap)

	if len(chunks) <= 1 {
		return nil // No need to chunk if only one piece
	}

	// Generate a unique symbol ID (project:file:name_path)
	symbolID := fmt.Sprintf("%s:%s:%s", projectID, filePath, sym.NamePath)

	result := make([]*storage.CodeChunk, len(chunks))
	offset := 0

	for i, chunkContent := range chunks {
		// Find actual offset in source
		startOffset := strings.Index(sourceCode[offset:], chunkContent)
		if startOffset == -1 {
			startOffset = offset
		} else {
			startOffset += offset
		}
		endOffset := startOffset + len(chunkContent)

		result[i] = &storage.CodeChunk{
			SymbolID:    symbolID,
			ProjectID:   projectID,
			FilePath:    filePath,
			ChunkIndex:  i,
			ChunkCount:  len(chunks),
			Content:     chunkContent,
			StartOffset: startOffset,
			EndOffset:   endOffset,
			SymbolName:  sym.Name,
			SymbolType:  string(sym.SymbolType),
			Language:    string(sym.Language),
		}

		offset = endOffset - ChunkOverlap
		if offset < 0 {
			offset = 0
		}
	}

	return result
}
