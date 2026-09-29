// The tests in this file stand in for a module outside the core: they import
// only the public packages, exactly as Alchemai Mem does, and they fail to
// compile if the alias layer stops being enough to write a storage wrapper.
package storage_test

import (
	"context"
	"errors"
	"testing"

	internalstorage "github.com/madeindigio/remembrances-mcp/v2/internal/storage"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/modules"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/storage"
)

// aclStorage is the shim a wrapper module needs: it embeds the primary storage
// so the whole contract is present by promotion, and shadows the methods it
// mediates. Embedding an interface named through the public package is the
// half that was impossible before the alias existed.
type aclStorage struct {
	storage.FullStorage

	denied error
}

// SaveFact is a mediated method: the gate runs before the primary is touched.
// It is written here to prove a wrapper can both name the signature and reach
// the embedded value, not to add behaviour to the core.
func (a *aclStorage) SaveFact(ctx context.Context, userID, key string, value interface{}) error {
	if a.denied != nil {
		return a.denied
	}
	return a.FullStorage.SaveFact(ctx, userID, key, value)
}

// GetStats is a mediated method whose result is a named type of the core, so
// it proves the result types are nameable too.
func (a *aclStorage) GetStats(ctx context.Context, userID string) (*storage.MemoryStats, error) {
	if a.denied != nil {
		return nil, a.denied
	}
	return a.FullStorage.GetStats(ctx, userID)
}

// wrapperModule is the module shape the core asks for. Declaring WrapStorage
// with the public spelling and having it satisfy modules.StorageWrapperProvider
// is the property ADR 004 is about.
type wrapperModule struct{ denied error }

func (wrapperModule) ModuleInfo() modules.ModuleInfo {
	return modules.ModuleInfo{
		ID:  "test.storage.wrapper",
		New: func() modules.Module { return wrapperModule{} },
	}
}

func (m wrapperModule) WrapStorage(primary storage.FullStorage) storage.FullStorage {
	return &aclStorage{FullStorage: primary, denied: m.denied}
}

// The core's interface is declared with its own spelling of the type; these
// assertions fail to compile if the alias ever stops being the same type.
var (
	_ modules.StorageWrapperProvider = wrapperModule{}
	_ storage.FullStorage            = (*aclStorage)(nil)
	_ internalstorage.FullStorage    = (*aclStorage)(nil)
)

// stubStorage stands in for the core's real storage. It embeds the contract
// rather than listing its methods, so it compiles as a FullStorage while
// implementing only what these tests call; anything else would be a nil call
// and no test makes one. Embedding the contract this way is itself only
// possible because the type is nameable.
type stubStorage struct {
	storage.FullStorage

	savedFacts   int
	deletedFacts int
}

func (s *stubStorage) SaveFact(ctx context.Context, userID, key string, value interface{}) error {
	s.savedFacts++
	return nil
}

func (s *stubStorage) DeleteFact(ctx context.Context, userID, key string) error {
	s.deletedFacts++
	return nil
}

// TestPublicAliasesAreTheInternalTypes asserts identity rather than
// assignability: a value of the internal type is used where the public type is
// required and back, which only compiles for identical types.
func TestPublicAliasesAreTheInternalTypes(t *testing.T) {
	var internalFull internalstorage.FullStorage
	var publicFull storage.FullStorage = internalFull
	internalFull = publicFull
	_ = internalFull

	var internalStats *internalstorage.MemoryStats
	var publicStats *storage.MemoryStats = internalStats
	internalStats = publicStats
	_ = internalStats

	var internalChange internalstorage.Change
	var publicChange storage.Change = internalChange
	internalChange = publicChange
	_ = internalChange

	if !errors.Is(storage.ErrNoPrincipal, internalstorage.ErrNoPrincipal) {
		t.Fatal("the re-exported ErrNoPrincipal must be the core's own error value")
	}
}

// TestWrapperMediatesAndForwards runs the shim over a stub primary: the
// mediated method is gated, an unmediated one reaches the primary by
// promotion. This is the behaviour a module gets for free once the type is
// nameable.
func TestWrapperMediatesAndForwards(t *testing.T) {
	primary := &stubStorage{}
	denied := errors.New("denied by acl")

	wrapped := wrapperModule{denied: denied}.WrapStorage(primary)

	if err := wrapped.SaveFact(context.Background(), "u", "k", "v"); !errors.Is(err, denied) {
		t.Fatalf("mediated method must be gated, got %v", err)
	}
	if primary.savedFacts != 0 {
		t.Fatal("a gated call must not reach the primary storage")
	}

	if err := wrapped.DeleteFact(context.Background(), "u", "k"); err != nil {
		t.Fatalf("promoted method must reach the primary: %v", err)
	}
	if primary.deletedFacts != 1 {
		t.Fatalf("promoted method did not reach the primary, deletes = %d", primary.deletedFacts)
	}

	// The pass-through wrapper, the shape a disabled module returns.
	if got := (wrapperModule{}).WrapStorage(primary); got == nil {
		t.Fatal("WrapStorage must always return a usable storage")
	}
}

// TestRequirePrincipalIsReachableFromOutsideTheCore covers the second half of
// ADR 004: a commercial module that installs a real principal has to be able
// to close the legacy path, which means reaching the switch (RM-10).
func TestRequirePrincipalIsReachableFromOutsideTheCore(t *testing.T) {
	before := storage.RequirePrincipal()
	t.Cleanup(func() { storage.SetRequirePrincipal(before) })

	storage.SetRequirePrincipal(true)
	if !storage.RequirePrincipal() || !internalstorage.RequirePrincipal() {
		t.Fatal("the public switch must set the core's own switch")
	}
	storage.SetRequirePrincipal(false)
	if storage.RequirePrincipal() || internalstorage.RequirePrincipal() {
		t.Fatal("the public switch must clear the core's own switch")
	}
}
