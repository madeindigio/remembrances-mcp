package transport

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/madeindigio/remembrances-mcp/v2/pkg/modules"
)

// endpointProvider is a test modules.HTTPEndpointProvider.
type endpointProvider struct {
	id       modules.ModuleID
	basePath string
	routes   []modules.HTTPRoute
}

func (p *endpointProvider) ModuleInfo() modules.ModuleInfo {
	return modules.ModuleInfo{ID: p.id, New: func() modules.Module { return p }}
}

func (p *endpointProvider) BasePath() string            { return p.basePath }
func (p *endpointProvider) Routes() []modules.HTTPRoute { return p.routes }

// authProvider is a test modules.HTTPAuthProvider.
type authProvider struct {
	id   modules.ModuleID
	info *modules.AuthInfo
	err  error
	mw   modules.HTTPMiddleware
}

func (p *authProvider) ModuleInfo() modules.ModuleInfo {
	return modules.ModuleInfo{ID: p.id, New: func() modules.Module { return p }}
}

func (p *authProvider) Authenticate(r *http.Request) (*modules.AuthInfo, error) {
	return p.info, p.err
}

func (p *authProvider) Middleware() modules.HTTPMiddleware { return p.mw }

func do(t *testing.T, h *HTTPTransport, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.Handler().ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("handler"))
}

func TestRegisterModuleRoutesAppliesMiddlewaresInDeclaredOrder(t *testing.T) {
	var order []string
	mark := func(name string) modules.HTTPMiddleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "in:"+name)
				next.ServeHTTP(w, r)
				order = append(order, "out:"+name)
			})
		}
	}

	tr := NewHTTPTransport(":0")
	tr.RegisterModuleRoutes([]modules.HTTPEndpointProvider{&endpointProvider{
		id:       "test.http.order",
		basePath: "/api/mod",
		routes: []modules.HTTPRoute{{
			Method:      http.MethodGet,
			Path:        "/thing",
			Handler:     func(w http.ResponseWriter, r *http.Request) { order = append(order, "handler") },
			Middlewares: []modules.HTTPMiddleware{mark("first"), mark("second")},
		}},
	}})

	if rec := do(t, tr, http.MethodGet, "/api/mod/thing"); rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}

	want := "in:first,in:second,handler,out:second,out:first"
	if got := strings.Join(order, ","); got != want {
		t.Fatalf("middleware order: got %q want %q", got, want)
	}
}

func TestRegisterModuleRoutesHonoursMethod(t *testing.T) {
	tr := NewHTTPTransport(":0")
	tr.RegisterModuleRoutes([]modules.HTTPEndpointProvider{&endpointProvider{
		id:       "test.http.method",
		basePath: "/api/mod",
		routes: []modules.HTTPRoute{
			{Method: http.MethodPost, Path: "/thing", Handler: okHandler},
			{Method: http.MethodDelete, Path: "/thing", Handler: okHandler},
		},
	}})

	if rec := do(t, tr, http.MethodPost, "/api/mod/thing"); rec.Code != http.StatusOK {
		t.Fatalf("POST: got %d want 200", rec.Code)
	}
	if rec := do(t, tr, http.MethodDelete, "/api/mod/thing"); rec.Code != http.StatusOK {
		t.Fatalf("DELETE: got %d want 200", rec.Code)
	}

	rec := do(t, tr, http.MethodGet, "/api/mod/thing")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET on a POST route: got %d want 405", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != "POST, DELETE" {
		t.Fatalf("Allow header: got %q want %q", allow, "POST, DELETE")
	}
}

func TestRegisterModuleRoutesWithoutMethodMatchesEveryMethod(t *testing.T) {
	tr := NewHTTPTransport(":0")
	tr.RegisterModuleRoutes([]modules.HTTPEndpointProvider{&endpointProvider{
		id:       "test.http.anymethod",
		basePath: "/api/mod",
		routes:   []modules.HTTPRoute{{Path: "/legacy", Handler: okHandler}},
	}})

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch} {
		if rec := do(t, tr, method, "/api/mod/legacy"); rec.Code != http.StatusOK {
			t.Fatalf("%s: got %d want 200", method, rec.Code)
		}
	}
}

func TestRegisterModuleRoutesWildcardPath(t *testing.T) {
	tr := NewHTTPTransport(":0")
	tr.RegisterModuleRoutes([]modules.HTTPEndpointProvider{&endpointProvider{
		id:       "test.http.wildcard",
		basePath: "/api/mod",
		routes:   []modules.HTTPRoute{{Method: http.MethodGet, Path: "/static/*", Handler: okHandler}},
	}})

	if rec := do(t, tr, http.MethodGet, "/api/mod/static/deep/file.css"); rec.Code != http.StatusOK {
		t.Fatalf("wildcard: got %d want 200", rec.Code)
	}
}

func TestRegisterModuleRoutesAuthProviderAllowsAndDenies(t *testing.T) {
	var seenUser string
	routes := []modules.HTTPRoute{{
		Method: http.MethodGet,
		Path:   "/secret",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			if info, ok := modules.AuthInfoFromContext(r.Context()); ok {
				seenUser = info.UserID
			}
			w.WriteHeader(http.StatusOK)
		},
	}}

	t.Run("allow", func(t *testing.T) {
		tr := NewHTTPTransport(":0")
		tr.RegisterModuleRoutes(
			[]modules.HTTPEndpointProvider{&endpointProvider{id: "test.http.auth.ok", basePath: "/api/mod", routes: routes}},
			// Tenancy claims, so the fixture describes an identity the
			// principal layer accepts whether require_principal is on or off.
			&authProvider{id: "test.http.auth.provider.ok", info: &modules.AuthInfo{
				UserID:   "alice",
				Metadata: map[string]any{"org": "org:acme", "user": "app_user:alice", "write_scope": "mem_scope:alice"},
			}},
		)
		if rec := do(t, tr, http.MethodGet, "/api/mod/secret"); rec.Code != http.StatusOK {
			t.Fatalf("status: got %d want 200", rec.Code)
		}
		if seenUser != "alice" {
			t.Fatalf("AuthInfo not visible to the handler, got %q", seenUser)
		}
	})

	t.Run("deny", func(t *testing.T) {
		seenUser = ""
		tr := NewHTTPTransport(":0")
		tr.RegisterModuleRoutes(
			[]modules.HTTPEndpointProvider{&endpointProvider{id: "test.http.auth.deny", basePath: "/api/mod", routes: routes}},
			&authProvider{id: "test.http.auth.provider.deny", err: errors.New("invalid token")},
		)
		rec := do(t, tr, http.MethodGet, "/api/mod/secret")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status: got %d want 401", rec.Code)
		}
		if seenUser != "" {
			t.Fatal("the handler must not run when the auth provider fails")
		}
	})

	t.Run("fail closed when the provider abstains", func(t *testing.T) {
		seenUser = ""
		tr := NewHTTPTransport(":0")
		tr.RegisterModuleRoutes(
			[]modules.HTTPEndpointProvider{&endpointProvider{id: "test.http.auth.abstain", basePath: "/api/mod", routes: routes}},
			&authProvider{id: "test.http.auth.provider.abstain"},
		)
		if rec := do(t, tr, http.MethodGet, "/api/mod/secret"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("status: got %d want 401", rec.Code)
		}
		if seenUser != "" {
			t.Fatal("an abstaining provider must not let the request through")
		}
	})
}

func TestRegisterModuleRoutesAuthRunsBeforeRouteMiddlewares(t *testing.T) {
	var order []string
	tr := NewHTTPTransport(":0")
	tr.RegisterModuleRoutes(
		[]modules.HTTPEndpointProvider{&endpointProvider{
			id:       "test.http.authorder",
			basePath: "/api/mod",
			routes: []modules.HTTPRoute{{
				Method:  http.MethodGet,
				Path:    "/thing",
				Handler: func(w http.ResponseWriter, r *http.Request) { order = append(order, "handler") },
				Middlewares: []modules.HTTPMiddleware{func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						order = append(order, "route-mw")
						next.ServeHTTP(w, r)
					})
				}},
			}},
		}},
		&authProvider{
			id: "test.http.authorder.provider",
			// Tenancy claims, so the layer order is asserted in both
			// require_principal states.
			info: &modules.AuthInfo{
				UserID:   "u",
				Metadata: map[string]any{"org": "org:acme", "user": "app_user:u", "write_scope": "mem_scope:u"},
			},
			mw: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					order = append(order, "auth-mw")
					next.ServeHTTP(w, r)
				})
			},
		},
	)

	do(t, tr, http.MethodGet, "/api/mod/thing")

	want := "auth-mw,route-mw,handler"
	if got := strings.Join(order, ","); got != want {
		t.Fatalf("layer order: got %q want %q", got, want)
	}
}

func TestRegisterModuleRoutesWrongMethodSkipsAuthAndMiddlewares(t *testing.T) {
	var order []string
	tr := NewHTTPTransport(":0")
	tr.RegisterModuleRoutes(
		[]modules.HTTPEndpointProvider{&endpointProvider{
			id:       "test.http.methodfirst",
			basePath: "/api/mod",
			routes: []modules.HTTPRoute{{
				Method:  http.MethodPost,
				Path:    "/thing",
				Handler: okHandler,
				Middlewares: []modules.HTTPMiddleware{func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						order = append(order, "route-mw")
						next.ServeHTTP(w, r)
					})
				}},
			}},
		}},
		&authProvider{id: "test.http.methodfirst.provider", err: errors.New("must not be consulted")},
	)

	if rec := do(t, tr, http.MethodGet, "/api/mod/thing"); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status: got %d want 405", rec.Code)
	}
	if len(order) != 0 {
		t.Fatalf("nothing must run for a wrong method, got %v", order)
	}
}

func TestRegisterModuleRoutesDuplicateRouteIsIgnored(t *testing.T) {
	tr := NewHTTPTransport(":0")
	first := func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("first")) }
	second := func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("second")) }

	tr.RegisterModuleRoutes([]modules.HTTPEndpointProvider{
		&endpointProvider{id: "test.http.dup.a", basePath: "/api/mod", routes: []modules.HTTPRoute{
			{Method: http.MethodGet, Path: "/thing", Handler: first},
		}},
		&endpointProvider{id: "test.http.dup.b", basePath: "/api/mod", routes: []modules.HTTPRoute{
			{Method: http.MethodGet, Path: "/thing", Handler: second},
		}},
	})

	if body := do(t, tr, http.MethodGet, "/api/mod/thing").Body.String(); body != "first" {
		t.Fatalf("the first registration must win, got %q", body)
	}
}

func TestRegisterModuleRoutesNilHandlerIsSkipped(t *testing.T) {
	tr := NewHTTPTransport(":0")
	tr.RegisterModuleRoutes([]modules.HTTPEndpointProvider{&endpointProvider{
		id:       "test.http.nil",
		basePath: "/api/mod",
		routes:   []modules.HTTPRoute{{Method: http.MethodGet, Path: "/thing"}},
	}})

	if rec := do(t, tr, http.MethodGet, "/api/mod/thing"); rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d want 404", rec.Code)
	}
}
