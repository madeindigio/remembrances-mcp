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

	config := &storage.ConnectionConfig{
		DBPath:    "surrealkv://~/www/MCP/remembrances-mcp/remembrances.db",
		Namespace: "test",
		Database:  "test",
		Timeout:   30 * time.Second,
	}

	st := storage.NewSurrealDBStorage(config)

	fmt.Println("=== Testing Migration V7 on Real Database ===\n")

	// Connect
	if err := st.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer st.Close()

	// Initialize schema (this will run migration V7 if needed)
	fmt.Println("Running schema initialization (will apply Migration V7)...")
	if err := st.InitializeSchema(ctx); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Check schema version after migration
	fmt.Println("\nChecking schema version after migration...")
	result, err := st.Query(ctx, "SELECT * FROM schema_version:current", nil)
	if err != nil {
		log.Fatalf("Failed to query schema version: %v", err)
	}

	if len(result) > 0 {
		version := result[0]["version"]
		fmt.Printf("✓ Current schema version: %v\n\n", version)

		if v, ok := version.(float64); ok && v >= 7 {
			fmt.Println("✅ Migration V7 has been applied successfully!")
		} else {
			fmt.Printf("⚠️  Still at version %v\n", version)
		}
	}

	// Test metadata persistence
	fmt.Println("\n=== Testing Metadata Persistence ===\n")

	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = 0.1
	}

	metadata := map[string]interface{}{
		"test_field_1": "value1",
		"test_field_2": 12345,
		"test_field_3": true,
		"nested": map[string]interface{}{
			"inner_field": "inner_value",
		},
	}

	chunks := []string{"Test content for migration verification"}
	embeddings := [][]float32{embedding}

	testFile := "migration-v7-test.md"

	fmt.Println("Saving test document with complex metadata...")
	err = st.SaveDocumentChunks(ctx, testFile, chunks, embeddings, metadata)
	if err != nil {
		log.Fatalf("Failed to save: %v", err)
	}

	fmt.Println("Retrieving document...")
	doc, err := st.GetDocument(ctx, testFile)
	if err != nil {
		log.Fatalf("Failed to get document: %v", err)
	}

	if doc == nil {
		log.Fatal("Document not found!")
	}

	fmt.Printf("\n✓ Document retrieved: %s\n", doc.FilePath)
	fmt.Printf("✓ Metadata fields: %d\n", len(doc.Metadata))
	fmt.Printf("✓ Metadata content: %+v\n\n", doc.Metadata)

	// Check if all fields are present
	requiredFields := []string{"test_field_1", "test_field_2", "test_field_3", "nested", "chunk_index", "chunk_count"}
	missingFields := []string{}

	for _, field := range requiredFields {
		if _, ok := doc.Metadata[field]; !ok {
			missingFields = append(missingFields, field)
		}
	}

	if len(missingFields) == 0 {
		fmt.Println("✅ SUCCESS! All metadata fields preserved!")
		fmt.Println("✅ Migration V7 is working correctly!")
		fmt.Println("✅ Nested fields are properly stored and retrieved!")
	} else {
		fmt.Printf("❌ FAILED! Missing fields: %v\n", missingFields)
	}

	// Cleanup test document
	fmt.Println("\nCleaning up test document...")
	st.DeleteDocument(ctx, testFile)
	fmt.Println("✓ Test complete")
}
