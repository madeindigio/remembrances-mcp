package modules

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// authModule is a test HTTPAuthProvider.
type authModule struct {
	id   ModuleID
	info *AuthInfo
	err  error
	mw   HTTPMiddleware
}

func (m *authModule) ModuleInfo() ModuleInfo {
	return ModuleInfo{ID: m.id, New: func() Module { return m }}
}

func (m *authModule) Authenticate(r *http.Request) (*AuthInfo, error) {
	return m.info, m.err
}

func (m *authModule) Middleware() HTTPMiddleware { return m.mw }

func serve(handler http.Handler) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	return rec
}

func TestAuthMiddlewareAllowsAndPopulatesAuthInfo(t *testing.T) {
	provider := &authModule{id: "test.auth.allow", info: &AuthInfo{UserID: "u1", Roles: []string{"admin"}}}

	var seen *AuthInfo
	handler := AuthMiddleware([]HTTPAuthProvider{provider}, nil)(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			info, ok := AuthInfoFromContext(r.Context())
			if !ok {
				t.Error("handler must see the AuthInfo in the request context")
			}
			seen = info
			w.WriteHeader(http.StatusOK)
		}))

	if rec := serve(handler); rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	if seen == nil || seen.UserID != "u1" {
		t.Fatalf("AuthInfo not propagated: %+v", seen)
	}
}

func TestAuthMiddlewareDeniesWhenProviderErrors(t *testing.T) {
	provider := &authModule{id: "test.auth.deny", err: errors.New("bad token")}

	called := false
	handler := AuthMiddleware([]HTTPAuthProvider{provider}, nil)(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) { called = true }))

	rec := serve(handler)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401", rec.Code)
	}
	if called {
		t.Fatal("the route handler must not run when authentication fails")
	}
}

func TestAuthMiddlewareFailsClosedWhenEveryProviderAbstains(t *testing.T) {
	provider := &authModule{id: "test.auth.abstain"}

	called := false
	handler := AuthMiddleware([]HTTPAuthProvider{provider}, nil)(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) { called = true }))

	rec := serve(handler)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401", rec.Code)
	}
	if called {
		t.Fatal("an abstaining provider must not let the request through")
	}
}

func TestAuthMiddlewareErroringProviderIsNotBypassedByALaterOne(t *testing.T) {
	strict := &authModule{id: "test.auth.strict", err: errors.New("nope")}
	lax := &authModule{id: "test.auth.lax", info: &AuthInfo{UserID: "anyone"}}

	called := false
	handler := AuthMiddleware([]HTTPAuthProvider{strict, lax}, nil)(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) { called = true }))

	rec := serve(handler)
	if rec.Code != http.StatusUnauthorized || called {
		t.Fatalf("a failing provider must be fail closed, status %d called %v", rec.Code, called)
	}
}

func TestAuthMiddlewareFirstAcceptingProviderWins(t *testing.T) {
	abstain := &authModule{id: "test.auth.first.abstain"}
	accept := &authModule{id: "test.auth.first.accept", info: &AuthInfo{UserID: "u2"}}
	never := &authModule{id: "test.auth.first.never", err: errors.New("must not be consulted")}

	handler := AuthMiddleware([]HTTPAuthProvider{abstain, accept, never}, nil)(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			info, _ := AuthInfoFromContext(r.Context())
			fmt.Fprint(w, info.UserID)
		}))

	rec := serve(handler)
	if rec.Code != http.StatusOK || rec.Body.String() != "u2" {
		t.Fatalf("status %d body %q", rec.Code, rec.Body.String())
	}
}

func TestAuthMiddlewareWithNoProvidersIsIdentity(t *testing.T) {
	handler := AuthMiddleware(nil, nil)(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if _, ok := AuthInfoFromContext(r.Context()); ok {
				t.Error("no provider means no AuthInfo")
			}
			w.WriteHeader(http.StatusTeapot)
		}))

	if rec := serve(handler); rec.Code != http.StatusTeapot {
		t.Fatalf("status: got %d want 418", rec.Code)
	}
}

func TestAuthMiddlewareAppliesProviderMiddlewaresOutsideAuthenticate(t *testing.T) {
	var order []string
	mark := func(name string) HTTPMiddleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	first := &authModule{id: "test.auth.mw.first", info: &AuthInfo{UserID: "u"}, mw: mark("first")}
	second := &authModule{id: "test.auth.mw.second", mw: mark("second")}

	handler := AuthMiddleware([]HTTPAuthProvider{first, second}, nil)(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) { order = append(order, "handler") }))

	serve(handler)
	want := "first,second,handler"
	if got := strings.Join(order, ","); got != want {
		t.Fatalf("provider middleware order: got %q want %q", got, want)
	}
}

func TestChainHTTPMiddlewaresAppliesFirstOutermost(t *testing.T) {
	var order []string
	mark := func(name string) HTTPMiddleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "in:"+name)
				next.ServeHTTP(w, r)
				order = append(order, "out:"+name)
			})
		}
	}

	handler := ChainHTTPMiddlewares(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { order = append(order, "handler") }),
		[]HTTPMiddleware{mark("a"), nil, mark("b")},
	)
	serve(handler)

	want := "in:a,in:b,handler,out:b,out:a"
	if got := strings.Join(order, ","); got != want {
		t.Fatalf("middleware order: got %q want %q", got, want)
	}
}

func TestGetHTTPAuthProvidersReturnsLoadOrder(t *testing.T) {
	a := registerForTest(t, &authModule{id: "test.auth.mgr.a"})
	plain := registerForTest(t, &plainModule{id: "test.auth.mgr.plain", rec: &recorder{}})
	b := registerForTest(t, &authModule{id: "test.auth.mgr.b"})

	mm := NewModuleManager(ModuleConfig{})
	loadAll(t, mm, a, plain, b)

	providers := mm.GetHTTPAuthProviders()
	if len(providers) != 2 {
		t.Fatalf("got %d auth providers, want 2", len(providers))
	}
	if providers[0].ModuleInfo().ID != a || providers[1].ModuleInfo().ID != b {
		t.Fatalf("auth providers out of load order: %v, %v",
			providers[0].ModuleInfo().ID, providers[1].ModuleInfo().ID)
	}
}
