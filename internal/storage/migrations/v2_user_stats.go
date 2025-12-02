package migrations

import (
	"context"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

// MigrationV2 implements the user_stats table migration
type MigrationV2 struct {
	*MigrationBase
}

// NewMigrationV2 creates a new V2 migration
func NewMigrationV2(db *surrealdb.DB) Migration {
	return &MigrationV2{
		MigrationBase: NewMigrationBase(db),
	}
}

// Version returns the migration version
func (m *MigrationV2) Version() int {
	return 2
}

// Description returns the migration description
func (m *MigrationV2) Description() string {
	return "Adding user_stats table"
}

// Apply executes the migration
func (m *MigrationV2) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration v2: Adding user_stats table")

	elements := []SchemaElement{
		// Add user_stats table
		{Type: "table", Statement: `DEFINE TABLE user_stats SCHEMAFULL;`},

		// Fields for user_stats
		{Type: "field", Statement: `DEFINE FIELD user_id ON user_stats TYPE string;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD key_value_count ON user_stats TYPE int VALUE 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD vector_count ON user_stats TYPE int VALUE 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD entity_count ON user_stats TYPE int VALUE 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD relationship_count ON user_stats TYPE int VALUE 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD document_count ON user_stats TYPE int VALUE 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD created_at ON user_stats TYPE datetime VALUE time::now();`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD updated_at ON user_stats TYPE datetime VALUE time::now();`, OnTable: "user_stats"},

		// Index for efficient user lookups
		{Type: "index", Statement: `DEFINE INDEX idx_user_stats_user_id ON user_stats FIELDS user_id UNIQUE;`, OnTable: "user_stats"},
	}

	return m.ApplyElements(ctx, elements)
}
