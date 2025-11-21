package main

import (
	"encoding/json"
	"fmt"
	"log"

	embedded "github.com/yourusername/surrealdb-embedded"
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

	// Get database info
	fmt.Println("=== INFO FOR DB ===")
	infoResults, err := db.Query("INFO FOR DB", nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		prettyJSON, _ := json.MarshalIndent(infoResults, "", "  ")
		fmt.Println(string(prettyJSON))
	}

	fmt.Println()
	fmt.Println("=== INFO FOR TABLE knowledge_base ===")
	tableInfo, err := db.Query("INFO FOR TABLE knowledge_base", nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		prettyJSON, _ := json.MarshalIndent(tableInfo, "", "  ")
		fmt.Println(string(prettyJSON))
	}

	fmt.Println()
	fmt.Println("=== Done ===")
}
