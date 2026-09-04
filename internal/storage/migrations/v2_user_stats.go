package migrations

import (
	"context"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

// MigrationV2 applies schema version 2.
// Its statements live in statements.go, the single SurrealQL source shared by
// the remote and the embedded execution paths.
type MigrationV2 struct {
	*MigrationBase
}

// NewMigrationV2 creates the v2 migration.
func NewMigrationV2(db *surrealdb.DB) Migration {
	return &MigrationV2{MigrationBase: NewMigrationBase(db)}
}

// Version returns the migration version.
func (m *MigrationV2) Version() int { return 2 }

// Description returns the migration description.
func (m *MigrationV2) Description() string {
	return "Adding user_stats table"
}

// Apply executes the migration.
func (m *MigrationV2) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration", "version", 2, "description", m.Description())
	return m.ApplyElements(ctx, Statements(2, EmbeddingDimension()))
}
