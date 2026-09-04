// Package modules implements the Remembrances module system: a registry of
// modules, a manager that loads, starts, stops and unloads them, and the
// provider interfaces through which a module extends the server.
//
// # Module life cycle
//
// A module goes through Provision (configuration), Validate (self check),
// Start (activation, see Lifecycle), Stop (deactivation) and Cleanup
// (resource release). Every step is optional: a module implements only the
// interfaces it needs. The manager loads modules in the order the host asks
// for them, starts them in that same order and stops them in reverse.
//
// # Middleware ordering
//
// There are two independent middleware paths, one per surface, and both put
// authentication first so that the principal is known before any module code
// runs:
//
// HTTP routes, applied by the transport when it registers the routes of an
// HTTPEndpointProvider. From outermost to innermost:
//
//  1. method dispatch: a request whose method does not match HTTPRoute.Method
//     is answered with 405 and never reaches auth or the handler. An empty
//     Method keeps the historical behaviour of matching every method;
//  2. the authentication layer built from the loaded HTTPAuthProvider modules
//     (AuthMiddleware): provider middlewares in load order, then Authenticate,
//     which stores the AuthInfo in the request context;
//  3. HTTPRoute.Middlewares in declared order, index 0 outermost;
//  4. HTTPRoute.Handler.
//
// MCP tools, applied by MiddlewareChain.Wrap. From outermost to innermost:
//
//  1. ToolValidator, in registration order, rejecting the call before any
//     middleware runs;
//  2. ToolMiddleware, sorted by ascending Priority, lowest priority number
//     outermost;
//  3. the tool handler;
//  4. on the way back, ToolTransformer then ResponseEnricher, both in
//     registration order.
//
// The two orders mirror each other: the check that decides whether the call
// happens at all is outermost, the module supplied wrappers sit in between,
// and response shaping happens innermost or on the way out. A module that
// authenticates HTTP requests with an HTTPAuthProvider and guards MCP tools
// with a ToolMiddleware therefore sees the same principal on both surfaces,
// provided it propagates the AuthInfo itself: the MCP transport carries no
// http.Request, so ContextWithAuthInfo is the only bridge between them.
package modules
