package migrations

import (
	"context"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

// V8FlexibleKVValue applies schema version 8.
// Its statements live in statements.go, the single SurrealQL source shared by
// the remote and the embedded execution paths.
type V8FlexibleKVValue struct {
	*MigrationBase
}

// NewV8FlexibleKVValue creates the v8 migration.
func NewV8FlexibleKVValue(db *surrealdb.DB) Migration {
	return &V8FlexibleKVValue{MigrationBase: NewMigrationBase(db)}
}

// Version returns the migration version.
func (m *V8FlexibleKVValue) Version() int { return 8 }

// Description returns the migration description.
func (m *V8FlexibleKVValue) Description() string {
	return "Fix kv_memories value field to accept every scalar and container shape"
}

// Apply executes the migration.
func (m *V8FlexibleKVValue) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration", "version", 8, "description", m.Description())
	return m.ApplyElements(ctx, Statements(8, EmbeddingDimension()))
}
