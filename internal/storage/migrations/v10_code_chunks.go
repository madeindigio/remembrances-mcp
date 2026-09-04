package migrations

import (
	"context"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

// V10CodeChunks applies schema version 10.
// Its statements live in statements.go, the single SurrealQL source shared by
// the remote and the embedded execution paths.
type V10CodeChunks struct {
	*MigrationBase
}

// NewV10CodeChunks creates the v10 migration.
func NewV10CodeChunks(db *surrealdb.DB) Migration {
	return &V10CodeChunks{MigrationBase: NewMigrationBase(db)}
}

// Version returns the migration version.
func (m *V10CodeChunks) Version() int { return 10 }

// Description returns the migration description.
func (m *V10CodeChunks) Description() string {
	return "Creating code_chunks table for large symbol chunking and semantic search"
}

// Apply executes the migration.
func (m *V10CodeChunks) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration", "version", 10, "description", m.Description())
	return m.ApplyElements(ctx, Statements(10, EmbeddingDimension()))
}
