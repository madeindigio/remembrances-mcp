package modules

import (
	"context"
	"log/slog"
	"net/http"
)

// authInfoContextKey is the private key under which the authenticated
// principal is stored in the request context.
type authInfoContextKey struct{}

// ContextWithAuthInfo returns a copy of ctx carrying info as the
// authenticated principal of the request.
func ContextWithAuthInfo(ctx context.Context, info *AuthInfo) context.Context {
	return context.WithValue(ctx, authInfoContextKey{}, info)
}

// AuthInfoFromContext returns the principal an HTTPAuthProvider produced for
// the request, if any. Handlers on routes that are not behind an auth
// provider get ok == false.
func AuthInfoFromContext(ctx context.Context) (*AuthInfo, bool) {
	info, ok := ctx.Value(authInfoContextKey{}).(*AuthInfo)
	if !ok || info == nil {
		return nil, false
	}
	return info, true
}

// AuthMiddleware builds the HTTP authentication layer for module routes out
// of the HTTPAuthProvider modules the manager loaded.
//
// The layer is fail closed. For each request the providers are consulted in
// load order and:
//
//   - a provider returning a non-nil *AuthInfo and a nil error authenticates
//     the request; the principal is put in the request context (see
//     AuthInfoFromContext) and no further provider is consulted;
//   - a provider returning an error rejects the request with 401 immediately.
//     Later providers are not consulted, so a strict provider can never be
//     bypassed by a laxer one registered after it;
//   - a provider returning (nil, nil) abstains and the next provider is
//     tried;
//   - if every provider abstains the request is rejected with 401, because a
//     route that is meant to be authenticated must never be served
//     anonymously.
//
// Each provider may also contribute a classic net/http middleware through
// Middleware(). Those wrap the Authenticate step from the outside, in load
// order, so the first provider's middleware is the outermost.
//
// With no providers the returned middleware is the identity, which is what
// keeps modules that declare no auth behaving exactly as before.
func AuthMiddleware(providers []HTTPAuthProvider, logger *slog.Logger) HTTPMiddleware {
	if len(providers) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, provider := range providers {
				info, err := provider.Authenticate(r)
				if err != nil {
					if logger != nil {
						logger.Warn("Module auth provider rejected request",
							"module_id", provider.ModuleInfo().ID,
							"path", r.URL.Path,
							"error", err)
					}
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				if info != nil {
					next.ServeHTTP(w, r.WithContext(ContextWithAuthInfo(r.Context(), info)))
					return
				}
			}
			// Every provider abstained: fail closed.
			if logger != nil {
				logger.Warn("No module auth provider authenticated the request", "path", r.URL.Path)
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		})

		for i := len(providers) - 1; i >= 0; i-- {
			if mw := providers[i].Middleware(); mw != nil {
				handler = mw(handler)
			}
		}
		return handler
	}
}

// ChainHTTPMiddlewares wraps handler with the given middlewares so that the
// first middleware of the slice is the outermost one: for
// [a, b, c] the request travels a -> b -> c -> handler and the response
// travels back the other way. A nil middleware is skipped.
func ChainHTTPMiddlewares(handler http.Handler, middlewares []HTTPMiddleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		if middlewares[i] == nil {
			continue
		}
		handler = middlewares[i](handler)
	}
	return handler
}
