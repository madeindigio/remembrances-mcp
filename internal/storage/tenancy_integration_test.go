//go:build integration

package storage

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/madeindigio/remembrances-mcp/pkg/principal"
	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// The RM-2, RM-3 and RM-9 acceptance tests, against the real engine.
//
//	go test -tags integration ./internal/storage/...
//
// Setting SURREALDB_URL runs the same suite against a remote SurrealDB 3.x
// instead of the embedded engine, which is how both paths are proved to apply
// identical predicates.

// tenants is the fixture every isolation test shares: two orgs, three scopes,
// and the principals that see them.
type tenants struct {
	st *SurrealDBStorage

	alice *principal.Principal // org a, writes into scope a1
	amy   *principal.Principal // org a, writes into scope a2, cannot read a1
	nosy  *principal.Principal // org a, writes into a2 and may read a1
	bob   *principal.Principal // org b, another tenant entirely
}

func newTenants(t *testing.T) *tenants {
	t.Helper()

	ctx := context.Background()
	st := newIntegrationStorage(t)
	if err := st.InitializeSchema(ctx); err != nil {
		t.Fatalf("schema initialisation failed: %v", err)
	}

	setup := []string{
		`UPSERT org:a SET name = 'a', status = 'active';`,
		`UPSERT org:b SET name = 'b', status = 'active';`,
		`UPSERT app_user:alice SET org = org:a, subject = 'alice', active = true;`,
		`UPSERT app_user:amy SET org = org:a, subject = 'amy', active = true;`,
		`UPSERT app_user:bob SET org = org:b, subject = 'bob', active = true;`,
		`UPSERT mem_scope:a1 SET org = org:a, kind = 'personal', owner_user = app_user:alice, label = 'a/alice';`,
		`UPSERT mem_scope:a2 SET org = org:a, kind = 'personal', owner_user = app_user:amy, label = 'a/amy';`,
		`UPSERT mem_scope:b1 SET org = org:b, kind = 'personal', owner_user = app_user:bob, label = 'b/bob';`,
	}
	for _, stmt := range setup {
		if _, err := st.query(ctx, stmt, nil); err != nil {
			t.Fatalf("fixture failed: %v\n  %s", err, stmt)
		}
	}

	return &tenants{
		st:    st,
		alice: &principal.Principal{Org: "org:a", User: "app_user:alice", WriteScope: "mem_scope:a1"},
		amy:   &principal.Principal{Org: "org:a", User: "app_user:amy", WriteScope: "mem_scope:a2"},
		nosy: &principal.Principal{Org: "org:a", User: "app_user:amy", WriteScope: "mem_scope:a2",
			Scopes: []string{"mem_scope:a1"}},
		bob: &principal.Principal{Org: "org:b", User: "app_user:bob", WriteScope: "mem_scope:b1"},
	}
}

// as returns a context carrying the given principal.
func as(p *principal.Principal) context.Context {
	return principal.NewContext(context.Background(), p)
}

// vec builds a deterministic unit embedding at the configured width.
func vec(seed int) []float32 {
	out := make([]float32, EmbeddingDimension())
	out[seed%len(out)] = 1
	return out
}

// TestNoPrincipalIsDeniedAgainstARealDatastore is the fail-closed acceptance
// test: with require_principal on, a call with no identity must return an
// error and must not return rows, even though the rows are right there.
func TestNoPrincipalIsDeniedAgainstARealDatastore(t *testing.T) {
	tn := newTenants(t)

	if err := tn.st.SaveFact(as(tn.alice), "", "colour", "green"); err != nil {
		t.Fatalf("SaveFact failed: %v", err)
	}

	withRequirePrincipal(t, true)
	ctx := context.Background()

	if _, err := tn.st.ListFacts(ctx, "alice"); !errors.Is(err, ErrNoPrincipal) {
		t.Errorf("ListFacts without a principal returned %v", err)
	}
	if _, err := tn.st.SearchDocuments(ctx, vec(1), 10); !errors.Is(err, ErrNoPrincipal) {
		t.Errorf("SearchDocuments without a principal returned %v", err)
	}
	if _, err := tn.st.ListEntityIDs(ctx); !errors.Is(err, ErrNoPrincipal) {
		t.Errorf("ListEntityIDs without a principal returned %v", err)
	}
	if _, _, err := tn.st.ListChanges(ctx, "", "", 10); !errors.Is(err, ErrNoPrincipal) {
		t.Errorf("ListChanges without a principal returned %v", err)
	}
}

// TestOneOrgNeverSeesAnother walks every memory kind: a row written by org a
// must be invisible to org b through every read path there is.
func TestOneOrgNeverSeesAnother(t *testing.T) {
	tn := newTenants(t)
	alice, bob := as(tn.alice), as(tn.bob)

	if err := tn.st.SaveFact(alice, "", "colour", "green"); err != nil {
		t.Fatalf("SaveFact: %v", err)
	}
	if err := tn.st.IndexVector(alice, "", "a private note", vec(2), nil); err != nil {
		t.Fatalf("IndexVector: %v", err)
	}
	if err := tn.st.SaveDocument(alice, "secret.md", "the plan", vec(3), nil); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	if _, _, err := tn.st.SaveEvent(alice, "", "standup", "we shipped it", vec(4), nil); err != nil {
		t.Fatalf("SaveEvent: %v", err)
	}
	if err := tn.st.CreateEntity(alice, "person", "Alice", nil); err != nil {
		t.Fatalf("CreateEntity: %v", err)
	}

	// Everything is visible to its own org.
	if facts, err := tn.st.ListFacts(alice, ""); err != nil || len(facts) != 1 {
		t.Fatalf("the owner cannot see its own fact: %v %v", facts, err)
	}

	t.Run("facts", func(t *testing.T) {
		facts, err := tn.st.ListFacts(bob, "")
		if err != nil {
			t.Fatalf("ListFacts: %v", err)
		}
		if len(facts) != 0 {
			t.Errorf("org b sees org a's facts: %v", facts)
		}
		if v, err := tn.st.GetFact(bob, "", "colour"); err != nil || v != nil {
			t.Errorf("GetFact leaked %v (%v)", v, err)
		}
		if keys, err := tn.st.ListFactKeys(bob, ""); err != nil || len(keys) != 0 {
			t.Errorf("ListFactKeys leaked %v (%v)", keys, err)
		}
	})

	t.Run("vectors", func(t *testing.T) {
		got, err := tn.st.SearchSimilar(bob, "", vec(2), 10)
		if err != nil {
			t.Fatalf("SearchSimilar: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("org b sees org a's vectors: %+v", got)
		}
	})

	t.Run("documents", func(t *testing.T) {
		got, err := tn.st.SearchDocuments(bob, vec(3), 10)
		if err != nil {
			t.Fatalf("SearchDocuments: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("org b sees org a's documents: %+v", got)
		}
		if doc, err := tn.st.GetDocument(bob, "secret.md"); err != nil || doc != nil {
			t.Errorf("GetDocument leaked %+v (%v)", doc, err)
		}
		if paths, err := tn.st.ListDocumentPaths(bob); err != nil || len(paths) != 0 {
			t.Errorf("ListDocumentPaths leaked %v (%v)", paths, err)
		}
	})

	t.Run("events", func(t *testing.T) {
		got, err := tn.st.SearchEvents(bob, EventSearchParams{Subject: "standup", Limit: 10})
		if err != nil {
			t.Fatalf("SearchEvents: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("org b sees org a's events: %+v", got)
		}
		if events, err := tn.st.GetEventsBySubject(bob, "", "standup", 10); err != nil || len(events) != 0 {
			t.Errorf("GetEventsBySubject leaked %+v (%v)", events, err)
		}
	})

	t.Run("entities", func(t *testing.T) {
		if e, err := tn.st.GetEntity(bob, "Alice"); err != nil || e != nil {
			t.Errorf("GetEntity leaked %+v (%v)", e, err)
		}
		if ids, err := tn.st.ListEntityIDs(bob); err != nil || len(ids) != 0 {
			t.Errorf("ListEntityIDs leaked %v (%v)", ids, err)
		}
	})

	t.Run("stats", func(t *testing.T) {
		stats, err := tn.st.GetStats(bob, "")
		if err != nil {
			t.Fatalf("GetStats: %v", err)
		}
		if stats.KeyValueCount+stats.VectorCount+stats.DocumentCount+stats.EventCount+stats.EntityCount != 0 {
			t.Errorf("GetStats leaked counts across the tenant boundary: %+v", stats)
		}
		counts, err := tn.st.CountByUserID(bob, "kv_memories")
		if err != nil {
			t.Fatalf("CountByUserID: %v", err)
		}
		if len(counts) != 0 {
			t.Errorf("CountByUserID leaked %v", counts)
		}
	})

	t.Run("hybrid", func(t *testing.T) {
		got, err := tn.st.HybridSearch(bob, "", vec(2), nil, 10)
		if err != nil {
			t.Fatalf("HybridSearch: %v", err)
		}
		if got.TotalResults != 0 {
			t.Errorf("HybridSearch leaked %+v", got)
		}
	})

	t.Run("change feed", func(t *testing.T) {
		changes, _, err := tn.st.ListChanges(bob, "", "", 100)
		if err != nil {
			t.Fatalf("ListChanges: %v", err)
		}
		if len(changes) != 0 {
			t.Errorf("the change feed leaked %d rows across the tenant boundary", len(changes))
		}
	})
}

// TestScopesAreIsolatedInsideOneOrg: same org, different scope, still no read.
// Granting the scope is what makes the row visible, and nothing else.
func TestScopesAreIsolatedInsideOneOrg(t *testing.T) {
	tn := newTenants(t)

	if err := tn.st.SaveFact(as(tn.alice), "", "colour", "green"); err != nil {
		t.Fatalf("SaveFact: %v", err)
	}

	facts, err := tn.st.ListFacts(as(tn.amy), "")
	if err != nil {
		t.Fatalf("ListFacts: %v", err)
	}
	if len(facts) != 0 {
		t.Fatalf("a sibling scope in the same org saw the fact: %v", facts)
	}

	facts, err = tn.st.ListFacts(as(tn.nosy), "")
	if err != nil {
		t.Fatalf("ListFacts: %v", err)
	}
	if len(facts) != 1 || facts["colour"] != "green" {
		t.Fatalf("a principal granted the scope could not read it: %v", facts)
	}
}

// TestTraversalStopsAtAScopeBoundary builds the exact structure RM-3 names: an
// edge that leaves the caller's scopes. Following it would be the leak.
func TestTraversalStopsAtAScopeBoundary(t *testing.T) {
	tn := newTenants(t)
	alice, nosy := as(tn.alice), as(tn.nosy)

	// Alice owns Anchor and Inside, both in scope a1.
	if err := tn.st.CreateEntity(alice, "doc", "Anchor", nil); err != nil {
		t.Fatalf("CreateEntity: %v", err)
	}
	if err := tn.st.CreateEntity(alice, "doc", "Inside", nil); err != nil {
		t.Fatalf("CreateEntity: %v", err)
	}
	// Amy owns Outside, in scope a2.
	if err := tn.st.CreateEntity(as(tn.amy), "doc", "Outside", nil); err != nil {
		t.Fatalf("CreateEntity: %v", err)
	}

	if err := tn.st.CreateRelationship(alice, "Anchor", "Inside", "related_to", nil); err != nil {
		t.Fatalf("CreateRelationship inside the scope: %v", err)
	}

	// The cross-scope edge cannot even be created from Alice's principal,
	// because resolving the far endpoint already fails. That is the first line
	// of defence; the traversal below is the second.
	if err := tn.st.CreateRelationship(alice, "Anchor", "Outside", "related_to", nil); err == nil {
		t.Fatal("an edge was created to an entity outside the caller's scopes")
	}

	// Plant the edge behind the API, as a mis-scoped row or a stale grant
	// would leave it, and assert the traversal still refuses to follow it.
	outside, err := tn.st.resolveEntityID(as(tn.amy), &tenancy{p: tn.amy}, "Outside")
	if err != nil {
		t.Fatalf("could not resolve Outside: %v", err)
	}
	anchor, err := tn.st.resolveEntityID(alice, &tenancy{p: tn.alice}, "Anchor")
	if err != nil {
		t.Fatalf("could not resolve Anchor: %v", err)
	}
	if _, err := tn.st.query(context.Background(), `
		CREATE related_to CONTENT {
			org_id: org:a, scope: mem_scope:a1,
			in: type::record($from), out: type::record($to),
			from_entity: type::record($from), to_entity: type::record($to),
			relationship_type: 'related_to', properties: {},
			version: 0, updated_at: time::now()
		}`, map[string]interface{}{"from": anchor, "to": outside}); err != nil {
		t.Fatalf("could not plant the cross-scope edge: %v", err)
	}

	results, err := tn.st.TraverseGraph(alice, "Anchor", "related_to", 3)
	if err != nil {
		t.Fatalf("TraverseGraph: %v", err)
	}
	for _, r := range results {
		if r.Entity.Name == "Outside" {
			t.Fatalf("the traversal crossed a scope boundary: %+v", r)
		}
	}
	if len(results) != 1 || results[0].Entity.Name != "Inside" {
		t.Fatalf("the traversal did not return the in-scope neighbour: %+v", results)
	}

	// A principal that can read both scopes does reach it, which proves the
	// walk stopped because of the scope and not because it is broken.
	results, err = tn.st.TraverseGraph(nosy, "Anchor", "related_to", 3)
	if err != nil {
		t.Fatalf("TraverseGraph as the granted principal: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("the granted principal saw %d neighbours, want 2: %+v", len(results), results)
	}
}

// TestChangeFeedIsMonotonicResumableAndCarriesTombstones is the RM-9
// acceptance test.
func TestChangeFeedIsMonotonicResumableAndCarriesTombstones(t *testing.T) {
	tn := newTenants(t)
	alice := as(tn.alice)

	for i := 0; i < 5; i++ {
		if err := tn.st.SaveFact(alice, "", fmt.Sprintf("k%d", i), fmt.Sprintf("v%d", i)); err != nil {
			t.Fatalf("SaveFact: %v", err)
		}
	}

	// Page through with a small limit; the cursor must never repeat or skip.
	seen := map[string]int{}
	var order []int64
	cursor := ""
	for page := 0; page < 20; page++ {
		changes, next, err := tn.st.ListChanges(alice, "", cursor, 2)
		if err != nil {
			t.Fatalf("ListChanges: %v", err)
		}
		if len(changes) == 0 {
			break
		}
		for _, c := range changes {
			seen[c.ID]++
			order = append(order, c.Version)
		}
		if next == cursor {
			t.Fatalf("the cursor did not advance past %q", cursor)
		}
		cursor = next
	}

	if len(order) < 5 {
		t.Fatalf("the feed returned %d rows for 5 facts", len(order))
	}
	for i := 1; i < len(order); i++ {
		if order[i] < order[i-1] {
			t.Fatalf("versions are not monotonic: %v", order)
		}
	}
	for id, count := range seen {
		if count != 1 {
			t.Errorf("%s appeared %d times across pages", id, count)
		}
	}

	// A delete must show up as a tombstone, with a fresh version and no
	// payload, and the row must survive.
	afterWrites := cursor
	if err := tn.st.DeleteFact(alice, "", "k0"); err != nil {
		t.Fatalf("DeleteFact: %v", err)
	}

	changes, _, err := tn.st.ListChanges(alice, "", afterWrites, 100)
	if err != nil {
		t.Fatalf("ListChanges after the delete: %v", err)
	}
	tombstones := 0
	for _, c := range changes {
		if c.Deleted {
			tombstones++
			if c.Payload != nil {
				t.Errorf("a tombstone carried a payload: %+v", c)
			}
		}
	}
	if tombstones == 0 {
		t.Fatalf("the delete produced no tombstone; got %+v", changes)
	}

	// The row is invalidated, not removed.
	rows, err := tn.st.query(context.Background(),
		"SELECT count() AS count FROM kv_memories WHERE key = 'k0' GROUP ALL", nil)
	if err != nil {
		t.Fatalf("counting the invalidated row: %v", err)
	}
	if n := (*rows)[0].Result[0]["count"]; convertToInt(n) == 0 {
		t.Fatal("the delete removed the row instead of invalidating it")
	}

	// A scope the principal cannot read is refused rather than ignored.
	if _, _, err := tn.st.ListChanges(alice, "mem_scope:b1", "", 10); err == nil {
		t.Fatal("ListChanges accepted a scope the principal cannot read")
	}
}

// TestChangeFeedCursorSurvivesARestart. The cursor is stateless, so a fresh
// process reading the same datastore must resume exactly where the previous
// one stopped. The datastore is on disk for this test precisely so the process
// can be replaced under it.
func TestChangeFeedCursorSurvivesARestart(t *testing.T) {
	if url := envOrEmpty("SURREALDB_URL"); url != "" {
		t.Skip("the restart test drives the embedded engine's on-disk datastore")
	}

	ctx := context.Background()
	// A plain path is treated as RocksDB by the embedded loader; the URL forms
	// need three slashes (rocksdb:///abs/path) and the plain path is clearer.
	path := filepath.Join(t.TempDir(), "feed")

	open := func() *SurrealDBStorage {
		cfg := &ConnectionConfig{
			Namespace:       "itest",
			Database:        "restart",
			DBPath:          path,
			UseEmbeddedLibs: true,
			Timeout:         60 * time.Second,
		}
		// The datastore lock the previous process held is released
		// asynchronously by the embedded engine, so the reopen is retried
		// briefly rather than being taken as "no on-disk backend here".
		var lastErr error
		for attempt := 0; attempt < 20; attempt++ {
			st := NewSurrealDBStorage(cfg)
			if lastErr = st.Connect(ctx); lastErr == nil {
				return st
			}
			time.Sleep(250 * time.Millisecond)
		}
		t.Skipf("the on-disk embedded engine is unavailable here: %v", lastErr)
		return nil
	}

	alice := &principal.Principal{Org: "org:a", User: "app_user:alice", WriteScope: "mem_scope:a1"}

	first := open()
	if err := first.InitializeSchema(ctx); err != nil {
		t.Fatalf("schema initialisation failed: %v", err)
	}
	for _, stmt := range []string{
		`UPSERT org:a SET name = 'a', status = 'active';`,
		`UPSERT app_user:alice SET org = org:a, subject = 'alice', active = true;`,
		`UPSERT mem_scope:a1 SET org = org:a, kind = 'personal', owner_user = app_user:alice, label = 'a/alice';`,
	} {
		if _, err := first.query(ctx, stmt, nil); err != nil {
			t.Fatalf("fixture failed: %v", err)
		}
	}

	for i := 0; i < 3; i++ {
		if err := first.SaveFact(as(alice), "", fmt.Sprintf("k%d", i), i); err != nil {
			t.Fatalf("SaveFact: %v", err)
		}
	}
	_, cursor, err := first.ListChanges(as(alice), "", "", 100)
	if err != nil {
		t.Fatalf("ListChanges: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	second := open()
	defer func() { _ = second.Close() }()

	// Nothing changed while the process was down.
	changes, _, err := second.ListChanges(as(alice), "", cursor, 100)
	if err != nil {
		t.Fatalf("ListChanges after the restart: %v", err)
	}
	if len(changes) != 0 {
		t.Fatalf("the resumed cursor replayed %d rows", len(changes))
	}

	// And a write made after the restart is picked up from the same cursor.
	if err := second.SaveFact(as(alice), "", "k9", 9); err != nil {
		t.Fatalf("SaveFact after the restart: %v", err)
	}
	changes, _, err = second.ListChanges(as(alice), "", cursor, 100)
	if err != nil {
		t.Fatalf("ListChanges after the restart: %v", err)
	}
	if len(changes) == 0 {
		t.Fatal("the resumed cursor missed a write made after the restart")
	}
}

// TestConcurrentWritersNeverShareAVersion is the race half of RM-9: two
// writers must not be able to produce two rows the feed cannot order.
func TestConcurrentWritersNeverShareAVersion(t *testing.T) {
	tn := newTenants(t)
	alice := as(tn.alice)

	const writers = 8
	var wg sync.WaitGroup
	errs := make([]error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = tn.st.SaveFact(alice, "", fmt.Sprintf("k%d", i), i)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("writer %d failed: %v", i, err)
		}
	}

	changes, _, err := tn.st.ListChanges(alice, "", "", 1000)
	if err != nil {
		t.Fatalf("ListChanges: %v", err)
	}

	versions := map[int64]string{}
	for _, c := range changes {
		if other, dup := versions[c.Version]; dup {
			t.Fatalf("version %d was handed to both %s and %s", c.Version, other, c.ID)
		}
		versions[c.Version] = c.ID
	}
	if len(changes) != writers {
		t.Fatalf("the feed reports %d rows for %d concurrent writes", len(changes), writers)
	}
}

// TestCodeStorageIsScoped covers the 27 method code surface at its edges: a
// project indexed by one tenant is not listable, readable or searchable by
// another.
func TestCodeStorageIsScoped(t *testing.T) {
	tn := newTenants(t)
	alice, bob := as(tn.alice), as(tn.bob)

	if err := tn.st.CreateCodeProject(alice, codeProjectFixture()); err != nil {
		t.Fatalf("CreateCodeProject: %v", err)
	}

	projects, err := tn.st.ListCodeProjects(bob)
	if err != nil {
		t.Fatalf("ListCodeProjects: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("org b sees org a's code projects: %+v", projects)
	}

	if p, err := tn.st.GetCodeProject(bob, "proj"); err != nil || p != nil {
		t.Errorf("GetCodeProject leaked %+v (%v)", p, err)
	}

	mine, err := tn.st.ListCodeProjects(alice)
	if err != nil {
		t.Fatalf("ListCodeProjects as the owner: %v", err)
	}
	if len(mine) != 1 {
		t.Fatalf("the owner cannot see its own project: %+v", mine)
	}
}

// codeProjectFixture is the smallest valid code project.
func codeProjectFixture() *treesitter.CodeProject {
	return &treesitter.CodeProject{
		ProjectID:      "proj",
		Name:           "proj",
		RootPath:       "/tmp/proj",
		LanguageStats:  map[treesitter.Language]int{},
		IndexingStatus: treesitter.IndexingStatusPending,
	}
}
