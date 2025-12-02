package migrations

import (
	"context"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

// MigrationV1 implements the initial schema migration
type MigrationV1 struct {
	*MigrationBase
}

// NewMigrationV1 creates a new V1 migration
func NewMigrationV1(db *surrealdb.DB) Migration {
	return &MigrationV1{
		MigrationBase: NewMigrationBase(db),
	}
}

// Version returns the migration version
func (m *MigrationV1) Version() int {
	return 1
}

// Description returns the migration description
func (m *MigrationV1) Description() string {
	return "Creating initial schema"
}

// Apply executes the migration
func (m *MigrationV1) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration v1: Creating initial schema")

	elements := []SchemaElement{
		// Tables first
		{Type: "table", Statement: `DEFINE TABLE kv_memories SCHEMAFULL;`},
		{Type: "table", Statement: `DEFINE TABLE vector_memories SCHEMAFULL;`},
		{Type: "table", Statement: `DEFINE TABLE knowledge_base SCHEMAFULL;`},
		{Type: "table", Statement: `DEFINE TABLE entities SCHEMAFULL;`},
		{Type: "table", Statement: `DEFINE TABLE wrote SCHEMALESS;`},
		{Type: "table", Statement: `DEFINE TABLE mentioned_in SCHEMALESS;`},
		{Type: "table", Statement: `DEFINE TABLE related_to SCHEMALESS;`},

		// Fields for kv_memories
		{Type: "field", Statement: `DEFINE FIELD user_id ON kv_memories TYPE string;`, OnTable: "kv_memories"},
		{Type: "field", Statement: `DEFINE FIELD key ON kv_memories TYPE string;`, OnTable: "kv_memories"},
		{Type: "field", Statement: `DEFINE FIELD value ON kv_memories TYPE any;`, OnTable: "kv_memories"},
		{Type: "field", Statement: `DEFINE FIELD created_at ON kv_memories TYPE datetime VALUE time::now();`, OnTable: "kv_memories"},
		// SurrealDB does not support `ON UPDATE` in field definitions in this client/schema version,
		// keep updated_at with a VALUE default and let the application set it on updates.
		{Type: "field", Statement: `DEFINE FIELD updated_at ON kv_memories TYPE datetime VALUE time::now();`, OnTable: "kv_memories"},

		// Fields for vector_memories
		{Type: "field", Statement: `DEFINE FIELD user_id ON vector_memories TYPE string;`, OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD content ON vector_memories TYPE string;`, OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD embedding ON vector_memories TYPE array<float>;`, OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD metadata ON vector_memories TYPE object;`, OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD created_at ON vector_memories TYPE datetime VALUE time::now();`, OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD updated_at ON vector_memories TYPE datetime VALUE time::now();`, OnTable: "vector_memories"},

		// Fields for knowledge_base
		{Type: "field", Statement: `DEFINE FIELD file_path ON knowledge_base TYPE string;`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD content ON knowledge_base TYPE string;`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD embedding ON knowledge_base TYPE array<float>;`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD metadata ON knowledge_base TYPE object;`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD created_at ON knowledge_base TYPE datetime VALUE time::now();`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD updated_at ON knowledge_base TYPE datetime VALUE time::now();`, OnTable: "knowledge_base"},

		// Fields for entities
		{Type: "field", Statement: `DEFINE FIELD name ON entities TYPE string;`, OnTable: "entities"},
		{Type: "field", Statement: `DEFINE FIELD type ON entities TYPE string;`, OnTable: "entities"},
		{Type: "field", Statement: `DEFINE FIELD properties ON entities TYPE object;`, OnTable: "entities"},
		{Type: "field", Statement: `DEFINE FIELD created_at ON entities TYPE datetime VALUE time::now();`, OnTable: "entities"},
		{Type: "field", Statement: `DEFINE FIELD updated_at ON entities TYPE datetime VALUE time::now();`, OnTable: "entities"},

		// Fields for relationship tables
		{Type: "field", Statement: `DEFINE FIELD in ON wrote TYPE record;`, OnTable: "wrote"},
		{Type: "field", Statement: `DEFINE FIELD out ON wrote TYPE record;`, OnTable: "wrote"},
		{Type: "field", Statement: `DEFINE FIELD timestamp ON wrote TYPE datetime VALUE time::now();`, OnTable: "wrote"},
		{Type: "field", Statement: `DEFINE FIELD properties ON wrote TYPE object;`, OnTable: "wrote"},

		{Type: "field", Statement: `DEFINE FIELD in ON mentioned_in TYPE string;`, OnTable: "mentioned_in"},
		{Type: "field", Statement: `DEFINE FIELD out ON mentioned_in TYPE string;`, OnTable: "mentioned_in"},
		{Type: "field", Statement: `DEFINE FIELD timestamp ON mentioned_in TYPE datetime VALUE time::now();`, OnTable: "mentioned_in"},
		{Type: "field", Statement: `DEFINE FIELD properties ON mentioned_in TYPE object;`, OnTable: "mentioned_in"},

		{Type: "field", Statement: `DEFINE FIELD in ON related_to TYPE string;`, OnTable: "related_to"},
		{Type: "field", Statement: `DEFINE FIELD out ON related_to TYPE string;`, OnTable: "related_to"},
		{Type: "field", Statement: `DEFINE FIELD timestamp ON related_to TYPE datetime VALUE time::now();`, OnTable: "related_to"},
		{Type: "field", Statement: `DEFINE FIELD properties ON related_to TYPE object;`, OnTable: "related_to"},

		// Indexes
		{Type: "index", Statement: `DEFINE INDEX idx_kv_user_key ON kv_memories FIELDS user_id, key UNIQUE;`, OnTable: "kv_memories"},
		{Type: "index", Statement: `DEFINE INDEX idx_vector_user ON vector_memories FIELDS user_id;`, OnTable: "vector_memories"},
		{Type: "index", Statement: `DEFINE INDEX idx_embedding ON vector_memories FIELDS embedding MTREE DIMENSION 768 DIST COSINE;`, OnTable: "vector_memories"},
		{Type: "index", Statement: `DEFINE INDEX idx_kb_path ON knowledge_base FIELDS file_path UNIQUE;`, OnTable: "knowledge_base"},
		{Type: "index", Statement: `DEFINE INDEX idx_kb_embedding ON knowledge_base FIELDS embedding MTREE DIMENSION 768 DIST COSINE;`, OnTable: "knowledge_base"},
		{Type: "index", Statement: `DEFINE INDEX idx_entity_name ON entities FIELDS name;`, OnTable: "entities"},
		{Type: "index", Statement: `DEFINE INDEX idx_entity_type ON entities FIELDS type;`, OnTable: "entities"},
	}

	return m.ApplyElements(ctx, elements)
}
