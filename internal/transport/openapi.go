package transport

import (
	"fmt"
	"sort"
	"strings"
)

// The API description (RM-5).
//
// The rule this file exists to enforce is that the document describes what the
// process serves, not what a plan hoped for. A hand-written specification rots
// the first time a module adds a route, and a stale specification is worse
// than none because a reader trusts it.
//
// So the route table is the registration: SetupMCPRoutes registers the core
// handlers by walking coreEndpoints, and every module route recorded in the
// dispatcher table records its documentation at the same time.
// RegisteredRoutes then reports what the transport is actually serving, the
// generator turns that into the document, and the tests fail when a served
// route has no operation or an operation has no route.
//
// Regenerate with:
//
//	go generate ./internal/transport/...
//
//go:generate go run ../../cmd/openapi-gen -out ../../docs/api

// DefaultMCPEndpoint is the path the MCP Streamable HTTP transport listens on
// when no endpoint is configured. It lives here so the document and the server
// cannot disagree about it.
const DefaultMCPEndpoint = "/mcp"

// APIVersion is the version of the HTTP surface this document describes. It is
// deliberately independent of the binary version, which release builds inject
// with -ldflags: the document describes a contract, and the contract does not
// change because a patch release was cut. Bump it when the surface changes.
const APIVersion = "1.0.0"

// SourceCore names the routes the core transport serves itself, as opposed to
// the module id a module route carries.
const SourceCore = "core"

// APIParameter is one path or query parameter of an operation.
type APIParameter struct {
	Name        string
	In          string // "path" or "query"
	Required    bool
	Description string
	Schema      string // JSON Schema type name, for example "string"
}

// APIRoute is one documented HTTP operation. It is produced by the code that
// registers the handler, never written by hand next to it.
type APIRoute struct {
	// Method is the upper case HTTP method.
	Method string
	// Path is the OpenAPI path, with {name} placeholders where the mux uses a
	// subtree pattern.
	Path string
	// MuxPath is the pattern the route is registered under, empty for a route
	// served by an embedded transport rather than by this mux.
	MuxPath string
	// OperationID is the stable name of the operation.
	OperationID string
	Summary     string
	Description string
	Tag         string
	// Source is SourceCore or the id of the module that provided the route.
	Source string
	// External marks a route served by an embedded library (the MCP
	// Streamable HTTP transport). It is documented because clients call it,
	// and excluded from the mux parity checks because this mux never sees it.
	External bool
	// Authenticated marks a route behind the module authentication layer.
	Authenticated bool
	Parameters    []APIParameter
	// RequestSchema and ResponseSchema name components of the document; an
	// empty ResponseSchema documents a free-form JSON object.
	RequestSchema  string
	ResponseSchema string
}

// MCPEndpointRoutes describes the MCP Streamable HTTP endpoint, which the
// go-mcp transport serves on its own listener. The operations are the three
// the MCP Streamable HTTP binding defines.
func MCPEndpointRoutes(endpoint string) []APIRoute {
	if endpoint == "" {
		endpoint = DefaultMCPEndpoint
	}
	return []APIRoute{
		{
			Method:      "POST",
			Path:        endpoint,
			OperationID: "mcpSendMessage",
			Summary:     "Send a JSON-RPC message to the MCP server",
			Description: "The MCP Streamable HTTP binding. The body is a JSON-RPC 2.0 request; the response is either a JSON-RPC response or an SSE stream, depending on the Accept header. Served by the embedded go-mcp transport on the --mcp-http listener, not by the HTTP JSON API listener.",
			Tag:         "mcp",
			Source:      SourceCore,
			External:    true,
		},
		{
			Method:      "GET",
			Path:        endpoint,
			OperationID: "mcpOpenStream",
			Summary:     "Open the server to client SSE stream",
			Description: "Opens the server-initiated event stream of a stateful MCP session. Requires the Mcp-Session-Id header returned by initialize.",
			Tag:         "mcp",
			Source:      SourceCore,
			External:    true,
		},
		{
			Method:      "DELETE",
			Path:        endpoint,
			OperationID: "mcpEndSession",
			Summary:     "End an MCP session",
			Description: "Terminates the session named by the Mcp-Session-Id header.",
			Tag:         "mcp",
			Source:      SourceCore,
			External:    true,
		},
	}
}

// moduleRouteDoc turns a module's HTTPRoute declaration into a documented
// operation. The base path and the wildcard convention are the transport's, so
// the translation belongs here rather than in every module.
func moduleRouteDoc(moduleID, basePath, routePath, method, description string, authenticated bool) APIRoute {
	muxPath := basePath + routePath
	docPath := muxPath
	var params []APIParameter
	if strings.HasSuffix(routePath, "/*") {
		// The mux serves the subtree; the document names the remainder.
		muxPath = strings.TrimSuffix(muxPath, "/*") + "/"
		docPath = muxPath + "{path}"
		params = append(params, APIParameter{
			Name:        "path",
			In:          "path",
			Required:    true,
			Description: "Path of the file below " + muxPath,
			Schema:      "string",
		})
	}

	summary := description
	if summary == "" {
		summary = strings.ToUpper(method) + " " + docPath
	}

	return APIRoute{
		Method:        strings.ToUpper(strings.TrimSpace(method)),
		Path:          docPath,
		MuxPath:       muxPath,
		OperationID:   operationID(method, docPath),
		Summary:       summary,
		Description:   description,
		Tag:           moduleTag(moduleID),
		Source:        moduleID,
		Authenticated: authenticated,
		Parameters:    params,
	}
}

// moduleTag groups a module's operations in the document.
func moduleTag(moduleID string) string {
	if moduleID == "" {
		return "modules"
	}
	return moduleID
}

// operationID derives a stable, unique operation id from a method and a path.
// It is deterministic so the generated document does not churn between runs.
func operationID(method, path string) string {
	parts := strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '{' || r == '}' || r == '-' || r == '.'
	})
	id := strings.ToLower(strings.TrimSpace(method))
	if id == "" {
		id = "any"
	}
	for _, part := range parts {
		if part == "" {
			continue
		}
		id += strings.ToUpper(part[:1]) + part[1:]
	}
	if len(parts) == 0 {
		id += "Root"
	}
	return id
}

// RegisteredRoutes returns every operation this transport serves, core and
// module, sorted by path and method. It reads the same tables the mux answers
// from, so it cannot describe a route the transport does not serve.
func (h *HTTPTransport) RegisteredRoutes() []APIRoute {
	h.moduleRoutesMu.RLock()
	routes := make([]APIRoute, len(h.routes))
	copy(routes, h.routes)
	h.moduleRoutesMu.RUnlock()

	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path != routes[j].Path {
			return routes[i].Path < routes[j].Path
		}
		return routes[i].Method < routes[j].Method
	})
	return routes
}

// recordRoutes adds operations to the documented set.
func (h *HTTPTransport) recordRoutes(routes ...APIRoute) {
	h.moduleRoutesMu.Lock()
	defer h.moduleRoutesMu.Unlock()
	h.routes = append(h.routes, routes...)
}

// DocumentedPaths reports the documented operations grouped by path, which is
// the shape the OpenAPI document needs.
func DocumentedPaths(routes []APIRoute) map[string][]APIRoute {
	byPath := make(map[string][]APIRoute, len(routes))
	for _, route := range routes {
		byPath[route.Path] = append(byPath[route.Path], route)
	}
	for path := range byPath {
		ops := byPath[path]
		sort.Slice(ops, func(i, j int) bool { return ops[i].Method < ops[j].Method })
		byPath[path] = ops
	}
	return byPath
}

// validateRoutes reports the inconsistencies that make a document dishonest: a
// duplicate operation id, an operation with no method, or a path that is not
// absolute. It is called by the generator so a broken table fails the build
// rather than producing a plausible looking document.
func validateRoutes(routes []APIRoute) error {
	seen := make(map[string]string, len(routes))
	for _, route := range routes {
		if route.Method == "" {
			return fmt.Errorf("route %q has no method", route.Path)
		}
		if !strings.HasPrefix(route.Path, "/") {
			return fmt.Errorf("route %q is not an absolute path", route.Path)
		}
		if route.OperationID == "" {
			return fmt.Errorf("route %s %s has no operation id", route.Method, route.Path)
		}
		if previous, ok := seen[route.OperationID]; ok {
			return fmt.Errorf("duplicate operation id %q on %s %s and %s", route.OperationID, route.Method, route.Path, previous)
		}
		seen[route.OperationID] = route.Method + " " + route.Path
	}
	return nil
}
