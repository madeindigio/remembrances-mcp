// Package principal carries the identity of the caller through
// context.Context, and is the single answer the storage layer consults before
// it touches a row.
//
// # Why the context and not a parameter
//
// Before RM-2 the only ownership signal Remembrances had was a caller-supplied
// userID string, filtered in some queries and ignored in others. A parameter
// that a caller may pass, forget, or invent is not an authorisation boundary.
// Moving the identity into the context means every storage method reads the
// same value from the same place, and a method that forgets to read it is
// caught by the interface coverage test rather than by a leak.
//
// The identifiers here are SurrealDB record ids of the v13 principal tables:
// Org is an `org` record, User an `app_user` record, Groups are `user_group`
// records and Scopes and WriteScope are `mem_scope` records. Resolving a token
// into this shape is Alchemai Mem's job (the ACL resolver); the core only
// consumes the result, validates it, and refuses to query without it.
//
// The type is in pkg/ rather than internal/ deliberately: Mem builds Principal
// values and must be able to import the definition rather than mirror it.
package principal

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrNoPrincipal is returned when a storage call has no principal in its
// context and the legacy fallback is disabled. It is the fail-closed error:
// no principal, deny, never "query every row".
var ErrNoPrincipal = errors.New("no principal in context")

// Principal is the identity a storage call runs as.
//
// Org and WriteScope are mandatory for a usable principal; Scopes is the read
// set and an empty read set means "see nothing", never "see everything".
type Principal struct {
	// Org is the tenant record id, for example "org:acme". Every query is
	// filtered by it and every write stamps it.
	Org string `json:"org"`

	// User is the app_user record id of the caller, empty for a service
	// principal that acts for no particular person.
	User string `json:"user,omitempty"`

	// Groups are the user_group record ids the caller belongs to. The core
	// does not resolve grants itself; the field is carried so an ACL wrapper
	// above it, and the audit log below it, see the same identity.
	Groups []string `json:"groups,omitempty"`

	// Scopes is the set of mem_scope record ids the caller may read. It is
	// resolved upstream from the grant table. An empty set is legal and means
	// the caller reads nothing.
	Scopes []string `json:"scopes,omitempty"`

	// WriteScope is the mem_scope record id new rows are created in, normally
	// the caller's personal scope.
	WriteScope string `json:"write_scope"`

	// IsAdmin marks an org administrator. It does not widen Scopes by itself:
	// the resolver decides what an admin sees and puts it in Scopes, so the
	// storage layer has exactly one rule to apply.
	IsAdmin bool `json:"is_admin,omitempty"`

	// Device identifies the client device, carried for audit only.
	Device string `json:"device,omitempty"`

	// IsService marks a machine caller (a worker, a sync job).
	IsService bool `json:"is_service,omitempty"`

	// LegacyUserID is the pre-v13 caller-supplied user_id label. It is kept so
	// the legacy user_id predicates can still be applied while the legacy path
	// exists, and is ignored once require_principal is on for everyone (RM-10).
	LegacyUserID string `json:"legacy_user_id,omitempty"`
}

// ctxKey is unexported so no other package can put a value under this key.
type ctxKey struct{}

// NewContext returns a copy of ctx carrying p.
func NewContext(ctx context.Context, p *Principal) context.Context {
	if p == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, p)
}

// FromContext returns the principal carried by ctx, if any.
func FromContext(ctx context.Context) (*Principal, bool) {
	if ctx == nil {
		return nil, false
	}
	p, ok := ctx.Value(ctxKey{}).(*Principal)
	if !ok || p == nil {
		return nil, false
	}
	return p, true
}

// Validate reports whether the principal can be used to build a query.
//
// It is strict on purpose. Every identifier must be a syntactically valid
// SurrealDB record id of the expected table, because a malformed id would
// otherwise reach type::record() and either error deep inside a query or, far
// worse, silently match nothing and look like an empty result.
func (p *Principal) Validate() error {
	if p == nil {
		return ErrNoPrincipal
	}
	if err := checkRecordID("org", p.Org, "org"); err != nil {
		return err
	}
	if err := checkRecordID("write scope", p.WriteScope, "mem_scope"); err != nil {
		return err
	}
	if p.User != "" {
		if err := checkRecordID("user", p.User, "app_user"); err != nil {
			return err
		}
	}
	for _, g := range p.Groups {
		if err := checkRecordID("group", g, "user_group"); err != nil {
			return err
		}
	}
	for _, s := range p.Scopes {
		if err := checkRecordID("scope", s, "mem_scope"); err != nil {
			return err
		}
	}
	return nil
}

// ReadScopes returns the scopes the principal may read, always including the
// write scope: a caller that can create a row in a scope can read it back.
// The result is a fresh slice, safe for the caller to pass as a query
// parameter.
func (p *Principal) ReadScopes() []string {
	if p == nil {
		return nil
	}
	out := make([]string, 0, len(p.Scopes)+1)
	seen := make(map[string]struct{}, len(p.Scopes)+1)
	for _, s := range append([]string{p.WriteScope}, p.Scopes...) {
		if s == "" {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// Clone returns a deep copy, so a caller can derive a narrower principal
// without aliasing the slices of the original.
func (p *Principal) Clone() *Principal {
	if p == nil {
		return nil
	}
	c := *p
	c.Groups = append([]string(nil), p.Groups...)
	c.Scopes = append([]string(nil), p.Scopes...)
	return &c
}

// String renders the principal for logs. It deliberately omits nothing
// sensitive because nothing here is a secret: these are record ids.
func (p *Principal) String() string {
	if p == nil {
		return "principal(none)"
	}
	return fmt.Sprintf("principal(org=%s user=%s scopes=%d admin=%t service=%t)",
		p.Org, p.User, len(p.ReadScopes()), p.IsAdmin, p.IsService)
}

// checkRecordID validates one "table:id" record id and that it names the
// expected table.
func checkRecordID(what, id, wantTable string) error {
	if id == "" {
		return fmt.Errorf("%w: %s is empty", ErrNoPrincipal, what)
	}
	table, key, ok := strings.Cut(id, ":")
	if !ok {
		return fmt.Errorf("invalid %s %q: not a record id of the form table:id", what, id)
	}
	if table != wantTable {
		return fmt.Errorf("invalid %s %q: expected a %s record", what, id, wantTable)
	}
	if key == "" || !isIdentifier(key) {
		return fmt.Errorf("invalid %s %q: %q is not a plain record key", what, id, key)
	}
	return nil
}

// isIdentifier reports whether s is a plain SurrealDB record key: ASCII
// letters, digits and underscore. Anything else (quotes, backticks, angle
// brackets, whitespace) is refused rather than escaped, because a principal is
// machine-generated and an exotic id is a bug, not a use case.
func isIdentifier(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_':
		default:
			return false
		}
	}
	return true
}
