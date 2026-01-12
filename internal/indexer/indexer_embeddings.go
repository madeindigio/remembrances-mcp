// Package indexer provides embedding generation functionality.
package indexer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/embedder"
	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// generateEmbeddings generates embeddings for symbols in batches
// This function continues processing even if some embeddings fail
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

	totalFailed := 0
	totalProcessed := 0

	// Generate embeddings in batches
	for i := 0; i < len(texts); i += idx.config.EmbeddingBatchSize {
		end := i + idx.config.EmbeddingBatchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embeddings, err := idx.embedder.EmbedDocuments(ctx, batch)
		if err != nil {
			// Log error but continue with next batch
			slog.Warn("Failed to generate embeddings for batch, skipping",
				"batch_start", i,
				"batch_size", len(batch),
				"error", err)
			totalFailed += len(batch)
			totalProcessed += len(batch)
			continue
		}

		// Assign embeddings to symbols (skip nil embeddings from failures)
		for j, embedding := range embeddings {
			if embedding == nil {
				// This embedding failed, log and skip
				symIdx := symbolIdxs[i+j]
				slog.Warn("Skipping nil embedding for symbol",
					"symbol", symbols[symIdx].Name,
					"type", symbols[symIdx].SymbolType)
				totalFailed++
			} else {
				symIdx := symbolIdxs[i+j]
				symbols[symIdx].Embedding = embedding
			}
			totalProcessed++
		}
	}

	// Log summary
	if totalFailed > 0 {
		slog.Warn("Some embeddings failed during generation",
			"failed_count", totalFailed,
			"total_count", totalProcessed,
			"success_rate", fmt.Sprintf("%.1f%%", float64(totalProcessed-totalFailed)/float64(totalProcessed)*100))
	}

	// Only return error if ALL embeddings failed
	if totalFailed == totalProcessed && totalProcessed > 0 {
		return fmt.Errorf("all %d embeddings failed", totalFailed)
	}

	return nil
}

// prepareSymbolText creates the text representation for embedding
func (idx *Indexer) prepareSymbolText(sym *treesitter.CodeSymbol) string {
	// Get dynamic limits from embedder (falls back to safe defaults)
	maxTextLength := 900 // Fallback default
	if ggufEmb, ok := idx.embedder.(*embedder.GGUFEmbedder); ok {
		maxTextLength = ggufEmb.MaxChars()
	}

	// Each part limited to avoid overflow when concatenated
	maxPartLength := maxTextLength / 3 // Divide by 3 to allow for 3 parts safely
	if maxPartLength < 100 {
		maxPartLength = 100 // Minimum reasonable size
	}

	var parts []string

	// Add symbol type and name (short, usually safe)
	parts = append(parts, fmt.Sprintf("%s %s", sym.SymbolType, sym.Name))

	// Add signature if available (truncate if too long)
	if sym.Signature != "" {
		sig := sym.Signature
		if len(sig) > maxPartLength {
			sig = sig[:maxPartLength]
		}
		parts = append(parts, sig)
	}

	// Add docstring if available (truncate if too long)
	if sym.DocString != "" {
		doc := sym.DocString
		if len(doc) > maxPartLength {
			doc = doc[:maxPartLength]
		}
		parts = append(parts, doc)
	}

	// Add source code if available and not too long (truncate if needed)
	if sym.SourceCode != "" && len(sym.SourceCode) < maxPartLength {
		parts = append(parts, sym.SourceCode)
	}

	text := strings.Join(parts, "\n")

	// Final safety check: truncate if still too long
	if len(text) > maxTextLength {
		text = text[:maxTextLength]
	}

	return text
}

// generateChunkEmbeddings generates embeddings for chunks in batches
// This function continues processing even if some embeddings fail
func (idx *Indexer) generateChunkEmbeddings(ctx context.Context, chunks []*storage.CodeChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	// Get dynamic limits from embedder (falls back to safe defaults)
	maxTextLength := 900 // Fallback default
	if ggufEmb, ok := idx.embedder.(*embedder.GGUFEmbedder); ok {
		maxTextLength = ggufEmb.MaxChars()
	}

	// Prepare texts with context
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		// Truncate chunk content first to avoid overflow
		// Leave ~50 chars for symbol type and name
		maxContentLength := maxTextLength - 50
		if maxContentLength < 100 {
			maxContentLength = 100
		}

		content := chunk.Content
		if len(content) > maxContentLength {
			content = content[:maxContentLength]
		}

		// Add symbol context to the chunk
		text := fmt.Sprintf("%s %s:\n%s", chunk.SymbolType, chunk.SymbolName, content)

		// Final safety check: truncate if still too long
		if len(text) > maxTextLength {
			text = text[:maxTextLength]
		}

		texts[i] = text
	}

	totalFailed := 0
	totalProcessed := 0

	// Generate embeddings in batches
	for i := 0; i < len(texts); i += idx.config.EmbeddingBatchSize {
		end := i + idx.config.EmbeddingBatchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embeddings, err := idx.embedder.EmbedDocuments(ctx, batch)
		if err != nil {
			// Log error but continue with next batch
			slog.Warn("Failed to generate chunk embeddings for batch, skipping",
				"batch_start", i,
				"batch_size", len(batch),
				"error", err)
			totalFailed += len(batch)
			totalProcessed += len(batch)
			continue
		}

		// Assign embeddings to chunks (skip nil embeddings from failures)
		for j, embedding := range embeddings {
			if embedding == nil {
				// This embedding failed, log and skip
				slog.Warn("Skipping nil embedding for chunk",
					"chunk_index", i+j,
					"symbol", chunks[i+j].SymbolName)
				totalFailed++
			} else {
				chunks[i+j].Embedding = embedding
			}
			totalProcessed++
		}
	}

	// Log summary
	if totalFailed > 0 {
		slog.Warn("Some chunk embeddings failed during generation",
			"failed_count", totalFailed,
			"total_count", totalProcessed,
			"success_rate", fmt.Sprintf("%.1f%%", float64(totalProcessed-totalFailed)/float64(totalProcessed)*100))
	}

	// Only return error if ALL embeddings failed
	if totalFailed == totalProcessed && totalProcessed > 0 {
		return fmt.Errorf("all %d chunk embeddings failed", totalFailed)
	}

	return nil
}
