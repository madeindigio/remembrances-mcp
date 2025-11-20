package embedder

import (
	"context"
	"strings"
	"unicode"
)

const (
	// DefaultMaxChunkSize is the default maximum number of characters per chunk.
	// This is a conservative estimate that should work for most embedding models.
	// Most models support 512-2048 tokens, and ~4 chars per token is a safe estimate.
	DefaultMaxChunkSize = 1500 // ~375 tokens with 4 chars/token ratio

	// DefaultChunkOverlap is the default overlap between consecutive chunks
	DefaultChunkOverlap = 200
)

// ChunkText splits a text into smaller chunks suitable for embedding models.
// It attempts to split on sentence boundaries when possible.
func ChunkText(text string, maxChunkSize, overlap int) []string {
	if maxChunkSize <= 0 {
		maxChunkSize = DefaultMaxChunkSize
	}
	if overlap < 0 {
		overlap = DefaultChunkOverlap
	}
	if overlap >= maxChunkSize {
		overlap = maxChunkSize / 4 // Overlap should be less than chunk size
	}

	// If text is smaller than max chunk size, return as-is
	if len(text) <= maxChunkSize {
		return []string{text}
	}

	var chunks []string
	start := 0
	textLen := len(text)
	lastEnd := -1 // Track the last end position to detect infinite loops

	for start < textLen {
		end := start + maxChunkSize
		if end > textLen {
			end = textLen
		}

		// Try to find a good breaking point (sentence end)
		if end < textLen {
			// Look for sentence terminators: . ! ? followed by space or newline
			breakPoint := findSentenceBreak(text, start, end)
			if breakPoint > start {
				end = breakPoint
			} else {
				// If no sentence break found, try to break at word boundary
				breakPoint = findWordBreak(text, start, end)
				if breakPoint > start {
					end = breakPoint
				}
			}
		}

		// Detect infinite loop: if end position hasn't changed, force progress
		if end == lastEnd {
			// Force move forward by at least maxChunkSize to avoid getting stuck
			end = start + maxChunkSize
			if end > textLen {
				end = textLen
			}
		}
		lastEnd = end

		chunk := strings.TrimSpace(text[start:end])
		if chunk != "" {
			chunks = append(chunks, chunk)
		}

		// If we've reached the end of text, we're done
		if end >= textLen {
			break
		}

		// Move start forward, accounting for overlap
		newStart := end - overlap

		// Ensure we're making progress - start must advance
		// If newStart would go backwards or stay the same, skip overlap and continue from end
		if newStart <= start {
			newStart = end
		}

		start = newStart
	}

	return chunks
}

// findSentenceBreak looks for a sentence terminator (. ! ?) followed by whitespace
// within the range [start, end] of the text, searching backwards from end.
func findSentenceBreak(text string, start, end int) int {
	for i := end - 1; i > start; i-- {
		if i >= len(text) {
			continue
		}
		ch := text[i]
		if ch == '.' || ch == '!' || ch == '?' {
			// Check if followed by whitespace or end of text
			if i+1 >= len(text) || unicode.IsSpace(rune(text[i+1])) {
				return i + 1
			}
		}
	}
	return -1
}

// findWordBreak looks for a whitespace character within the range [start, end]
// of the text, searching backwards from end.
func findWordBreak(text string, start, end int) int {
	for i := end - 1; i > start; i-- {
		if i >= len(text) {
			continue
		}
		if unicode.IsSpace(rune(text[i])) {
			return i + 1
		}
	}
	return -1
}

// AverageEmbeddings computes the average of multiple embeddings.
// This is useful for combining embeddings from multiple text chunks.
func AverageEmbeddings(embeddings [][]float32) []float32 {
	if len(embeddings) == 0 {
		return nil
	}
	if len(embeddings) == 1 {
		return embeddings[0]
	}

	dim := len(embeddings[0])
	result := make([]float32, dim)

	for _, emb := range embeddings {
		for i, val := range emb {
			result[i] += val
		}
	}

	// Divide by count to get average
	count := float32(len(embeddings))
	for i := range result {
		result[i] /= count
	}

	return result
}

// EmbedLongText is a helper function that chunks text if needed and returns
// a single averaged embedding. This should be used for texts that may exceed
// the model's context window.
func EmbedLongText(ctx context.Context, embedder Embedder, text string, maxChunkSize int) ([]float32, error) {
	if maxChunkSize <= 0 {
		maxChunkSize = DefaultMaxChunkSize
	}

	// If text is short enough, embed directly
	if len(text) <= maxChunkSize {
		return embedder.EmbedQuery(ctx, text)
	}

	// Chunk the text
	chunks := ChunkText(text, maxChunkSize, DefaultChunkOverlap)
	if len(chunks) == 0 {
		return embedder.EmbedQuery(ctx, text)
	}

	// Embed each chunk
	embeddings := make([][]float32, len(chunks))
	for i, chunk := range chunks {
		emb, err := embedder.EmbedQuery(ctx, chunk)
		if err != nil {
			return nil, err
		}
		embeddings[i] = emb
	}

	// Return averaged embedding
	return AverageEmbeddings(embeddings), nil
}

// EmbedTextChunksWithOverlap chunks text and returns both the chunks and their individual embeddings.
// This allows storing each chunk separately for more precise retrieval.
func EmbedTextChunksWithOverlap(ctx context.Context, embedder Embedder, text string, maxChunkSize, overlap int) ([]string, [][]float32, error) {
	if maxChunkSize <= 0 {
		maxChunkSize = DefaultMaxChunkSize
	}
	if overlap < 0 {
		overlap = DefaultChunkOverlap
	}

	// Chunk the text with specified overlap
	chunks := ChunkText(text, maxChunkSize, overlap)
	if len(chunks) == 0 {
		// If no chunks, embed the whole text and return as single chunk
		emb, err := embedder.EmbedQuery(ctx, text)
		if err != nil {
			return nil, nil, err
		}
		return []string{text}, [][]float32{emb}, nil
	}

	// Embed each chunk
	embeddings := make([][]float32, len(chunks))
	for i, chunk := range chunks {
		emb, err := embedder.EmbedQuery(ctx, chunk)
		if err != nil {
			return nil, nil, err
		}
		embeddings[i] = emb
	}

	return chunks, embeddings, nil
}

// EmbedLongTextWithOverlap is similar to EmbedLongText but allows specifying both
// chunk size and overlap. This provides more control over how text is split.
func EmbedLongTextWithOverlap(ctx context.Context, embedder Embedder, text string, maxChunkSize, overlap int) ([]float32, error) {
	if maxChunkSize <= 0 {
		maxChunkSize = DefaultMaxChunkSize
	}
	if overlap < 0 {
		overlap = DefaultChunkOverlap
	}

	// If text is short enough, embed directly
	if len(text) <= maxChunkSize {
		return embedder.EmbedQuery(ctx, text)
	}

	// Chunk the text with specified overlap
	chunks := ChunkText(text, maxChunkSize, overlap)
	if len(chunks) == 0 {
		return embedder.EmbedQuery(ctx, text)
	}

	// Embed each chunk
	embeddings := make([][]float32, len(chunks))
	for i, chunk := range chunks {
		emb, err := embedder.EmbedQuery(ctx, chunk)
		if err != nil {
			return nil, err
		}
		embeddings[i] = emb
	}

	// Return averaged embedding
	return AverageEmbeddings(embeddings), nil
}
