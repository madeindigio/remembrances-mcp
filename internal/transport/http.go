// Package transport contains custom HTTP transport implementation for MCP servers.
package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	mcpserver "github.com/ThinkInAIXYZ/go-mcp/server"
	"github.com/madeindigio/remembrances-mcp/pkg/modules"
)

const (
	contentTypeJSON   = "application/json"
	headerContentType = "Content-Type"
)

// HTTPTransport implements a simple HTTP JSON API transport for MCP
type HTTPTransport struct {
	addr   string
	server *http.Server
	mux    *http.ServeMux

	// moduleRoutes maps a full mux path to the module routes registered on
	// it, one per HTTP method. The mux gets a single dispatcher per path.
	moduleRoutes map[string][]moduleRouteEntry
	// routes is the documented view of everything registered above: one entry
	// per operation the transport serves. It is written by the registration
	// itself, which is what keeps the OpenAPI document from drifting (RM-5).
	routes         []APIRoute
	moduleRoutesMu sync.RWMutex
}

// NewHTTPTransport creates a new HTTP transport for MCP server
func NewHTTPTransport(addr string) *HTTPTransport {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	transport := &HTTPTransport{
		addr:   addr,
		server: server,
		mux:    mux,
	}

	return transport
}

// coreEndpoint binds one mux pattern to the handler that serves it and to the
// operations that handler answers. Registering and documenting from the same
// value is what keeps the two in step: a handler added without an operation,
// or an operation without a handler, does not compile into this table.
type coreEndpoint struct {
	path       string
	handler    func(h *HTTPTransport, mcpServer *mcpserver.Server) http.HandlerFunc
	operations []APIRoute
}

// coreEndpoints is the HTTP JSON API the core transport serves itself. It is
// the single source for both the mux registration and the API document.
//
// There is deliberately no tool endpoint here. `GET /mcp/tools` and
// `POST /mcp/tools/call` existed and were documented as placeholders: they
// answered with a hard coded tool and a "(placeholder response)" string and
// never reached the MCP tool registry, which go-mcp does not expose to a
// transport in the first place. A documented placeholder is a promise, so they
// were removed rather than left to be discovered by a caller in production
// (defect 12 of the v1 list). The real tool surface is the MCP Streamable HTTP
// endpoint, which authenticates with the module auth providers; these two did
// not authenticate at all, so implementing them would have published every
// tool in the process, unauthenticated, on the JSON API port. Rebuilding them
// needs that authentication layer and a registry the core owns, and that is a
// work item, not a handler.
func coreEndpoints() []coreEndpoint {
	return []coreEndpoint{
		{
			path: "/health",
			handler: func(h *HTTPTransport, _ *mcpserver.Server) http.HandlerFunc {
				return h.handleHealth
			},
			operations: []APIRoute{{
				Method:         http.MethodGet,
				Path:           "/health",
				MuxPath:        "/health",
				OperationID:    "getHealth",
				Summary:        "Liveness probe",
				Description:    "Returns 200 with {\"status\":\"ok\"} while the process is serving. It does not check the datastore, so it is a liveness probe and not a readiness one.",
				Tag:            "core",
				Source:         SourceCore,
				ResponseSchema: "HealthResponse",
			}},
		},
	}
}

// SetupMCPRoutes configures the HTTP routes for MCP protocol. It walks
// coreEndpoints, so the mux and the API document are built from one table.
func (h *HTTPTransport) SetupMCPRoutes(mcpServer *mcpserver.Server) error {
	for _, endpoint := range coreEndpoints() {
		h.mux.HandleFunc(endpoint.path, endpoint.handler(h, mcpServer))
		h.recordRoutes(endpoint.operations...)
	}
	return nil
}

func (h *HTTPTransport) handleHealth(w http.ResponseWriter, r *http.Request) {
	// The document says this path answers GET, so the handler answers GET.
	// It used to reply 200 to any method, including POST and DELETE, which the
	// route parity test found the moment the document was generated.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		slog.Error("failed to encode health response", "error", err)
	}
}

// Start starts the HTTP server
func (h *HTTPTransport) Start() error {
	slog.Info("Starting HTTP transport server", "address", h.addr)
	return h.server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server
func (h *HTTPTransport) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down HTTP transport server")
	return h.server.Shutdown(ctx)
}

// CreateHTTPServerTransport creates a custom HTTP transport that integrates with go-mcp
func CreateHTTPServerTransport(addr string, mcpServer *mcpserver.Server) (*HTTPTransport, error) {
	transport := NewHTTPTransport(addr)
	if err := transport.SetupMCPRoutes(mcpServer); err != nil {
		return nil, fmt.Errorf("failed to setup MCP routes: %w", err)
	}
	return transport, nil
}

// moduleRouteEntry is one registered module route: the HTTP method it answers
// and the fully wrapped handler (authentication layer plus the route
// middlewares around the module handler).
type moduleRouteEntry struct {
	// method is the upper case HTTP method, or "" when the route declares no
	// method and therefore answers every method, as it did before RM-4.
	method  string
	handler http.Handler
}

// RegisterModuleRoutes registers HTTP routes from module providers.
//
// Every route is wrapped, from outermost to innermost, as:
//
//  1. method dispatch, which answers 405 with an Allow header when the
//     request method does not match any route registered on the path;
//  2. the authentication layer built from authProviders, which populates the
//     AuthInfo in the request context and fails closed (see
//     modules.AuthMiddleware). With no auth providers this layer is the
//     identity;
//  3. the principal layer, which turns that AuthInfo into the storage
//     principal the tenancy predicates read (see PrincipalMiddleware). It runs
//     inside the authentication layer because it consumes its result, and
//     outside the route middlewares because a route middleware that touches
//     storage must already run as the caller;
//  4. route.Middlewares in declared order, index 0 outermost;
//  5. route.Handler.
//
// authProviders is variadic so existing callers that pass only the endpoint
// providers keep compiling and keep their previous, unauthenticated
// behaviour.
func (h *HTTPTransport) RegisterModuleRoutes(providers []modules.HTTPEndpointProvider, authProviders ...modules.HTTPAuthProvider) {
	auth := modules.AuthMiddleware(authProviders, slog.Default())
	installPrincipal := PrincipalMiddleware(slog.Default())

	for _, provider := range providers {
		basePath := provider.BasePath()
		routes := provider.Routes()

		slog.Info("Registering module HTTP routes", "base_path", basePath, "routes", len(routes))

		for _, route := range routes {
			if route.Handler == nil {
				slog.Warn("Skipping module route with nil handler", "base_path", basePath, "path", route.Path)
				continue
			}

			fullPath := basePath + route.Path
			// Handle wildcard paths for static files: the mux subtree pattern
			// keeps its trailing slash.
			if strings.HasSuffix(route.Path, "/*") {
				fullPath = strings.TrimSuffix(fullPath, "/*") + "/"
			}

			handler := modules.ChainHTTPMiddlewares(http.Handler(route.Handler), route.Middlewares)
			handler = installPrincipal(handler)
			handler = auth(handler)

			// The documentation is written by the registration, so a module
			// route cannot exist undocumented (RM-5).
			doc := moduleRouteDoc(string(provider.ModuleInfo().ID), basePath, route.Path,
				route.Method, route.Description, len(authProviders) > 0)

			h.addModuleRoute(fullPath, strings.ToUpper(strings.TrimSpace(route.Method)), handler, doc)
			slog.Debug("Registered module route", "method", route.Method, "path", fullPath)
		}
	}
}

// addModuleRoute stores one entry in the module route table and, the first
// time a path is used, installs the method dispatcher for it on the mux.
// Registering the dispatcher once per path is what allows several methods to
// share a path without the mux rejecting the duplicate pattern.
func (h *HTTPTransport) addModuleRoute(path, method string, handler http.Handler, doc APIRoute) {
	h.moduleRoutesMu.Lock()
	defer h.moduleRoutesMu.Unlock()

	if h.moduleRoutes == nil {
		h.moduleRoutes = make(map[string][]moduleRouteEntry)
	}

	for _, existing := range h.moduleRoutes[path] {
		if existing.method == method {
			slog.Warn("Duplicate module route ignored", "method", method, "path", path)
			return
		}
	}

	_, known := h.moduleRoutes[path]
	h.moduleRoutes[path] = append(h.moduleRoutes[path], moduleRouteEntry{method: method, handler: handler})
	h.routes = append(h.routes, doc)
	if !known {
		h.mux.Handle(path, h.moduleRouteDispatcher(path))
	}
}

// moduleRouteDispatcher returns the handler registered on the mux for a path.
// It selects the entry matching the request method, falls back to an entry
// that declares no method, and otherwise answers 405 with an Allow header.
func (h *HTTPTransport) moduleRouteDispatcher(path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.moduleRoutesMu.RLock()
		entries := h.moduleRoutes[path]
		h.moduleRoutesMu.RUnlock()

		var fallback http.Handler
		allowed := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.method == "" {
				if fallback == nil {
					fallback = entry.handler
				}
				continue
			}
			allowed = append(allowed, entry.method)
			if entry.method == strings.ToUpper(r.Method) {
				entry.handler.ServeHTTP(w, r)
				return
			}
		}

		if fallback != nil {
			fallback.ServeHTTP(w, r)
			return
		}

		if len(allowed) > 0 {
			w.Header().Set("Allow", strings.Join(allowed, ", "))
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
}

// Handler exposes the transport mux so tests and embedders can drive the
// registered routes without binding a socket.
func (h *HTTPTransport) Handler() http.Handler {
	return h.mux
}
