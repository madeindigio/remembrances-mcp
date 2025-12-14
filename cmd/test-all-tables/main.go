package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/madeindigio/remembrances-mcp/internal/surrealembedded"
)

func main() {
	ctx := context.Background()
	// Connect to embedded DB
	db, err := surrealembedded.NewFromURL(ctx, "~/www/MCP/remembrances-mcp/remembrances.db")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Use the same namespace and database
	if err := db.Use("test", "test"); err != nil {
		log.Fatalf("Failed to use namespace/database: %v", err)
	}

	fmt.Println("=== Connected to embedded SurrealDB ===")
	fmt.Println()

	// List of tables to check
	tables := []string{
		"knowledge_base",
		"vector_memories",
		"kv_memories",
		"entities",
		"user_stats",
		"schema_version",
	}

	for _, table := range tables {
		fmt.Printf("=== Table: %s ===\n", table)

		// Count
		countQuery := fmt.Sprintf("SELECT count() FROM %s GROUP ALL", table)
		countResults, err := db.Query(countQuery, nil)
		if err != nil {
			fmt.Printf("Error counting: %v\n", err)
		} else {
			prettyJSON, _ := json.MarshalIndent(countResults, "", "  ")
			fmt.Printf("Count: %s\n", string(prettyJSON))
		}

		// Sample records
		sampleQuery := fmt.Sprintf("SELECT * FROM %s LIMIT 2", table)
		sampleResults, err := db.Query(sampleQuery, nil)
		if err != nil {
			fmt.Printf("Error querying: %v\n", err)
		} else if len(sampleResults) > 0 {
			prettyJSON, _ := json.MarshalIndent(sampleResults, "", "  ")
			fmt.Printf("Sample: %s\n", string(prettyJSON))
		}

		fmt.Println()
	}

	fmt.Println("=== Done ===")
}
