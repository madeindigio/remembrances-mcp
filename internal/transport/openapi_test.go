package transport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/madeindigio/remembrances-mcp/v2/pkg/modules"
)

// The tests that keep the API document honest (RM-5).
//
// A generated document is only worth having if something fails when it stops
// matching the server. These tests are that something: they drive a real
// transport, ask it what it serves, and compare that with what the document
// says. A route added without an operation, an operation without a route, a
// method the server answers and the document omits, all fail here.

// documentedTransport builds a transport with the core routes and a module
// whose route table exercises the shapes a module can declare: a plain route,
// a second method on the same path, and a wildcard subtree.
func documentedTransport(t *testing.T) *HTTPTransport {
	t.Helper()
	provider := &endpointProvider{
		id:       "test.docs",
		basePath: "/mem",
		routes: []modules.HTTPRoute{
			{Method: http.MethodGet, Path: "/facts", Handler: okHandler, Description: "List the facts of the caller"},
			{Method: http.MethodPost, Path: "/facts", Handler: okHandler, Description: "Save a fact"},
			{Method: http.MethodGet, Path: "/static/*", Handler: okHandler, Description: "Serve static assets"},
		},
	}

	h := NewHTTPTransport(":0")
	if err := h.SetupMCPRoutes(nil); err != nil {
		t.Fatalf("core route registration failed: %v", err)
	}
	h.RegisterModuleRoutes([]modules.HTTPEndpointProvider{provider})
	return h
}

// key is the identity of an operation for set comparisons.
func key(method, path string) string { return strings.ToUpper(method) + " " + path }

func TestTheDocumentDescribesExactlyTheRegisteredRoutes(t *testing.T) {
	h := documentedTransport(t)
	routes := append(h.RegisteredRoutes(), MCPEndpointRoutes(DefaultMCPEndpoint)...)

	doc, err := BuildDocument(routes, "test")
	if err != nil {
		t.Fatalf("building the document failed: %v", err)
	}

	registered := map[string]bool{}
	for _, route := range routes {
		registered[key(route.Method, route.Path)] = true
	}

	documented := map[string]bool{}
	for path, item := range doc.Paths {
		for method := range item {
			documented[key(method, path)] = true
		}
	}

	for k := range registered {
		if !documented[k] {
			t.Errorf("route %s is served with no documented operation", k)
		}
	}
	for k := range documented {
		if !registered[k] {
			t.Errorf("operation %s is documented but no route serves it", k)
		}
	}
}

func TestEveryDocumentedRouteOfThisMuxIsActuallyServed(t *testing.T) {
	// The set comparison above proves the document matches the table. This one
	// proves the table matches the mux, which is what stops the table becoming
	// a second hand-written list.
	h := documentedTransport(t)

	for _, route := range h.RegisteredRoutes() {
		if route.External {
			continue
		}
		path := route.MuxPath
		if path == "" {
			t.Errorf("route %s %s declares no mux path", route.Method, route.Path)
			continue
		}
		if strings.HasSuffix(path, "/") && strings.Contains(route.Path, "{") {
			// A subtree route: address a file inside it.
			path += "probe.css"
		}

		rec := httptest.NewRecorder()
		h.Handler().ServeHTTP(rec, httptest.NewRequest(route.Method, path, strings.NewReader("{}")))
		switch rec.Code {
		case http.StatusNotFound, http.StatusMethodNotAllowed:
			t.Errorf("documented operation %s %s is not served: status %d", route.Method, path, rec.Code)
		}
	}
}

func TestAMethodTheDocumentDoesNotClaimIsRefused(t *testing.T) {
	// The other half of parity: the server must not quietly answer methods the
	// document never mentions.
	h := documentedTransport(t)

	documented := map[string]bool{}
	paths := map[string]bool{}
	for _, route := range h.RegisteredRoutes() {
		if route.External || route.MuxPath == "" {
			continue
		}
		documented[key(route.Method, route.MuxPath)] = true
		paths[route.MuxPath] = true
	}

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for path := range paths {
		if strings.HasSuffix(path, "/") {
			continue // a subtree pattern also matches its parent, tested above
		}
		for _, method := range methods {
			if documented[key(method, path)] {
				continue
			}
			rec := httptest.NewRecorder()
			h.Handler().ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader("{}")))
			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s %s is not documented but answered %d instead of 405", method, path, rec.Code)
			}
		}
	}
}

func TestAWildcardRouteIsDocumentedAsAPathParameter(t *testing.T) {
	h := documentedTransport(t)

	var found bool
	for _, route := range h.RegisteredRoutes() {
		if route.Path != "/mem/static/{path}" {
			continue
		}
		found = true
		if route.MuxPath != "/mem/static/" {
			t.Errorf("the subtree pattern must stay the mux path, got %q", route.MuxPath)
		}
		if len(route.Parameters) != 1 || route.Parameters[0].Name != "path" || route.Parameters[0].In != "path" {
			t.Errorf("the wildcard must become one path parameter, got %+v", route.Parameters)
		}
	}
	if !found {
		t.Fatal("the wildcard route was not documented")
	}
}

func TestTheDocumentIsAWellFormedOpenAPI31(t *testing.T) {
	h := documentedTransport(t)
	routes := append(h.RegisteredRoutes(), MCPEndpointRoutes(DefaultMCPEndpoint)...)
	doc, err := BuildDocument(routes, "test")
	if err != nil {
		t.Fatalf("building the document failed: %v", err)
	}

	if doc.OpenAPI != "3.1.0" {
		t.Errorf("wrong OpenAPI version %q", doc.OpenAPI)
	}
	if doc.Info.Title == "" || doc.Info.Version == "" {
		t.Error("info must carry a title and a version")
	}

	ids := map[string]string{}
	for path, item := range doc.Paths {
		for method, op := range item {
			where := key(method, path)
			if op.OperationID == "" {
				t.Errorf("%s has no operation id", where)
			}
			if previous, ok := ids[op.OperationID]; ok {
				t.Errorf("operation id %q is used by both %s and %s", op.OperationID, where, previous)
			}
			ids[op.OperationID] = where
			if len(op.Responses) == 0 {
				t.Errorf("%s documents no response", where)
			}
			if _, ok := op.Responses["200"]; !ok {
				t.Errorf("%s documents no success response", where)
			}
			// Every placeholder in the path must be a declared parameter.
			for _, segment := range strings.Split(path, "/") {
				if !strings.HasPrefix(segment, "{") {
					continue
				}
				name := strings.Trim(segment, "{}")
				var declared bool
				for _, p := range op.Parameters {
					if p.Name == name && p.In == "path" && p.Required {
						declared = true
					}
				}
				if !declared {
					t.Errorf("%s uses {%s} without declaring it as a required path parameter", where, name)
				}
			}
		}
	}

	// A referenced component must exist, or the document does not resolve.
	for path, item := range doc.Paths {
		for method, op := range item {
			for _, response := range op.Responses {
				for _, media := range response.Content {
					checkRef(t, doc, media.Schema, key(method, path))
				}
			}
			if op.RequestBody != nil {
				for _, media := range op.RequestBody.Content {
					checkRef(t, doc, media.Schema, key(method, path))
				}
			}
		}
	}
	for name, schema := range doc.Components.Schemas {
		checkNestedRefs(t, doc, schema, "component "+name)
	}
}

// checkRef fails when a $ref names a component the document does not declare.
func checkRef(t *testing.T, doc *Document, schema map[string]any, where string) {
	t.Helper()
	raw, ok := schema["$ref"].(string)
	if !ok {
		checkNestedRefs(t, doc, schema, where)
		return
	}
	name := strings.TrimPrefix(raw, "#/components/schemas/")
	if _, ok := doc.Components.Schemas[name]; !ok {
		t.Errorf("%s references undeclared component %q", where, name)
	}
}

// checkNestedRefs walks a schema fragment looking for references.
func checkNestedRefs(t *testing.T, doc *Document, node any, where string) {
	t.Helper()
	switch value := node.(type) {
	case map[string]any:
		if _, ok := value["$ref"]; ok {
			checkRef(t, doc, value, where)
			return
		}
		for _, child := range value {
			checkNestedRefs(t, doc, child, where)
		}
	case []any:
		for _, child := range value {
			checkNestedRefs(t, doc, child, where)
		}
	}
}

func TestThePostmanCollectionCarriesEveryOperation(t *testing.T) {
	h := documentedTransport(t)
	routes := append(h.RegisteredRoutes(), MCPEndpointRoutes(DefaultMCPEndpoint)...)

	collection, err := BuildPostmanCollection(routes, "test")
	if err != nil {
		t.Fatalf("building the collection failed: %v", err)
	}

	requests := 0
	for _, folder := range collection.Item {
		for _, item := range folder.Item {
			if item.Request == nil {
				t.Errorf("collection item %q carries no request", item.Name)
				continue
			}
			requests++
			if item.Request.URL.Raw == "" {
				t.Errorf("request %q has no URL", item.Name)
			}
			// The MCP endpoint is on its own listener and must not be sent to
			// the JSON API base, which would return 404 in a review.
			wantBase := "{{base}}"
			if strings.HasPrefix(item.Request.URL.Raw, "{{mcpBase}}") {
				wantBase = "{{mcpBase}}"
			}
			if !strings.HasPrefix(item.Request.URL.Raw, wantBase) {
				t.Errorf("request %q does not use a base variable: %s", item.Name, item.Request.URL.Raw)
			}
		}
	}
	if requests != len(routes) {
		t.Errorf("the collection has %d requests for %d documented operations", requests, len(routes))
	}
	if collection.Info.Schema == "" {
		t.Error("the collection must name the Postman schema it follows")
	}
}

func TestDuplicateOperationIdsAreRefused(t *testing.T) {
	// The generator must fail rather than emit a document whose operations
	// collide, because a collision silently drops one of them.
	routes := []APIRoute{
		{Method: http.MethodGet, Path: "/a", OperationID: "same"},
		{Method: http.MethodGet, Path: "/b", OperationID: "same"},
	}
	if _, err := BuildDocument(routes, "test"); err == nil {
		t.Fatal("a duplicate operation id must be an error")
	}
}
