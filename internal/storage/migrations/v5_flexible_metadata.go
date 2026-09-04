package migrations

import (
	"context"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

// MigrationV5 applies schema version 5.
// Its statements live in statements.go, the single SurrealQL source shared by
// the remote and the embedded execution paths.
type MigrationV5 struct {
	*MigrationBase
}

// NewMigrationV5 creates the v5 migration.
func NewMigrationV5(db *surrealdb.DB) Migration {
	return &MigrationV5{MigrationBase: NewMigrationBase(db)}
}

// Version returns the migration version.
func (m *MigrationV5) Version() int { return 5 }

// Description returns the migration description.
func (m *MigrationV5) Description() string {
	return "Ensure metadata and property fields are flexible objects"
}

// Apply executes the migration.
func (m *MigrationV5) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration", "version", 5, "description", m.Description())
	return m.ApplyElements(ctx, Statements(5, EmbeddingDimension()))
}
