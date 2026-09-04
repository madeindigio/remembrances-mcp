package migrations

import (
	"context"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

// MigrationV1 applies schema version 1.
// Its statements live in statements.go, the single SurrealQL source shared by
// the remote and the embedded execution paths.
type MigrationV1 struct {
	*MigrationBase
}

// NewMigrationV1 creates the v1 migration.
func NewMigrationV1(db *surrealdb.DB) Migration {
	return &MigrationV1{MigrationBase: NewMigrationBase(db)}
}

// Version returns the migration version.
func (m *MigrationV1) Version() int { return 1 }

// Description returns the migration description.
func (m *MigrationV1) Description() string {
	return "Creating initial schema"
}

// Apply executes the migration.
func (m *MigrationV1) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration", "version", 1, "description", m.Description())
	return m.ApplyElements(ctx, Statements(1, EmbeddingDimension()))
}
