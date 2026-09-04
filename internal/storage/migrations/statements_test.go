package migrations

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// bareParamAssignment matches a parameter assignment that is not introduced by
// LET. 3.x made LET mandatory for parameter declarations.
var bareParamAssignment = regexp.MustCompile(`(?m)(^|[;\n]\s*)\$[A-Za-z_][A-Za-z0-9_]*\s*=`)

// TestNoLegacyDialect is the regression guard for the three clauses that changed
// between SurrealDB 2.x and 3.x. It runs over every migration, not just v13:
// a 2.x spelling anywhere is a parse error at boot on the 3.x engine.
func TestNoLegacyDialect(t *testing.T) {
	forbidden := []struct {
		needle string
		why    string
	}{
		{"MTREE", "MTREE is removed in 3.x; use HNSW"},
		{"SEARCH ANALYZER", "SEARCH ANALYZER is removed in 3.x; use FULLTEXT ANALYZER"},
		{"FLEXIBLE TYPE", "FLEXIBLE is a postfix modifier in 3.x: TYPE object FLEXIBLE"},
		{"DEFER", "DEFER is 2.x only and is rejected at parse time on 3.x; use CONCURRENTLY"},
	}

	for version := 1; version <= TargetVersion; version++ {
		for _, stmt := range StatementStrings(version, DefaultEmbeddingDimension) {
			upper := strings.ToUpper(stmt)
			for _, f := range forbidden {
				if strings.Contains(upper, f.needle) {
					t.Errorf("v%d statement uses a 2.x clause (%s): %s", version, f.why, stmt)
				}
			}
			if !strings.Contains(upper, "LET ") && bareParamAssignment.MatchString(stmt) {
				t.Errorf("v%d statement assigns a parameter without LET: %s", version, stmt)
			}
		}
	}
}

// TestFlexibleOnlyOnSchemafullTables guards the constraint the 3.2.4 engine
// enforces: "FLEXIBLE can only be used in SCHEMAFULL tables".
func TestFlexibleOnlyOnSchemafullTables(t *testing.T) {
	for version := 1; version <= TargetVersion; version++ {
		for _, e := range Statements(version, DefaultEmbeddingDimension) {
			if !strings.Contains(e.Statement, "FLEXIBLE") {
				continue
			}
			if schemalessTables[e.OnTable] {
				t.Errorf("v%d puts FLEXIBLE on the SCHEMALESS table %q: %s", version, e.OnTable, e.Statement)
			}
		}
	}
}

// TestDimensionIsRenderedEverywhere asserts the 768 hard-wiring is gone: at any
// configured dimension the literal must not appear, and every vector field and
// HNSW index must carry the configured value.
func TestDimensionIsRenderedEverywhere(t *testing.T) {
	for _, dim := range []int{384, 768, 1024} {
		dim := dim
		t.Run(fmt.Sprint(dim), func(t *testing.T) {
			var all []string
			for version := 1; version <= TargetVersion; version++ {
				all = append(all, StatementStrings(version, dim)...)
			}
			joined := strings.Join(all, "\n")

			if dim != 768 && strings.Contains(joined, "768") {
				t.Errorf("the literal 768 survives at dimension %d", dim)
			}

			wantField := fmt.Sprintf("array<float, %d>", dim)
			if !strings.Contains(joined, wantField) {
				t.Errorf("no statement declares %s", wantField)
			}

			wantIndex := fmt.Sprintf("HNSW DIMENSION %d", dim)
			for _, vi := range VectorIndexes {
				found := false
				for _, s := range all {
					if strings.Contains(s, vi.Index) && strings.Contains(s, wantIndex) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("index %s on %s is not defined as %s", vi.Index, vi.Table, wantIndex)
				}
			}
		})
	}
}

// TestEveryVectorIndexIsHNSW asserts that every index touching an embedding
// field is HNSW with the clause order the 3.x grammar requires
// (HNSW DIMENSION n TYPE t DIST d EFC e M m).
func TestEveryVectorIndexIsHNSW(t *testing.T) {
	want := regexp.MustCompile(`HNSW DIMENSION \d+ TYPE F32 DIST COSINE EFC \d+ M \d+`)
	for version := 1; version <= TargetVersion; version++ {
		for _, stmt := range StatementStrings(version, DefaultEmbeddingDimension) {
			if !strings.HasPrefix(stmt, "DEFINE INDEX") || !strings.Contains(stmt, "embedding") {
				continue
			}
			if !want.MatchString(stmt) {
				t.Errorf("v%d embedding index is not a well-formed HNSW definition: %s", version, stmt)
			}
		}
	}
}

// TestV13AddsEveryColumnAndIndexToEveryTable is the table-driven check that the
// tenant, scope, principal and validity contract has no holes: a table missing a
// column is a hole in the ACL wrapper's uniform preamble.
func TestV13AddsEveryColumnAndIndexToEveryTable(t *testing.T) {
	statements := StatementStrings(13, DefaultEmbeddingDimension)

	for _, table := range MemoryTables {
		for _, column := range ScopeColumns {
			needle := fmt.Sprintf("DEFINE FIELD OVERWRITE %s ON %s ", column, table)
			if !containsAny(statements, needle) {
				t.Errorf("v13 does not add %s to %s", column, table)
			}
		}

		prefix := tableIndexPrefix[table]
		if prefix == "" {
			t.Fatalf("no index prefix registered for %s", table)
		}
		for _, suffix := range []string{"org_scope", "org_creator", "org_version", "valid"} {
			needle := fmt.Sprintf("idx_%s_%s ON %s ", prefix, suffix, table)
			if !containsAny(statements, needle) {
				t.Errorf("v13 does not define idx_%s_%s on %s", prefix, suffix, table)
			}
		}
	}
}

// TestV13DefinesEveryPrincipalTable asserts the eight principal tables exist.
func TestV13DefinesEveryPrincipalTable(t *testing.T) {
	statements := StatementStrings(13, DefaultEmbeddingDimension)
	for _, table := range PrincipalTables {
		name := table
		if name == "grant" {
			name = "`grant`"
		}
		if !containsAny(statements, "DEFINE TABLE IF NOT EXISTS "+name+" ") {
			t.Errorf("v13 does not define the principal table %s", table)
		}
	}
}

// TestV13MovesTheKnowledgeBaseUniqueIndex asserts both legacy index names are
// dropped and the scoped one is created. A datastore carries one name or the
// other depending on which side of the pre-RM-7 schema fork it was created on.
func TestV13MovesTheKnowledgeBaseUniqueIndex(t *testing.T) {
	statements := StatementStrings(13, DefaultEmbeddingDimension)
	for _, needle := range []string{
		"REMOVE INDEX IF EXISTS idx_kb_path ON knowledge_base;",
		"REMOVE INDEX IF EXISTS idx_kb_file_path ON knowledge_base;",
		"idx_kb_scope_path ON knowledge_base FIELDS org_id, scope, file_path UNIQUE;",
	} {
		if !containsAny(statements, needle) {
			t.Errorf("v13 is missing: %s", needle)
		}
	}
}

// TestV13UsesFullTextNotSearch asserts the BM25 index is retained in its 3.x
// spelling, with highlights for Control's memory browser.
func TestV13UsesFullTextNotSearch(t *testing.T) {
	statements := StatementStrings(13, DefaultEmbeddingDimension)
	if !containsAny(statements, "FULLTEXT ANALYZER events_analyzer BM25 HIGHLIGHTS") {
		t.Error("v13 does not define the BM25 full-text index on events")
	}
	if !containsAny(statements, "DEFINE ANALYZER OVERWRITE events_analyzer") {
		t.Error("v13 does not redefine events_analyzer with OVERWRITE")
	}
}

// TestEverySchemaVersionIsReachable asserts no version in the range renders an
// empty statement list, which would mean a silently skipped migration.
func TestEverySchemaVersionIsReachable(t *testing.T) {
	for version := 1; version <= TargetVersion; version++ {
		if len(StatementStrings(version, DefaultEmbeddingDimension)) == 0 {
			t.Errorf("v%d renders no statements", version)
		}
	}
	if StatementStrings(TargetVersion+1, DefaultEmbeddingDimension) != nil {
		t.Errorf("v%d should not exist", TargetVersion+1)
	}
}

// TestStatementStringsMatchesStatements asserts the embedded path (which reads
// StatementStrings) executes exactly what the remote path (which reads
// Statements) executes. This is the test that keeps the schema fork closed.
func TestStatementStringsMatchesStatements(t *testing.T) {
	for version := 1; version <= TargetVersion; version++ {
		elements := Statements(version, DefaultEmbeddingDimension)
		strs := StatementStrings(version, DefaultEmbeddingDimension)
		if len(elements) != len(strs) {
			t.Fatalf("v%d: %d elements but %d strings", version, len(elements), len(strs))
		}
		for i := range elements {
			if elements[i].Statement != strs[i] {
				t.Errorf("v%d statement %d differs between the two accessors", version, i)
			}
		}
	}
}

// TestEmbeddingDimensionIsTheSingleAuthority asserts the setter round-trips and
// that a nonsensical value cannot destroy the schema.
func TestEmbeddingDimensionIsTheSingleAuthority(t *testing.T) {
	original := EmbeddingDimension()
	defer SetEmbeddingDimension(original)

	SetEmbeddingDimension(1024)
	if got := EmbeddingDimension(); got != 1024 {
		t.Fatalf("expected 1024, got %d", got)
	}
	SetEmbeddingDimension(0)
	if got := EmbeddingDimension(); got != 1024 {
		t.Fatalf("a zero dimension must be ignored, got %d", got)
	}
	SetEmbeddingDimension(-5)
	if got := EmbeddingDimension(); got != 1024 {
		t.Fatalf("a negative dimension must be ignored, got %d", got)
	}
}

func containsAny(statements []string, needle string) bool {
	for _, s := range statements {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}
