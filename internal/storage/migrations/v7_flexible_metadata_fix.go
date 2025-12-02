package migrations

import (
	"context"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

type V7FlexibleMetadataFix struct {
	db *surrealdb.DB
}

func NewV7FlexibleMetadataFix(db *surrealdb.DB) *V7FlexibleMetadataFix {
	return &V7FlexibleMetadataFix{db: db}
}

func (m *V7FlexibleMetadataFix) Version() int { return 7 }

func (m *V7FlexibleMetadataFix) Description() string {
	return "Fix metadata/properties fields to use FLEXIBLE TYPE object for dynamic nested fields"
}

func (m *V7FlexibleMetadataFix) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration v7: fixing metadata/properties fields to be FLEXIBLE")

	// Note: We need to REMOVE the old field definitions first, then redefine them
	// This ensures the FLEXIBLE keyword is properly applied

	// Remove old field definitions
	removeStatements := []string{
		`REMOVE FIELD metadata ON vector_memories;`,
		`REMOVE FIELD metadata ON knowledge_base;`,
		`REMOVE FIELD properties ON entities;`,
	}

	for _, stmt := range removeStatements {
		slog.Debug("Removing old field definition: %s", stmt)
		_, err := surrealdb.Query[[]map[string]interface{}](db, stmt, nil)
		if err != nil {
			// Log warning but continue - field might not exist
			slog.Debug("Warning: Could not remove field (may not exist): %v", err)
		}
	}

	// Redefine with FLEXIBLE keyword
	elements := []SchemaElement{
		{Type: "field", Statement: `DEFINE FIELD metadata ON vector_memories FLEXIBLE TYPE object DEFAULT {};`, OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD metadata ON knowledge_base FLEXIBLE TYPE object DEFAULT {};`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD properties ON entities FLEXIBLE TYPE object DEFAULT {};`, OnTable: "entities"},
	}

	slog.Info("Redefining fields with FLEXIBLE keyword")
	return NewMigrationBase(db).ApplyElements(ctx, elements)
}
