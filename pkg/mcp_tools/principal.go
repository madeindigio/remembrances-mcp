package mcp_tools

import (
	"context"
	"sync/atomic"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/madeindigio/remembrances-mcp/pkg/principal"
)

// The principal on the MCP tool path (RM-10).
//
// The HTTP module routes get their principal from the authentication layer
// (see internal/transport.PrincipalMiddleware). MCP tools have no
// http.Request: a stdio server has no token at all, and a Streamable HTTP
// server hands the handler a context the transport built. Both therefore need
// the same thing, an installer the process configures once, or every tool call
// reaches storage with no identity and fails the moment require_principal is
// on.
//
// The resolver is a function rather than a fixed rule because the answer
// differs by deployment: the local single-user server runs as the local
// principal, while a multi-tenant server resolves the caller's token. The core
// supplies the first; a commercial module supplies the second.
//
// It is also a process-wide default rather than a constructor argument,
// because the tools are registered by the standard modules, each of which
// builds its own ToolManager in Provision. A resolver that had to be passed in
// would reach some tools and not others, which for an identity is the worst of
// both worlds.

// ToolPrincipalResolver returns the principal a tool call runs as. Returning
// (nil, nil) means "no identity for this call", which leaves the context
// untouched and lets the storage layer apply its own fail-closed rule.
type ToolPrincipalResolver func(ctx context.Context, req *protocol.CallToolRequest) (*principal.Principal, error)

// toolRegistrar is the registration callback shape every Register*With method
// takes. It is named here so the wrapper below has something to say.
type toolRegistrar func(string, *protocol.Tool, func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error

// defaultResolver holds the process-wide resolver. It is consulted per call
// rather than captured at registration, so a resolver installed after the
// modules registered their tools still takes effect.
var defaultResolver atomic.Pointer[ToolPrincipalResolver]

// SetDefaultPrincipalResolver installs the process-wide resolver used by every
// tool that has no manager-specific one. A nil resolver clears it, which
// restores the previous behaviour of leaving the context alone.
func SetDefaultPrincipalResolver(resolver ToolPrincipalResolver) {
	if resolver == nil {
		defaultResolver.Store(nil)
		return
	}
	defaultResolver.Store(&resolver)
}

// DefaultPrincipalResolver returns the process-wide resolver, or nil.
func DefaultPrincipalResolver() ToolPrincipalResolver {
	if r := defaultResolver.Load(); r != nil {
		return *r
	}
	return nil
}

// SetPrincipalResolver installs a resolver for the tools this manager
// registers, overriding the process-wide default.
func (tm *ToolManager) SetPrincipalResolver(resolver ToolPrincipalResolver) {
	tm.principalResolver = resolver
}

// withPrincipal decorates a registration callback so that every handler it
// registers runs with the resolved principal in its context.
//
// A principal already in the context is never replaced: a transport or module
// that resolved a real identity outranks the process-wide resolver. A resolver
// error fails the call rather than falling back, because a resolver that was
// configured and cannot answer is a broken deployment, not an anonymous one.
func (tm *ToolManager) withPrincipal(reg toolRegistrar) toolRegistrar {
	var own ToolPrincipalResolver
	if tm != nil {
		own = tm.principalResolver
	}
	return withPrincipalResolver(reg, own)
}

// withPrincipalResolver is the manager-independent half, so the code search
// and code manipulation managers install the same identity as the rest.
func withPrincipalResolver(reg toolRegistrar, own ToolPrincipalResolver) toolRegistrar {
	return func(name string, tool *protocol.Tool, handler func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error {
		wrapped := func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
			resolver := own
			if resolver == nil {
				resolver = DefaultPrincipalResolver()
			}
			if resolver == nil {
				return handler(ctx, req)
			}
			if _, ok := principal.FromContext(ctx); ok {
				return handler(ctx, req)
			}
			p, err := resolver(ctx, req)
			if err != nil {
				return nil, err
			}
			if p != nil {
				ctx = principal.NewContext(ctx, p)
			}
			return handler(ctx, req)
		}
		return reg(name, tool, wrapped)
	}
}
