package main

import (
	"fmt"
	"log"
	"time"

	embedded "github.com/madeindigio/surrealdb-embedded-golang"
)

func main() {
	// Connect to embedded DB
	db, err := embedded.NewRocksDB("/www/MCP/remembrances-mcp/test-simple.db")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Use namespace and database
	if err := db.Use("test", "test"); err != nil {
		log.Fatalf("Failed to use: %v", err)
	}

	fmt.Println("✓ Connected")

	// Create a simple table
	fmt.Println("\n1. Creating table...")
	result, err := db.Query("DEFINE TABLE test_table SCHEMAFULL", nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}

	// Add a field
	fmt.Println("\n2. Adding field...")
	result, err = db.Query("DEFINE FIELD name ON test_table TYPE string", nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}

	// Insert a record
	fmt.Println("\n3. Inserting record...")
	result, err = db.Query("CREATE test_table CONTENT { name: 'test record' }", nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}

	// Select the record
	fmt.Println("\n4. Selecting record...")
	result, err = db.Query("SELECT * FROM test_table", nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}

	// Close
	fmt.Println("\n5. Closing...")
	db.Close()

	time.Sleep(time.Second)

	// Reopen and check
	fmt.Println("\n6. Reopening...")
	db2, err := embedded.NewRocksDB("/www/MCP/remembrances-mcp/test-simple.db")
	if err != nil {
		log.Fatalf("Failed to reopen: %v", err)
	}
	defer db2.Close()

	if err := db2.Use("test", "test"); err != nil {
		log.Fatalf("Failed to use after reopen: %v", err)
	}

	fmt.Println("\n7. Selecting after reopen...")
	result, err = db2.Query("SELECT * FROM test_table", nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}

	fmt.Println("\n✓ Done")
}
