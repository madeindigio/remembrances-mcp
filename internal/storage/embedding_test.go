package storage

import (
	"testing"
)

func TestConvertEmbeddingToFloat64_Nil(t *testing.T) {
	emb := convertEmbeddingToFloat64(nil)
	if len(emb) != defaultMtreeDim {
		t.Fatalf("expected length %d, got %d", defaultMtreeDim, len(emb))
	}
	for i, v := range emb {
		if v != 0.0 {
			t.Fatalf("expected zero at index %d, got %v", i, v)
		}
	}
}

func TestConvertEmbeddingToFloat64_Pad(t *testing.T) {
	src := []float32{0.1, 0.2, 0.3}
	emb := convertEmbeddingToFloat64(src)
	if len(emb) != defaultMtreeDim {
		t.Fatalf("expected length %d, got %d", defaultMtreeDim, len(emb))
	}
	if emb[0] != float64(src[0]) || emb[1] != float64(src[1]) || emb[2] != float64(src[2]) {
		t.Fatalf("expected first values to match source: got %v", emb[:3])
	}
	for i := len(src); i < len(emb); i++ {
		if emb[i] != 0.0 {
			t.Fatalf("expected zero at padded index %d, got %v", i, emb[i])
		}
	}
}

func TestConvertEmbeddingToFloat64_Truncate(t *testing.T) {
	// Create longer than defaultMtreeDim
	src := make([]float32, defaultMtreeDim+5)
	for i := range src {
		src[i] = float32(i) * 0.01
	}
	emb := convertEmbeddingToFloat64(src)
	if len(emb) != defaultMtreeDim {
		t.Fatalf("expected length %d, got %d", defaultMtreeDim, len(emb))
	}
	for i := 0; i < defaultMtreeDim; i++ {
		if emb[i] != float64(src[i]) {
			t.Fatalf("mismatch at index %d: expected %v got %v", i, src[i], emb[i])
		}
	}
}
