package migrations

import (
	"context"
	"log"

	"github.com/surrealdb/surrealdb.go"
)

// V12CodeProjectsWatcher implements the migration to add watcher_enabled field to code_projects
type V12CodeProjectsWatcher struct {
	*MigrationBase
}

// NewV12CodeProjectsWatcher creates a new V12 migration
func NewV12CodeProjectsWatcher(db *surrealdb.DB) Migration {
	return &V12CodeProjectsWatcher{
		MigrationBase: NewMigrationBase(db),
	}
}

// Version returns the migration version
func (m *V12CodeProjectsWatcher) Version() int {
	return 12
}

// Description returns the migration description
func (m *V12CodeProjectsWatcher) Description() string {
	return "Adding watcher_enabled field to code_projects table"
}

// Apply executes the migration
func (m *V12CodeProjectsWatcher) Apply(ctx context.Context, db *surrealdb.DB) error {
	log.Println("Applying migration v12: Adding watcher_enabled field to code_projects")

	elements := []SchemaElement{
		// Add watcher_enabled field to code_projects
		{Type: "field", Statement: `DEFINE FIELD watcher_enabled ON code_projects TYPE bool DEFAULT false;`, OnTable: "code_projects"},
	}

	return m.ApplyElements(ctx, elements)
}
