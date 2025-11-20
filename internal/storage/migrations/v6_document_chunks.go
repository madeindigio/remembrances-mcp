package migrations

import (
	"context"
	"log"

	"github.com/surrealdb/surrealdb.go"
)

type V6DocumentChunks struct {
	db *surrealdb.DB
}

func NewV6DocumentChunks(db *surrealdb.DB) *V6DocumentChunks {
	return &V6DocumentChunks{db: db}
}

func (m *V6DocumentChunks) Version() int { return 6 }

func (m *V6DocumentChunks) Description() string {
	return "Add document chunking support to knowledge_base table"
}

func (m *V6DocumentChunks) Apply(ctx context.Context, db *surrealdb.DB) error {
	log.Println("Applying migration v6: adding document chunking support")
	elements := []SchemaElement{
		{Type: "field", Statement: `DEFINE FIELD chunk_index ON knowledge_base TYPE int DEFAULT 0;`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD chunk_count ON knowledge_base TYPE int DEFAULT 0;`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD source_file ON knowledge_base TYPE option<string>;`, OnTable: "knowledge_base"},
	}
	return NewMigrationBase(db).ApplyElements(ctx, elements)
}
