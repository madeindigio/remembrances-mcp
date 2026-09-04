package transport

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/modules"
	"github.com/madeindigio/remembrances-mcp/pkg/principal"
)

// The bridge between the module authentication layer and the storage tenancy
// layer (RM-10).
//
// RM-2 put the principal in the context and made the storage layer refuse to
// query without one, but nothing on the HTTP path put a principal there: an
// authenticated module route reached storage with an AuthInfo and no
// principal, so every call ran on the legacy fallback and
// SetRequirePrincipal(true) would have failed every request. This file closes
// that gap for module routes, which is what makes the fail-closed switch
// usable rather than only announced.
//
// The order of resolution is deliberate:
//
//  1. a principal already in the context wins. An auth provider that resolves
//     grants itself (Alchemai Mem does) installs the real principal in its own
//     Middleware, and a derivation from AuthInfo claims must never overwrite
//     it with something coarser;
//  2. otherwise the AuthInfo claims are used, and a claim set that is present
//     but malformed rejects the request rather than degrading to the legacy
//     identity;
//  3. otherwise, the legacy identity derived from AuthInfo.UserID, but only
//     while the legacy path is enabled. With require_principal on, a request
//     whose authentication carries no tenancy claims is rejected here rather
//     than failing later inside a query.

// Metadata keys an HTTPAuthProvider may set on modules.AuthInfo to describe
// the tenancy of the caller. They are the JSON field names of
// principal.Principal so that a provider mapping token claims has one obvious
// spelling to use.
const (
	metaOrg        = "org"
	metaUser       = "user"
	metaGroups     = "groups"
	metaScopes     = "scopes"
	metaWriteScope = "write_scope"
	metaDevice     = "device"
	metaIsService  = "is_service"
	metaIsAdmin    = "is_admin"
	// metaOrgAdmin is the spelling Alchemai Mem's verifier already emits.
	metaOrgAdmin = "org_admin"
)

// errNoTenancyClaims reports that an AuthInfo carries no org claim, so no
// principal can be derived from it. It is not a failure by itself: the caller
// decides between the legacy identity and a rejection.
var errNoTenancyClaims = fmt.Errorf("auth info carries no %q claim", metaOrg)

// principalFromAuthInfo builds a principal from the tenancy claims of an
// AuthInfo. It returns errNoTenancyClaims when there is no org claim, and a
// validation error when the claims are present but do not form a usable
// principal.
func principalFromAuthInfo(info *modules.AuthInfo) (*principal.Principal, error) {
	if info == nil {
		return nil, errNoTenancyClaims
	}
	org := metaString(info.Metadata, metaOrg)
	if org == "" {
		return nil, errNoTenancyClaims
	}

	p := &principal.Principal{
		Org:          org,
		User:         metaString(info.Metadata, metaUser),
		Groups:       metaStrings(info.Metadata, metaGroups),
		Scopes:       metaStrings(info.Metadata, metaScopes),
		WriteScope:   metaString(info.Metadata, metaWriteScope),
		IsAdmin:      metaBool(info.Metadata, metaIsAdmin) || metaBool(info.Metadata, metaOrgAdmin),
		Device:       metaString(info.Metadata, metaDevice),
		IsService:    metaBool(info.Metadata, metaIsService),
		LegacyUserID: info.UserID,
	}
	if len(p.Groups) == 0 {
		// Roles are the group list of providers that do not spell it out
		// separately; they are carried for audit, never for a grant decision.
		p.Groups = info.Roles
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return p, nil
}

// PrincipalMiddleware installs the storage principal for module routes.
//
// It runs inside the authentication layer, so an AuthInfo is already in the
// context when it does. It never widens an identity: a principal installed
// upstream is left untouched, and a request it cannot resolve is either left
// on the legacy path or rejected, according to storage.RequirePrincipal.
func PrincipalMiddleware(logger *slog.Logger) modules.HTTPMiddleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if _, ok := principal.FromContext(ctx); ok {
				next.ServeHTTP(w, r)
				return
			}

			info, authenticated := modules.AuthInfoFromContext(ctx)
			if !authenticated {
				// No authentication layer on this route. Nothing can be
				// derived; the storage layer applies its own rule, which with
				// require_principal on is a refusal.
				next.ServeHTTP(w, r)
				return
			}

			p, err := principalFromAuthInfo(info)
			switch {
			case err == nil:
				next.ServeHTTP(w, r.WithContext(principal.NewContext(ctx, p)))
				return
			case err == errNoTenancyClaims && !storage.RequirePrincipal():
				// The transitional path: an authenticated caller with no
				// tenancy claims runs as the legacy identity of its user id,
				// which is the same identity the pre-RM-2 API gave it.
				legacy := storage.PrincipalForUser(info.UserID)
				next.ServeHTTP(w, r.WithContext(principal.NewContext(ctx, legacy)))
				return
			default:
				if logger != nil {
					logger.Warn("Rejecting request without a usable principal",
						"path", r.URL.Path, "user_id", info.UserID, "error", err)
				}
				http.Error(w, "Forbidden: no usable principal", http.StatusForbidden)
			}
		})
	}
}

// metaString reads a string metadata value, tolerating a missing map.
func metaString(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	s, _ := meta[key].(string)
	return strings.TrimSpace(s)
}

// metaStrings reads a string list metadata value, accepting both []string and
// the []any a JSON decoder produces.
func metaStrings(meta map[string]any, key string) []string {
	if meta == nil {
		return nil
	}
	switch v := meta[key].(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	default:
		return nil
	}
}

// metaBool reads a boolean metadata value.
func metaBool(meta map[string]any, key string) bool {
	if meta == nil {
		return false
	}
	b, _ := meta[key].(bool)
	return b
}
