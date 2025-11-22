package migrations

import (
	"context"
	"log"

	"github.com/surrealdb/surrealdb.go"
)

type V8FlexibleKVValue struct {
	db *surrealdb.DB
}

func NewV8FlexibleKVValue(db *surrealdb.DB) *V8FlexibleKVValue {
	return &V8FlexibleKVValue{db: db}
}

func (m *V8FlexibleKVValue) Version() int { return 8 }

func (m *V8FlexibleKVValue) Description() string {
	return "Fix kv_memories value field to use FLEXIBLE TYPE to handle strings with newlines"
}

func (m *V8FlexibleKVValue) Apply(ctx context.Context, db *surrealdb.DB) error {
	log.Println("Applying migration v8: fixing kv_memories value field to be FLEXIBLE")

	// Remove old field definition
	removeStatement := `REMOVE FIELD value ON kv_memories;`
	log.Printf("Removing old field definition: %s", removeStatement)
	_, err := surrealdb.Query[[]map[string]interface{}](db, removeStatement, nil)
	if err != nil {
		// Log warning but continue - field might not exist
		log.Printf("Warning: Could not remove field (may not exist): %v", err)
	}

	// Redefine with FLEXIBLE keyword to allow strings with newlines
	element := SchemaElement{
		Type:      "field",
		Statement: `DEFINE FIELD value ON kv_memories FLEXIBLE TYPE option<string | int | float | bool | object | array>;`,
		OnTable:   "kv_memories",
	}

	log.Printf("Creating field: %s", element.Statement)
	_, err = surrealdb.Query[[]map[string]interface{}](db, element.Statement, nil)
	if err != nil {
		return err
	}

	log.Println("Migration v8 completed successfully")
	return nil
}
