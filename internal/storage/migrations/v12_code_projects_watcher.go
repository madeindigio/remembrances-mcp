package migrations

import (
	"context"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

// V12CodeProjectsWatcher applies schema version 12.
// Its statements live in statements.go, the single SurrealQL source shared by
// the remote and the embedded execution paths.
type V12CodeProjectsWatcher struct {
	*MigrationBase
}

// NewV12CodeProjectsWatcher creates the v12 migration.
func NewV12CodeProjectsWatcher(db *surrealdb.DB) Migration {
	return &V12CodeProjectsWatcher{MigrationBase: NewMigrationBase(db)}
}

// Version returns the migration version.
func (m *V12CodeProjectsWatcher) Version() int { return 12 }

// Description returns the migration description.
func (m *V12CodeProjectsWatcher) Description() string {
	return "Adding watcher_enabled field to code_projects table"
}

// Apply executes the migration.
func (m *V12CodeProjectsWatcher) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration", "version", 12, "description", m.Description())
	return m.ApplyElements(ctx, Statements(12, EmbeddingDimension()))
}
