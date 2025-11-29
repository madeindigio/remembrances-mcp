package migrations

import (
	"context"
	"log"

	"github.com/surrealdb/surrealdb.go"
)

// V10CodeChunks implements the code chunks schema migration for large symbol chunking
type V10CodeChunks struct {
	*MigrationBase
}

// NewV10CodeChunks creates a new V10 migration
func NewV10CodeChunks(db *surrealdb.DB) Migration {
	return &V10CodeChunks{
		MigrationBase: NewMigrationBase(db),
	}
}

// Version returns the migration version
func (m *V10CodeChunks) Version() int {
	return 10
}

// Description returns the migration description
func (m *V10CodeChunks) Description() string {
	return "Creating code_chunks table for large symbol chunking and semantic search"
}

// Apply executes the migration
func (m *V10CodeChunks) Apply(ctx context.Context, db *surrealdb.DB) error {
	log.Println("Applying migration v10: Creating code_chunks table")

	elements := []SchemaElement{
		// ===========================================
		// TABLE: code_chunks
		// For storing chunks of large symbols for better semantic search
		// ===========================================
		{Type: "table", Statement: `DEFINE TABLE code_chunks SCHEMAFULL;`},

		// Fields for code_chunks - Reference to parent symbol
		{Type: "field", Statement: `DEFINE FIELD symbol_id ON code_chunks TYPE string;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD project_id ON code_chunks TYPE string;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD file_path ON code_chunks TYPE string;`, OnTable: "code_chunks"},

		// Fields for code_chunks - Chunk identification
		{Type: "field", Statement: `DEFINE FIELD chunk_index ON code_chunks TYPE int;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD chunk_count ON code_chunks TYPE int;`, OnTable: "code_chunks"},

		// Fields for code_chunks - Content
		{Type: "field", Statement: `DEFINE FIELD content ON code_chunks TYPE string;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD start_offset ON code_chunks TYPE int;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD end_offset ON code_chunks TYPE int;`, OnTable: "code_chunks"},

		// Fields for code_chunks - Vector embedding
		{Type: "field", Statement: `DEFINE FIELD embedding ON code_chunks TYPE option<array<float>>;`, OnTable: "code_chunks"},

		// Fields for code_chunks - Metadata for search context
		{Type: "field", Statement: `DEFINE FIELD symbol_name ON code_chunks TYPE string;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD symbol_type ON code_chunks TYPE string;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD language ON code_chunks TYPE string;`, OnTable: "code_chunks"},

		// Fields for code_chunks - Timestamps
		{Type: "field", Statement: `DEFINE FIELD created_at ON code_chunks TYPE datetime VALUE time::now();`, OnTable: "code_chunks"},

		// ===========================================
		// INDEXES for code_chunks
		// ===========================================
		// Primary lookups
		{Type: "index", Statement: `DEFINE INDEX idx_code_chunk_symbol ON code_chunks FIELDS symbol_id;`, OnTable: "code_chunks"},
		{Type: "index", Statement: `DEFINE INDEX idx_code_chunk_project ON code_chunks FIELDS project_id;`, OnTable: "code_chunks"},
		{Type: "index", Statement: `DEFINE INDEX idx_code_chunk_file ON code_chunks FIELDS project_id, file_path;`, OnTable: "code_chunks"},

		// Unique constraint for symbol + chunk index
		{Type: "index", Statement: `DEFINE INDEX idx_code_chunk_unique ON code_chunks FIELDS symbol_id, chunk_index UNIQUE;`, OnTable: "code_chunks"},

		// Vector search - MTREE with 768 dimensions
		{Type: "index", Statement: `DEFINE INDEX idx_code_chunk_embedding ON code_chunks FIELDS embedding MTREE DIMENSION 768 DIST COSINE;`, OnTable: "code_chunks"},
	}

	return m.ApplyElements(ctx, elements)
}
