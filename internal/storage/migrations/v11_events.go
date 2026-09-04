package migrations

import (
	"context"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

// V11Events applies schema version 11.
// Its statements live in statements.go, the single SurrealQL source shared by
// the remote and the embedded execution paths.
type V11Events struct {
	*MigrationBase
}

// NewV11Events creates the v11 migration.
func NewV11Events(db *surrealdb.DB) Migration {
	return &V11Events{MigrationBase: NewMigrationBase(db)}
}

// Version returns the migration version.
func (m *V11Events) Version() int { return 11 }

// Description returns the migration description.
func (m *V11Events) Description() string {
	return "Creating events table for temporal event storage with hybrid search"
}

// Apply executes the migration.
func (m *V11Events) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration", "version", 11, "description", m.Description())
	return m.ApplyElements(ctx, Statements(11, EmbeddingDimension()))
}
