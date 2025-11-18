package embedder

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestGGUFEmbedder tests the GGUF embedder with a real model file.
// This test requires a GGUF model file to be present.
// Set GGUF_TEST_MODEL_PATH environment variable to run this test.
func TestGGUFEmbedder(t *testing.T) {
	modelPath := os.Getenv("GGUF_TEST_MODEL_PATH")
	if modelPath == "" {
		t.Skip("Skipping GGUF test: GGUF_TEST_MODEL_PATH not set")
	}

	// Check if model file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Fatalf("Model file not found: %s", modelPath)
	}

	t.Logf("Testing with model: %s", modelPath)

	// Create embedder
	embedder, err := NewGGUFEmbedder(modelPath, 4, 0)
	if err != nil {
		t.Fatalf("Failed to create GGUF embedder: %v", err)
	}
	defer embedder.Close()

	ctx := context.Background()

	// Test single embedding
	t.Run("SingleEmbedding", func(t *testing.T) {
		text := "Hello, world!"
		embedding, err := embedder.EmbedQuery(ctx, text)
		if err != nil {
			t.Fatalf("Failed to embed query: %v", err)
		}

		if len(embedding) == 0 {
			t.Fatal("Embedding is empty")
		}

		t.Logf("Embedding dimension: %d", len(embedding))
		t.Logf("First 5 values: %v", embedding[:min(5, len(embedding))])
	})

	// Test batch embeddings
	t.Run("BatchEmbeddings", func(t *testing.T) {
		texts := []string{
			"The quick brown fox jumps over the lazy dog",
			"Machine learning is fascinating",
			"Go is a great programming language",
		}

		embeddings, err := embedder.EmbedDocuments(ctx, texts)
		if err != nil {
			t.Fatalf("Failed to embed documents: %v", err)
		}

		if len(embeddings) != len(texts) {
			t.Fatalf("Expected %d embeddings, got %d", len(texts), len(embeddings))
		}

		for i, emb := range embeddings {
			if len(emb) == 0 {
				t.Fatalf("Embedding %d is empty", i)
			}
			t.Logf("Embedding %d dimension: %d", i, len(emb))
		}
	})

	// Test dimension consistency
	t.Run("DimensionConsistency", func(t *testing.T) {
		dim := embedder.Dimension()
		if dim == 0 {
			t.Fatal("Dimension is 0")
		}

		text := "Test text for dimension check"
		embedding, err := embedder.EmbedQuery(ctx, text)
		if err != nil {
			t.Fatalf("Failed to embed query: %v", err)
		}

		if len(embedding) != dim {
			t.Fatalf("Expected dimension %d, got %d", dim, len(embedding))
		}

		t.Logf("Dimension: %d", dim)
	})

	// Test empty text handling
	t.Run("EmptyText", func(t *testing.T) {
		_, err := embedder.EmbedQuery(ctx, "")
		if err == nil {
			t.Fatal("Expected error for empty text, got nil")
		}
		t.Logf("Correctly rejected empty text: %v", err)
	})

	// Test context cancellation
	t.Run("ContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := embedder.EmbedQuery(ctx, "Test text")
		if err == nil {
			t.Log("Warning: Context cancellation not properly handled")
		} else {
			t.Logf("Context cancellation handled: %v", err)
		}
	})
}

// TestGGUFConfig tests the configuration creation.
func TestGGUFConfig(t *testing.T) {
	cfg := GGUFConfig{
		ModelPath: "/path/to/model.gguf",
		Threads:   8,
		GPULayers: 32,
	}

	if cfg.ModelPath == "" {
		t.Fatal("ModelPath is empty")
	}

	if cfg.Threads != 8 {
		t.Fatalf("Expected Threads=8, got %d", cfg.Threads)
	}

	if cfg.GPULayers != 32 {
		t.Fatalf("Expected GPULayers=32, got %d", cfg.GPULayers)
	}
}

// TestGGUFEmbedderFromConfig tests creating embedder from config.
func TestGGUFEmbedderFromConfig(t *testing.T) {
	modelPath := os.Getenv("GGUF_TEST_MODEL_PATH")
	if modelPath == "" {
		t.Skip("Skipping GGUF config test: GGUF_TEST_MODEL_PATH not set")
	}

	cfg := GGUFConfig{
		ModelPath: modelPath,
		Threads:   4,
		GPULayers: 0,
	}

	embedder, err := NewGGUFEmbedderFromConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create embedder from config: %v", err)
	}
	defer embedder.Close()

	if embedder.ModelPath() != modelPath {
		t.Fatalf("Expected model path %s, got %s", modelPath, embedder.ModelPath())
	}

	if embedder.Threads() != 4 {
		t.Fatalf("Expected 4 threads, got %d", embedder.Threads())
	}

	if embedder.GPULayers() != 0 {
		t.Fatalf("Expected 0 GPU layers, got %d", embedder.GPULayers())
	}
}

// TestGGUFEmbedderInvalidModel tests error handling for invalid model path.
func TestGGUFEmbedderInvalidModel(t *testing.T) {
	_, err := NewGGUFEmbedder("/nonexistent/model.gguf", 4, 0)
	if err == nil {
		t.Fatal("Expected error for nonexistent model, got nil")
	}
	t.Logf("Correctly rejected invalid model: %v", err)
}

// TestGGUFEmbedderEmptyPath tests error handling for empty model path.
func TestGGUFEmbedderEmptyPath(t *testing.T) {
	_, err := NewGGUFEmbedder("", 4, 0)
	if err == nil {
		t.Fatal("Expected error for empty model path, got nil")
	}
	t.Logf("Correctly rejected empty path: %v", err)
}

// BenchmarkGGUFEmbedder benchmarks the GGUF embedder performance.
func BenchmarkGGUFEmbedder(b *testing.B) {
	modelPath := os.Getenv("GGUF_TEST_MODEL_PATH")
	if modelPath == "" {
		b.Skip("Skipping GGUF benchmark: GGUF_TEST_MODEL_PATH not set")
	}

	embedder, err := NewGGUFEmbedder(modelPath, 8, 0)
	if err != nil {
		b.Fatalf("Failed to create embedder: %v", err)
	}
	defer embedder.Close()

	ctx := context.Background()
	text := "The quick brown fox jumps over the lazy dog"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := embedder.EmbedQuery(ctx, text)
		if err != nil {
			b.Fatalf("Failed to embed: %v", err)
		}
	}
}

// BenchmarkGGUFEmbedderBatch benchmarks batch embedding performance.
func BenchmarkGGUFEmbedderBatch(b *testing.B) {
	modelPath := os.Getenv("GGUF_TEST_MODEL_PATH")
	if modelPath == "" {
		b.Skip("Skipping GGUF batch benchmark: GGUF_TEST_MODEL_PATH not set")
	}

	embedder, err := NewGGUFEmbedder(modelPath, 8, 0)
	if err != nil {
		b.Fatalf("Failed to create embedder: %v", err)
	}
	defer embedder.Close()

	ctx := context.Background()
	texts := []string{
		"The quick brown fox jumps over the lazy dog",
		"Machine learning is fascinating",
		"Go is a great programming language",
		"Embeddings are useful for semantic search",
		"Natural language processing with transformers",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := embedder.EmbedDocuments(ctx, texts)
		if err != nil {
			b.Fatalf("Failed to embed batch: %v", err)
		}
	}
}

// ExampleGGUFEmbedder demonstrates how to use the GGUF embedder.
func ExampleGGUFEmbedder() {
	// Create embedder with model file
	embedder, err := NewGGUFEmbedder("/path/to/model.gguf", 8, 32)
	if err != nil {
		panic(err)
	}
	defer embedder.Close()

	// Generate embedding for a single query
	ctx := context.Background()
	embedding, err := embedder.EmbedQuery(ctx, "Hello, world!")
	if err != nil {
		panic(err)
	}

	println("Embedding dimension:", len(embedding))

	// Generate embeddings for multiple documents
	texts := []string{
		"First document",
		"Second document",
		"Third document",
	}
	embeddings, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		panic(err)
	}

	println("Generated", len(embeddings), "embeddings")
}

// Helper function for min (Go 1.21+)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Example test runner script
func init() {
	// Print helpful message when running tests
	if os.Getenv("GGUF_TEST_MODEL_PATH") == "" {
		// Check common model locations
		commonPaths := []string{
			"/www/Remembrances/nomic-embed-text-v1.5.Q4_K_M.gguf",
			filepath.Join(os.Getenv("HOME"), "models/nomic-embed-text-v1.5.Q4_K_M.gguf"),
			"./models/nomic-embed-text-v1.5.Q4_K_M.gguf",
		}

		for _, path := range commonPaths {
			if _, err := os.Stat(path); err == nil {
				os.Setenv("GGUF_TEST_MODEL_PATH", path)
				return
			}
		}
	}
}
