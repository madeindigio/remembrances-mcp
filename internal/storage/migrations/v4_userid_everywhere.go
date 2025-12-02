package migrations

import (
	"context"
	"github.com/surrealdb/surrealdb.go"
	"log/slog"
)

type MigrationV4 struct {
	db *surrealdb.DB
}

func NewMigrationV4(db *surrealdb.DB) *MigrationV4 {
	return &MigrationV4{db: db}
}

func (m *MigrationV4) Version() int { return 4 }
func (m *MigrationV4) Description() string {
	return "Add optional user_id to all major tables and ensure dynamic metadata"
}

func (m *MigrationV4) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration v4: Add user_id to all tables and dynamic metadata")
	elements := []SchemaElement{
		// Add user_id to entities
		{Type: "field", Statement: `DEFINE FIELD user_id ON entities TYPE option<string>;`, OnTable: "entities"},
		// Add user_id to knowledge_base
		{Type: "field", Statement: `DEFINE FIELD user_id ON knowledge_base TYPE option<string>;`, OnTable: "knowledge_base"},
		// Add user_id to relationship tables
		{Type: "field", Statement: `DEFINE FIELD user_id ON wrote TYPE option<string>;`, OnTable: "wrote"},
		{Type: "field", Statement: `DEFINE FIELD user_id ON mentioned_in TYPE option<string>;`, OnTable: "mentioned_in"},
		{Type: "field", Statement: `DEFINE FIELD user_id ON related_to TYPE option<string>;`, OnTable: "related_to"},
		// Ensure metadata is object (already is, but reinforce)
		{Type: "field", Statement: `DEFINE FIELD metadata ON knowledge_base TYPE object;`, OnTable: "knowledge_base"},
	}
	return NewMigrationBase(db).ApplyElements(ctx, elements)
}
