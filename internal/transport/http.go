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

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	mcpserver "github.com/ThinkInAIXYZ/go-mcp/server"
	"github.com/madeindigio/remembrances-mcp/pkg/modules"
)

const (
	contentTypeJSON   = "application/json"
	headerContentType = "Content-Type"
	headerCORSOrigin  = "Access-Control-Allow-Origin"
	headerCORSMethods = "Access-Control-Allow-Methods"
	headerCORSHeaders = "Access-Control-Allow-Headers"
	corsMethods       = "GET, POST, OPTIONS"
	corsOrigin        = "*"
	corsHeaders       = "Content-Type"
)

// HTTPTransport implements a simple HTTP JSON API transport for MCP
type HTTPTransport struct {
	addr   string
	server *http.Server
	mux    *http.ServeMux

	// moduleRoutes maps a full mux path to the module routes registered on
	// it, one per HTTP method. The mux gets a single dispatcher per path.
	moduleRoutes   map[string][]moduleRouteEntry
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

// SetupMCPRoutes configures the HTTP routes for MCP protocol
func (h *HTTPTransport) SetupMCPRoutes(mcpServer *mcpserver.Server) error {
	h.mux.HandleFunc("/health", h.handleHealth)
	h.mux.HandleFunc("/mcp/tools", h.handleListTools(mcpServer))
	h.mux.HandleFunc("/mcp/tools/call", h.handleCallTool(mcpServer))
	return nil
}

func (h *HTTPTransport) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *HTTPTransport) handleListTools(mcpServer *mcpserver.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		h.setCORSHeaders(w)
		w.Header().Set(headerContentType, contentTypeJSON)

		// For now, we'll return a basic tools list since we can't directly access the server's tools
		// In a real implementation, you'd need to expose the tools through the server interface
		response := map[string]interface{}{
			"tools": []map[string]interface{}{
				{
					"name":        "list_available_tools",
					"description": "List all available MCP tools",
					"inputSchema": map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
					},
				},
			},
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			slog.Error("failed to encode tools response", "error", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

func (h *HTTPTransport) handleCallTool(mcpServer *mcpserver.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			h.setCORSHeaders(w)
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		h.setCORSHeaders(w)
		w.Header().Set(headerContentType, contentTypeJSON)

		var callReq protocol.CallToolRequest
		if err := json.NewDecoder(r.Body).Decode(&callReq); err != nil {
			slog.Error("failed to decode tool call request", "error", err)
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		// For now, we'll return a placeholder response since direct access to tool handlers
		// requires a different approach. This would need to be integrated with the MCP server's
		// tool registry.
		response := map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("HTTP transport called tool: %s (placeholder response)", callReq.Name),
				},
			},
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			slog.Error("failed to encode tool call response", "error", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

func (h *HTTPTransport) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set(headerCORSOrigin, corsOrigin)
	w.Header().Set(headerCORSMethods, corsMethods)
	w.Header().Set(headerCORSHeaders, corsHeaders)
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
//  3. route.Middlewares in declared order, index 0 outermost;
//  4. route.Handler.
//
// authProviders is variadic so existing callers that pass only the endpoint
// providers keep compiling and keep their previous, unauthenticated
// behaviour.
func (h *HTTPTransport) RegisterModuleRoutes(providers []modules.HTTPEndpointProvider, authProviders ...modules.HTTPAuthProvider) {
	auth := modules.AuthMiddleware(authProviders, slog.Default())

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
			handler = auth(handler)

			h.addModuleRoute(fullPath, strings.ToUpper(strings.TrimSpace(route.Method)), handler)
			slog.Debug("Registered module route", "method", route.Method, "path", fullPath)
		}
	}
}

// addModuleRoute stores one entry in the module route table and, the first
// time a path is used, installs the method dispatcher for it on the mux.
// Registering the dispatcher once per path is what allows several methods to
// share a path without the mux rejecting the duplicate pattern.
func (h *HTTPTransport) addModuleRoute(path, method string, handler http.Handler) {
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
