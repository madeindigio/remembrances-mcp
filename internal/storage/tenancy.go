package storage

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/madeindigio/remembrances-mcp/v2/internal/storage/migrations"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/principal"
)

// The tenancy layer (RM-2 and RM-3).
//
// Every statement the storage layer emits carries the same two clauses:
//
//	reads:  org_id = type::record($__org) AND scope IN $__scopes.map(...)
//	writes: org_id: type::record($__org), scope: type::record($__scope), ...
//
// They are built here rather than spelled out per query so that a new query
// cannot quietly omit them, and so the exact SurrealQL is validated in one
// place. The shapes below were checked against the linked SurrealDB 3.x engine
// with `go run -tags surrealprobe ./cmd/surrealql-probe rm239`; in particular a
// record link compared to a plain string parameter parses and matches nothing,
// which is why every id goes through type::record().
//
// Two rules are absolute:
//
//   - No principal, deny. A missing principal is an error, never an unfiltered
//     query. See requirePrincipal below for the one deliberate exception, the
//     single-user local path.
//   - An empty visible scope set selects nothing, never everything.

// Parameter names the tenancy clauses bind. They are prefixed so they cannot
// collide with a query's own parameters.
const (
	paramOrg    = "__org"
	paramScopes = "__scopes"
	paramScope  = "__scope"
	paramUser   = "__user"
)

// scopePredicate filters a table to the rows of the caller's org that live in
// a scope the caller may read. It is served by idx_<prefix>_org_scope.
const scopePredicate = "org_id = type::record($" + paramOrg + ") AND scope IN $" + paramScopes + ".map(|$s| type::record($s))"

// livePredicate is scopePredicate restricted to rows that have not been
// invalidated. It is the shape idx_<prefix>_valid (org_id, scope, valid_to)
// exists to serve. Invalidation rather than deletion is the standing rule, so
// almost every read wants this one.
const livePredicate = scopePredicate + " AND valid_to = NONE"

// knnEF is the search-time ef parameter of the HNSW KNN operator. SurrealDB
// 3.x removed the bare `<|k|>` spelling with the M-Tree it belonged to: the
// operator now needs either an ef (HNSW) or a distance (brute force). It is
// pinned rather than left to the engine default for the same reason the index
// build parameters are, and matches the EFC the indexes are built with.
const knnEF = 150

// changeFeedSequence is the org-wide monotonic counter every write draws its
// `version` from. A per-row counter would make (org_id, version) useless as a
// feed order; a shared sequence makes it a total order and turns
// idx_<prefix>_org_version into the range scan RM-9 asks for.
const changeFeedSequence = "mem_version"

// versionExpr is the SurrealQL that allocates the next version.
const versionExpr = "sequence::nextval('" + changeFeedSequence + "')"

// requirePrincipal is the fail-closed switch of RM-2. When it is set, a
// storage call whose context carries no principal returns ErrNoPrincipal
// instead of falling back to the synthetic local principal.
//
// It is a package-level setting rather than a field so that it applies to
// every storage implementation in the process, including the wrappers modules
// install; the same reasoning as SetEmbeddingDimension.
var requirePrincipal atomic.Bool

// SetRequirePrincipal turns the legacy no-principal path on or off.
//
// The end state (RM-10) is that v1 builds ship with it on and the legacy path
// is removed. It defaults to off in this release so that stdio mode, where
// Remembrances runs single-user inside the agent and no token exists, keeps
// working: those calls run as LocalPrincipal, which is a real principal in the
// legacy org rather than an absence of one. Isolation is enforced either way;
// this flag only decides whether the fallback identity may be used at all.
func SetRequirePrincipal(v bool) { requirePrincipal.Store(v) }

// RequirePrincipal reports whether the legacy no-principal path is disabled.
func RequirePrincipal() bool { return requirePrincipal.Load() }

// LegacyLineEndOfSupport is the date the pre-principal line stops working
// (RM-10). It is the date the README's end-of-support section and the CHANGELOG
// deprecation entry name, declared once here so the running binary and the two
// documents cannot drift apart.
const LegacyLineEndOfSupport = "15 December 2026"

// LegacyLineNotice returns the end-of-support warning an operator should see at
// startup, or the empty string when the deployment is already on the supported
// line.
//
// A notice that lives only in a README reaches the people who read READMEs. The
// deployments this one is for are the others: they run with the legacy fallback
// enabled, they stop working on the date above, and their operator finds out
// either at startup or on the day. Callers log whatever this returns; the
// emptiness of it is the whole condition, so a deployment that has already
// migrated is never nagged.
func LegacyLineNotice() string {
	if RequirePrincipal() {
		return ""
	}
	return "the pre-principal legacy path is enabled and loses support on " + LegacyLineEndOfSupport +
		"; set --require-principal (GOMEM_REQUIRE_PRINCIPAL=true) once every caller supplies an identity, " +
		`and see the README section "End of support for the legacy line" for the migration path`
}

// ErrNoPrincipal is re-exported so storage callers need not import the
// principal package just to test for the fail-closed error.
var ErrNoPrincipal = principal.ErrNoPrincipal

// LocalUserID is the legacy user_id the synthetic local principal stands for.
// It matches the label the single-user stdio path has always written.
const LocalUserID = "local"

var (
	localPrincipalOnce sync.Once
	localPrincipal     *principal.Principal
)

// LocalPrincipal returns the synthetic principal for the single-user local
// path. It resolves to exactly the records the v13 backfill created for a
// legacy instance, so a datastore migrated from v12 keeps serving its own rows
// to the local user rather than looking empty.
func LocalPrincipal() *principal.Principal {
	localPrincipalOnce.Do(func() {
		slug := migrations.Slug(LocalUserID)
		localPrincipal = &principal.Principal{
			Org:          migrations.LegacyOrgID,
			User:         "app_user:legacy_" + slug,
			WriteScope:   "mem_scope:legacy_personal_" + slug,
			Scopes:       []string{migrations.LegacyUnassignedScopeID},
			LegacyUserID: LocalUserID,
		}
	})
	return localPrincipal.Clone()
}

// PrincipalForUser builds the legacy principal for a caller-supplied user_id.
//
// It exists for the transition only: the pre-v13 API is "pass me a user label
// and I will treat it as the owner", and until every caller carries a real
// principal that label has to map onto the records the v13 backfill derived
// from it. Once require_principal is on everywhere (RM-10) this function and
// its callers go away together.
func PrincipalForUser(userID string) *principal.Principal {
	if strings.TrimSpace(userID) == "" {
		return LocalPrincipal()
	}
	slug := migrations.Slug(userID)
	return &principal.Principal{
		Org:          migrations.LegacyOrgID,
		User:         "app_user:legacy_" + slug,
		WriteScope:   "mem_scope:legacy_personal_" + slug,
		Scopes:       []string{migrations.LegacyUnassignedScopeID},
		LegacyUserID: userID,
	}
}

// tenancy is the resolved tenant context of one storage call: the principal
// plus the parameters its clauses bind. Building it is the first thing every
// storage method does, and its error is returned unchanged, so "no principal"
// surfaces as a failure of the call rather than as an empty result.
type tenancy struct {
	p *principal.Principal
}

// scope resolves the tenancy for a call.
//
// Order of resolution:
//  1. a principal in the context, which is the only path once RM-10 lands;
//  2. otherwise, if the legacy path is enabled, the principal derived from the
//     caller-supplied legacy user id (empty means the local user);
//  3. otherwise ErrNoPrincipal.
//
// legacyUserID is the userID parameter the pre-RM-2 signatures still carry. It
// is consulted only in case 2: when the context has a principal, the parameter
// is ignored outright, which is what "demoted to legacy" means.
func (s *SurrealDBStorage) scope(ctx context.Context, legacyUserID string) (*tenancy, error) {
	if p, ok := principal.FromContext(ctx); ok {
		if err := p.Validate(); err != nil {
			return nil, fmt.Errorf("invalid principal: %w", err)
		}
		return &tenancy{p: p}, nil
	}
	if RequirePrincipal() {
		return nil, fmt.Errorf("%w: require_principal is on and this call carries no identity", ErrNoPrincipal)
	}
	p := PrincipalForUser(legacyUserID)
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("invalid legacy principal for %q: %w", legacyUserID, err)
	}
	s.ensureLegacyPrincipal(ctx, p)
	return &tenancy{p: p}, nil
}

// ensureLegacyPrincipal creates the org, app_user and mem_scope records a
// derived legacy principal points at, once per process per label.
//
// The v13 backfill created these for every user_id that existed when the
// migration ran; a label that appears afterwards has none, and while
// SurrealDB is happy to store a link to a record that does not exist, an ACL
// resolver reading mem_scope would then find nothing to resolve. Failures are
// logged rather than returned: the write path works either way, and refusing
// to serve a read because a convenience row could not be created would be the
// wrong trade.
func (s *SurrealDBStorage) ensureLegacyPrincipal(ctx context.Context, p *principal.Principal) {
	if _, done := s.legacyPrincipals.Load(p.WriteScope); done {
		return
	}

	stmt := `
BEGIN TRANSACTION;
UPSERT type::record($org) SET name = 'legacy', status = 'active';
UPSERT type::record($scope_all) SET org = type::record($org), kind = 'org', label = 'legacy/unassigned';
UPSERT type::record($user) SET org = type::record($org), subject = $label, display_name = $label, active = true, is_org_admin = false;
UPSERT type::record($scope) SET org = type::record($org), kind = 'personal', owner_user = type::record($user), label = 'legacy/personal/' + $label;
COMMIT TRANSACTION;`

	params := map[string]interface{}{
		"org":       p.Org,
		"scope_all": migrations.LegacyUnassignedScopeID,
		"user":      p.User,
		"scope":     p.WriteScope,
		"label":     p.LegacyUserID,
	}
	if _, err := s.query(ctx, stmt, params); err != nil {
		slog.Debug("could not materialise the legacy principal", "principal", p.String(), "error", err)
		return
	}
	s.legacyPrincipals.Store(p.WriteScope, struct{}{})
}

// Principal returns the identity this call runs as.
func (t *tenancy) Principal() *principal.Principal { return t.p }

// params merges the tenancy parameters into a query's own parameters. The
// argument may be nil; the returned map is always non-nil and is never the
// argument itself, so a caller can build parameters once and scope them twice.
func (t *tenancy) params(own map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(own)+4)
	for k, v := range own {
		out[k] = v
	}
	out[paramOrg] = t.p.Org
	out[paramScopes] = t.p.ReadScopes()
	out[paramScope] = t.p.WriteScope
	if t.p.User != "" {
		out[paramUser] = t.p.User
	}
	return out
}

// where renders a WHERE clause that is scoped and, unless includeInvalidated
// is set, restricted to live rows. Additional conditions are ANDed after the
// tenant clauses so the index-serving prefix comes first.
func (t *tenancy) where(conditions ...string) string {
	return t.whereWith(livePredicate, conditions...)
}

// whereAny is where without the valid_to restriction, for the few callers that
// must see invalidated rows: the change feed and the dedup paths.
func (t *tenancy) whereAny(conditions ...string) string {
	return t.whereWith(scopePredicate, conditions...)
}

func (t *tenancy) whereWith(base string, conditions ...string) string {
	var b strings.Builder
	b.WriteString("WHERE ")
	b.WriteString(base)
	for _, c := range conditions {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		b.WriteString(" AND ")
		b.WriteString(c)
	}
	return b.String()
}

// contentFields renders the tenant columns for a CREATE ... CONTENT { } or an
// INSERT INTO ... { } body, with a trailing comma so it can be prepended to
// the row's own fields. Every write stamps org, scope, creator, validity start
// and a fresh feed version: a row without them is refused by the schema after
// v13, which is the fail-closed property at the database level.
func (t *tenancy) contentFields() string {
	parts := []string{
		"org_id: type::record($" + paramOrg + ")",
		"scope: type::record($" + paramScope + ")",
		"valid_from: time::now()",
		"updated_at: time::now()",
		"version: " + versionExpr,
	}
	if t.p.User != "" {
		parts = append(parts, "created_by: type::record($"+paramUser+")")
	}
	return strings.Join(parts, ",\n\t\t\t") + ",\n\t\t\t"
}

// updateSet renders the tenant columns for an UPDATE ... SET, with a trailing
// comma. It bumps the version and the timestamp but never rewrites org_id,
// scope or created_by: an update must not be able to move a row into another
// tenant.
func (t *tenancy) updateSet() string {
	return "updated_at = time::now(),\n\t\t\tversion = " + versionExpr + ",\n\t\t\t"
}

// invalidateSet renders the SET clause of a soft delete: the row stays, gains
// a valid_to and a new feed version, and so becomes the tombstone RM-9's
// consumers mirror.
func (t *tenancy) invalidateSet() string {
	return "valid_to = time::now(), updated_at = time::now(), version = " + versionExpr
}
