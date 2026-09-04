package storage

import "testing"

// TestNormalizeEmbeddingFollowsTheConfiguredDimension asserts there is one
// dimension authority: changing it changes what the write path produces, so the
// vectors and the array<float, N> schema can never disagree.
func TestNormalizeEmbeddingFollowsTheConfiguredDimension(t *testing.T) {
	original := EmbeddingDimension()
	defer SetEmbeddingDimension(original)

	for _, dim := range []int{384, 768, 1024} {
		SetEmbeddingDimension(dim)

		if got := len(normalizeEmbedding(nil)); got != dim {
			t.Errorf("a nil embedding must become %d zeros, got %d", dim, got)
		}

		short := []float32{0.1, 0.2, 0.3}
		padded := normalizeEmbedding(short)
		if len(padded) != dim {
			t.Fatalf("expected %d elements, got %d", dim, len(padded))
		}
		for i, v := range short {
			if padded[i] != float64(v) {
				t.Errorf("value %d changed during padding: %v != %v", i, padded[i], v)
			}
		}
		for i := len(short); i < dim; i++ {
			if padded[i] != 0 {
				t.Errorf("padding must be zero, got %v at %d", padded[i], i)
			}
		}

		long := make([]float32, dim+16)
		for i := range long {
			long[i] = float32(i) * 0.01
		}
		truncated := normalizeEmbedding(long)
		if len(truncated) != dim {
			t.Fatalf("expected %d elements after truncation, got %d", dim, len(truncated))
		}
		for i := 0; i < dim; i++ {
			if truncated[i] != float64(long[i]) {
				t.Errorf("value %d changed during truncation", i)
			}
		}
	}
}

// TestSetEmbeddingDimensionIgnoresNonsense asserts a zero-valued configuration
// keeps the default rather than producing an unusable schema.
func TestSetEmbeddingDimensionIgnoresNonsense(t *testing.T) {
	original := EmbeddingDimension()
	defer SetEmbeddingDimension(original)

	SetEmbeddingDimension(512)
	SetEmbeddingDimension(0)
	SetEmbeddingDimension(-1)
	if got := EmbeddingDimension(); got != 512 {
		t.Fatalf("expected the dimension to stay at 512, got %d", got)
	}
}
