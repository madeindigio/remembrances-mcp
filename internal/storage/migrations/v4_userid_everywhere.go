package migrations

import (
	"context"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

// MigrationV4 applies schema version 4.
// Its statements live in statements.go, the single SurrealQL source shared by
// the remote and the embedded execution paths.
type MigrationV4 struct {
	*MigrationBase
}

// NewMigrationV4 creates the v4 migration.
func NewMigrationV4(db *surrealdb.DB) Migration {
	return &MigrationV4{MigrationBase: NewMigrationBase(db)}
}

// Version returns the migration version.
func (m *MigrationV4) Version() int { return 4 }

// Description returns the migration description.
func (m *MigrationV4) Description() string {
	return "Add optional user_id to all major tables and ensure dynamic metadata"
}

// Apply executes the migration.
func (m *MigrationV4) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration", "version", 4, "description", m.Description())
	return m.ApplyElements(ctx, Statements(4, EmbeddingDimension()))
}
