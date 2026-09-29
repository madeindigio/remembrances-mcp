package mcp_tools

import (
	"context"
	"errors"
	"testing"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/principal"
)

// collector is a registration callback that keeps the handler it is given, so
// a test can call it the way the MCP server would.
type collector struct {
	handlers map[string]func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)
}

func newCollector() *collector {
	return &collector{handlers: map[string]func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error){}}
}

func (c *collector) reg(name string, _ *protocol.Tool, handler func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error {
	c.handlers[name] = handler
	return nil
}

// principalReporter is a tool handler that reports the principal of its call.
func principalReporter(out **principal.Principal, ran *bool) func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	return func(ctx context.Context, _ *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		*ran = true
		if p, ok := principal.FromContext(ctx); ok {
			*out = p
		}
		return &protocol.CallToolResult{}, nil
	}
}

func localish() *principal.Principal {
	return &principal.Principal{
		Org:          "org:legacy",
		User:         "app_user:legacy_local",
		WriteScope:   "mem_scope:legacy_personal_local",
		Scopes:       []string{"mem_scope:legacy_unassigned"},
		LegacyUserID: "local",
	}
}

func TestToolsRunWithTheResolvedPrincipal(t *testing.T) {
	t.Cleanup(func() { SetDefaultPrincipalResolver(nil) })
	want := localish()
	SetDefaultPrincipalResolver(func(context.Context, *protocol.CallToolRequest) (*principal.Principal, error) {
		return want, nil
	})

	var got *principal.Principal
	var ran bool
	c := newCollector()
	if err := withPrincipalResolver(c.reg, nil)("save_fact", nil, principalReporter(&got, &ran)); err != nil {
		t.Fatal(err)
	}
	if _, err := c.handlers["save_fact"](context.Background(), &protocol.CallToolRequest{}); err != nil {
		t.Fatal(err)
	}
	if !ran || got != want {
		t.Fatalf("the tool must run as the resolved principal, ran %v got %+v", ran, got)
	}
}

func TestTheDefaultResolverIsReadPerCallNotAtRegistration(t *testing.T) {
	// The standard modules register their tools during Provision, so a
	// resolver installed by main after that must still take effect.
	t.Cleanup(func() { SetDefaultPrincipalResolver(nil) })
	SetDefaultPrincipalResolver(nil)

	var got *principal.Principal
	var ran bool
	c := newCollector()
	if err := withPrincipalResolver(c.reg, nil)("get_fact", nil, principalReporter(&got, &ran)); err != nil {
		t.Fatal(err)
	}

	want := localish()
	SetDefaultPrincipalResolver(func(context.Context, *protocol.CallToolRequest) (*principal.Principal, error) {
		return want, nil
	})
	if _, err := c.handlers["get_fact"](context.Background(), &protocol.CallToolRequest{}); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("a resolver installed after registration must apply, got %+v", got)
	}
}

func TestAPrincipalInTheContextIsNeverReplaced(t *testing.T) {
	t.Cleanup(func() { SetDefaultPrincipalResolver(nil) })
	SetDefaultPrincipalResolver(func(context.Context, *protocol.CallToolRequest) (*principal.Principal, error) {
		return localish(), nil
	})

	upstream := &principal.Principal{Org: "org:acme", User: "app_user:ada", WriteScope: "mem_scope:ada", Scopes: []string{"mem_scope:ada"}}
	var got *principal.Principal
	var ran bool
	c := newCollector()
	if err := withPrincipalResolver(c.reg, nil)("list_facts", nil, principalReporter(&got, &ran)); err != nil {
		t.Fatal(err)
	}
	ctx := principal.NewContext(context.Background(), upstream)
	if _, err := c.handlers["list_facts"](ctx, &protocol.CallToolRequest{}); err != nil {
		t.Fatal(err)
	}
	if got != upstream {
		t.Fatalf("the caller's own principal must win, got %+v", got)
	}
}

func TestAResolverErrorFailsTheCall(t *testing.T) {
	t.Cleanup(func() { SetDefaultPrincipalResolver(nil) })
	boom := errors.New("token verification unavailable")
	SetDefaultPrincipalResolver(func(context.Context, *protocol.CallToolRequest) (*principal.Principal, error) {
		return nil, boom
	})

	var got *principal.Principal
	var ran bool
	c := newCollector()
	if err := withPrincipalResolver(c.reg, nil)("delete_fact", nil, principalReporter(&got, &ran)); err != nil {
		t.Fatal(err)
	}
	if _, err := c.handlers["delete_fact"](context.Background(), &protocol.CallToolRequest{}); !errors.Is(err, boom) {
		t.Fatalf("a resolver error must fail the call, got %v", err)
	}
	if ran {
		t.Fatal("the tool must not run without an identity the resolver was meant to supply")
	}
}

func TestWithoutAResolverTheContextIsUntouched(t *testing.T) {
	// Networked deployments with no auth module keep the pre-RM-10 behaviour:
	// nothing is invented here and the storage layer applies its own rule.
	t.Cleanup(func() { SetDefaultPrincipalResolver(nil) })
	SetDefaultPrincipalResolver(nil)

	var got *principal.Principal
	var ran bool
	c := newCollector()
	if err := withPrincipalResolver(c.reg, nil)("hybrid_search", nil, principalReporter(&got, &ran)); err != nil {
		t.Fatal(err)
	}
	if _, err := c.handlers["hybrid_search"](context.Background(), &protocol.CallToolRequest{}); err != nil {
		t.Fatal(err)
	}
	if !ran || got != nil {
		t.Fatalf("no principal may be invented, ran %v got %+v", ran, got)
	}
}

func TestAManagerResolverOverridesTheDefault(t *testing.T) {
	t.Cleanup(func() { SetDefaultPrincipalResolver(nil) })
	SetDefaultPrincipalResolver(func(context.Context, *protocol.CallToolRequest) (*principal.Principal, error) {
		return localish(), nil
	})
	own := &principal.Principal{Org: "org:acme", User: "app_user:svc", WriteScope: "mem_scope:svc", IsService: true}

	tm := &ToolManager{}
	tm.SetPrincipalResolver(func(context.Context, *protocol.CallToolRequest) (*principal.Principal, error) {
		return own, nil
	})

	var got *principal.Principal
	var ran bool
	c := newCollector()
	if err := tm.withPrincipal(c.reg)("get_stats", nil, principalReporter(&got, &ran)); err != nil {
		t.Fatal(err)
	}
	if _, err := c.handlers["get_stats"](context.Background(), &protocol.CallToolRequest{}); err != nil {
		t.Fatal(err)
	}
	if got != own {
		t.Fatalf("the manager resolver must win, got %+v", got)
	}
}

func TestEveryRegisteredToolCarriesThePrincipalInstaller(t *testing.T) {
	// The point of the decoration is that it is not per tool: a tool added
	// later gets the identity without anyone remembering, and a registration
	// path that forgot the wrapper fails here.
	//
	// The probe is a resolver that always fails. The wrapper short-circuits on
	// a resolver error, so a handler reached through it returns that error and
	// never touches storage; a handler registered without the wrapper would
	// run for real. That makes the assertion decisive without giving the tools
	// a database.
	t.Cleanup(func() { SetDefaultPrincipalResolver(nil) })
	boom := errors.New("resolver probe")
	SetDefaultPrincipalResolver(func(context.Context, *protocol.CallToolRequest) (*principal.Principal, error) {
		return nil, boom
	})

	tm := NewToolManager(nil, nil, "")
	c := newCollector()
	if err := tm.RegisterToolsWith(c.reg); err != nil {
		t.Fatalf("registration failed: %v", err)
	}
	if len(c.handlers) == 0 {
		t.Fatal("no tools registered")
	}

	for name, handler := range c.handlers {
		if _, err := handler(context.Background(), &protocol.CallToolRequest{}); !errors.Is(err, boom) {
			t.Errorf("tool %q was registered without the principal installer, got %v", name, err)
		}
	}
}
