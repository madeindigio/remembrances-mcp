package migrations

import (
	"context"
	"log"

	"github.com/surrealdb/surrealdb.go"
)

// V11Events implements the events table schema migration for temporal event storage
type V11Events struct {
	*MigrationBase
}

// NewV11Events creates a new V11 migration
func NewV11Events(db *surrealdb.DB) Migration {
	return &V11Events{
		MigrationBase: NewMigrationBase(db),
	}
}

// Version returns the migration version
func (m *V11Events) Version() int {
	return 11
}

// Description returns the migration description
func (m *V11Events) Description() string {
	return "Creating events table for temporal event storage with hybrid search"
}

// Apply executes the migration
func (m *V11Events) Apply(ctx context.Context, db *surrealdb.DB) error {
	log.Println("Applying migration v11: Creating events table")

	elements := []SchemaElement{
		// ===========================================
		// TABLE: events
		// For storing temporal events with semantic search
		// ===========================================
		{Type: "table", Statement: `DEFINE TABLE events SCHEMAFULL;`},

		// Fields for events - User/Project identification
		{Type: "field", Statement: `DEFINE FIELD user_id ON events TYPE string;`, OnTable: "events"},

		// Fields for events - Subject categorization (e.g., "conversation:session_123", "log:build")
		{Type: "field", Statement: `DEFINE FIELD subject ON events TYPE string;`, OnTable: "events"},

		// Fields for events - Content
		{Type: "field", Statement: `DEFINE FIELD content ON events TYPE string;`, OnTable: "events"},

		// Fields for events - Vector embedding for semantic search
		{Type: "field", Statement: `DEFINE FIELD embedding ON events TYPE array<float>;`, OnTable: "events"},

		// Fields for events - Optional metadata
		{Type: "field", Statement: `DEFINE FIELD metadata ON events FLEXIBLE TYPE option<object>;`, OnTable: "events"},

		// Fields for events - Timestamp
		{Type: "field", Statement: `DEFINE FIELD created_at ON events TYPE datetime DEFAULT time::now();`, OnTable: "events"},

		// ===========================================
		// INDEXES for events
		// ===========================================
		// Primary lookups by user
		{Type: "index", Statement: `DEFINE INDEX idx_events_user ON events FIELDS user_id;`, OnTable: "events"},

		// Subject filter (e.g., filter by "log:build" or "conversation:*")
		{Type: "index", Statement: `DEFINE INDEX idx_events_subject ON events FIELDS subject;`, OnTable: "events"},

		// Temporal queries
		{Type: "index", Statement: `DEFINE INDEX idx_events_created ON events FIELDS created_at;`, OnTable: "events"},

		// Compound index for user + subject queries
		{Type: "index", Statement: `DEFINE INDEX idx_events_user_subject ON events FIELDS user_id, subject;`, OnTable: "events"},

		// Vector search - MTREE with 768 dimensions for semantic search
		{Type: "index", Statement: `DEFINE INDEX idx_events_embedding ON events FIELDS embedding MTREE DIMENSION 768 DIST COSINE;`, OnTable: "events"},

		// Full-text search with BM25 for hybrid search
		{Type: "index", Statement: `DEFINE ANALYZER events_analyzer TOKENIZERS blank, class FILTERS lowercase, snowball(english);`, OnTable: "events"},
		{Type: "index", Statement: `DEFINE INDEX idx_events_content ON events FIELDS content SEARCH ANALYZER events_analyzer BM25;`, OnTable: "events"},
	}

	return m.ApplyElements(ctx, elements)
}
