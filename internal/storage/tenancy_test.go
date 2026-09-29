package storage

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/madeindigio/remembrances-mcp/v2/internal/storage/migrations"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/principal"
)

// withRequirePrincipal turns the fail-closed switch on for one test and
// restores it afterwards, so the tests can run in any order.
func withRequirePrincipal(t *testing.T, v bool) {
	t.Helper()
	previous := RequirePrincipal()
	SetRequirePrincipal(v)
	t.Cleanup(func() { SetRequirePrincipal(previous) })
}

// unconnectedStorage is enough to exercise everything that happens before the
// first query: the principal resolution and the predicate building. A method
// that reaches the database without resolving a principal fails these tests
// with a connection error instead of ErrNoPrincipal, which is exactly the
// signal wanted.
func unconnectedStorage() *SurrealDBStorage {
	return NewSurrealDBStorage(&ConnectionConfig{Namespace: "t", Database: "t"})
}

// notPrincipalGoverned are the storage methods that legitimately run without a
// principal.
//
// Connect, Close, Ping and InitializeSchema are process lifecycle, not data
// access. Query is the raw escape hatch: it hands the caller the SurrealQL
// pen, so it cannot be scoped from here and every caller of it is expected to
// be reviewed on its own. Adding a name to this list is therefore a security
// decision, and the list is deliberately short enough to read in one glance.
var notPrincipalGoverned = map[string]bool{
	"Connect":          true,
	"Close":            true,
	"Ping":             true,
	"InitializeSchema": true,
	"Query":            true,
}

// storageInterfaces are the interfaces whose every method must be covered.
func storageInterfaces() map[string]reflect.Type {
	return map[string]reflect.Type{
		"Storage":       reflect.TypeOf((*Storage)(nil)).Elem(),
		"StatsProvider": reflect.TypeOf((*StatsProvider)(nil)).Elem(),
		"CodeStorage":   reflect.TypeOf((*CodeStorage)(nil)).Elem(),
		"ChangeLister":  reflect.TypeOf((*ChangeLister)(nil)).Elem(),
	}
}

// TestEveryStorageMethodFailsClosedWithoutAPrincipal is the interface coverage
// test RM-2 asks for: it enumerates the storage surface by reflection and
// asserts that each method refuses to run without an identity. A method added
// later that forgets its predicate fails here rather than in production.
func TestEveryStorageMethodFailsClosedWithoutAPrincipal(t *testing.T) {
	withRequirePrincipal(t, true)

	st := unconnectedStorage()
	value := reflect.ValueOf(st)
	ctx := context.Background()

	covered := 0
	for name, iface := range storageInterfaces() {
		for i := 0; i < iface.NumMethod(); i++ {
			method := iface.Method(i)
			if notPrincipalGoverned[method.Name] {
				continue
			}
			covered++

			t.Run(name+"."+method.Name, func(t *testing.T) {
				impl := value.MethodByName(method.Name)
				if !impl.IsValid() {
					t.Fatalf("*SurrealDBStorage does not implement %s", method.Name)
				}

				out := impl.Call(zeroArgs(impl.Type(), ctx))

				err := lastError(out)
				if err == nil {
					t.Fatalf("%s returned no error without a principal; it must fail closed", method.Name)
				}
				if !errors.Is(err, ErrNoPrincipal) {
					t.Fatalf("%s failed with %v, want an error wrapping ErrNoPrincipal", method.Name, err)
				}
			})
		}
	}

	if covered < 55 {
		t.Fatalf("only %d methods were checked; the reflection walk is not seeing the whole surface", covered)
	}
}

// zeroArgs builds a call argument list: the context first, zero values after.
func zeroArgs(fn reflect.Type, ctx context.Context) []reflect.Value {
	args := make([]reflect.Value, 0, fn.NumIn())
	for i := 0; i < fn.NumIn(); i++ {
		if i == 0 {
			args = append(args, reflect.ValueOf(ctx))
			continue
		}
		args = append(args, reflect.Zero(fn.In(i)))
	}
	return args
}

// lastError returns the trailing error of a call result, if there is one.
func lastError(out []reflect.Value) error {
	if len(out) == 0 {
		return nil
	}
	last := out[len(out)-1]
	if last.Kind() != reflect.Interface || last.IsNil() {
		return nil
	}
	err, _ := last.Interface().(error)
	return err
}

// TestChangeFeedTablesAreMemoryTables keeps the feed honest: a table that does
// not carry the v13 scope columns cannot be filtered or ordered, so listing
// one here would produce a feed nobody can scope.
func TestChangeFeedTablesAreMemoryTables(t *testing.T) {
	for _, table := range ChangeFeedTables() {
		if !containsString(migrations.MemoryTables, table) {
			t.Errorf("change feed table %q is not one of migrations.MemoryTables", table)
		}
	}
}

func TestPrincipalValidation(t *testing.T) {
	tests := []struct {
		name    string
		p       *principal.Principal
		wantErr bool
	}{
		{name: "nil", p: nil, wantErr: true},
		{name: "empty", p: &principal.Principal{}, wantErr: true},
		{
			name:    "no org",
			p:       &principal.Principal{WriteScope: "mem_scope:a"},
			wantErr: true,
		},
		{
			name:    "no write scope",
			p:       &principal.Principal{Org: "org:acme"},
			wantErr: true,
		},
		{
			name:    "org named as the wrong table",
			p:       &principal.Principal{Org: "app_user:acme", WriteScope: "mem_scope:a"},
			wantErr: true,
		},
		{
			name:    "record key with an escape attempt",
			p:       &principal.Principal{Org: "org:acme", WriteScope: "mem_scope:a' OR true"},
			wantErr: true,
		},
		{
			name:    "scope of the wrong table",
			p:       &principal.Principal{Org: "org:acme", WriteScope: "mem_scope:a", Scopes: []string{"org:acme"}},
			wantErr: true,
		},
		{
			name: "complete",
			p: &principal.Principal{
				Org:        "org:acme",
				User:       "app_user:jose",
				Groups:     []string{"user_group:eng"},
				Scopes:     []string{"mem_scope:team"},
				WriteScope: "mem_scope:jose",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.p.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestReadScopesAlwaysIncludeTheWriteScope: a caller that can create a row in
// a scope must be able to read it back, and the set must not repeat itself.
func TestReadScopesAlwaysIncludeTheWriteScope(t *testing.T) {
	p := &principal.Principal{
		Org:        "org:acme",
		WriteScope: "mem_scope:jose",
		Scopes:     []string{"mem_scope:team", "mem_scope:jose", "mem_scope:team"},
	}

	got := p.ReadScopes()
	want := []string{"mem_scope:jose", "mem_scope:team"}
	if len(got) != len(want) {
		t.Fatalf("ReadScopes() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ReadScopes() = %v, want %v", got, want)
		}
	}
}

// TestContextPrincipalWinsOverTheLegacyLabel is the "demoted to legacy" rule:
// once an identity is in the context, the userID parameter is ignored.
func TestContextPrincipalWinsOverTheLegacyLabel(t *testing.T) {
	withRequirePrincipal(t, false)

	p := &principal.Principal{Org: "org:acme", WriteScope: "mem_scope:jose"}
	ctx := principal.NewContext(context.Background(), p)

	st := unconnectedStorage()
	tn, err := st.scope(ctx, "somebody-else")
	if err != nil {
		t.Fatalf("scope() failed: %v", err)
	}
	if tn.Principal().Org != "org:acme" {
		t.Fatalf("the legacy label overrode the context principal: %s", tn.Principal())
	}
	if tn.Principal().LegacyUserID != "" {
		t.Fatalf("the legacy label leaked into the context principal: %q", tn.Principal().LegacyUserID)
	}
}

// TestLegacyLabelMapsOntoTheBackfilledRecords: the derived principal has to
// point at exactly the ids migration v13 wrote, or an upgraded datastore looks
// empty to its own user.
func TestLegacyLabelMapsOntoTheBackfilledRecords(t *testing.T) {
	p := PrincipalForUser("Jose F")
	slug := migrations.Slug("Jose F")

	if p.Org != migrations.LegacyOrgID {
		t.Errorf("org = %q, want %q", p.Org, migrations.LegacyOrgID)
	}
	if want := "app_user:legacy_" + slug; p.User != want {
		t.Errorf("user = %q, want %q", p.User, want)
	}
	if want := "mem_scope:legacy_personal_" + slug; p.WriteScope != want {
		t.Errorf("write scope = %q, want %q", p.WriteScope, want)
	}
	if !containsString(p.Scopes, migrations.LegacyUnassignedScopeID) {
		t.Errorf("the unassigned scope is missing from %v", p.Scopes)
	}
	if err := p.Validate(); err != nil {
		t.Errorf("the derived principal does not validate: %v", err)
	}
}

// TestPredicatesBindEveryIdentifier: no record id may reach the statement text
// itself, and the live restriction must be present unless it was opted out of.
func TestPredicatesBindEveryIdentifier(t *testing.T) {
	tn := &tenancy{p: &principal.Principal{
		Org:        "org:acme",
		WriteScope: "mem_scope:jose",
		User:       "app_user:jose",
	}}

	where := tn.where("key = $key")
	for _, want := range []string{
		"org_id = type::record($" + paramOrg + ")",
		"scope IN $" + paramScopes,
		"valid_to = NONE",
		"key = $key",
	} {
		if !strings.Contains(where, want) {
			t.Errorf("where() = %q, missing %q", where, want)
		}
	}
	if strings.Contains(where, "org:acme") {
		t.Errorf("where() interpolated a record id instead of binding it: %q", where)
	}

	if strings.Contains(tn.whereAny(), "valid_to = NONE") {
		t.Errorf("whereAny() must not restrict to live rows: %q", tn.whereAny())
	}

	fields := tn.contentFields()
	for _, want := range []string{"org_id:", "scope:", "created_by:", "version:", versionExpr} {
		if !strings.Contains(fields, want) {
			t.Errorf("contentFields() = %q, missing %q", fields, want)
		}
	}

	if strings.Contains(tn.updateSet(), "org_id") {
		t.Errorf("updateSet() must not rewrite org_id: %q", tn.updateSet())
	}
	if !strings.Contains(tn.invalidateSet(), "valid_to = time::now()") {
		t.Errorf("invalidateSet() does not stamp valid_to: %q", tn.invalidateSet())
	}
}

// TestParamsCarryTheScopeSetAndNeverAliasTheCaller.
func TestParamsCarryTheScopeSetAndNeverAliasTheCaller(t *testing.T) {
	tn := &tenancy{p: &principal.Principal{
		Org:        "org:acme",
		WriteScope: "mem_scope:jose",
		Scopes:     []string{"mem_scope:team"},
	}}

	own := map[string]interface{}{"key": "colour"}
	got := tn.params(own)

	if got[paramOrg] != "org:acme" {
		t.Errorf("org parameter = %v", got[paramOrg])
	}
	scopes, ok := got[paramScopes].([]string)
	if !ok || len(scopes) != 2 {
		t.Fatalf("scope parameter = %v, want the two readable scopes", got[paramScopes])
	}
	if _, aliased := own[paramOrg]; aliased {
		t.Errorf("params() mutated the caller's map")
	}
	if got["key"] != "colour" {
		t.Errorf("params() dropped the caller's own parameter")
	}
}

// TestAnEmptyScopeSetSelectsNothing. The principal here can read exactly one
// scope, its own; the point of the test is that ReadScopes never yields an
// empty list that a query would treat as "no filter".
func TestAnEmptyScopeSetSelectsNothing(t *testing.T) {
	p := &principal.Principal{Org: "org:acme", WriteScope: "mem_scope:jose"}
	if got := len(p.ReadScopes()); got != 1 {
		t.Fatalf("ReadScopes() has %d entries, want just the write scope", got)
	}

	empty := &principal.Principal{}
	if got := len(empty.ReadScopes()); got != 0 {
		t.Fatalf("an identity-free principal yielded %d scopes", got)
	}
}

func TestCursorRoundTrip(t *testing.T) {
	want := cursor{
		Version:   4242,
		UpdatedAt: time.Date(2026, 9, 4, 10, 30, 0, 123456789, time.UTC),
		ID:        "01hxyz",
	}

	got, err := decodeCursor(want.encode())
	if err != nil {
		t.Fatalf("decodeCursor() failed: %v", err)
	}
	if got.Version != want.Version || got.ID != want.ID || !got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Fatalf("round trip = %+v, want %+v", got, want)
	}
}

func TestCursorRejectsGarbage(t *testing.T) {
	if c, err := decodeCursor(""); err != nil || c.Version != 0 || c.ID != "" {
		t.Fatalf("the empty cursor must be the start of the feed, got %+v %v", c, err)
	}
	// Not base64; too few fields; an unknown format marker; a bad version.
	for _, bad := range []string{"!!!", "YWJj", "MnwxfDF8eA", "MXxub3BlfDF8eA"} {
		if _, err := decodeCursor(bad); err == nil {
			t.Errorf("decodeCursor(%q) accepted a malformed token", bad)
		}
	}
}

func TestEdgeTableNameRefusesInjection(t *testing.T) {
	for _, bad := range []string{"", "  ", "wrote; DELETE FROM entities", "wrote WHERE", "1wrote", "wro-te", "`grant`"} {
		if _, err := edgeTableName(bad); err == nil {
			t.Errorf("edgeTableName(%q) was accepted", bad)
		}
	}
	for _, good := range []string{"wrote", "mentioned_in", "related_to", "worksWith2"} {
		if got, err := edgeTableName(good); err != nil || got != good {
			t.Errorf("edgeTableName(%q) = %q, %v", good, got, err)
		}
	}
}

// TestContentHashMatchesTheSurrealExpression documents the contract the unique
// dedup index rests on: the Go side and the SurrealQL side must name the same
// function over the same trimmed input.
func TestContentHashMatchesTheSurrealExpression(t *testing.T) {
	expr := migrations.ContentHashSurrealExpr("$content")
	if !strings.Contains(expr, "crypto::sha256") || !strings.Contains(expr, "string::trim") {
		t.Fatalf("the SurrealQL content hash drifted: %q", expr)
	}
	if migrations.ContentHash("  hello  ") != migrations.ContentHash("hello") {
		t.Fatal("ContentHash does not trim, but the SurrealQL side does")
	}
}
