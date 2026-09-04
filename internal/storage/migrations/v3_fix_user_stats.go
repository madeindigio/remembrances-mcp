package migrations

import (
	"context"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

// MigrationV3 applies schema version 3.
// Its statements live in statements.go, the single SurrealQL source shared by
// the remote and the embedded execution paths.
type MigrationV3 struct {
	*MigrationBase
}

// NewMigrationV3 creates the v3 migration.
func NewMigrationV3(db *surrealdb.DB) Migration {
	return &MigrationV3{MigrationBase: NewMigrationBase(db)}
}

// Version returns the migration version.
func (m *MigrationV3) Version() int { return 3 }

// Description returns the migration description.
func (m *MigrationV3) Description() string {
	return "Fixing user_stats field definitions"
}

// Apply executes the migration.
func (m *MigrationV3) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration", "version", 3, "description", m.Description())
	return m.ApplyElements(ctx, Statements(3, EmbeddingDimension()))
}
