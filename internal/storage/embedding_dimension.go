package storage

import (
	"github.com/madeindigio/remembrances-mcp/v2/internal/storage/migrations"
)

// DefaultEmbeddingDimension is the embedding width assumed when nothing else is
// configured. It matches nomic-embed-text-v1.5, the model shipped by default.
const DefaultEmbeddingDimension = migrations.DefaultEmbeddingDimension

// SetEmbeddingDimension configures the vector width used by the schema (HNSW
// indexes and array<float, N> field types) and by embedding normalisation.
// Values below 1 are ignored so a zero-valued configuration keeps the default.
//
// Call it before InitializeSchema. Changing the dimension of a populated
// datastore requires re-embedding: SurrealDB 3.x reads array<float, N> as
// exactly N elements, so vectors written at the old width no longer coerce.
func SetEmbeddingDimension(dim int) {
	migrations.SetEmbeddingDimension(dim)
}

// EmbeddingDimension returns the configured vector width. This is the single
// dimension authority for the storage layer: nothing else may hard-code 768.
func EmbeddingDimension() int {
	return migrations.EmbeddingDimension()
}

// normalizeEmbedding pads with zeros or truncates so the vector always matches
// the configured dimension, and converts to []float64 for SurrealDB's JSON
// numeric handling. Under SurrealDB 3.x array<float, N> means exactly N
// elements, so this is a correctness requirement rather than a convenience: a
// vector of the wrong width is rejected at write time.
func normalizeEmbedding(embedding []float32) []float64 {
	dim := EmbeddingDimension()
	out := make([]float64, dim)
	for i := 0; i < dim && i < len(embedding); i++ {
		out[i] = float64(embedding[i])
	}
	return out
}
