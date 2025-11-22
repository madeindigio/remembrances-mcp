package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
)

func main() {
	ctx := context.Background()

	// Read config from file or use defaults
	dbPath := "/www/MCP/remembrances-mcp/remembrances.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	// Create storage instance
	storageConfig := &storage.ConnectionConfig{
		DBPath:    dbPath,
		Namespace: "test",
		Database:  "test",
		Timeout:   30 * time.Second,
	}

	st := storage.NewSurrealDBStorage(storageConfig)

	// Connect
	if err := st.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer st.Close()

	fmt.Println("=== Connected to database ===")
	fmt.Printf("Database path: %s\n", dbPath)
	fmt.Println()

	// Test files to check
	testFiles := []string{
		"BUILD_INSTRUCTIONS.md",
		"kb_timestamp_optimization.md",
		"remembrances-mcp_project_overview.md",
		"COMPILE_CUDA.md",
	}

	fmt.Println("=== Testing GetDocument for specific files ===")
	fmt.Println()

	foundCount := 0
	notFoundCount := 0

	for _, testFile := range testFiles {
		doc, err := st.GetDocument(ctx, testFile)
		if err != nil {
			fmt.Printf("❌ Error getting document '%s': %v\n", testFile, err)
		} else if doc == nil {
			fmt.Printf("⚠️  Document '%s' NOT FOUND in database\n", testFile)
			notFoundCount++
		} else {
			fmt.Printf("✅ Document '%s' FOUND!\n", testFile)
			fmt.Printf("   ID: %s\n", doc.ID)
			fmt.Printf("   FilePath: %s\n", doc.FilePath)
			fmt.Printf("   Content length: %d chars\n", len(doc.Content))
			fmt.Printf("   Embedding length: %d\n", len(doc.Embedding))
			if doc.Metadata != nil {
				if lastMod, ok := doc.Metadata["last_modified"].(string); ok {
					fmt.Printf("   last_modified: %s\n", lastMod)
				} else {
					fmt.Printf("   last_modified: NOT SET\n")
				}
				if source, ok := doc.Metadata["source"]; ok {
					fmt.Printf("   source: %v\n", source)
				}
				if chunkIndex, ok := doc.Metadata["chunk_index"]; ok {
					fmt.Printf("   chunk_index: %v\n", chunkIndex)
				}
				if chunkCount, ok := doc.Metadata["chunk_count"]; ok {
					fmt.Printf("   chunk_count: %v\n", chunkCount)
				}
			} else {
				fmt.Printf("   Metadata: nil\n")
			}
			fmt.Printf("   CreatedAt: %s\n", doc.CreatedAt)
			fmt.Printf("   UpdatedAt: %s\n", doc.UpdatedAt)
			foundCount++
		}
		fmt.Println()
	}

	fmt.Println("=== Summary ===")
	fmt.Printf("Documents found: %d\n", foundCount)
	fmt.Printf("Documents not found: %d\n", notFoundCount)

	fmt.Println()
	fmt.Println("=== Done ===")
}
