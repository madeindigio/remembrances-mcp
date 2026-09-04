package migrations

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

// ChangeFeedSequence is the name of the org-wide monotonic counter every write
// draws its `version` from.
//
// # Why a sequence and not a per-row counter
//
// v13 gave every memory row a `version` and indexed (org_id, version). A
// counter that only has to increase per row makes that index useless as a feed
// order: two rows can carry version 1 for ever, so "give me everything after
// cursor X" has no answer. A single shared sequence makes (org_id, version) a
// strict total order across the whole datastore, which is what turns
// idx_<prefix>_org_version into the range scan RM-9 specifies and what makes a
// cursor resumable with no gaps.
//
// BATCH 1 is deliberate. A larger batch lets a node cache a block of ids and
// hand them out later than ids another node has already published, which for a
// change feed means a row can appear below a cursor a client has already
// passed. With BATCH 1 every allocation is a coordinated step, which is the
// correct trade for a feed: allocation is not on the hot path of a read, and
// correctness of the cursor is the whole point of the feature.
const ChangeFeedSequence = "mem_version"

// V14ChangeFeed adds the change feed sequence and re-cuts the v13 content hash
// uniqueness so that tombstones do not block re-creation.
type V14ChangeFeed struct {
	*MigrationBase
}

// NewV14ChangeFeed creates a new V14 migration.
func NewV14ChangeFeed(db *surrealdb.DB) Migration {
	return &V14ChangeFeed{MigrationBase: NewMigrationBase(db)}
}

// Version returns the migration version.
func (m *V14ChangeFeed) Version() int { return 14 }

// Description returns the migration description.
func (m *V14ChangeFeed) Description() string {
	return "Adding the change feed sequence and tombstone-tolerant hash uniqueness (v14)"
}

// Apply executes the migration.
func (m *V14ChangeFeed) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration v14: change feed sequence and tombstone-tolerant uniqueness")
	return m.ApplyElements(ctx, Statements(14, EmbeddingDimension()))
}

// v14Statements renders the whole v14 DDL.
func v14Statements() []SchemaElement {
	out := []SchemaElement{
		{
			Type:      "sequence",
			Statement: fmt.Sprintf(`DEFINE SEQUENCE IF NOT EXISTS %s BATCH 1 START 1;`, ChangeFeedSequence),
			OnTable:   "",
		},
	}

	// The v13 index was UNIQUE (org_id, scope, content_hash). Soft deletion is
	// the rule everywhere (invalidate, never delete), so a tombstone keeps its
	// content_hash and would permanently block storing that content in that
	// scope again. Adding valid_to to the key keeps exactly one *live* row per
	// hash, which is the constraint that was actually wanted, while letting any
	// number of invalidated ones sit behind it.
	for _, table := range HashUniqueTables {
		p := tableIndexPrefix[table]
		out = append(out,
			SchemaElement{
				Type:      "index",
				Statement: fmt.Sprintf(`REMOVE INDEX IF EXISTS idx_%s_hash ON %s;`, p, table),
				OnTable:   table,
			},
			SchemaElement{
				Type:      "index",
				Statement: fmt.Sprintf(`DEFINE INDEX IF NOT EXISTS idx_%s_hash ON %s FIELDS org_id, scope, content_hash, valid_to UNIQUE;`, p, table),
				OnTable:   table,
			},
		)
	}

	// knowledge_base uniqueness needs the same treatment for the same reason:
	// v13 made it UNIQUE (org_id, scope, file_path), and re-saving a document
	// after it was invalidated would collide with its own tombstone. The index
	// keeps its name, so nothing that reads the index list has to change.
	out = append(out,
		SchemaElement{Type: "index", Statement: `REMOVE INDEX IF EXISTS idx_kb_scope_path ON knowledge_base;`, OnTable: "knowledge_base"},
		SchemaElement{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_kb_scope_path ON knowledge_base FIELDS org_id, scope, file_path, valid_to UNIQUE;`, OnTable: "knowledge_base"},
	)

	// The graph edge tables are created lazily by CreateRelationship, which
	// names the table after the relationship type. The three canonical ones are
	// declared here so their scope columns and indexes exist before the first
	// edge is written rather than only after v13 has seen the table.
	for _, table := range EdgeTables {
		p := tableIndexPrefix[table]
		out = append(out,
			SchemaElement{Type: "table", Statement: fmt.Sprintf(`DEFINE TABLE IF NOT EXISTS %s SCHEMALESS;`, table), OnTable: table},
			SchemaElement{Type: "index", Statement: fmt.Sprintf(`DEFINE INDEX IF NOT EXISTS idx_%s_from ON %s FIELDS org_id, scope, from_entity;`, p, table), OnTable: table},
		)
	}

	return out
}

// EdgeTables are the relationship tables the graph layer writes by default.
// CreateRelationship can create others on demand; these are the ones v13 gave
// scope columns to and the ones TraverseGraph walks when no relationship type
// is named.
var EdgeTables = []string{"wrote", "mentioned_in", "related_to"}
