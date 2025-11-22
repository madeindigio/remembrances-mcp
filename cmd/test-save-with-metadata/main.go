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

	cfg := &storage.ConnectionConfig{
		DBPath:    "surrealkv:///www/MCP/remembrances-mcp/remembrances.db",
		Namespace: "test",
		Database:  "test",
	}

	store := storage.NewSurrealDBStorage(cfg)
	if err := store.Connect(ctx); err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer store.Close()

	fmt.Println("=== TEST: Save document with nested metadata ===")
	
	testMetadata := map[string]interface{}{
		"source":        "test",
		"last_modified": "2025-11-21T23:00:00Z",
		"nested": map[string]interface{}{
			"level1": "value1",
			"level2": map[string]interface{}{
				"deep": "value2",
			},
		},
		"array": []interface{}{"item1", "item2"},
	}

	testFilePath := "TEST_METADATA_FILE.md"
	testContent := "Test document to verify metadata persistence after migration V7."
	testEmbedding := make([]float32, 768)
	
	if err := store.SaveDocument(ctx, testFilePath, testContent, testEmbedding, testMetadata); err != nil {
		log.Fatal("Failed to save:", err)
	}
	
	fmt.Println("✓ Document saved")

	fmt.Println("\n=== TEST: Retrieve and check metadata ===")
	doc, err := store.GetDocument(ctx, testFilePath)
	if err != nil {
		log.Fatal("Failed to retrieve:", err)
	}
	if doc == nil {
		log.Fatal("Document not found!")
	}

	fmt.Printf("Document file_path: %s\n", doc.FilePath)
	fmt.Printf("Metadata keys: %d\n", len(doc.Metadata))
	
	metaJSON, _ := json.MarshalIndent(doc.Metadata, "", "  ")
	fmt.Printf("Metadata:\n%s\n", string(metaJSON))

	if nested, ok := doc.Metadata["nested"].(map[string]interface{}); ok {
		fmt.Println("\n✓ Nested metadata preserved!")
		if level2, ok := nested["level2"].(map[string]interface{}); ok {
			fmt.Printf("✓ Deep nested value: %v\n", level2["deep"])
		} else {
			fmt.Println("✗ Level2 metadata NOT preserved")
		}
	} else {
		fmt.Println("✗ Nested metadata NOT preserved")
	}

	fmt.Println("\n=== Cleanup ===")
	if err := store.DeleteDocument(ctx, testFilePath); err != nil {
		log.Printf("Warning: %v\n", err)
	} else {
		fmt.Println("✓ Test document deleted")
	}
}
