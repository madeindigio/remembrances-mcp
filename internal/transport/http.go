// Package transport contains custom HTTP transport implementation for MCP servers.
package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	mcpserver "github.com/ThinkInAIXYZ/go-mcp/server"
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
