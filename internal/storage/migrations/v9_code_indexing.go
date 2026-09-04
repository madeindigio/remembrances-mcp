package migrations

import (
	"context"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

// V9CodeIndexing applies schema version 9.
// Its statements live in statements.go, the single SurrealQL source shared by
// the remote and the embedded execution paths.
type V9CodeIndexing struct {
	*MigrationBase
}

// NewV9CodeIndexing creates the v9 migration.
func NewV9CodeIndexing(db *surrealdb.DB) Migration {
	return &V9CodeIndexing{MigrationBase: NewMigrationBase(db)}
}

// Version returns the migration version.
func (m *V9CodeIndexing) Version() int { return 9 }

// Description returns the migration description.
func (m *V9CodeIndexing) Description() string {
	return "Creating code indexing schema: code_projects, code_files, code_symbols"
}

// Apply executes the migration.
func (m *V9CodeIndexing) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration", "version", 9, "description", m.Description())
	return m.ApplyElements(ctx, Statements(9, EmbeddingDimension()))
}
