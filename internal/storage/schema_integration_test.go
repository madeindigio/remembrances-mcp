//go:build integration

package storage

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/madeindigio/remembrances-mcp/internal/storage/migrations"
)

// These tests execute real SurrealQL. They need the prebuilt embedded library at
// runtime, so they are behind the `integration` build tag and the default
// `go test ./...` stays hermetic:
//
//	go test -tags integration ./internal/storage/...
//
// Setting SURREALDB_URL points the same suite at a remote surrealdb/surrealdb:v3.2
// instead of the embedded engine, which is how the CI matrix runs both paths
// against identical statements.

func newIntegrationStorage(t *testing.T) *SurrealDBStorage {
	t.Helper()

	cfg := &ConnectionConfig{
		Namespace:       "itest",
		Database:        fmt.Sprintf("d%d", time.Now().UnixNano()),
		UseEmbeddedLibs: true,
		Timeout:         60 * time.Second,
	}
	if url := envOrEmpty("SURREALDB_URL"); url != "" {
		cfg.URL = url
		cfg.Username = envOr("SURREALDB_USER", "root")
		cfg.Password = envOr("SURREALDB_PASS", "root")
	} else {
		cfg.DBPath = "memory"
	}

	st := NewSurrealDBStorage(cfg)
	if err := st.Connect(context.Background()); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// TestSchemaMigratesToV13OnAnEmptyDatastore is the acceptance test for the whole
// chain: v1 through v13 must apply against a fresh 3.x datastore, and the
// version must land on 13.
func TestSchemaMigratesToV13OnAnEmptyDatastore(t *testing.T) {
	ctx := context.Background()
	st := newIntegrationStorage(t)

	if err := st.InitializeSchema(ctx); err != nil {
		t.Fatalf("schema initialisation failed: %v", err)
	}

	version, err := st.getCurrentSchemaVersion(ctx)
	if err != nil {
		t.Fatalf("failed to read the schema version: %v", err)
	}
	if version != migrations.TargetVersion {
		t.Fatalf("expected schema version %d, got %d", migrations.TargetVersion, version)
	}
}

// TestSchemaIsIdempotent asserts what OVERWRITE and IF NOT EXISTS buy: running
// the whole chain twice changes nothing and reports no error.
func TestSchemaIsIdempotent(t *testing.T) {
	ctx := context.Background()
	st := newIntegrationStorage(t)

	if err := st.InitializeSchema(ctx); err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	for version := 1; version <= migrations.TargetVersion; version++ {
		if err := st.applyMigrationWithData(ctx, version, migrations.V13Options{}); err != nil {
			t.Fatalf("replaying v%d failed: %v", version, err)
		}
	}
}

// TestEveryMigrationStatementParses executes every statement this tree emits
// against the real engine. It is the test that would have caught FLEXIBLE on a
// SCHEMALESS table, MTREE, and SEARCH ANALYZER before a boot did.
func TestEveryMigrationStatementParses(t *testing.T) {
	ctx := context.Background()
	st := newIntegrationStorage(t)

	for version := 1; version <= migrations.TargetVersion; version++ {
		for _, stmt := range migrations.StatementStrings(version, migrations.EmbeddingDimension()) {
			if _, err := st.query(ctx, stmt, nil); err != nil {
				t.Errorf("v%d rejected by the engine: %v\n  %s", version, err, stmt)
			}
		}
	}
}

// TestHNSWSearchWorksAfterMigration asserts the replacement for MTREE actually
// serves a KNN query at the configured dimension.
func TestHNSWSearchWorksAfterMigration(t *testing.T) {
	ctx := context.Background()
	st := newIntegrationStorage(t)
	if err := st.InitializeSchema(ctx); err != nil {
		t.Fatalf("schema initialisation failed: %v", err)
	}

	dim := EmbeddingDimension()
	for i := 0; i < 20; i++ {
		vec := make([]float64, dim)
		vec[i%dim] = 1.0
		if _, err := st.query(ctx, `
			CREATE vector_memories SET
				content = $content, embedding = $embedding, metadata = {},
				content_hash = crypto::sha256(string::trim($content)),
				org_id = org:legacy, scope = mem_scope:legacy_unassigned;`,
			map[string]interface{}{
				"content":   fmt.Sprintf("document %d", i),
				"embedding": vec,
			}); err != nil {
			t.Fatalf("failed to write vector %d: %v", i, err)
		}
	}

	probe := make([]float64, dim)
	probe[3] = 1.0
	rows, err := st.Query(ctx,
		"SELECT content FROM vector_memories WHERE embedding <|5,COSINE|> $vec;",
		map[string]interface{}{"vec": probe})
	if err != nil {
		t.Fatalf("HNSW search failed: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("HNSW search returned nothing")
	}
}

// TestFullTextSearchWorksAfterMigration asserts BM25 is retained in its 3.x
// spelling and scores a rare term.
func TestFullTextSearchWorksAfterMigration(t *testing.T) {
	ctx := context.Background()
	st := newIntegrationStorage(t)
	if err := st.InitializeSchema(ctx); err != nil {
		t.Fatalf("schema initialisation failed: %v", err)
	}

	dim := EmbeddingDimension()
	for _, content := range []string{
		"the quick brown fox jumps",
		"a completely unrelated sentence about zarquon",
	} {
		if _, err := st.query(ctx, `
			CREATE events SET subject = 'test', content = $content,
				embedding = $embedding, org_id = org:legacy, scope = mem_scope:legacy_unassigned;`,
			map[string]interface{}{"content": content, "embedding": make([]float64, dim)}); err != nil {
			t.Fatalf("failed to write event: %v", err)
		}
	}

	rows, err := st.Query(ctx,
		"SELECT content, search::score(1) AS score FROM events WHERE content @1@ 'zarquon';", nil)
	if err != nil {
		t.Fatalf("full-text search failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected exactly one match for a rare term, got %d", len(rows))
	}
	if !strings.Contains(fmt.Sprint(rows[0]["content"]), "zarquon") {
		t.Errorf("unexpected match: %v", rows[0])
	}
}

// TestV13BackfillAttributesLegacyRows seeds a pre-v13 dataset and asserts the
// mapping the RM-1 design specifies: a labelled row becomes personal to its
// derived principal, an unlabelled one lands in the org-wide legacy scope.
func TestV13BackfillAttributesLegacyRows(t *testing.T) {
	ctx := context.Background()
	st := newIntegrationStorage(t)

	// Bring the schema up to v12 only, then seed, then run v13.
	for version := 1; version <= 12; version++ {
		if err := st.applyMigration(ctx, version); err != nil {
			t.Fatalf("v%d failed: %v", version, err)
		}
	}

	for _, uid := range []string{"alice", "bob"} {
		if _, err := st.query(ctx,
			"CREATE kv_memories SET user_id = $uid, key = $key, value = 'x';",
			map[string]interface{}{"uid": uid, "key": "k-" + uid}); err != nil {
			t.Fatalf("failed to seed a row for %s: %v", uid, err)
		}
	}
	if _, err := st.query(ctx,
		"CREATE kv_memories SET key = 'orphan', value = 'x';", nil); err != nil {
		t.Fatalf("failed to seed the unlabelled row: %v", err)
	}

	if err := st.applyMigrationWithData(ctx, 13, migrations.V13Options{BatchSize: 100}); err != nil {
		t.Fatalf("v13 failed: %v", err)
	}

	rows, err := st.Query(ctx, "SELECT key, scope, created_by, visibility, version FROM kv_memories;", nil)
	if err != nil {
		t.Fatalf("failed to read back: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	for _, row := range rows {
		key := fmt.Sprint(row["key"])
		scope := fmt.Sprint(row["scope"])
		if scope == "" || scope == "<nil>" {
			t.Errorf("row %s has no scope after the backfill", key)
		}
		if toIntValue(row["version"]) != 1 {
			t.Errorf("row %s is not at version 1: %v", key, row["version"])
		}
		switch key {
		case "k-alice", "k-bob":
			want := "mem_scope:legacy_personal_" + strings.TrimPrefix(key, "k-")
			if !strings.Contains(scope, want) {
				t.Errorf("row %s landed in %s, expected %s", key, scope, want)
			}
			if fmt.Sprint(row["visibility"]) != "private" {
				t.Errorf("a labelled row must be private, got %v", row["visibility"])
			}
		case "orphan":
			if !strings.Contains(scope, "legacy_unassigned") {
				t.Errorf("the unlabelled row landed in %s, expected legacy_unassigned", scope)
			}
			if fmt.Sprint(row["visibility"]) != "org" {
				t.Errorf("an unlabelled row must be org-visible, got %v", row["visibility"])
			}
		}
	}
}

// TestV13ContentHashMatchesTheGoImplementation is the cross-check the RM-1 plan
// requires: the SurrealQL the migration runs and the Go ContentHash used on
// write must agree, or the unique dedup index stops deduplicating.
func TestV13ContentHashMatchesTheGoImplementation(t *testing.T) {
	ctx := context.Background()
	st := newIntegrationStorage(t)

	for _, content := range []string{"hello", "  padded  ", "señor", "multi\nline"} {
		rows, err := st.Query(ctx,
			"RETURN [{ h: crypto::sha256(string::trim($c)) }];", map[string]interface{}{"c": content})
		if err != nil {
			t.Fatalf("hash query failed for %q: %v", content, err)
		}
		if len(rows) == 0 {
			t.Fatalf("no result for %q", content)
		}
		got := fmt.Sprint(rows[0]["h"])
		if want := migrations.ContentHash(content); got != want {
			t.Errorf("hash drift for %q: SurrealQL gave %s, Go gave %s", content, got, want)
		}
	}
}

func toIntValue(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return -1
	}
}

func envOrEmpty(key string) string { return os.Getenv(key) }

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
