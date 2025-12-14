//go:build tools

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
)

func main() {
	ctx := context.Background()

	// Create storage instance
	storageConfig := &storage.ConnectionConfig{
		DBPath:    "~/www/MCP/remembrances-mcp/remembrances.db",
		Namespace: "test",
		Database:  "test",
		Timeout:   30 * time.Second,
	}

	st := storage.NewSurrealDBStorage(storageConfig)

	// Connect
	if err := st.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	// Initialize schema
	if err := st.InitializeSchema(ctx); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	fmt.Println("✓ Connected and schema initialized")

	// Create test embeddings
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = 0.5
	}

	// Save a test document chunk
	chunks := []string{"Test chunk 1", "Test chunk 2"}
	embeddings := [][]float32{embedding, embedding}
	metadata := map[string]interface{}{
		"source":        "test",
		"last_modified": time.Now().Format(time.RFC3339),
	}

	fmt.Println("Saving test document chunks...")
	err := st.SaveDocumentChunks(ctx, "test-file.md", chunks, embeddings, metadata)
	if err != nil {
		log.Fatalf("Failed to save chunks: %v", err)
	}

	fmt.Println("✓ Chunks saved successfully")

	// Try to retrieve immediately
	fmt.Println("Retrieving document before close...")
	doc, err := st.GetDocument(ctx, "test-file.md")
	if err != nil {
		fmt.Printf("Error getting document: %v\n", err)
	} else if doc == nil {
		fmt.Println("⚠️  Document not found (before close)")
	} else {
		fmt.Printf("✓ Document found before close: %s\n", doc.FilePath)
	}

	// Close connection
	fmt.Println("\nClosing connection...")
	if err := st.Close(); err != nil {
		log.Fatalf("Failed to close: %v", err)
	}

	fmt.Println("✓ Connection closed")
	fmt.Println("\n=== Now reopening to check persistence ===\n")

	// Reopen connection
	st2 := storage.NewSurrealDBStorage(storageConfig)
	if err := st2.Connect(ctx); err != nil {
		log.Fatalf("Failed to reconnect: %v", err)
	}
	defer st2.Close()

	fmt.Println("✓ Reconnected")

	// Try to retrieve after reopening
	fmt.Println("Retrieving document after reopen...")
	doc2, err := st2.GetDocument(ctx, "test-file.md")
	if err != nil {
		fmt.Printf("❌ Error getting document: %v\n", err)
	} else if doc2 == nil {
		fmt.Println("❌ Document NOT FOUND (data was lost!)")
	} else {
		fmt.Printf("✅ SUCCESS! Document persisted: %s\n", doc2.FilePath)
		fmt.Printf("   Content: %s\n", doc2.Content)
		fmt.Printf("   Metadata: %+v\n", doc2.Metadata)
	}
}
