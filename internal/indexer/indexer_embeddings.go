// Package indexer provides embedding generation functionality.
package indexer

import (
	"context"
	"fmt"
	"strings"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// generateEmbeddings generates embeddings for symbols in batches
func (idx *Indexer) generateEmbeddings(ctx context.Context, symbols []*treesitter.CodeSymbol) error {
	if len(symbols) == 0 {
		return nil
	}

	// Prepare texts for embedding
	texts := make([]string, 0, len(symbols))
	symbolIdxs := make([]int, 0, len(symbols))

	for i, sym := range symbols {
		text := idx.prepareSymbolText(sym)
		if text != "" {
			texts = append(texts, text)
			symbolIdxs = append(symbolIdxs, i)
		}
	}

	if len(texts) == 0 {
		return nil
	}

	// Generate embeddings in batches
	for i := 0; i < len(texts); i += idx.config.EmbeddingBatchSize {
		end := i + idx.config.EmbeddingBatchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embeddings, err := idx.embedder.EmbedDocuments(ctx, batch)
		if err != nil {
			return fmt.Errorf("failed to generate embeddings: %w", err)
		}

		// Assign embeddings to symbols
		for j, embedding := range embeddings {
			symIdx := symbolIdxs[i+j]
			symbols[symIdx].Embedding = embedding
		}
	}

	return nil
}

// prepareSymbolText creates the text representation for embedding
func (idx *Indexer) prepareSymbolText(sym *treesitter.CodeSymbol) string {
	var parts []string

	// Add symbol type and name
	parts = append(parts, fmt.Sprintf("%s %s", sym.SymbolType, sym.Name))

	// Add signature if available
	if sym.Signature != "" {
		parts = append(parts, sym.Signature)
	}

	// Add docstring if available
	if sym.DocString != "" {
		parts = append(parts, sym.DocString)
	}

	// Add source code if available and not too long
	if sym.SourceCode != "" && len(sym.SourceCode) < 500 {
		parts = append(parts, sym.SourceCode)
	}

	return strings.Join(parts, "\n")
}

// generateChunkEmbeddings generates embeddings for chunks in batches
func (idx *Indexer) generateChunkEmbeddings(ctx context.Context, chunks []*storage.CodeChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	// Prepare texts with context
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		// Add symbol context to the chunk
		texts[i] = fmt.Sprintf("%s %s:\n%s", chunk.SymbolType, chunk.SymbolName, chunk.Content)
	}

	// Generate embeddings in batches
	for i := 0; i < len(texts); i += idx.config.EmbeddingBatchSize {
		end := i + idx.config.EmbeddingBatchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embeddings, err := idx.embedder.EmbedDocuments(ctx, batch)
		if err != nil {
			return fmt.Errorf("failed to generate embeddings: %w", err)
		}

		// Assign embeddings to chunks
		for j, embedding := range embeddings {
			chunks[i+j].Embedding = embedding
		}
	}

	return nil
}
