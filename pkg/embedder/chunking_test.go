package embedder

import (
	"strings"
	"testing"
)

func TestChunkText_NoInfiniteLoop(t *testing.T) {
	// Test case that previously caused infinite loop
	text := strings.Repeat("x", 5000)
	chunks := ChunkText(text, 1000, 200)

	// Should create ~5-6 chunks, not 200+
	if len(chunks) > 20 {
		t.Errorf("Too many chunks: got %d, want <= 20 (likely infinite loop)", len(chunks))
	}

	t.Logf("Created %d chunks from %d chars", len(chunks), len(text))
}

func TestChunkText_NormalText(t *testing.T) {
	text := strings.Repeat("Hello world. ", 200)
	chunks := ChunkText(text, 1000, 200)

	t.Logf("Normal text: %d chars -> %d chunks", len(text), len(chunks))

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}
