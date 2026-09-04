package migrations

import (
	"context"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

// V6DocumentChunks applies schema version 6.
// Its statements live in statements.go, the single SurrealQL source shared by
// the remote and the embedded execution paths.
type V6DocumentChunks struct {
	*MigrationBase
}

// NewV6DocumentChunks creates the v6 migration.
func NewV6DocumentChunks(db *surrealdb.DB) Migration {
	return &V6DocumentChunks{MigrationBase: NewMigrationBase(db)}
}

// Version returns the migration version.
func (m *V6DocumentChunks) Version() int { return 6 }

// Description returns the migration description.
func (m *V6DocumentChunks) Description() string {
	return "Add document chunking support to knowledge_base table"
}

// Apply executes the migration.
func (m *V6DocumentChunks) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration", "version", 6, "description", m.Description())
	return m.ApplyElements(ctx, Statements(6, EmbeddingDimension()))
}
