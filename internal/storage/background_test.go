package storage

import (
	"context"
	"testing"
	"time"

	"github.com/madeindigio/remembrances-mcp/v2/pkg/principal"
)

func TestServicePrincipalIsUsableAndMarkedAsAService(t *testing.T) {
	SetServicePrincipalProvider(nil)

	p := ServicePrincipal()
	if p == nil {
		t.Fatal("ServicePrincipal returned nil")
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("the default service principal is not usable: %v", err)
	}
	if !p.IsService {
		t.Error("the service principal must be marked IsService, or an audit row cannot tell it from a user")
	}

	local := LocalPrincipal()
	if p.Org != local.Org || p.WriteScope != local.WriteScope {
		t.Errorf("the default service principal must write where the local user reads: got org=%s scope=%s, want org=%s scope=%s",
			p.Org, p.WriteScope, local.Org, local.WriteScope)
	}

	// The value handed out is a copy: narrowing it must not narrow the next
	// caller's.
	p.Scopes = nil
	if len(ServicePrincipal().Scopes) == 0 {
		t.Error("ServicePrincipal handed out its own value instead of a copy")
	}
}

func TestAModuleCanReplaceTheServicePrincipal(t *testing.T) {
	t.Cleanup(func() { SetServicePrincipalProvider(nil) })

	want := &principal.Principal{
		Org:        "org:acme",
		WriteScope: "mem_scope:acme_system",
		Scopes:     []string{"mem_scope:acme_system"},
		IsService:  true,
	}
	SetServicePrincipalProvider(func() *principal.Principal { return want })

	got := ServicePrincipal()
	if got.Org != want.Org || got.WriteScope != want.WriteScope {
		t.Fatalf("provider ignored: got %s, want %s", got, want)
	}

	SetServicePrincipalProvider(nil)
	if ServicePrincipal().Org == want.Org {
		t.Fatal("a nil provider must restore the default")
	}
}

func TestServiceContextNeverReplacesAnIdentity(t *testing.T) {
	SetServicePrincipalProvider(nil)

	caller := &principal.Principal{
		Org:        "org:acme",
		User:       "app_user:alice",
		WriteScope: "mem_scope:alice",
	}
	ctx := principal.NewContext(context.Background(), caller)

	got, ok := principal.FromContext(ServiceContext(ctx))
	if !ok {
		t.Fatal("ServiceContext lost the principal it was given")
	}
	if got.User != caller.User {
		t.Fatalf("ServiceContext overwrote the caller: got %s, want %s", got.User, caller.User)
	}

	// And it installs one when there is none.
	if _, ok := principal.FromContext(ServiceContext(context.Background())); !ok {
		t.Fatal("ServiceContext left a context with no identity")
	}
	//nolint:staticcheck // a nil context is exactly what this guards against.
	if _, ok := principal.FromContext(ServiceContext(nil)); !ok {
		t.Fatal("ServiceContext(nil) must still produce a scoped context")
	}
}

func TestBackgroundContextDropsCancellationAndKeepsTheIdentity(t *testing.T) {
	SetServicePrincipalProvider(nil)

	caller := &principal.Principal{
		Org:        "org:acme",
		User:       "app_user:alice",
		WriteScope: "mem_scope:alice",
	}
	request, cancel := context.WithCancel(principal.NewContext(context.Background(), caller))
	bg := BackgroundContext(request)

	cancel()
	select {
	case <-request.Done():
	case <-time.After(time.Second):
		t.Fatal("the request context did not cancel")
	}

	if bg.Err() != nil {
		t.Fatalf("BackgroundContext inherited the cancellation: %v", bg.Err())
	}
	got, ok := principal.FromContext(bg)
	if !ok {
		t.Fatal("BackgroundContext dropped the principal, which is the defect it exists to prevent")
	}
	if got.User != caller.User {
		t.Fatalf("BackgroundContext substituted an identity: got %s, want %s", got.User, caller.User)
	}

	// Work that starts from nowhere becomes the service rather than nobody.
	if _, ok := principal.FromContext(BackgroundContext(context.Background())); !ok {
		t.Fatal("BackgroundContext left an unscoped context")
	}
}
