package migrations

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

// tableIndexPrefix maps a memory table to the prefix used in its v13 index
// names. Index names are part of the contract RM-2, RM-3 and RM-9 query
// against, so they live in one table rather than being derived ad hoc.
var tableIndexPrefix = map[string]string{
	"kv_memories":        "kv",
	"vector_memories":    "vector",
	"knowledge_base":     "kb",
	"events":             "events",
	"entities":           "entity",
	"wrote":              "wrote",
	"mentioned_in":       "mentioned",
	"related_to":         "related",
	"code_projects":      "code_project",
	"code_files":         "code_file",
	"code_symbols":       "code_symbol",
	"code_chunks":        "code_chunk",
	"code_indexing_jobs": "code_job",
}

// HashUniqueTables are the tables that gain a UNIQUE
// (org_id, scope, content_hash) index in v13.
//
// Deviation from the RM-1 design, which named the constraint only for
// kv_memories: it is applied to kv_memories and vector_memories, the two tables
// where storing the same content twice in one scope is always a bug. It is
// deliberately NOT applied to knowledge_base (already unique on
// (org_id, scope, file_path); two chunks of one document can legitimately share
// content), to events (append-only temporal records, where the same content at
// two times is the normal case) or to the code_* family (content_hash stays
// NONE there, and NONE values would collide).
var HashUniqueTables = []string{"kv_memories", "vector_memories"}

// ScopeColumns is the ordered list of columns v13 adds to every memory table.
// Callers that need to know the tenant/scope/principal/validity contract read
// this rather than re-deriving it.
var ScopeColumns = []string{
	"org_id",
	"scope",
	"created_by",
	"source",
	"visibility",
	"valid_from",
	"valid_to",
	"superseded_by",
	"content_hash",
	"updated_at",
	"version",
}

// PrincipalTables are the tables Alchemai Mem resolves permissions against.
var PrincipalTables = []string{
	"org",
	"app_user",
	"user_group",
	"member_of",
	"mem_scope",
	"grant",
	"audit_log",
	"quota_usage",
}

// V13TenantScopePrincipal implements the tenant, scope, principal and validity
// migration. It is one-way: there is no v13 to v12 path and none will be
// written (see the RM-1 plan, "Downgrade policy: none").
type V13TenantScopePrincipal struct {
	*MigrationBase
}

// NewV13TenantScopePrincipal creates a new V13 migration.
func NewV13TenantScopePrincipal(db *surrealdb.DB) Migration {
	return &V13TenantScopePrincipal{
		MigrationBase: NewMigrationBase(db),
	}
}

// Version returns the migration version.
func (m *V13TenantScopePrincipal) Version() int { return 13 }

// Description returns the migration description.
func (m *V13TenantScopePrincipal) Description() string {
	return "Adding tenant, scope, principal and validity model (v13)"
}

// Apply executes the DDL half of v13 through the remote client. The backfill
// half needs an Executor and runs from ApplyV13, which the storage layer calls
// for both the remote and the embedded path.
func (m *V13TenantScopePrincipal) Apply(ctx context.Context, db *surrealdb.DB) error {
	slog.Info("Applying migration v13: tenant, scope, principal and validity model")
	return m.ApplyElements(ctx, Statements(13, EmbeddingDimension()))
}

// v13Statements renders the whole v13 DDL: the principal tables, the permissive
// columns on every memory table, the scope indexes, the HNSW rebuild and the
// full-text index. Phases 3 and 4 of the design (backfill and dedup) are data
// operations and live in v13_backfill.go.
func v13Statements(dim int) []SchemaElement {
	var out []SchemaElement
	out = append(out, v13PrincipalTables()...)
	out = append(out, v13ScopeColumns()...)
	out = append(out, v13ScopeIndexes()...)
	out = append(out, v13VectorIndexes(dim)...)
	out = append(out, v13FullText()...)
	return out
}

// v13PrincipalTables is phase 1: org, app_user, user_group, member_of,
// mem_scope, grant, audit_log and quota_usage.
func v13PrincipalTables() []SchemaElement {
	return []SchemaElement{
		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS org SCHEMAFULL;`, OnTable: "org"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE name ON org TYPE string;`, OnTable: "org"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE oidc_issuer ON org TYPE option<string>;`, OnTable: "org"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE status ON org TYPE string DEFAULT 'active' ASSERT $value IN ['active', 'suspended', 'archived'];`, OnTable: "org"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE created_at ON org TYPE datetime DEFAULT time::now() READONLY;`, OnTable: "org"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_org_name ON org FIELDS name UNIQUE;`, OnTable: "org"},

		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS app_user SCHEMAFULL;`, OnTable: "app_user"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE org ON app_user TYPE record<org>;`, OnTable: "app_user"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE subject ON app_user TYPE string;`, OnTable: "app_user"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE email ON app_user TYPE option<string>;`, OnTable: "app_user"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE display_name ON app_user TYPE option<string>;`, OnTable: "app_user"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE is_org_admin ON app_user TYPE bool DEFAULT false;`, OnTable: "app_user"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE active ON app_user TYPE bool DEFAULT true;`, OnTable: "app_user"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE external_id ON app_user TYPE option<string>;`, OnTable: "app_user"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE created_at ON app_user TYPE datetime DEFAULT time::now() READONLY;`, OnTable: "app_user"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE updated_at ON app_user TYPE datetime DEFAULT time::now();`, OnTable: "app_user"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_app_user_subject ON app_user FIELDS org, subject UNIQUE;`, OnTable: "app_user"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_app_user_email ON app_user FIELDS org, email;`, OnTable: "app_user"},

		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS user_group SCHEMAFULL;`, OnTable: "user_group"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE org ON user_group TYPE record<org>;`, OnTable: "user_group"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE name ON user_group TYPE string;`, OnTable: "user_group"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE kind ON user_group TYPE string DEFAULT 'static' ASSERT $value IN ['static', 'dynamic', 'mirrored'];`, OnTable: "user_group"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE source ON user_group TYPE option<string>;`, OnTable: "user_group"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE external_id ON user_group TYPE option<string>;`, OnTable: "user_group"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE created_at ON user_group TYPE datetime DEFAULT time::now() READONLY;`, OnTable: "user_group"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_user_group_name ON user_group FIELDS org, name UNIQUE;`, OnTable: "user_group"},

		// A real RELATION table. ENFORCED refuses an edge to a record that does
		// not exist, which is what stops a stale membership granting access.
		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS member_of TYPE RELATION IN app_user OUT user_group ENFORCED SCHEMAFULL;`, OnTable: "member_of"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE role ON member_of TYPE string DEFAULT 'member' ASSERT $value IN ['member', 'owner'];`, OnTable: "member_of"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE org ON member_of TYPE record<org>;`, OnTable: "member_of"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE joined_at ON member_of TYPE datetime DEFAULT time::now() READONLY;`, OnTable: "member_of"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_member_of_org ON member_of FIELDS org, in, out UNIQUE;`, OnTable: "member_of"},

		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS mem_scope SCHEMAFULL;`, OnTable: "mem_scope"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE org ON mem_scope TYPE record<org>;`, OnTable: "mem_scope"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE kind ON mem_scope TYPE string ASSERT $value IN ['personal', 'group', 'org', 'project'];`, OnTable: "mem_scope"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE owner_user ON mem_scope TYPE option<record<app_user>>;`, OnTable: "mem_scope"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE owner_group ON mem_scope TYPE option<record<user_group>>;`, OnTable: "mem_scope"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE label ON mem_scope TYPE string;`, OnTable: "mem_scope"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE retention_policy ON mem_scope TYPE option<string>;`, OnTable: "mem_scope"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE created_at ON mem_scope TYPE datetime DEFAULT time::now() READONLY;`, OnTable: "mem_scope"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_mem_scope_label ON mem_scope FIELDS org, label UNIQUE;`, OnTable: "mem_scope"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_mem_scope_owner ON mem_scope FIELDS org, owner_user;`, OnTable: "mem_scope"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_mem_scope_group ON mem_scope FIELDS org, owner_group;`, OnTable: "mem_scope"},

		// `grant` is escaped throughout. It is not a documented reserved word,
		// but the escape costs nothing and removes any risk of a future GRANT
		// statement colliding with the table name.
		{Type: "table", Statement: "DEFINE TABLE IF NOT EXISTS `grant` SCHEMAFULL;", OnTable: "grant"},
		{Type: "field", Statement: "DEFINE FIELD OVERWRITE org ON `grant` TYPE record<org>;", OnTable: "grant"},
		{Type: "field", Statement: "DEFINE FIELD OVERWRITE scope ON `grant` TYPE record<mem_scope>;", OnTable: "grant"},
		{Type: "field", Statement: "DEFINE FIELD OVERWRITE principal_kind ON `grant` TYPE string ASSERT $value IN ['user', 'group', 'org'];", OnTable: "grant"},
		{Type: "field", Statement: "DEFINE FIELD OVERWRITE principal ON `grant` TYPE string;", OnTable: "grant"},
		{Type: "field", Statement: "DEFINE FIELD OVERWRITE perms ON `grant` TYPE array<string> ASSERT $value.all(|$p| $p IN ['read', 'write', 'delete', 'share']);", OnTable: "grant"},
		{Type: "field", Statement: "DEFINE FIELD OVERWRITE granted_by ON `grant` TYPE option<record<app_user>>;", OnTable: "grant"},
		{Type: "field", Statement: "DEFINE FIELD OVERWRITE granted_at ON `grant` TYPE datetime DEFAULT time::now() READONLY;", OnTable: "grant"},
		{Type: "field", Statement: "DEFINE FIELD OVERWRITE expires_at ON `grant` TYPE option<datetime>;", OnTable: "grant"},
		{Type: "field", Statement: "DEFINE FIELD OVERWRITE reason ON `grant` TYPE option<string>;", OnTable: "grant"},
		{Type: "index", Statement: "DEFINE INDEX IF NOT EXISTS idx_grant_scope ON `grant` FIELDS org, scope;", OnTable: "grant"},
		{Type: "index", Statement: "DEFINE INDEX IF NOT EXISTS idx_grant_principal ON `grant` FIELDS org, principal_kind, principal;", OnTable: "grant"},
		{Type: "index", Statement: "DEFINE INDEX IF NOT EXISTS idx_grant_unique ON `grant` FIELDS org, scope, principal_kind, principal UNIQUE;", OnTable: "grant"},

		// Append only. There is no update or delete path in the storage layer.
		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS audit_log SCHEMAFULL;`, OnTable: "audit_log"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE org ON audit_log TYPE record<org>;`, OnTable: "audit_log"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE actor ON audit_log TYPE string;`, OnTable: "audit_log"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE actor_kind ON audit_log TYPE string ASSERT $value IN ['user', 'service', 'agent', 'worker'];`, OnTable: "audit_log"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE action ON audit_log TYPE string;`, OnTable: "audit_log"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE target_table ON audit_log TYPE option<string>;`, OnTable: "audit_log"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE target_ids ON audit_log TYPE option<array<string>>;`, OnTable: "audit_log"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE scope ON audit_log TYPE option<record<mem_scope>>;`, OnTable: "audit_log"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE decision ON audit_log TYPE string ASSERT $value IN ['allow', 'deny'];`, OnTable: "audit_log"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE reason ON audit_log TYPE option<string>;`, OnTable: "audit_log"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE request_id ON audit_log TYPE option<string>;`, OnTable: "audit_log"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE instance_id ON audit_log TYPE option<string>;`, OnTable: "audit_log"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE ts ON audit_log TYPE datetime DEFAULT time::now() READONLY;`, OnTable: "audit_log"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_audit_org_ts ON audit_log FIELDS org, ts;`, OnTable: "audit_log"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_audit_org_actor ON audit_log FIELDS org, actor, ts;`, OnTable: "audit_log"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_audit_org_scope ON audit_log FIELDS org, scope, ts;`, OnTable: "audit_log"},

		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS quota_usage SCHEMAFULL;`, OnTable: "quota_usage"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE org ON quota_usage TYPE record<org>;`, OnTable: "quota_usage"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE principal ON quota_usage TYPE string;`, OnTable: "quota_usage"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE period ON quota_usage TYPE string;`, OnTable: "quota_usage"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE ingest_bytes ON quota_usage TYPE int DEFAULT 0;`, OnTable: "quota_usage"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE records ON quota_usage TYPE int DEFAULT 0;`, OnTable: "quota_usage"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE embed_calls ON quota_usage TYPE int DEFAULT 0;`, OnTable: "quota_usage"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_quota_unique ON quota_usage FIELDS org, principal, period UNIQUE;`, OnTable: "quota_usage"},
	}
}

// v13ScopeColumns is phase 2: the eleven columns, added permissively so that a
// SCHEMAFULL table with existing rows can gain them in one step. They are
// tightened after the backfill (phase 4, in v13_backfill.go).
func v13ScopeColumns() []SchemaElement {
	var out []SchemaElement
	for _, table := range MemoryTables {
		out = append(out,
			SchemaElement{Type: "field", Statement: fmt.Sprintf(`DEFINE FIELD OVERWRITE org_id ON %s TYPE option<record<org>>;`, table), OnTable: table},
			SchemaElement{Type: "field", Statement: fmt.Sprintf(`DEFINE FIELD OVERWRITE scope ON %s TYPE option<record<mem_scope>>;`, table), OnTable: table},
			SchemaElement{Type: "field", Statement: fmt.Sprintf(`DEFINE FIELD OVERWRITE created_by ON %s TYPE option<record<app_user>>;`, table), OnTable: table},
			SchemaElement{Type: "field", Statement: objectField("source", table, "object", "{}"), OnTable: table},
			SchemaElement{Type: "field", Statement: fmt.Sprintf(`DEFINE FIELD OVERWRITE visibility ON %s TYPE string DEFAULT 'private' ASSERT $value IN ['private', 'group', 'org'];`, table), OnTable: table},
			SchemaElement{Type: "field", Statement: fmt.Sprintf(`DEFINE FIELD OVERWRITE valid_from ON %s TYPE datetime DEFAULT time::now();`, table), OnTable: table},
			SchemaElement{Type: "field", Statement: fmt.Sprintf(`DEFINE FIELD OVERWRITE valid_to ON %s TYPE option<datetime>;`, table), OnTable: table},
			SchemaElement{Type: "field", Statement: fmt.Sprintf(`DEFINE FIELD OVERWRITE superseded_by ON %s TYPE option<record>;`, table), OnTable: table},
			SchemaElement{Type: "field", Statement: fmt.Sprintf(`DEFINE FIELD OVERWRITE content_hash ON %s TYPE option<string>;`, table), OnTable: table},
			SchemaElement{Type: "field", Statement: fmt.Sprintf(`DEFINE FIELD OVERWRITE updated_at ON %s TYPE datetime DEFAULT time::now();`, table), OnTable: table},
			SchemaElement{Type: "field", Statement: fmt.Sprintf(`DEFINE FIELD OVERWRITE version ON %s TYPE int DEFAULT 0;`, table), OnTable: table},
		)
	}
	return out
}

// v13ScopeIndexes is phase 5. idx_<p>_org_version is what makes the RM-9 change
// feed a range scan rather than a table scan; idx_<p>_valid is what makes "the
// live rows of this scope" cheap, which is the shape of nearly every
// ACL-filtered query.
func v13ScopeIndexes() []SchemaElement {
	var out []SchemaElement
	for _, table := range MemoryTables {
		p := tableIndexPrefix[table]
		out = append(out,
			SchemaElement{Type: "index", Statement: fmt.Sprintf(`DEFINE INDEX IF NOT EXISTS idx_%s_org_scope ON %s FIELDS org_id, scope;`, p, table), OnTable: table},
			SchemaElement{Type: "index", Statement: fmt.Sprintf(`DEFINE INDEX IF NOT EXISTS idx_%s_org_creator ON %s FIELDS org_id, created_by;`, p, table), OnTable: table},
			SchemaElement{Type: "index", Statement: fmt.Sprintf(`DEFINE INDEX IF NOT EXISTS idx_%s_org_version ON %s FIELDS org_id, version;`, p, table), OnTable: table},
			SchemaElement{Type: "index", Statement: fmt.Sprintf(`DEFINE INDEX IF NOT EXISTS idx_%s_valid ON %s FIELDS org_id, scope, valid_to;`, p, table), OnTable: table},
		)
	}

	// The knowledge_base uniqueness moves from file_path to
	// (org_id, scope, file_path). Both legacy names are removed because the
	// remote/embedded fork means a given datastore carries one or the other and
	// the migration cannot know which.
	out = append(out,
		SchemaElement{Type: "index", Statement: `REMOVE INDEX IF EXISTS idx_kb_path ON knowledge_base;`, OnTable: "knowledge_base"},
		SchemaElement{Type: "index", Statement: `REMOVE INDEX IF EXISTS idx_kb_file_path ON knowledge_base;`, OnTable: "knowledge_base"},
		SchemaElement{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_kb_scope_path ON knowledge_base FIELDS org_id, scope, file_path UNIQUE;`, OnTable: "knowledge_base"},
	)
	return out
}

// v13VectorIndexes is phase 6: MTREE is removed in 3.x, so every vector index is
// dropped by both of its legacy names and rebuilt as HNSW at the configured
// dimension. CONCURRENTLY keeps the build off the migration's critical path;
// DEFER is 2.x only and is a parse error on 3.x.
func v13VectorIndexes(dim int) []SchemaElement {
	out := []SchemaElement{
		// The v1 remote schema called the vector_memories index idx_embedding.
		{Type: "index", Statement: `REMOVE INDEX IF EXISTS idx_embedding ON vector_memories;`, OnTable: "vector_memories"},
	}
	for _, vi := range VectorIndexes {
		out = append(out,
			SchemaElement{Type: "index", Statement: fmt.Sprintf(`REMOVE INDEX IF EXISTS %s ON %s;`, vi.Index, vi.Table), OnTable: vi.Table},
			SchemaElement{Type: "index", Statement: hnswIndex(vi.Index, vi.Table, vi.Field, dim, true), OnTable: vi.Table},
		)
	}
	return out
}

// v13FullText is phase 7. FULLTEXT replaces SEARCH; it also enables the OR
// operator and concurrent full-text search, which the hybrid path wants.
// HIGHLIGHTS is added because Control's memory browser shows the matching span.
//
// This also fixes a v11 quirk: the analyzer was registered as an index element,
// so the existence probe looked for an index named events_analyzer, never found
// one, and re-executed the statement on every boot. DEFINE ANALYZER OVERWRITE
// removes the probe and the accident together.
func v13FullText() []SchemaElement {
	return []SchemaElement{
		{Type: "analyzer", Statement: `DEFINE ANALYZER OVERWRITE events_analyzer TOKENIZERS blank, class FILTERS lowercase, snowball(english);`, OnTable: ""},
		{Type: "index", Statement: `REMOVE INDEX IF EXISTS idx_events_content ON events;`, OnTable: "events"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_events_content ON events FIELDS content FULLTEXT ANALYZER events_analyzer BM25 HIGHLIGHTS;`, OnTable: "events"},
	}
}
