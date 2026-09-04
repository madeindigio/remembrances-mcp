package migrations

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// fakeExecutor records every statement and answers the queries the backfill
// asks, so the whole data phase can be exercised without a database.
type fakeExecutor struct {
	statements []string
	// userIDs is what the legacy user_id discovery query returns.
	userIDs []string
	// pending is the per-table count of rows still lacking an org_id. It drops
	// to zero after the first batch of each table.
	pending map[string]int
	// duplicates is what the dedup grouping query returns per table.
	duplicates map[string][]map[string]interface{}
	// failOn makes a statement containing this substring return an error.
	failOn string
}

func newFakeExecutor() *fakeExecutor {
	return &fakeExecutor{
		pending:    map[string]int{},
		duplicates: map[string][]map[string]interface{}{},
	}
}

func (f *fakeExecutor) Query(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	f.statements = append(f.statements, query)
	if f.failOn != "" && strings.Contains(query, f.failOn) {
		return nil, errors.New("injected failure")
	}

	switch {
	case strings.Contains(query, "SELECT user_id FROM"):
		var out []map[string]interface{}
		for _, id := range f.userIDs {
			out = append(out, map[string]interface{}{"user_id": id})
		}
		return out, nil

	case strings.Contains(query, "count() AS n FROM") && strings.Contains(query, "GROUP ALL"):
		table := tableFromQuery(query, "FROM ")
		return []map[string]interface{}{{"n": float64(f.pending[table])}}, nil

	case strings.Contains(query, "GROUP BY scope, content_hash"):
		return f.duplicates[tableFromQuery(query, "FROM ")], nil

	case strings.Contains(query, "LET $batch"):
		// One non-empty batch per table pass, then empty, so the loop ends.
		table := tableFromQuery(query, "FROM ")
		if f.pending[table] > 0 {
			f.pending[table] = 0
			return []map[string]interface{}{{"id": table + ":one"}}, nil
		}
		return nil, nil
	}
	return nil, nil
}

func tableFromQuery(query, after string) string {
	i := strings.Index(query, after)
	if i < 0 {
		return ""
	}
	rest := strings.TrimSpace(query[i+len(after):])
	if j := strings.IndexAny(rest, " \n\t;"); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

func TestSlugIsDeterministicAndSafe(t *testing.T) {
	cases := map[string]string{
		"alice":        "alice",
		"Alice":        "alice",
		"alice@x.com":  "alice_x_com",
		"jose garcia":  "jose_garcia",
		"  padded  ":   "padded",
		"":             "unknown",
		"señor":        "se_or",
		"user-123":     "user_123",
		"MiXeD_Case_9": "mixed_case_9",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
		if got := Slug(in); got != Slug(in) {
			t.Errorf("Slug(%q) is not deterministic", in)
		}
	}
}

// TestPreflightSlugCollisions is the guard against merging two people's memory.
// Two user_id values that slug to the same fragment must be named, not resolved.
func TestPreflightSlugCollisions(t *testing.T) {
	cases := []struct {
		name    string
		userIDs []string
		want    int
	}{
		{"no collision", []string{"alice", "bob", "carol"}, 0},
		{"case collision", []string{"Alice", "alice"}, 1},
		{"punctuation collision", []string{"a.b", "a-b", "a b"}, 1},
		{"two independent collisions", []string{"A", "a", "B", "b"}, 2},
		{"duplicates are not collisions", []string{"alice", "alice"}, 0},
		{"empty set", nil, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := PreflightSlugCollisions(tc.userIDs)
			if len(got) != tc.want {
				t.Fatalf("got %d collisions, want %d: %+v", len(got), tc.want, got)
			}
			for _, c := range got {
				if len(c.UserIDs) < 2 {
					t.Errorf("a reported collision must name at least two ids: %+v", c)
				}
			}
		})
	}
}

// TestContentHashMatchesTheSurrealExpression asserts the Go hash and the
// SurrealQL the migration runs describe the same normalisation. A drift here
// means the unique dedup index silently stops deduplicating.
func TestContentHashMatchesTheSurrealExpression(t *testing.T) {
	expr := ContentHashSurrealExpr("content")
	if expr != "crypto::sha256(string::trim(content))" {
		t.Fatalf("unexpected expression %q: the Go side trims then sha256s, so the SurrealQL must too", expr)
	}

	// The Go side must be trim-then-hash, and must be stable across the inputs
	// that historically differ between implementations.
	cases := []struct{ a, b string }{
		{"hello", "  hello  "},
		{"hello", "\thello\n"},
		{"señor", " señor "},
		{"", "   "},
	}
	for _, c := range cases {
		if ContentHash(c.a) != ContentHash(c.b) {
			t.Errorf("ContentHash(%q) != ContentHash(%q): trimming is not applied consistently", c.a, c.b)
		}
	}
	if ContentHash("a") == ContentHash("b") {
		t.Error("distinct content must hash differently")
	}
	if len(ContentHash("x")) != 64 {
		t.Errorf("expected a 64 character hex digest, got %d characters", len(ContentHash("x")))
	}
}

// TestApplyV13DataDryRunWritesNothing asserts the dry run reports the mapping
// and the row counts without issuing a single write.
func TestApplyV13DataDryRunWritesNothing(t *testing.T) {
	f := newFakeExecutor()
	f.userIDs = []string{"alice", "bob"}
	for _, table := range MemoryTables {
		f.pending[table] = 7
	}

	report, err := ApplyV13Data(context.Background(), f, V13Options{DryRun: true})
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}
	if len(report.LegacyUsers) != 2 {
		t.Errorf("expected 2 legacy users, got %d", len(report.LegacyUsers))
	}
	if report.LegacyUsers["alice"] != "alice" {
		t.Errorf("unexpected slug mapping: %+v", report.LegacyUsers)
	}
	if report.RowsPerTable["kv_memories"] != 7 {
		t.Errorf("expected 7 pending rows on kv_memories, got %d", report.RowsPerTable["kv_memories"])
	}
	for _, stmt := range f.statements {
		upper := strings.ToUpper(stmt)
		if strings.Contains(upper, "UPSERT") || strings.Contains(upper, "UPDATE ") ||
			strings.Contains(upper, "DEFINE ") || strings.Contains(upper, "DELETE ") {
			t.Errorf("dry run issued a write: %s", stmt)
		}
	}
}

// TestApplyV13DataRefusesOnSlugCollision asserts the migration aborts before
// writing anything and names the colliding pairs.
func TestApplyV13DataRefusesOnSlugCollision(t *testing.T) {
	f := newFakeExecutor()
	f.userIDs = []string{"Alice", "alice"}

	_, err := ApplyV13Data(context.Background(), f, V13Options{})
	if err == nil {
		t.Fatal("expected the migration to refuse a slug collision")
	}
	if !strings.Contains(err.Error(), "Alice") || !strings.Contains(err.Error(), "alice") {
		t.Errorf("the error must name the colliding pair, got: %v", err)
	}
	for _, stmt := range f.statements {
		if strings.Contains(strings.ToUpper(stmt), "UPSERT") {
			t.Errorf("a write was issued before the preflight aborted: %s", stmt)
		}
	}
}

// TestApplyV13DataBackfillsEveryTable walks the full data phase and asserts the
// shape of what it emits: the legacy principals, one pass per table, the
// tightening, and the unique hash index on the tables that get one.
func TestApplyV13DataBackfillsEveryTable(t *testing.T) {
	f := newFakeExecutor()
	f.userIDs = []string{"alice"}
	for _, table := range MemoryTables {
		f.pending[table] = 3
	}

	if _, err := ApplyV13Data(context.Background(), f, V13Options{BatchSize: 100}); err != nil {
		t.Fatalf("backfill failed: %v", err)
	}
	joined := strings.Join(f.statements, "\n")

	for _, needle := range []string{
		"UPSERT org:legacy SET name = 'legacy'",
		"UPSERT mem_scope:legacy_unassigned SET org = org:legacy",
		"UPSERT app_user:legacy_alice SET org = org:legacy",
		"UPSERT mem_scope:legacy_personal_alice SET org = org:legacy",
	} {
		if !strings.Contains(joined, needle) {
			t.Errorf("the backfill never issued: %s", needle)
		}
	}

	for _, table := range MemoryTables {
		if !strings.Contains(joined, fmt.Sprintf("SELECT VALUE id FROM %s WHERE org_id = NONE", table)) {
			t.Errorf("no unassigned pass for %s", table)
		}
		if !strings.Contains(joined, fmt.Sprintf("DEFINE FIELD OVERWRITE org_id ON %s TYPE record<org>;", table)) {
			t.Errorf("org_id was never tightened on %s", table)
		}
		if !strings.Contains(joined, fmt.Sprintf("DEFINE FIELD OVERWRITE scope ON %s TYPE record<mem_scope>;", table)) {
			t.Errorf("scope was never tightened on %s", table)
		}
	}

	for _, table := range HashUniqueTables {
		if !strings.Contains(joined, fmt.Sprintf("idx_%s_hash ON %s FIELDS org_id, scope, content_hash UNIQUE",
			tableIndexPrefix[table], table)) {
			t.Errorf("no unique content hash index on %s", table)
		}
	}

	// The user pass must exist for the tables that carry a legacy label, and
	// must attribute the rows to the derived principal.
	for _, table := range userIDTables {
		if !strings.Contains(joined, fmt.Sprintf("SELECT VALUE id FROM %s WHERE org_id = NONE AND user_id = $uid", table)) {
			t.Errorf("no per-user pass for %s", table)
		}
	}
	if !strings.Contains(joined, "created_by = app_user:legacy_alice") {
		t.Error("the per-user pass does not set created_by")
	}
	if !strings.Contains(joined, "visibility = 'org'") {
		t.Error("the unassigned pass does not set org visibility")
	}
	if !strings.Contains(joined, ContentHashSurrealExpr("<string> value")) {
		t.Error("kv_memories rows are not given a content hash")
	}
}

// TestApplyV13DataIsIdempotentInShape asserts a second run over a converged
// datastore issues no UPDATE passes, which is what the WHERE org_id = NONE
// guard buys.
func TestApplyV13DataIsIdempotentInShape(t *testing.T) {
	f := newFakeExecutor()
	f.userIDs = nil // everything already attributed, nothing left to map

	if _, err := ApplyV13Data(context.Background(), f, V13Options{}); err != nil {
		t.Fatalf("second run failed: %v", err)
	}
	for _, stmt := range f.statements {
		if strings.Contains(stmt, "UPDATE $batch") {
			t.Errorf("a converged datastore must not be updated again: %s", stmt)
		}
	}
}

// TestApplyV13DataSurfacesFailures asserts a failing statement aborts the run
// rather than being swallowed, so the version never bumps over a partial state.
func TestApplyV13DataSurfacesFailures(t *testing.T) {
	f := newFakeExecutor()
	f.failOn = "UPSERT org:legacy"

	if _, err := ApplyV13Data(context.Background(), f, V13Options{}); err == nil {
		t.Fatal("expected the run to fail when the legacy org cannot be created")
	}
}

// TestMissingTableIsNotAFailure covers the 3.x behaviour change where selecting
// from a table that was never defined is an error rather than an empty result.
func TestMissingTableIsNotAFailure(t *testing.T) {
	if !isMissingTableError(errors.New("The table 'kv_memories' does not exist")) {
		t.Error("the 3.x missing-table error must be recognised")
	}
	if isMissingTableError(errors.New("some other failure")) {
		t.Error("an unrelated error must not be treated as a missing table")
	}
}

func TestValidFromExprUsesTheRightTimestampColumn(t *testing.T) {
	if got := validFromExpr("wrote"); !strings.HasPrefix(got, "timestamp") {
		t.Errorf("edge tables record their creation time as timestamp, got %q", got)
	}
	if got := validFromExpr("kv_memories"); !strings.HasPrefix(got, "created_at") {
		t.Errorf("memory tables record their creation time as created_at, got %q", got)
	}
}
