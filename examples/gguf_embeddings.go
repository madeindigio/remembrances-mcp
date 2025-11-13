package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/madeindigio/remembrances-mcp/pkg/embedder"
)

func main() {
	// Parse command-line flags
	modelPath := flag.String("model", "", "Path to GGUF model file (required)")
	threads := flag.Int("threads", 8, "Number of threads to use")
	gpuLayers := flag.Int("gpu-layers", 0, "Number of GPU layers (0 = CPU only)")
	text := flag.String("text", "Hello, world!", "Text to embed")
	benchmark := flag.Bool("benchmark", false, "Run benchmark")
	flag.Parse()

	if *modelPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --model flag is required")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example:")
		fmt.Fprintln(os.Stderr, "  go run examples/gguf_embeddings.go --model /path/to/model.gguf --text \"Hello world\"")
		os.Exit(1)
	}

	// Create GGUF embedder
	fmt.Printf("Loading GGUF model: %s\n", *modelPath)
	fmt.Printf("Configuration: threads=%d, gpu_layers=%d\n", *threads, *gpuLayers)

	start := time.Now()
	emb, err := embedder.NewGGUFEmbedder(*modelPath, *threads, *gpuLayers)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}
	defer emb.Close()

	loadTime := time.Since(start)
	fmt.Printf("Model loaded in %v\n", loadTime)
	fmt.Printf("Embedding dimension: %d\n\n", emb.Dimension())

	ctx := context.Background()

	if *benchmark {
		runBenchmark(ctx, emb)
		return
	}

	// Generate embedding for the text
	fmt.Printf("Generating embedding for: %q\n", *text)

	start = time.Now()
	embedding, err := emb.EmbedQuery(ctx, *text)
	if err != nil {
		log.Fatalf("Failed to generate embedding: %v", err)
	}
	embedTime := time.Since(start)

	fmt.Printf("\nEmbedding generated in %v\n", embedTime)
	fmt.Printf("Dimension: %d\n", len(embedding))
	fmt.Printf("First 10 values: %v\n", embedding[:min(10, len(embedding))])
	fmt.Printf("Norm: %.6f\n", norm(embedding))

	// Test batch embeddings
	fmt.Println("\n--- Testing batch embeddings ---")
	testBatch(ctx, emb)
}

func runBenchmark(ctx context.Context, emb *embedder.GGUFEmbedder) {
	fmt.Println("Running benchmark...")

	texts := []string{
		"The quick brown fox jumps over the lazy dog",
		"Machine learning is a fascinating field of study",
		"Go programming language is efficient and concurrent",
		"Embeddings capture semantic meaning of text",
		"Natural language processing has many applications",
	}

	// Warm-up
	fmt.Println("Warming up...")
	for i := 0; i < 3; i++ {
		_, _ = emb.EmbedQuery(ctx, texts[0])
	}

	// Single embedding benchmark
	fmt.Println("\nBenchmarking single embeddings (100 iterations)...")
	start := time.Now()
	for i := 0; i < 100; i++ {
		_, err := emb.EmbedQuery(ctx, texts[i%len(texts)])
		if err != nil {
			log.Fatalf("Benchmark failed: %v", err)
		}
	}
	elapsed := time.Since(start)
	fmt.Printf("Total time: %v\n", elapsed)
	fmt.Printf("Average per embedding: %v\n", elapsed/100)
	fmt.Printf("Embeddings per second: %.2f\n", 100.0/elapsed.Seconds())

	// Batch embedding benchmark
	fmt.Println("\nBenchmarking batch embeddings (20 batches of 5)...")
	start = time.Now()
	for i := 0; i < 20; i++ {
		_, err := emb.EmbedDocuments(ctx, texts)
		if err != nil {
			log.Fatalf("Batch benchmark failed: %v", err)
		}
	}
	elapsed = time.Since(start)
	totalEmbeddings := 20 * len(texts)
	fmt.Printf("Total time: %v\n", elapsed)
	fmt.Printf("Average per batch: %v\n", elapsed/20)
	fmt.Printf("Average per embedding: %v\n", elapsed/time.Duration(totalEmbeddings))
	fmt.Printf("Embeddings per second: %.2f\n", float64(totalEmbeddings)/elapsed.Seconds())
}

func testBatch(ctx context.Context, emb *embedder.GGUFEmbedder) {
	texts := []string{
		"First example text",
		"Second example text",
		"Third example text",
	}

	fmt.Printf("Generating embeddings for %d texts...\n", len(texts))

	start := time.Now()
	embeddings, err := emb.EmbedDocuments(ctx, texts)
	if err != nil {
		log.Fatalf("Failed to generate batch embeddings: %v", err)
	}
	batchTime := time.Since(start)

	fmt.Printf("Batch embeddings generated in %v\n", batchTime)
	fmt.Printf("Average per text: %v\n", batchTime/time.Duration(len(texts)))

	for i, embedding := range embeddings {
		fmt.Printf("\nText %d: %q\n", i+1, texts[i])
		fmt.Printf("  Dimension: %d\n", len(embedding))
		fmt.Printf("  First 5 values: %v\n", embedding[:min(5, len(embedding))])
		fmt.Printf("  Norm: %.6f\n", norm(embedding))
	}

	// Compute cosine similarity between first two embeddings
	if len(embeddings) >= 2 {
		sim := cosineSimilarity(embeddings[0], embeddings[1])
		fmt.Printf("\nCosine similarity between text 1 and 2: %.6f\n", sim)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func norm(vec []float32) float32 {
	var sum float32
	for _, v := range vec {
		sum += v * v
	}
	return float32(sqrt(float64(sum)))
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(sqrt(float64(normA))) * float32(sqrt(float64(normB))))
}

func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	// Simple Newton-Raphson method
	z := x
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}
