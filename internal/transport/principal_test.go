package transport

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/modules"
	"github.com/madeindigio/remembrances-mcp/pkg/principal"
)

// seen records the principal a handler ran with, or its absence.
type seen struct {
	principal *principal.Principal
	called    bool
}

func recordingHandler(s *seen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.called = true
		if p, ok := principal.FromContext(r.Context()); ok {
			s.principal = p
		}
		w.WriteHeader(http.StatusOK)
	}
}

// claims is a full, valid tenancy claim set of the shape an auth provider
// resolving a Control token produces.
func claims() map[string]any {
	return map[string]any{
		metaOrg:        "org:acme",
		metaUser:       "app_user:ada",
		metaScopes:     []string{"mem_scope:ada_personal", "mem_scope:acme_shared"},
		metaWriteScope: "mem_scope:ada_personal",
		metaGroups:     []any{"user_group:engineers"},
		metaOrgAdmin:   true,
		metaDevice:     "laptop-1",
	}
}

func withAuthInfo(t *testing.T, info *modules.AuthInfo, handler http.Handler) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/api/stats", nil)
	if info != nil {
		req = req.WithContext(modules.ContextWithAuthInfo(req.Context(), info))
	}
	PrincipalMiddleware(slog.Default())(handler).ServeHTTP(rec, req)
	return rec
}

func TestPrincipalIsBuiltFromTheAuthInfoClaims(t *testing.T) {
	var got seen
	rec := withAuthInfo(t, &modules.AuthInfo{UserID: "ada", Metadata: claims()}, recordingHandler(&got))

	if rec.Code != http.StatusOK || !got.called {
		t.Fatalf("handler must run, status %d called %v", rec.Code, got.called)
	}
	if got.principal == nil {
		t.Fatal("no principal installed")
	}
	if got.principal.Org != "org:acme" || got.principal.User != "app_user:ada" {
		t.Fatalf("wrong identity: %+v", got.principal)
	}
	if len(got.principal.Scopes) != 2 || got.principal.WriteScope != "mem_scope:ada_personal" {
		t.Fatalf("wrong scopes: %+v", got.principal)
	}
	if !got.principal.IsAdmin {
		t.Error("org_admin claim must map to IsAdmin")
	}
	if got.principal.LegacyUserID != "ada" {
		t.Errorf("the legacy user id is carried for audit, got %q", got.principal.LegacyUserID)
	}
}

func TestAnUpstreamPrincipalIsNeverReplaced(t *testing.T) {
	// A provider that resolves grants itself installs the real principal in
	// its own middleware. A derivation from coarse claims must not overwrite
	// it, which is the difference between a scoped identity and a wrong one.
	upstream := &principal.Principal{Org: "org:acme", User: "app_user:ada", WriteScope: "mem_scope:team", Scopes: []string{"mem_scope:team"}}

	var got seen
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/api/stats", nil)
	ctx := modules.ContextWithAuthInfo(req.Context(), &modules.AuthInfo{UserID: "ada", Metadata: claims()})
	ctx = principal.NewContext(ctx, upstream)
	PrincipalMiddleware(nil)(recordingHandler(&got)).ServeHTTP(rec, req.WithContext(ctx))

	if got.principal != upstream {
		t.Fatalf("the upstream principal must survive, got %+v", got.principal)
	}
}

func TestAuthInfoWithoutClaimsFallsBackToTheLegacyIdentity(t *testing.T) {
	restore := storage.RequirePrincipal()
	t.Cleanup(func() { storage.SetRequirePrincipal(restore) })
	storage.SetRequirePrincipal(false)

	var got seen
	rec := withAuthInfo(t, &modules.AuthInfo{UserID: "ada"}, recordingHandler(&got))

	if rec.Code != http.StatusOK {
		t.Fatalf("the legacy path must still serve, status %d", rec.Code)
	}
	want := storage.PrincipalForUser("ada")
	if got.principal == nil || got.principal.Org != want.Org || got.principal.User != want.User || got.principal.WriteScope != want.WriteScope {
		t.Fatalf("legacy identity mismatch: got %+v want %+v", got.principal, want)
	}
}

func TestAuthInfoWithoutClaimsIsRejectedWhenPrincipalsAreRequired(t *testing.T) {
	restore := storage.RequirePrincipal()
	t.Cleanup(func() { storage.SetRequirePrincipal(restore) })
	storage.SetRequirePrincipal(true)

	var got seen
	rec := withAuthInfo(t, &modules.AuthInfo{UserID: "ada"}, recordingHandler(&got))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 with require_principal on, got %d", rec.Code)
	}
	if got.called {
		t.Fatal("the handler must not run without a principal")
	}
}

func TestMalformedClaimsAreRejectedRatherThanDegraded(t *testing.T) {
	restore := storage.RequirePrincipal()
	t.Cleanup(func() { storage.SetRequirePrincipal(restore) })
	// Explicitly with the legacy path enabled: a claim set that is present but
	// broken is a misconfiguration, and silently running it as the legacy
	// identity would hide the break behind wrong-tenant data.
	storage.SetRequirePrincipal(false)

	bad := claims()
	bad[metaOrg] = "not a record id"

	var got seen
	rec := withAuthInfo(t, &modules.AuthInfo{UserID: "ada", Metadata: bad}, recordingHandler(&got))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 for malformed claims, got %d", rec.Code)
	}
	if got.called {
		t.Fatal("the handler must not run with malformed claims")
	}
}

func TestRouteWithNoAuthenticationInstallsNoPrincipal(t *testing.T) {
	// Unauthenticated routes are unchanged: nothing is invented for them and
	// the storage layer applies its own rule.
	var got seen
	rec := withAuthInfo(t, nil, recordingHandler(&got))

	if rec.Code != http.StatusOK || !got.called {
		t.Fatalf("handler must run, status %d", rec.Code)
	}
	if got.principal != nil {
		t.Fatalf("no principal may be invented, got %+v", got.principal)
	}
}

func TestRegisteredModuleRoutesRunWithThePrincipal(t *testing.T) {
	// The end-to-end shape: an auth provider authenticates, the principal
	// layer installs the identity, the module handler sees it.
	var got seen
	provider := &endpointProvider{
		id:       "test.principal.routes",
		basePath: "/mem",
		routes: []modules.HTTPRoute{{
			Method:  http.MethodGet,
			Path:    "/facts",
			Handler: recordingHandler(&got),
		}},
	}
	auth := &authProvider{id: "test.principal.auth", info: &modules.AuthInfo{UserID: "ada", Metadata: claims()}}

	h := NewHTTPTransport(":0")
	h.RegisterModuleRoutes([]modules.HTTPEndpointProvider{provider}, auth)

	if rec := do(t, h, http.MethodGet, "/mem/facts"); rec.Code != http.StatusOK {
		t.Fatalf("route must serve, got %d", rec.Code)
	}
	if got.principal == nil || got.principal.Org != "org:acme" {
		t.Fatalf("the module handler must run as the authenticated principal, got %+v", got.principal)
	}
}
