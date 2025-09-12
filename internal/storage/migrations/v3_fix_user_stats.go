package migrations

import (
	"context"
	"fmt"
	"log"

	"github.com/surrealdb/surrealdb.go"
)

// MigrationV3 implements the user_stats field fixes migration
type MigrationV3 struct {
	*MigrationBase
}

// NewMigrationV3 creates a new V3 migration
func NewMigrationV3(db *surrealdb.DB) Migration {
	return &MigrationV3{
		MigrationBase: NewMigrationBase(db),
	}
}

// Version returns the migration version
func (m *MigrationV3) Version() int {
	return 3
}

// Description returns the migration description
func (m *MigrationV3) Description() string {
	return "Fixing user_stats field definitions"
}

// Apply executes the migration
func (m *MigrationV3) Apply(ctx context.Context, db *surrealdb.DB) error {
	log.Println("Applying migration v3: Fixing user_stats field definitions")

	// Remove the problematic fields that have VALUE constraints
	dropStatements := []string{
		"REMOVE FIELD key_value_count ON user_stats;",
		"REMOVE FIELD vector_count ON user_stats;",
		"REMOVE FIELD entity_count ON user_stats;",
		"REMOVE FIELD relationship_count ON user_stats;",
		"REMOVE FIELD document_count ON user_stats;",
	}

	// Execute drop statements
	for _, stmt := range dropStatements {
		log.Printf("Executing: %s", stmt)
		_, err := surrealdb.Query[[]map[string]interface{}](db, stmt, nil)
		if err != nil {
			// Log warning but continue - field might not exist
			log.Printf("Warning: Failed to drop field: %v", err)
		}
	}

	// Define the corrected fields without VALUE constraints
	elements := []SchemaElement{
		{Type: "field", Statement: `DEFINE FIELD key_value_count ON user_stats TYPE int;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD vector_count ON user_stats TYPE int;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD entity_count ON user_stats TYPE int;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD relationship_count ON user_stats TYPE int;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD document_count ON user_stats TYPE int;`, OnTable: "user_stats"},
	}

	// Apply the corrected field definitions
	for i, element := range elements {
		log.Printf("Creating corrected field: %s", element.Statement)
		_, err := surrealdb.Query[[]map[string]interface{}](db, element.Statement, nil)
		if err != nil {
			// This should succeed since we removed the old definitions
			return fmt.Errorf("failed to execute migration v3 statement %d '%s': %w", i+1, element.Statement, err)
		}
	}

	log.Println("Migration v3 completed successfully")
	return nil
}
