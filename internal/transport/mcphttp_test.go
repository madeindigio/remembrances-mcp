package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	mcpserver "github.com/ThinkInAIXYZ/go-mcp/server"
	mcptransport "github.com/ThinkInAIXYZ/go-mcp/transport"

	"github.com/madeindigio/remembrances-mcp/v2/internal/storage"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/mcp_tools"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/modules"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/principal"
)

// The MCP Streamable HTTP endpoint was the one entry point require_principal
// could not be turned on for: go-mcp gives a tool handler a context with no
// request in it, so a bearer token arriving there could not be mapped to an
// identity. MCPHTTPServer closes that by owning the http.Server and running
// the authentication, principal and request layers in front of the transport.
//
// These tests drive the real transport over real HTTP and assert what the tool
// handler sees, because the claim being made is precisely that a value put in
// the request context survives go-mcp's dispatch.

// bearerAuthProvider authenticates one token and emits the tenancy claims
// principalFromAuthInfo reads, which is the shape a commercial auth module
// produces.
type bearerAuthProvider struct{}

func (bearerAuthProvider) ModuleInfo() modules.ModuleInfo {
	return modules.ModuleInfo{ID: "test.auth", Name: "test auth"}
}

func (bearerAuthProvider) Middleware() modules.HTTPMiddleware { return nil }

func (bearerAuthProvider) Authenticate(r *http.Request) (*modules.AuthInfo, error) {
	info := mcp_tools.NewRequestInfo(r)
	if info.BearerToken() != "s3cret" {
		return nil, fmt.Errorf("bad token")
	}
	return &modules.AuthInfo{
		UserID: "alice",
		Metadata: map[string]any{
			"org":         "org:acme",
			"user":        "app_user:alice",
			"write_scope": "mem_scope:alice",
			"scopes":      []string{"mem_scope:alice"},
		},
	}, nil
}

// observedCall is what the tool handler saw.
type observedCall struct {
	mu        sync.Mutex
	principal *principal.Principal
	request   *mcp_tools.RequestInfo
}

func (o *observedCall) set(ctx context.Context) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.principal, _ = principal.FromContext(ctx)
	o.request, _ = mcp_tools.RequestFromContext(ctx)
}

func (o *observedCall) get() (*principal.Principal, *mcp_tools.RequestInfo) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.principal, o.request
}

// startMCPOverHTTP wires an MCPHTTPServer to a real listener and registers one
// tool that reports its context.
func startMCPOverHTTP(t *testing.T, providers []modules.HTTPAuthProvider) (*httptest.Server, *observedCall) {
	t.Helper()

	mcpHTTP, err := NewMCPHTTPServer("127.0.0.1:0", DefaultMCPEndpoint, mcptransport.Stateful, nil)
	if err != nil {
		t.Fatalf("NewMCPHTTPServer: %v", err)
	}
	mcpHTTP.SetAuthProviders(providers)

	srv, err := mcpserver.NewServer(mcpHTTP.Transport(),
		mcpserver.WithServerInfo(protocol.Implementation{Name: "test", Version: "0"}))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	observed := &observedCall{}
	tool, err := protocol.NewTool("whoami", "reports the caller", struct{}{})
	if err != nil {
		t.Fatalf("NewTool: %v", err)
	}
	srv.RegisterTool(tool, func(ctx context.Context, _ *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		observed.set(ctx)
		return protocol.NewCallToolResult([]protocol.Content{&protocol.TextContent{Type: "text", Text: "ok"}}, false), nil
	})

	// The listener is httptest's, the handler is ours: the chain under test is
	// the same one Serve installs.
	http := httptest.NewServer(mcpHTTP)
	t.Cleanup(http.Close)
	return http, observed
}

// post sends one JSON-RPC message and returns the status, the session id and
// the first SSE data line of the response.
func post(t *testing.T, server *httptest.Server, sessionID, token, body string) (int, string, string) {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, server.URL+DefaultMCPEndpoint, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var data string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data = strings.TrimPrefix(line, "data: ")
			break
		}
	}
	return resp.StatusCode, resp.Header.Get("Mcp-Session-Id"), data
}

const initializeMessage = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`

const callMessage = `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"whoami","arguments":{}}}`

func TestTheMCPBearerTokenBecomesAPrincipalInTheToolContext(t *testing.T) {
	server, observed := startMCPOverHTTP(t, []modules.HTTPAuthProvider{bearerAuthProvider{}})

	status, session, _ := post(t, server, "", "s3cret", initializeMessage)
	if status != http.StatusOK {
		t.Fatalf("initialize: status %d", status)
	}
	if session == "" {
		t.Fatal("initialize returned no session id")
	}

	if status, _, data := post(t, server, session, "s3cret", callMessage); status != http.StatusOK {
		t.Fatalf("tools/call: status %d, body %s", status, data)
	}

	p, info := observed.get()
	if p == nil {
		t.Fatal("the tool ran with no principal: the request identity did not reach the handler")
	}
	if p.Org != "org:acme" || p.User != "app_user:alice" {
		t.Fatalf("the tool ran as %s, want the token's claims", p)
	}
	if info == nil {
		t.Fatal("the tool saw no request metadata")
	}
	if info.BearerToken() != "s3cret" {
		t.Fatalf("the tool saw bearer %q, want the one that was sent", info.BearerToken())
	}
}

func TestTheMCPEndpointRefusesABadToken(t *testing.T) {
	server, observed := startMCPOverHTTP(t, []modules.HTTPAuthProvider{bearerAuthProvider{}})

	if status, _, _ := post(t, server, "", "wrong", initializeMessage); status != http.StatusUnauthorized {
		t.Fatalf("a bad token got status %d, want 401", status)
	}
	if p, _ := observed.get(); p != nil {
		t.Fatal("a rejected request reached the tools")
	}
}

// TestTheMCPEndpointWithoutAuthProvidersStillCarriesTheRequest keeps the OSS
// deployment working: with no auth module there is no principal to derive, but
// the request metadata is there so a tool resolver can map the token itself.
func TestTheMCPEndpointWithoutAuthProvidersStillCarriesTheRequest(t *testing.T) {
	previous := storage.RequirePrincipal()
	storage.SetRequirePrincipal(false)
	t.Cleanup(func() { storage.SetRequirePrincipal(previous) })

	server, observed := startMCPOverHTTP(t, nil)

	status, session, _ := post(t, server, "", "anything", initializeMessage)
	if status != http.StatusOK {
		t.Fatalf("initialize: status %d", status)
	}
	if status, _, data := post(t, server, session, "anything", callMessage); status != http.StatusOK {
		t.Fatalf("tools/call: status %d, body %s", status, data)
	}

	_, info := observed.get()
	if info == nil {
		t.Fatal("the tool saw no request metadata")
	}
	if info.BearerToken() != "anything" {
		t.Fatalf("the tool saw bearer %q", info.BearerToken())
	}
}

// TestTheInitializeResultIsWellFormed guards the plumbing itself: a change that
// broke the handler chain would otherwise show up only as a missing principal.
func TestTheInitializeResultIsWellFormed(t *testing.T) {
	server, _ := startMCPOverHTTP(t, nil)

	_, _, data := post(t, server, "", "", initializeMessage)
	var response struct {
		Result struct {
			ServerInfo protocol.Implementation `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		t.Fatalf("initialize response %q: %v", data, err)
	}
	if response.Result.ServerInfo.Name != "test" {
		t.Fatalf("initialize returned %+v", response.Result.ServerInfo)
	}
}
