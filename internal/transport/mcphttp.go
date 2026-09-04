package transport

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	mcppkg "github.com/ThinkInAIXYZ/go-mcp/pkg"
	mcptransport "github.com/ThinkInAIXYZ/go-mcp/transport"

	"github.com/madeindigio/remembrances-mcp/pkg/mcp_tools"
	"github.com/madeindigio/remembrances-mcp/pkg/modules"
)

// The MCP Streamable HTTP endpoint, served so that the caller's identity
// reaches the tool handlers (RM-10, MM-1).
//
// The gap this closes: go-mcp's own NewStreamableHTTPServerTransport owns its
// http.Server, so nothing can run before its handler, and a tool handler
// therefore sees a context with no request, no headers and no bearer token. A
// resolver installed with mcp_tools.SetDefaultPrincipalResolver had nothing to
// resolve from, which is why "MCP Streamable HTTP" was the one entry point
// require_principal could not be turned on for.
//
// It is closed without touching go-mcp. The library also exposes
// NewStreamableHTTPServerTransportAndHandler, which returns the transport and
// an http.Handler and leaves the server to us; and its dispatch passes
// r.Context() down to the tool handler (through a cancel-shielding wrapper
// that keeps the values). So a middleware in front of that handler is all that
// is needed: whatever it puts in the request context, a tool sees.
//
// Three things go in, in this order:
//
//  1. the module authentication layer, exactly the one module HTTP routes get,
//     so a deployment that authenticates its API authenticates its MCP
//     endpoint with the same providers and the same fail-closed rule;
//  2. the principal layer, which turns the resulting AuthInfo into the storage
//     principal (see PrincipalMiddleware); and
//  3. the request metadata, so a module resolver can map a token the core does
//     not understand (mcp_tools.RequestFromContext).
//
// A build with no auth providers is unchanged: the auth layer is the identity
// and the tools resolve their principal however they did before.

// MCPHTTPServer serves the MCP Streamable HTTP endpoint on an http.Server this
// package owns, with the middleware chain above in front of the transport.
type MCPHTTPServer struct {
	transport mcptransport.ServerTransport
	endpoint  string
	addr      string
	srv       *http.Server
	logger    *slog.Logger

	// inner is the chain below the authentication layer; chain is the whole
	// thing. The authentication layer is replaceable because the auth
	// providers are only known once the modules have loaded, which is after
	// the transport has to exist.
	inner http.Handler
	chain atomic.Pointer[http.Handler]
}

// NewMCPHTTPServer builds the transport and the server that fronts it.
//
// addr and endpoint are the listen address and the single MCP endpoint path;
// stateMode and logger are passed through to go-mcp unchanged.
func NewMCPHTTPServer(addr, endpoint string, stateMode mcptransport.StateMode, logger mcppkg.Logger) (*MCPHTTPServer, error) {
	if endpoint == "" {
		endpoint = DefaultMCPEndpoint
	}

	opts := []mcptransport.StreamableHTTPServerTransportAndHandlerOption{
		mcptransport.WithStreamableHTTPServerTransportAndHandlerOptionStateMode(stateMode),
	}
	if logger != nil {
		opts = append(opts, mcptransport.WithStreamableHTTPServerTransportAndHandlerOptionLogger(logger))
	}

	t, handler, err := mcptransport.NewStreamableHTTPServerTransportAndHandler(opts...)
	if err != nil {
		return nil, err
	}

	s := &MCPHTTPServer{
		transport: t,
		endpoint:  endpoint,
		addr:      addr,
		logger:    slog.Default(),
	}
	s.inner = mcpRequestMiddleware()(PrincipalMiddleware(s.logger)(handler.HandleMCP()))
	s.SetAuthProviders(nil)

	mux := http.NewServeMux()
	mux.Handle(endpoint, s)
	s.srv = &http.Server{
		Addr:        addr,
		Handler:     mux,
		IdleTimeout: time.Minute,
	}
	return s, nil
}

// Transport returns the go-mcp transport to hand to mcpserver.NewServer.
func (s *MCPHTTPServer) Transport() mcptransport.ServerTransport { return s.transport }

// Endpoint returns the path the endpoint is served on.
func (s *MCPHTTPServer) Endpoint() string { return s.endpoint }

// Addr returns the listen address.
func (s *MCPHTTPServer) Addr() string { return s.addr }

// SetAuthProviders installs the module authentication layer. It may be called
// once the modules have loaded and before Serve; with no providers the layer
// is the identity, which leaves an unauthenticated deployment exactly as it
// was.
func (s *MCPHTTPServer) SetAuthProviders(providers []modules.HTTPAuthProvider) {
	h := modules.AuthMiddleware(providers, s.logger)(s.inner)
	s.chain.Store(&h)
}

// ServeHTTP dispatches through the current chain.
func (s *MCPHTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h := s.chain.Load(); h != nil {
		(*h).ServeHTTP(w, r)
		return
	}
	s.inner.ServeHTTP(w, r)
}

// Serve runs the HTTP server and blocks until it is shut down. It returns
// http.ErrServerClosed on a clean shutdown, like http.Server does.
func (s *MCPHTTPServer) Serve() error {
	s.logger.Info("MCP Streamable HTTP endpoint listening", "address", s.addr, "endpoint", s.endpoint)
	return s.srv.ListenAndServe()
}

// Shutdown stops the HTTP server. The MCP transport is shut down separately by
// the MCP server that owns it.
func (s *MCPHTTPServer) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

// mcpRequestMiddleware puts the request metadata in the context so a tool
// principal resolver can read the bearer token the transport would otherwise
// hide. It is the innermost of the three layers: it must not run before the
// authentication layer has had its chance to reject the request.
func mcpRequestMiddleware() modules.HTTPMiddleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := mcp_tools.NewRequestContext(r.Context(), mcp_tools.NewRequestInfo(r))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
