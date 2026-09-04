package mcp_tools

import (
	"context"
	"net/http"
	"strings"
)

// The HTTP request behind an MCP tool call.
//
// go-mcp hands a tool handler a context and the decoded call, and nothing
// else: neither the http.Request nor its headers. On a networked transport
// that is the whole identity of the caller, so a resolver installed with
// SetDefaultPrincipalResolver could see a bearer token only if something put
// it in the context first. This type is that something.
//
// It carries a copy of the metadata rather than the *http.Request because the
// body has already been consumed by the transport by the time a tool runs, and
// because a resolver has no business writing to the response.

// RequestInfo is the HTTP metadata of an MCP call that arrived over a
// networked transport. It is absent on stdio, where there is no request.
type RequestInfo struct {
	// Header is a copy of the request headers, so a resolver may read the
	// Authorization header (or any deployment-specific one) without holding
	// the request alive.
	Header http.Header

	// Method and Path are the HTTP method and the endpoint path the call
	// arrived on.
	Method string
	Path   string

	// RemoteAddr is the client address as the server saw it, for audit.
	RemoteAddr string

	// TLS reports whether the connection reaching this process was encrypted.
	// A proxy terminating TLS makes this false, which is why it is a fact
	// about the connection and not an authorisation input.
	TLS bool
}

// requestKey is unexported so no other package can put a value under this key.
type requestKey struct{}

// NewRequestInfo builds the carried metadata from an incoming request.
func NewRequestInfo(r *http.Request) *RequestInfo {
	if r == nil {
		return nil
	}
	info := &RequestInfo{
		Header:     r.Header.Clone(),
		Method:     r.Method,
		RemoteAddr: r.RemoteAddr,
		TLS:        r.TLS != nil,
	}
	if r.URL != nil {
		info.Path = r.URL.Path
	}
	return info
}

// NewRequestContext returns a copy of ctx carrying info.
func NewRequestContext(ctx context.Context, info *RequestInfo) context.Context {
	if info == nil {
		return ctx
	}
	return context.WithValue(ctx, requestKey{}, info)
}

// RequestFromContext returns the HTTP metadata of the MCP call, if the
// transport carried any.
func RequestFromContext(ctx context.Context) (*RequestInfo, bool) {
	if ctx == nil {
		return nil, false
	}
	info, ok := ctx.Value(requestKey{}).(*RequestInfo)
	if !ok || info == nil {
		return nil, false
	}
	return info, true
}

// BearerToken returns the token of an "Authorization: Bearer …" header, or the
// empty string. The scheme comparison is case insensitive because RFC 7235
// says it is.
func (i *RequestInfo) BearerToken() string {
	if i == nil || i.Header == nil {
		return ""
	}
	value := strings.TrimSpace(i.Header.Get("Authorization"))
	const prefix = "bearer "
	if len(value) <= len(prefix) || !strings.EqualFold(value[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(value[len(prefix):])
}
