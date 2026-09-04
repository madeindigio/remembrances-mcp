package migrations

import (
	"context"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

// V7FlexibleMetadataFix applies schema version 7.
// Its statements live in statements.go, the single SurrealQL source shared by
// the remote and the embedded execution paths.
type V7FlexibleMetadataFix struct {
	*MigrationBase
}

// NewV7FlexibleMetadataFix creates the v7 migration.
func NewV7FlexibleMetadataFix(db *surrealdb.DB) Migration {
	return &V7FlexibleMetadataFix{MigrationBase: NewMigrationBase(db)}
}

// Version returns the migration version.
func (m *V7FlexibleMetadataFix) Version() int { return 7 }

// Description returns the migration description.
func (m *V7FlexibleMetadataFix) Description() string {
	return "Fix metadata and properties fields to be flexible objects"
}

// Apply executes the migration.
func (m *V7FlexibleMetadataFix) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration", "version", 7, "description", m.Description())
	return m.ApplyElements(ctx, Statements(7, EmbeddingDimension()))
}
