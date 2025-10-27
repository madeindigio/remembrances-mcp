package migrations

import (
	"context"
	"log"

	"github.com/surrealdb/surrealdb.go"
)

type MigrationV5 struct {
	db *surrealdb.DB
}

func NewMigrationV5(db *surrealdb.DB) *MigrationV5 {
	return &MigrationV5{db: db}
}

func (m *MigrationV5) Version() int { return 5 }
func (m *MigrationV5) Description() string {
	return "Ensure metadata and property fields are flexible objects"
}

func (m *MigrationV5) Apply(ctx context.Context, db *surrealdb.DB) error {
	log.Println("Applying migration v5: setting metadata and property fields to FLEXIBLE objects")
	elements := []SchemaElement{
		{Type: "field", Statement: `DEFINE FIELD metadata ON knowledge_base TYPE object FLEXIBLE;`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD metadata ON vector_memories TYPE object FLEXIBLE;`, OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD properties ON entities TYPE object FLEXIBLE;`, OnTable: "entities"},
		{Type: "field", Statement: `DEFINE FIELD properties ON wrote TYPE object FLEXIBLE;`, OnTable: "wrote"},
		{Type: "field", Statement: `DEFINE FIELD properties ON mentioned_in TYPE object FLEXIBLE;`, OnTable: "mentioned_in"},
		{Type: "field", Statement: `DEFINE FIELD properties ON related_to TYPE object FLEXIBLE;`, OnTable: "related_to"},
	}
	return NewMigrationBase(db).ApplyElements(ctx, elements)
}
