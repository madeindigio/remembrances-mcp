package migrations

import (
	"context"
	"log"

	"github.com/surrealdb/surrealdb.go"
)

// V9CodeIndexing implements the code indexing schema migration
type V9CodeIndexing struct {
	*MigrationBase
}

// NewV9CodeIndexing creates a new V9 migration
func NewV9CodeIndexing(db *surrealdb.DB) Migration {
	return &V9CodeIndexing{
		MigrationBase: NewMigrationBase(db),
	}
}

// Version returns the migration version
func (m *V9CodeIndexing) Version() int {
	return 9
}

// Description returns the migration description
func (m *V9CodeIndexing) Description() string {
	return "Creating code indexing schema: code_projects, code_files, code_symbols"
}

// Apply executes the migration
func (m *V9CodeIndexing) Apply(ctx context.Context, db *surrealdb.DB) error {
	log.Println("Applying migration v9: Creating code indexing schema")

	elements := []SchemaElement{
		// ===========================================
		// TABLE: code_projects
		// ===========================================
		{Type: "table", Statement: `DEFINE TABLE code_projects SCHEMAFULL;`},

		// Fields for code_projects
		{Type: "field", Statement: `DEFINE FIELD project_id ON code_projects TYPE string;`, OnTable: "code_projects"},
		{Type: "field", Statement: `DEFINE FIELD name ON code_projects TYPE string;`, OnTable: "code_projects"},
		{Type: "field", Statement: `DEFINE FIELD root_path ON code_projects TYPE string;`, OnTable: "code_projects"},
		{Type: "field", Statement: `DEFINE FIELD language_stats ON code_projects FLEXIBLE TYPE option<object>;`, OnTable: "code_projects"},
		{Type: "field", Statement: `DEFINE FIELD last_indexed_at ON code_projects TYPE option<datetime>;`, OnTable: "code_projects"},
		{Type: "field", Statement: `DEFINE FIELD indexing_status ON code_projects TYPE string DEFAULT 'pending';`, OnTable: "code_projects"},
		{Type: "field", Statement: `DEFINE FIELD created_at ON code_projects TYPE datetime VALUE time::now();`, OnTable: "code_projects"},
		{Type: "field", Statement: `DEFINE FIELD updated_at ON code_projects TYPE datetime VALUE time::now();`, OnTable: "code_projects"},

		// Indexes for code_projects
		{Type: "index", Statement: `DEFINE INDEX idx_code_project_id ON code_projects FIELDS project_id UNIQUE;`, OnTable: "code_projects"},
		{Type: "index", Statement: `DEFINE INDEX idx_code_project_name ON code_projects FIELDS name;`, OnTable: "code_projects"},
		{Type: "index", Statement: `DEFINE INDEX idx_code_project_status ON code_projects FIELDS indexing_status;`, OnTable: "code_projects"},

		// ===========================================
		// TABLE: code_files
		// ===========================================
		{Type: "table", Statement: `DEFINE TABLE code_files SCHEMAFULL;`},

		// Fields for code_files
		{Type: "field", Statement: `DEFINE FIELD project_id ON code_files TYPE string;`, OnTable: "code_files"},
		{Type: "field", Statement: `DEFINE FIELD file_path ON code_files TYPE string;`, OnTable: "code_files"},
		{Type: "field", Statement: `DEFINE FIELD language ON code_files TYPE string;`, OnTable: "code_files"},
		{Type: "field", Statement: `DEFINE FIELD file_hash ON code_files TYPE string;`, OnTable: "code_files"},
		{Type: "field", Statement: `DEFINE FIELD symbols_count ON code_files TYPE int DEFAULT 0;`, OnTable: "code_files"},
		{Type: "field", Statement: `DEFINE FIELD indexed_at ON code_files TYPE datetime VALUE time::now();`, OnTable: "code_files"},

		// Indexes for code_files
		{Type: "index", Statement: `DEFINE INDEX idx_code_file_project_path ON code_files FIELDS project_id, file_path UNIQUE;`, OnTable: "code_files"},
		{Type: "index", Statement: `DEFINE INDEX idx_code_file_project ON code_files FIELDS project_id;`, OnTable: "code_files"},
		{Type: "index", Statement: `DEFINE INDEX idx_code_file_language ON code_files FIELDS language;`, OnTable: "code_files"},
		{Type: "index", Statement: `DEFINE INDEX idx_code_file_hash ON code_files FIELDS file_hash;`, OnTable: "code_files"},

		// ===========================================
		// TABLE: code_symbols
		// ===========================================
		{Type: "table", Statement: `DEFINE TABLE code_symbols SCHEMAFULL;`},

		// Fields for code_symbols - Core identification
		{Type: "field", Statement: `DEFINE FIELD project_id ON code_symbols TYPE string;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD file_path ON code_symbols TYPE string;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD language ON code_symbols TYPE string;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD symbol_type ON code_symbols TYPE string;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD name ON code_symbols TYPE string;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD name_path ON code_symbols TYPE string;`, OnTable: "code_symbols"},

		// Fields for code_symbols - Location
		{Type: "field", Statement: `DEFINE FIELD start_line ON code_symbols TYPE int;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD end_line ON code_symbols TYPE int;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD start_byte ON code_symbols TYPE int;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD end_byte ON code_symbols TYPE int;`, OnTable: "code_symbols"},

		// Fields for code_symbols - Content
		{Type: "field", Statement: `DEFINE FIELD source_code ON code_symbols TYPE option<string>;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD signature ON code_symbols TYPE option<string>;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD doc_string ON code_symbols TYPE option<string>;`, OnTable: "code_symbols"},

		// Fields for code_symbols - Vector embedding for semantic search
		{Type: "field", Statement: `DEFINE FIELD embedding ON code_symbols TYPE option<array<float>>;`, OnTable: "code_symbols"},

		// Fields for code_symbols - Hierarchy
		{Type: "field", Statement: `DEFINE FIELD parent_id ON code_symbols TYPE option<string>;`, OnTable: "code_symbols"},

		// Fields for code_symbols - Metadata
		{Type: "field", Statement: `DEFINE FIELD metadata ON code_symbols FLEXIBLE TYPE option<object>;`, OnTable: "code_symbols"},

		// Fields for code_symbols - Timestamps
		{Type: "field", Statement: `DEFINE FIELD created_at ON code_symbols TYPE datetime VALUE time::now();`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD updated_at ON code_symbols TYPE datetime VALUE time::now();`, OnTable: "code_symbols"},

		// ===========================================
		// INDEXES for code_symbols
		// ===========================================
		// Primary lookups
		{Type: "index", Statement: `DEFINE INDEX idx_code_symbol_project ON code_symbols FIELDS project_id;`, OnTable: "code_symbols"},
		{Type: "index", Statement: `DEFINE INDEX idx_code_symbol_file ON code_symbols FIELDS project_id, file_path;`, OnTable: "code_symbols"},
		{Type: "index", Statement: `DEFINE INDEX idx_code_symbol_name_path ON code_symbols FIELDS project_id, name_path UNIQUE;`, OnTable: "code_symbols"},

		// Type-based queries
		{Type: "index", Statement: `DEFINE INDEX idx_code_symbol_type ON code_symbols FIELDS symbol_type;`, OnTable: "code_symbols"},
		{Type: "index", Statement: `DEFINE INDEX idx_code_symbol_language ON code_symbols FIELDS language;`, OnTable: "code_symbols"},

		// Name-based search (for find_symbol functionality)
		{Type: "index", Statement: `DEFINE INDEX idx_code_symbol_name ON code_symbols FIELDS name;`, OnTable: "code_symbols"},

		// Parent-child relationships
		{Type: "index", Statement: `DEFINE INDEX idx_code_symbol_parent ON code_symbols FIELDS parent_id;`, OnTable: "code_symbols"},

		// Vector search - MTREE with 768 dimensions (matching existing embedding dimension)
		{Type: "index", Statement: `DEFINE INDEX idx_code_symbol_embedding ON code_symbols FIELDS embedding MTREE DIMENSION 768 DIST COSINE;`, OnTable: "code_symbols"},

		// ===========================================
		// TABLE: code_indexing_jobs (for async job tracking)
		// ===========================================
		{Type: "table", Statement: `DEFINE TABLE code_indexing_jobs SCHEMAFULL;`},

		// Fields for code_indexing_jobs
		{Type: "field", Statement: `DEFINE FIELD project_id ON code_indexing_jobs TYPE string;`, OnTable: "code_indexing_jobs"},
		{Type: "field", Statement: `DEFINE FIELD project_path ON code_indexing_jobs TYPE string;`, OnTable: "code_indexing_jobs"},
		{Type: "field", Statement: `DEFINE FIELD status ON code_indexing_jobs TYPE string DEFAULT 'pending';`, OnTable: "code_indexing_jobs"},
		{Type: "field", Statement: `DEFINE FIELD progress ON code_indexing_jobs TYPE float DEFAULT 0.0;`, OnTable: "code_indexing_jobs"},
		{Type: "field", Statement: `DEFINE FIELD files_total ON code_indexing_jobs TYPE int DEFAULT 0;`, OnTable: "code_indexing_jobs"},
		{Type: "field", Statement: `DEFINE FIELD files_indexed ON code_indexing_jobs TYPE int DEFAULT 0;`, OnTable: "code_indexing_jobs"},
		{Type: "field", Statement: `DEFINE FIELD started_at ON code_indexing_jobs TYPE datetime VALUE time::now();`, OnTable: "code_indexing_jobs"},
		{Type: "field", Statement: `DEFINE FIELD completed_at ON code_indexing_jobs TYPE option<datetime>;`, OnTable: "code_indexing_jobs"},
		{Type: "field", Statement: `DEFINE FIELD error ON code_indexing_jobs TYPE option<string>;`, OnTable: "code_indexing_jobs"},

		// Indexes for code_indexing_jobs
		{Type: "index", Statement: `DEFINE INDEX idx_code_job_project ON code_indexing_jobs FIELDS project_id;`, OnTable: "code_indexing_jobs"},
		{Type: "index", Statement: `DEFINE INDEX idx_code_job_status ON code_indexing_jobs FIELDS status;`, OnTable: "code_indexing_jobs"},
	}

	return m.ApplyElements(ctx, elements)
}
