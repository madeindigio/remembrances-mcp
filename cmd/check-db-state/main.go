package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
)

func main() {
	ctx := context.Background()

	// Create storage config (same as in config.sample.gguf.yaml)
	cfg := &storage.ConnectionConfig{
		DBPath:    "surrealkv://~/www/MCP/remembrances-mcp/remembrances.db",
		Namespace: "test",
		Database:  "test",
	}

	// Create storage
	store := storage.NewSurrealDBStorage(cfg)
	if err := store.Connect(ctx); err != nil {
		log.Fatal("Failed to connect to storage:", err)
	}
	defer store.Close()

	// Get all documents
	fmt.Println("=== KNOWLEDGE BASE DOCUMENTS ===")
	results, err := store.SearchDocuments(ctx, nil, 100)
	if err != nil {
		log.Fatal("Failed to search documents:", err)
	}

	fmt.Printf("Found %d documents\n\n", len(results))
	for i, result := range results {
		doc := result.Document
		if doc == nil {
			continue
		}
		
		fmt.Printf("Document #%d:\n", i+1)
		fmt.Printf("  file_path: %s\n", doc.FilePath)
		fmt.Printf("  ID: %s\n", doc.ID)
		
		// Check metadata
		fmt.Printf("  metadata type: %T\n", doc.Metadata)
		fmt.Printf("  metadata keys: %d\n", len(doc.Metadata))
		if len(doc.Metadata) > 0 {
			metaJSON, _ := json.MarshalIndent(doc.Metadata, "    ", "  ")
			fmt.Printf("  metadata content:\n    %s\n", string(metaJSON))
		} else {
			fmt.Printf("  metadata: EMPTY MAP\n")
		}
		fmt.Printf("  content length: %d bytes\n", len(doc.Content))
		fmt.Println()
	}

	// Try to get a specific document
	fmt.Println("\n=== GET SPECIFIC DOCUMENT (BUILD_INSTRUCTIONS.md) ===")
	doc, err := store.GetDocument(ctx, "BUILD_INSTRUCTIONS.md")
	if err != nil {
		log.Printf("Error getting document: %v\n", err)
	} else if doc != nil {
		fmt.Printf("Document found:\n")
		fmt.Printf("  file_path: %s\n", doc.FilePath)
		fmt.Printf("  ID: %s\n", doc.ID)
		fmt.Printf("  metadata keys: %d\n", len(doc.Metadata))
		if len(doc.Metadata) > 0 {
			metaJSON, _ := json.MarshalIndent(doc.Metadata, "  ", "  ")
			fmt.Printf("  metadata:\n%s\n", string(metaJSON))
		} else {
			fmt.Printf("  metadata: EMPTY\n")
		}
		fmt.Printf("  content length: %d bytes\n", len(doc.Content))
	} else {
		fmt.Println("Document not found")
	}
}
