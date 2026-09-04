package migrations

import "sync/atomic"

// DefaultEmbeddingDimension is the vector width assumed when nothing else is
// configured. It matches nomic-embed-text-v1.5, the model Remembrances ships
// with by default.
const DefaultEmbeddingDimension = 768

// embeddingDimension is the single authority for the vector width used by every
// schema statement in this package. It is process wide because the SurrealQL
// schema is process wide: one binary talks to one datastore, and SurrealDB 3.x
// enforces `array<float, N>` as exactly N elements, so the schema and the write
// path must agree on N or every insert fails.
//
// Nothing else in the code base may hard-code a dimension; storage.SetEmbeddingDimension
// delegates here and storage.EmbeddingDimension reads back from here.
var embeddingDimension atomic.Int64

func init() {
	embeddingDimension.Store(DefaultEmbeddingDimension)
}

// SetEmbeddingDimension configures the vector width rendered into HNSW index
// definitions and `array<float, N>` field types. Values below 1 are ignored so
// that a zero-valued configuration keeps the default instead of producing an
// unusable schema.
func SetEmbeddingDimension(dim int) {
	if dim < 1 {
		return
	}
	embeddingDimension.Store(int64(dim))
}

// EmbeddingDimension returns the configured vector width.
func EmbeddingDimension() int {
	return int(embeddingDimension.Load())
}
