package storage

import (
	"context"
	"sync/atomic"

	"github.com/madeindigio/remembrances-mcp/pkg/principal"
)

// The identity of the core's own background work.
//
// RM-2 made a storage call without a principal an error once
// require_principal is on, and RM-10 put a principal on both request-borne
// entry points. What was left over is the work the core starts by itself: the
// knowledge-base watcher, the asynchronous code-indexing jobs, the code file
// watcher and the maintenance commands. None of them has a user, but every one
// of them has a tenant and a scope, so the answer is not "let them run
// unscoped" (that is exactly the hole require_principal closes) but "give them
// an explicit service principal, derived once and passed down".
//
// Two helpers cover every case:
//
//   - ServiceContext installs the service principal when, and only when, the
//     context has none. Work that genuinely starts from nowhere (a watcher
//     started at boot, a maintenance command) uses it.
//   - BackgroundContext additionally detaches cancellation and deadlines.
//     Work that a request starts but that outlives it (an indexing job, the
//     goroutine that catches up on outdated files) uses it, and keeps the
//     requesting principal instead of silently becoming the service.
//
// Neither ever replaces a principal already in the context. Dropping the
// caller's identity, as watcher_manager.ProcessOutdatedFiles used to by
// substituting context.Background(), is the defect these two exist to make
// impossible: TestNoBackgroundContextIsBuiltByHand in internal/indexer fails
// the build if a background context is assembled by hand again.

// ServicePrincipalProvider returns the principal the core's background work
// runs as. It is a function rather than a value because the answer depends on
// the deployment: a single-tenant OSS server derives it from the legacy org,
// while a commercial multi-tenant build (Alchemai Mem) mints a real service
// principal of the org that owns the process.
type ServicePrincipalProvider func() *principal.Principal

// serviceProvider holds the process-wide provider. It is package level for the
// same reason requirePrincipal is: it has to apply to every storage
// implementation in the process, including the wrappers modules install.
var serviceProvider atomic.Pointer[ServicePrincipalProvider]

// SetServicePrincipalProvider installs the process-wide provider used by
// ServicePrincipal. A nil provider restores the default, which is the local
// principal marked as a service.
//
// A module that turns require_principal on is expected to install one: without
// it the core's background work runs in the legacy org, which in a
// multi-tenant deployment is an org its ACL layer will refuse.
func SetServicePrincipalProvider(provider ServicePrincipalProvider) {
	if provider == nil {
		serviceProvider.Store(nil)
		return
	}
	serviceProvider.Store(&provider)
}

// ServicePrincipal returns the principal the core's own background operations
// run as.
//
// The default is the local principal of the legacy org marked IsService, which
// is deliberate: it is exactly the identity these operations already ran as
// through the legacy fallback, so turning require_principal on moves no row
// between scopes and a migrated datastore keeps serving the documents its
// watcher indexed. What changes is that the identity is now explicit and
// auditable rather than a fallback nobody chose.
//
// The result is a fresh value, safe for the caller to narrow.
func ServicePrincipal() *principal.Principal {
	if p := serviceProvider.Load(); p != nil {
		if sp := (*p)(); sp != nil {
			return sp.Clone()
		}
	}
	sp := LocalPrincipal()
	sp.IsService = true
	return sp
}

// ServiceContext returns ctx carrying the service principal.
//
// A principal already in the context wins outright: a request that reached
// this far with a real identity keeps it, and only work that has none at all
// becomes the service.
func ServiceContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := principal.FromContext(ctx); ok {
		return ctx
	}
	return principal.NewContext(ctx, ServicePrincipal())
}

// BackgroundContext derives the context for work that outlives the call that
// started it.
//
// Cancellation and deadlines are dropped, because the caller's request is
// about to return and the work must not die with it; the values, and above all
// the principal, are kept, because the work is still being done for whoever
// asked for it. When the starting context carried no principal, the service
// principal is installed, so the result is never unscoped.
//
// This is the replacement for `context.Background()` inside a goroutine: that
// spelling keeps the work alive and throws the identity away, which is the one
// combination that is always wrong.
func BackgroundContext(ctx context.Context) context.Context {
	if ctx == nil {
		return ServiceContext(context.Background())
	}
	return ServiceContext(context.WithoutCancel(ctx))
}
