package main

import (
	"encoding/json"
	"fmt"
	"log"

	embedded "github.com/madeindigio/surrealdb-embedded-golang"
)

func main() {
	// Connect to embedded DB
	db, err := embedded.NewRocksDB("/www/MCP/remembrances-mcp/remembrances.db")
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

	// Query all knowledge_base documents
	fmt.Println("=== Querying all knowledge_base documents (LIMIT 5) ===")
	results, err := db.Query("SELECT * FROM knowledge_base LIMIT 5", nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	prettyJSON, _ := json.MarshalIndent(results, "", "  ")
	fmt.Println(string(prettyJSON))
	fmt.Println()

	// Count documents
	fmt.Println("=== Counting knowledge_base documents ===")
	countResults, err := db.Query("SELECT count() FROM knowledge_base GROUP ALL", nil)
	if err != nil {
		log.Printf("Count failed: %v", err)
	} else {
		prettyJSON, _ := json.MarshalIndent(countResults, "", "  ")
		fmt.Println(string(prettyJSON))
	}
	fmt.Println()

	// Query for a specific file by source_file
	fmt.Println("=== Querying by source_file ===")
	sourceResults, err := db.Query("SELECT * FROM knowledge_base WHERE source_file = $file LIMIT 1", map[string]interface{}{
		"file": "BUILD_INSTRUCTIONS.md",
	})
	if err != nil {
		log.Printf("Source file query failed: %v", err)
	} else {
		prettyJSON, _ := json.MarshalIndent(sourceResults, "", "  ")
		fmt.Println(string(prettyJSON))
	}
	fmt.Println()

	// Query for a specific file by file_path
	fmt.Println("=== Querying by file_path (with chunk) ===")
	pathResults, err := db.Query("SELECT * FROM knowledge_base WHERE file_path = $file LIMIT 1", map[string]interface{}{
		"file": "BUILD_INSTRUCTIONS.md#chunk0",
	})
	if err != nil {
		log.Printf("File path query failed: %v", err)
	} else {
		prettyJSON, _ := json.MarshalIndent(pathResults, "", "  ")
		fmt.Println(string(prettyJSON))
	}

	fmt.Println()
	fmt.Println("=== Done ===")
}
