package migrations

import "fmt"

// TargetVersion is the schema version a current binary migrates to.
const TargetVersion = 14

// MemoryTables lists every table that carries the v13 tenant, scope, principal
// and validity columns. The code_* family is included for uniformity: a table
// without the columns would be a hole in the ACL wrapper's uniform preamble.
var MemoryTables = []string{
	"kv_memories",
	"vector_memories",
	"knowledge_base",
	"events",
	"entities",
	"wrote",
	"mentioned_in",
	"related_to",
	"code_projects",
	"code_files",
	"code_symbols",
	"code_chunks",
	"code_indexing_jobs",
}

// VectorIndexes maps each table carrying an embedding to the HNSW index name
// v13 defines on it. It is the single list the schema, the tests and the
// migration all read, so an index cannot be added in one place and forgotten in
// another.
var VectorIndexes = []struct {
	Table string
	Index string
	Field string
}{
	{Table: "vector_memories", Index: "idx_vector_embedding", Field: "embedding"},
	{Table: "knowledge_base", Index: "idx_kb_embedding", Field: "embedding"},
	{Table: "events", Index: "idx_events_embedding", Field: "embedding"},
	{Table: "code_symbols", Index: "idx_code_symbol_embedding", Field: "embedding"},
	{Table: "code_chunks", Index: "idx_code_chunk_embedding", Field: "embedding"},
}

// HNSW tuning defaults. SurrealDB 3.x computes M0 and LM itself and they are not
// settable; M and EFC default to 12 and 150 and are pinned explicitly here so a
// future change of the engine default cannot silently reshape an existing
// deployment's index. Revisit in the week 11 performance pass (risk R6).
const (
	hnswM   = 12
	hnswEFC = 150
)

// hnswIndex renders a 3.x HNSW vector index definition. The clause order is
// `HNSW DIMENSION n [TYPE t] [DIST d] [EFC e] [M m]`: TYPE precedes DIST in the
// grammar. TYPE F32 matches the array<float> storage and is the engine default.
func hnswIndex(name, table, field string, dim int, concurrently bool) string {
	stmt := fmt.Sprintf(
		"DEFINE INDEX IF NOT EXISTS %s ON %s FIELDS %s HNSW DIMENSION %d TYPE F32 DIST COSINE EFC %d M %d",
		name, table, field, dim, hnswEFC, hnswM,
	)
	if concurrently {
		stmt += " CONCURRENTLY"
	}
	return stmt + ";"
}

// schemalessTables are declared SCHEMALESS, and SurrealDB 3.x refuses FLEXIBLE
// on a table that is not SCHEMAFULL ("FLEXIBLE can only be used in SCHEMAFULL
// tables"). A SCHEMALESS table already accepts an arbitrary object, so the
// modifier is redundant there rather than merely unavailable.
var schemalessTables = map[string]bool{
	"wrote":        true,
	"mentioned_in": true,
	"related_to":   true,
}

// objectField renders an object field, adding FLEXIBLE only where the engine
// accepts it.
func objectField(field, table, typ, def string) string {
	stmt := fmt.Sprintf("DEFINE FIELD OVERWRITE %s ON %s TYPE %s", field, table, typ)
	if !schemalessTables[table] {
		stmt += " FLEXIBLE"
	}
	if def != "" {
		stmt += " DEFAULT " + def
	}
	return stmt + ";"
}

// vectorField renders an embedding field type. SurrealDB 3.x reads
// `array<float, N>` as exactly N elements, which is why storage normalises every
// embedding to EmbeddingDimension() before writing.
func vectorField(field, table string, dim int, optional bool) string {
	typ := fmt.Sprintf("array<float, %d>", dim)
	if optional {
		typ = "option<" + typ + ">"
	}
	return fmt.Sprintf("DEFINE FIELD OVERWRITE %s ON %s TYPE %s;", field, table, typ)
}

// Statements returns the canonical statement list for a schema version rendered
// at the given embedding dimension. It is the single source of SurrealQL for
// both the remote and the embedded execution paths: the embedded path is a thin
// wrapper over this function, so the two can no longer diverge.
//
// Dialect is SurrealDB 3.x throughout: HNSW rather than MTREE, FULLTEXT rather
// than SEARCH ANALYZER, and FLEXIBLE as a postfix modifier after TYPE.
func Statements(version, dim int) []SchemaElement {
	switch version {
	case 1:
		return v1Statements(dim)
	case 2:
		return v2Statements()
	case 3:
		return v3Statements()
	case 4:
		return v4Statements()
	case 5:
		return v5Statements()
	case 6:
		return v6Statements()
	case 7:
		return v7Statements()
	case 8:
		return v8Statements()
	case 9:
		return v9Statements(dim)
	case 10:
		return v10Statements(dim)
	case 11:
		return v11Statements(dim)
	case 12:
		return v12Statements()
	case 13:
		return v13Statements(dim)
	case 14:
		return v14Statements()
	default:
		return nil
	}
}

// StatementStrings returns the raw SurrealQL for a version, for callers that
// execute statements directly rather than through MigrationBase.
func StatementStrings(version, dim int) []string {
	elements := Statements(version, dim)
	if elements == nil {
		return nil
	}
	out := make([]string, 0, len(elements))
	for _, e := range elements {
		out = append(out, e.Statement)
	}
	return out
}

func v1Statements(dim int) []SchemaElement {
	return []SchemaElement{
		// Tables.
		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS kv_memories SCHEMAFULL;`, OnTable: "kv_memories"},
		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS vector_memories SCHEMAFULL;`, OnTable: "vector_memories"},
		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS knowledge_base SCHEMAFULL;`, OnTable: "knowledge_base"},
		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS entities SCHEMAFULL;`, OnTable: "entities"},
		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS wrote SCHEMALESS;`, OnTable: "wrote"},
		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS mentioned_in SCHEMALESS;`, OnTable: "mentioned_in"},
		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS related_to SCHEMALESS;`, OnTable: "related_to"},

		// kv_memories. user_id is optional: the write path omits it when the
		// caller supplies none, and 3.x rejects a missing required field.
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE user_id ON kv_memories TYPE option<string>;`, OnTable: "kv_memories"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE key ON kv_memories TYPE string;`, OnTable: "kv_memories"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE value ON kv_memories TYPE any;`, OnTable: "kv_memories"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE created_at ON kv_memories TYPE datetime DEFAULT time::now();`, OnTable: "kv_memories"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE updated_at ON kv_memories TYPE datetime DEFAULT time::now();`, OnTable: "kv_memories"},

		// vector_memories.
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE user_id ON vector_memories TYPE option<string>;`, OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE content ON vector_memories TYPE string;`, OnTable: "vector_memories"},
		{Type: "field", Statement: vectorField("embedding", "vector_memories", dim, false), OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE metadata ON vector_memories TYPE object FLEXIBLE DEFAULT {};`, OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE created_at ON vector_memories TYPE datetime DEFAULT time::now();`, OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE updated_at ON vector_memories TYPE datetime DEFAULT time::now();`, OnTable: "vector_memories"},

		// knowledge_base.
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE file_path ON knowledge_base TYPE string;`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE content ON knowledge_base TYPE string;`, OnTable: "knowledge_base"},
		{Type: "field", Statement: vectorField("embedding", "knowledge_base", dim, false), OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE metadata ON knowledge_base TYPE object FLEXIBLE DEFAULT {};`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE created_at ON knowledge_base TYPE datetime DEFAULT time::now();`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE updated_at ON knowledge_base TYPE datetime DEFAULT time::now();`, OnTable: "knowledge_base"},

		// entities. entity_type is the canonical spelling; it is what the Go
		// write path sends. The remote schema used to call it `type`, which no
		// write ever populated.
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE name ON entities TYPE string;`, OnTable: "entities"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE entity_type ON entities TYPE string;`, OnTable: "entities"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE properties ON entities TYPE object FLEXIBLE DEFAULT {};`, OnTable: "entities"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE created_at ON entities TYPE datetime DEFAULT time::now();`, OnTable: "entities"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE updated_at ON entities TYPE datetime DEFAULT time::now();`, OnTable: "entities"},

		// Edge tables. in and out are records on all three: they are written by
		// RELATE, which stores record links, not strings.
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE in ON wrote TYPE record;`, OnTable: "wrote"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE out ON wrote TYPE record;`, OnTable: "wrote"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE timestamp ON wrote TYPE datetime DEFAULT time::now();`, OnTable: "wrote"},
		{Type: "field", Statement: objectField("properties", "wrote", "object", "{}"), OnTable: "wrote"},

		{Type: "field", Statement: `DEFINE FIELD OVERWRITE in ON mentioned_in TYPE record;`, OnTable: "mentioned_in"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE out ON mentioned_in TYPE record;`, OnTable: "mentioned_in"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE timestamp ON mentioned_in TYPE datetime DEFAULT time::now();`, OnTable: "mentioned_in"},
		{Type: "field", Statement: objectField("properties", "mentioned_in", "object", "{}"), OnTable: "mentioned_in"},

		{Type: "field", Statement: `DEFINE FIELD OVERWRITE in ON related_to TYPE record;`, OnTable: "related_to"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE out ON related_to TYPE record;`, OnTable: "related_to"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE timestamp ON related_to TYPE datetime DEFAULT time::now();`, OnTable: "related_to"},
		{Type: "field", Statement: objectField("properties", "related_to", "object", "{}"), OnTable: "related_to"},

		// Indexes.
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_kv_user_key ON kv_memories FIELDS user_id, key UNIQUE;`, OnTable: "kv_memories"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_vector_user ON vector_memories FIELDS user_id;`, OnTable: "vector_memories"},
		{Type: "index", Statement: hnswIndex("idx_vector_embedding", "vector_memories", "embedding", dim, false), OnTable: "vector_memories"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_kb_file_path ON knowledge_base FIELDS file_path UNIQUE;`, OnTable: "knowledge_base"},
		{Type: "index", Statement: hnswIndex("idx_kb_embedding", "knowledge_base", "embedding", dim, false), OnTable: "knowledge_base"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_entity_name ON entities FIELDS name;`, OnTable: "entities"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_entity_type ON entities FIELDS entity_type;`, OnTable: "entities"},
	}
}

func v2Statements() []SchemaElement {
	return []SchemaElement{
		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS user_stats SCHEMAFULL;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE user_id ON user_stats TYPE string;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE key_value_count ON user_stats TYPE int DEFAULT 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE vector_count ON user_stats TYPE int DEFAULT 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE entity_count ON user_stats TYPE int DEFAULT 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE relationship_count ON user_stats TYPE int DEFAULT 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE document_count ON user_stats TYPE int DEFAULT 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE fact_count ON user_stats TYPE int DEFAULT 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE created_at ON user_stats TYPE datetime DEFAULT time::now();`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE updated_at ON user_stats TYPE datetime DEFAULT time::now();`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE last_updated ON user_stats TYPE datetime DEFAULT time::now();`, OnTable: "user_stats"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_user_stats_user_id ON user_stats FIELDS user_id UNIQUE;`, OnTable: "user_stats"},
	}
}

// v3Statements drops the `VALUE 0` clauses v2 originally carried. VALUE
// recomputes on every write, so the counters were reset to zero on each update.
func v3Statements() []SchemaElement {
	return []SchemaElement{
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE key_value_count ON user_stats TYPE int DEFAULT 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE vector_count ON user_stats TYPE int DEFAULT 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE entity_count ON user_stats TYPE int DEFAULT 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE relationship_count ON user_stats TYPE int DEFAULT 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE document_count ON user_stats TYPE int DEFAULT 0;`, OnTable: "user_stats"},
	}
}

func v4Statements() []SchemaElement {
	return []SchemaElement{
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE user_id ON entities TYPE option<string>;`, OnTable: "entities"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE user_id ON knowledge_base TYPE option<string>;`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE user_id ON wrote TYPE option<string>;`, OnTable: "wrote"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE user_id ON mentioned_in TYPE option<string>;`, OnTable: "mentioned_in"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE user_id ON related_to TYPE option<string>;`, OnTable: "related_to"},
	}
}

func v5Statements() []SchemaElement {
	return []SchemaElement{
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE metadata ON knowledge_base TYPE object FLEXIBLE DEFAULT {};`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE metadata ON vector_memories TYPE object FLEXIBLE DEFAULT {};`, OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE properties ON entities TYPE object FLEXIBLE DEFAULT {};`, OnTable: "entities"},
		{Type: "field", Statement: objectField("properties", "wrote", "object", "{}"), OnTable: "wrote"},
		{Type: "field", Statement: objectField("properties", "mentioned_in", "object", "{}"), OnTable: "mentioned_in"},
		{Type: "field", Statement: objectField("properties", "related_to", "object", "{}"), OnTable: "related_to"},
	}
}

func v6Statements() []SchemaElement {
	return []SchemaElement{
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE chunk_index ON knowledge_base TYPE int DEFAULT 0;`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE chunk_count ON knowledge_base TYPE int DEFAULT 0;`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE source_file ON knowledge_base TYPE option<string>;`, OnTable: "knowledge_base"},
	}
}

func v7Statements() []SchemaElement {
	return []SchemaElement{
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE metadata ON vector_memories TYPE object FLEXIBLE DEFAULT {};`, OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE metadata ON knowledge_base TYPE object FLEXIBLE DEFAULT {};`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE properties ON entities TYPE object FLEXIBLE DEFAULT {};`, OnTable: "entities"},
	}
}

func v8Statements() []SchemaElement {
	return []SchemaElement{
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE value ON kv_memories TYPE option<string | int | float | bool | object | array> FLEXIBLE;`, OnTable: "kv_memories"},
	}
}

func v9Statements(dim int) []SchemaElement {
	return []SchemaElement{
		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS code_projects SCHEMAFULL;`, OnTable: "code_projects"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE project_id ON code_projects TYPE string;`, OnTable: "code_projects"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE name ON code_projects TYPE string;`, OnTable: "code_projects"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE root_path ON code_projects TYPE string;`, OnTable: "code_projects"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE language_stats ON code_projects TYPE option<object> FLEXIBLE;`, OnTable: "code_projects"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE last_indexed_at ON code_projects TYPE option<datetime>;`, OnTable: "code_projects"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE indexing_status ON code_projects TYPE string DEFAULT 'pending';`, OnTable: "code_projects"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE created_at ON code_projects TYPE datetime DEFAULT time::now();`, OnTable: "code_projects"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE updated_at ON code_projects TYPE datetime DEFAULT time::now();`, OnTable: "code_projects"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_project_id ON code_projects FIELDS project_id UNIQUE;`, OnTable: "code_projects"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_project_name ON code_projects FIELDS name;`, OnTable: "code_projects"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_project_status ON code_projects FIELDS indexing_status;`, OnTable: "code_projects"},

		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS code_files SCHEMAFULL;`, OnTable: "code_files"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE project_id ON code_files TYPE string;`, OnTable: "code_files"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE file_path ON code_files TYPE string;`, OnTable: "code_files"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE language ON code_files TYPE string;`, OnTable: "code_files"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE file_hash ON code_files TYPE string;`, OnTable: "code_files"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE symbols_count ON code_files TYPE int DEFAULT 0;`, OnTable: "code_files"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE indexed_at ON code_files TYPE datetime DEFAULT time::now();`, OnTable: "code_files"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_file_project_path ON code_files FIELDS project_id, file_path UNIQUE;`, OnTable: "code_files"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_file_project ON code_files FIELDS project_id;`, OnTable: "code_files"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_file_language ON code_files FIELDS language;`, OnTable: "code_files"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_file_hash ON code_files FIELDS file_hash;`, OnTable: "code_files"},

		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS code_symbols SCHEMAFULL;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE project_id ON code_symbols TYPE string;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE file_path ON code_symbols TYPE string;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE language ON code_symbols TYPE string;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE symbol_type ON code_symbols TYPE string;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE name ON code_symbols TYPE string;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE name_path ON code_symbols TYPE string;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE start_line ON code_symbols TYPE int;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE end_line ON code_symbols TYPE int;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE start_byte ON code_symbols TYPE int;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE end_byte ON code_symbols TYPE int;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE source_code ON code_symbols TYPE option<string>;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE signature ON code_symbols TYPE option<string>;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE doc_string ON code_symbols TYPE option<string>;`, OnTable: "code_symbols"},
		{Type: "field", Statement: vectorField("embedding", "code_symbols", dim, true), OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE parent_id ON code_symbols TYPE option<string>;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE metadata ON code_symbols TYPE option<object> FLEXIBLE;`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE created_at ON code_symbols TYPE datetime DEFAULT time::now();`, OnTable: "code_symbols"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE updated_at ON code_symbols TYPE datetime DEFAULT time::now();`, OnTable: "code_symbols"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_symbol_project ON code_symbols FIELDS project_id;`, OnTable: "code_symbols"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_symbol_file ON code_symbols FIELDS project_id, file_path;`, OnTable: "code_symbols"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_symbol_name_path ON code_symbols FIELDS project_id, name_path UNIQUE;`, OnTable: "code_symbols"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_symbol_type ON code_symbols FIELDS symbol_type;`, OnTable: "code_symbols"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_symbol_language ON code_symbols FIELDS language;`, OnTable: "code_symbols"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_symbol_name ON code_symbols FIELDS name;`, OnTable: "code_symbols"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_symbol_parent ON code_symbols FIELDS parent_id;`, OnTable: "code_symbols"},
		{Type: "index", Statement: hnswIndex("idx_code_symbol_embedding", "code_symbols", "embedding", dim, false), OnTable: "code_symbols"},

		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS code_indexing_jobs SCHEMAFULL;`, OnTable: "code_indexing_jobs"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE project_id ON code_indexing_jobs TYPE string;`, OnTable: "code_indexing_jobs"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE project_path ON code_indexing_jobs TYPE string;`, OnTable: "code_indexing_jobs"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE status ON code_indexing_jobs TYPE string DEFAULT 'pending';`, OnTable: "code_indexing_jobs"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE progress ON code_indexing_jobs TYPE float DEFAULT 0.0;`, OnTable: "code_indexing_jobs"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE files_total ON code_indexing_jobs TYPE int DEFAULT 0;`, OnTable: "code_indexing_jobs"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE files_indexed ON code_indexing_jobs TYPE int DEFAULT 0;`, OnTable: "code_indexing_jobs"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE started_at ON code_indexing_jobs TYPE datetime DEFAULT time::now();`, OnTable: "code_indexing_jobs"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE completed_at ON code_indexing_jobs TYPE option<datetime>;`, OnTable: "code_indexing_jobs"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE error ON code_indexing_jobs TYPE option<string>;`, OnTable: "code_indexing_jobs"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_job_project ON code_indexing_jobs FIELDS project_id;`, OnTable: "code_indexing_jobs"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_job_status ON code_indexing_jobs FIELDS status;`, OnTable: "code_indexing_jobs"},
	}
}

func v10Statements(dim int) []SchemaElement {
	return []SchemaElement{
		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS code_chunks SCHEMAFULL;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE symbol_id ON code_chunks TYPE string;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE project_id ON code_chunks TYPE string;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE file_path ON code_chunks TYPE string;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE chunk_index ON code_chunks TYPE int;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE chunk_count ON code_chunks TYPE int;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE content ON code_chunks TYPE string;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE start_offset ON code_chunks TYPE int;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE end_offset ON code_chunks TYPE int;`, OnTable: "code_chunks"},
		{Type: "field", Statement: vectorField("embedding", "code_chunks", dim, true), OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE symbol_name ON code_chunks TYPE string;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE symbol_type ON code_chunks TYPE string;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE language ON code_chunks TYPE string;`, OnTable: "code_chunks"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE created_at ON code_chunks TYPE datetime DEFAULT time::now();`, OnTable: "code_chunks"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_chunk_symbol ON code_chunks FIELDS symbol_id;`, OnTable: "code_chunks"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_chunk_project ON code_chunks FIELDS project_id;`, OnTable: "code_chunks"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_chunk_file ON code_chunks FIELDS project_id, file_path;`, OnTable: "code_chunks"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_code_chunk_unique ON code_chunks FIELDS symbol_id, chunk_index UNIQUE;`, OnTable: "code_chunks"},
		{Type: "index", Statement: hnswIndex("idx_code_chunk_embedding", "code_chunks", "embedding", dim, false), OnTable: "code_chunks"},
	}
}

func v11Statements(dim int) []SchemaElement {
	return []SchemaElement{
		{Type: "table", Statement: `DEFINE TABLE IF NOT EXISTS events SCHEMAFULL;`, OnTable: "events"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE user_id ON events TYPE option<string>;`, OnTable: "events"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE subject ON events TYPE string;`, OnTable: "events"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE content ON events TYPE string;`, OnTable: "events"},
		{Type: "field", Statement: vectorField("embedding", "events", dim, false), OnTable: "events"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE metadata ON events TYPE option<object> FLEXIBLE;`, OnTable: "events"},
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE created_at ON events TYPE datetime DEFAULT time::now();`, OnTable: "events"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_events_user ON events FIELDS user_id;`, OnTable: "events"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_events_subject ON events FIELDS subject;`, OnTable: "events"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_events_created ON events FIELDS created_at;`, OnTable: "events"},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_events_user_subject ON events FIELDS user_id, subject;`, OnTable: "events"},
		{Type: "index", Statement: hnswIndex("idx_events_embedding", "events", "embedding", dim, false), OnTable: "events"},
		// Full text. 3.x spells the clause FULLTEXT ANALYZER, not SEARCH ANALYZER.
		{Type: "analyzer", Statement: `DEFINE ANALYZER OVERWRITE events_analyzer TOKENIZERS blank, class FILTERS lowercase, snowball(english);`, OnTable: ""},
		{Type: "index", Statement: `DEFINE INDEX IF NOT EXISTS idx_events_content ON events FIELDS content FULLTEXT ANALYZER events_analyzer BM25 HIGHLIGHTS;`, OnTable: "events"},
	}
}

func v12Statements() []SchemaElement {
	return []SchemaElement{
		{Type: "field", Statement: `DEFINE FIELD OVERWRITE watcher_enabled ON code_projects TYPE bool DEFAULT false;`, OnTable: "code_projects"},
	}
}
