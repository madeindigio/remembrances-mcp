package modules

import (
	"context"
	"net/http"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	mcpserver "github.com/ThinkInAIXYZ/go-mcp/server"
	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/embedder"
)

// ToolHandler is the signature for a tool handler.
type ToolHandler = mcpserver.ToolHandlerFunc

// ToolDefinition bundles a tool definition with its handler.
type ToolDefinition struct {
	Tool    *protocol.Tool
	Handler ToolHandler
}

// ToolProvider adds MCP tools.
type ToolProvider interface {
	Module
	Tools() []ToolDefinition
}

// StorageProvider adds storage backends.
type StorageProvider interface {
	Module
	StorageType() string
	NewStorage(cfg map[string]any) (storage.FullStorage, error)
}

// EmbedderProvider adds embedding providers.
type EmbedderProvider interface {
	Module
	EmbedderType() string
	NewEmbedder(cfg map[string]any) (embedder.Embedder, error)
}

// ToolMiddleware intercepts tool calls.
type ToolMiddleware interface {
	Module
	Wrap(next ToolHandler) ToolHandler
	Priority() int
	ToolFilter() []string
}

// ToolTransformer transforms tool responses.
type ToolTransformer interface {
	Module
	Transform(ctx context.Context, req *protocol.CallToolRequest, res *protocol.CallToolResult) (*protocol.CallToolResult, error)
	ToolFilter() []string
}

// ToolValidator validates tool requests.
type ToolValidator interface {
	Module
	Validate(ctx context.Context, req *protocol.CallToolRequest) error
	ToolFilter() []string
}

// ResponseEnricher adds extra information to tool responses.
type ResponseEnricher interface {
	Module
	Enrich(ctx context.Context, req *protocol.CallToolRequest, res *protocol.CallToolResult) (*protocol.CallToolResult, error)
	ToolFilter() []string
}

// ConfigProvider loads configuration from additional sources.
type ConfigProvider interface {
	Module
	LoadConfig() (map[string]any, error)
}

// HTTPEndpointProvider exposes additional HTTP routes.
type HTTPEndpointProvider interface {
	Module
	Routes() []HTTPRoute
	BasePath() string
}

// HTTPRoute defines an HTTP route.
type HTTPRoute struct {
	Method      string
	Path        string
	Handler     http.HandlerFunc
	Middlewares []HTTPMiddleware
	Description string
}

// HTTPMiddleware is standard HTTP middleware.
type HTTPMiddleware func(http.Handler) http.Handler

// HTTPAuthProvider provides auth for HTTP endpoints.
type HTTPAuthProvider interface {
	Module
	Authenticate(r *http.Request) (*AuthInfo, error)
	Middleware() HTTPMiddleware
}

// AuthInfo contains user auth data.
type AuthInfo struct {
	UserID   string
	Roles    []string
	Metadata map[string]any
}

// WebhookHandler handles incoming webhooks.
type WebhookHandler interface {
	Module
	HandleWebhook(ctx context.Context, source string, payload []byte) error
	Validate(r *http.Request) error
	Sources() []string
}
