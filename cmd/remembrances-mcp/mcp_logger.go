package main

import (
	"strings"

	mcppkg "github.com/ThinkInAIXYZ/go-mcp/pkg"
)

// streamableHTTPLogger returns a go-mcp Logger that downgrades some common client-side
// session handshake errors (HTTP 400) from Error -> Info.
//
// Why: In Stateful Streamable HTTP mode, a client must first POST an initialize request,
// then reuse the returned Mcp-Session-Id header for subsequent requests. If a client
// reconnects with a stale session ID (or probes the endpoint incorrectly), the upstream
// transport returns 400 (e.g. "lack session"). Those are expected client errors and
// shouldn't look like server failures in logs.
func streamableHTTPLogger() mcppkg.Logger {
	return &filteredMCPLogger{base: mcppkg.DefaultLogger}
}

type filteredMCPLogger struct {
	base mcppkg.Logger
}

func (l *filteredMCPLogger) Debugf(format string, a ...any) { l.base.Debugf(format, a...) }
func (l *filteredMCPLogger) Infof(format string, a ...any)  { l.base.Infof(format, a...) }
func (l *filteredMCPLogger) Warnf(format string, a ...any)  { l.base.Warnf(format, a...) }

func (l *filteredMCPLogger) Errorf(format string, a ...any) {
	// The go-mcp Streamable HTTP transport currently logs many 400 responses as [Error].
	// We only downgrade the known noisy cases.
	if shouldDowngradeStreamableHTTPError(format, a...) {
		l.base.Infof(format, a...)
		return
	}
	l.base.Errorf(format, a...)
}

func shouldDowngradeStreamableHTTPError(format string, a ...any) bool {
	// Upstream format in streamable_http_server.go:
	// "streamableHTTPServerTransport Error: code: %d, message: %s"
	if !strings.Contains(format, "streamableHTTPServerTransport Error:") {
		return false
	}
	if len(a) < 2 {
		return false
	}

	// Arg0: status code (int). Arg1: message (string).
	code, ok := a[0].(int)
	if !ok {
		return false
	}
	msg, ok := a[1].(string)
	if !ok {
		return false
	}

	if code != 400 {
		return false
	}

	m := strings.ToLower(msg)
	// Known noisy client mistakes / stale sessions.
	if m == "lack session" || strings.Contains(m, "missing session") {
		return true
	}
	return false
}
