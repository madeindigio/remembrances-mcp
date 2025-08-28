package embedder_test

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/madeindigio/remembrances-mcp/pkg/embedder"
)

// Example demonstrates how to use the embedder service with Ollama
func ExampleNewOllamaEmbedder() {
	// Create Ollama embedder
	ollamaEmbedder, err := embedder.NewOllamaEmbedder(
		"http://localhost:11434",
		"nomic-embed-text",
	)
	if err != nil {
		log.Fatalf("Failed to create Ollama embedder: %v", err)
	}

	ctx := context.Background()

	// Embed a single query
	query := "What is machine learning?"
	embedding, err := ollamaEmbedder.EmbedQuery(ctx, query)
	if err != nil {
		log.Fatalf("Failed to embed query: %v", err)
	}

	fmt.Printf("Query embedding dimension: %d\n", len(embedding))
	fmt.Printf("Embedder dimension: %d\n", ollamaEmbedder.Dimension())

	// Embed multiple documents
	documents := []string{
		"Machine learning is a subset of artificial intelligence.",
		"Deep learning uses neural networks with multiple layers.",
		"Natural language processing helps computers understand text.",
	}

	embeddings, err := ollamaEmbedder.EmbedDocuments(ctx, documents)
	if err != nil {
		log.Fatalf("Failed to embed documents: %v", err)
	}

	fmt.Printf("Number of document embeddings: %d\n", len(embeddings))
	// Output: Query embedding dimension: 768
	// Embedder dimension: 768
	// Number of document embeddings: 3
}

// Example demonstrates how to use the embedder service with OpenAI
func ExampleNewOpenAIEmbedder() {
	// Create OpenAI embedder
	openaiEmbedder, err := embedder.NewOpenAIEmbedder(
		"your-api-key-here",
		"", // Use default OpenAI base URL
		"text-embedding-3-large",
	)
	if err != nil {
		log.Fatalf("Failed to create OpenAI embedder: %v", err)
	}

	ctx := context.Background()

	// Embed a single query
	query := "What is artificial intelligence?"
	embedding, err := openaiEmbedder.EmbedQuery(ctx, query)
	if err != nil {
		log.Fatalf("Failed to embed query: %v", err)
	}

	fmt.Printf("Query embedding dimension: %d\n", len(embedding))
	fmt.Printf("Embedder dimension: %d\n", openaiEmbedder.Dimension())
	// Output: Query embedding dimension: 3072
	// Embedder dimension: 3072
}

// Example demonstrates how to use the factory function with configuration
func ExampleNewEmbedderFromConfig() {
	// Create config for Ollama
	cfg := &embedder.Config{
		OllamaURL:   "http://localhost:11434",
		OllamaModel: "nomic-embed-text",
	}

	embedderInstance, err := embedder.NewEmbedderFromConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}

	fmt.Printf("Embedder type: %s\n", embedder.GetEmbedderType(cfg))
	fmt.Printf("Embedder dimension: %d\n", embedderInstance.Dimension())
	// Output: Embedder type: ollama
	// Embedder dimension: 768
}

// Example demonstrates how to use the factory function with environment variables
func ExampleNewEmbedderFromEnv() {
	// Set environment variables (in real usage, these would be set outside the program)
	os.Setenv("OLLAMA_URL", "http://localhost:11434")
	os.Setenv("OLLAMA_EMBEDDING_MODEL", "nomic-embed-text")

	embedderInstance, err := embedder.NewEmbedderFromEnv()
	if err != nil {
		log.Fatalf("Failed to create embedder from env: %v", err)
	}

	fmt.Printf("Embedder dimension: %d\n", embedderInstance.Dimension())
	// Output: Embedder dimension: 768
}

// Example demonstrates configuration validation
func ExampleValidateConfig() {
	// Valid Ollama config
	validConfig := &embedder.Config{
		OllamaURL:   "http://localhost:11434",
		OllamaModel: "nomic-embed-text",
	}

	if err := embedder.ValidateConfig(validConfig); err != nil {
		fmt.Printf("Invalid config: %v\n", err)
	} else {
		fmt.Println("Config is valid")
	}

	// Invalid config (missing model)
	invalidConfig := &embedder.Config{
		OllamaURL: "http://localhost:11434",
		// Missing OllamaModel
	}

	if err := embedder.ValidateConfig(invalidConfig); err != nil {
		fmt.Printf("Invalid config: %v\n", err)
	} else {
		fmt.Println("Config is valid")
	}

	// Output: Config is valid
	// Invalid config: ollama model is required when ollama URL is provided
}
