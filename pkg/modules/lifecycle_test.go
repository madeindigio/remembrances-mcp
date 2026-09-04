package modules

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// recorder collects lifecycle events in the order they happen.
type recorder struct {
	mu     sync.Mutex
	events []string
}

func (r *recorder) add(event string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *recorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.events))
	copy(out, r.events)
	return out
}

// lifecycleModule is a test module implementing Lifecycle and CleanerUpper.
type lifecycleModule struct {
	id       ModuleID
	rec      *recorder
	startErr error
	stopErr  error
}

func (m *lifecycleModule) ModuleInfo() ModuleInfo {
	return ModuleInfo{
		ID:  m.id,
		New: func() Module { return m },
	}
}

func (m *lifecycleModule) Start(ctx context.Context) error {
	m.rec.add("start:" + string(m.id))
	return m.startErr
}

func (m *lifecycleModule) Stop(ctx context.Context) error {
	m.rec.add("stop:" + string(m.id))
	return m.stopErr
}

func (m *lifecycleModule) Cleanup() error {
	m.rec.add("cleanup:" + string(m.id))
	return nil
}

// plainModule implements nothing beyond Module: it must be untouched by the
// lifecycle machinery.
type plainModule struct {
	id  ModuleID
	rec *recorder
}

func (m *plainModule) ModuleInfo() ModuleInfo {
	return ModuleInfo{ID: m.id, New: func() Module { return m }}
}

// registerForTest registers a module under a unique id and removes it from the
// global registry when the test ends.
func registerForTest(t *testing.T, instance Module) ModuleID {
	t.Helper()
	info := instance.ModuleInfo()
	RegisterModule(instance)
	t.Cleanup(func() {
		modulesMu.Lock()
		delete(modules, info.ID)
		modulesMu.Unlock()
	})
	return info.ID
}

func loadAll(t *testing.T, mm *ModuleManager, ids ...ModuleID) {
	t.Helper()
	for _, id := range ids {
		if _, err := mm.LoadModule(context.Background(), id, nil); err != nil {
			t.Fatalf("LoadModule(%s): %v", id, err)
		}
	}
}

func TestStartModulesRunsInLoadOrderAndStopsInReverse(t *testing.T) {
	rec := &recorder{}
	a := registerForTest(t, &lifecycleModule{id: "test.lifecycle.order.a", rec: rec})
	b := registerForTest(t, &lifecycleModule{id: "test.lifecycle.order.b", rec: rec})
	c := registerForTest(t, &plainModule{id: "test.lifecycle.order.c", rec: rec})
	d := registerForTest(t, &lifecycleModule{id: "test.lifecycle.order.d", rec: rec})

	mm := NewModuleManager(ModuleConfig{})
	loadAll(t, mm, a, b, c, d)

	if err := mm.StartModules(context.Background()); err != nil {
		t.Fatalf("StartModules: %v", err)
	}
	if err := mm.StopModules(context.Background()); err != nil {
		t.Fatalf("StopModules: %v", err)
	}

	want := []string{
		"start:test.lifecycle.order.a",
		"start:test.lifecycle.order.b",
		"start:test.lifecycle.order.d",
		"stop:test.lifecycle.order.d",
		"stop:test.lifecycle.order.b",
		"stop:test.lifecycle.order.a",
	}
	got := rec.snapshot()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("lifecycle order:\n got %v\nwant %v", got, want)
	}
}

func TestStartModulesFailureRollsBackAndAborts(t *testing.T) {
	rec := &recorder{}
	boom := errors.New("boom")
	a := registerForTest(t, &lifecycleModule{id: "test.lifecycle.fail.a", rec: rec})
	b := registerForTest(t, &lifecycleModule{id: "test.lifecycle.fail.b", rec: rec, startErr: boom})
	c := registerForTest(t, &lifecycleModule{id: "test.lifecycle.fail.c", rec: rec})

	mm := NewModuleManager(ModuleConfig{})
	loadAll(t, mm, a, b, c)

	err := mm.StartModules(context.Background())
	if err == nil {
		t.Fatal("StartModules must fail when a module Start fails")
	}
	if !errors.Is(err, boom) {
		t.Fatalf("error must wrap the module error, got %v", err)
	}
	if !strings.Contains(err.Error(), "test.lifecycle.fail.b") {
		t.Fatalf("error must name the failing module, got %v", err)
	}

	// c must never start, and a must be stopped again.
	want := []string{
		"start:test.lifecycle.fail.a",
		"start:test.lifecycle.fail.b",
		"stop:test.lifecycle.fail.a",
	}
	got := rec.snapshot()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("rollback order:\n got %v\nwant %v", got, want)
	}
}

func TestStopModulesIsIdempotentAndJoinsErrors(t *testing.T) {
	rec := &recorder{}
	stopErr := errors.New("stop failed")
	a := registerForTest(t, &lifecycleModule{id: "test.lifecycle.idem.a", rec: rec, stopErr: stopErr})
	b := registerForTest(t, &lifecycleModule{id: "test.lifecycle.idem.b", rec: rec})

	mm := NewModuleManager(ModuleConfig{})
	loadAll(t, mm, a, b)

	if err := mm.StartModules(context.Background()); err != nil {
		t.Fatalf("StartModules: %v", err)
	}
	err := mm.StopModules(context.Background())
	if !errors.Is(err, stopErr) {
		t.Fatalf("StopModules must report the module error, got %v", err)
	}
	// b must have been stopped even though a failed.
	if len(rec.snapshot()) != 4 {
		t.Fatalf("every module must be stopped, events: %v", rec.snapshot())
	}

	// A second stop is a no-op.
	if err := mm.StopModules(context.Background()); err != nil {
		t.Fatalf("second StopModules: %v", err)
	}
	if got := len(rec.snapshot()); got != 4 {
		t.Fatalf("second StopModules must not re-stop modules, got %d events", got)
	}
}

func TestCleanupStopsStartedModulesFirst(t *testing.T) {
	rec := &recorder{}
	a := registerForTest(t, &lifecycleModule{id: "test.lifecycle.cleanup.a", rec: rec})
	b := registerForTest(t, &lifecycleModule{id: "test.lifecycle.cleanup.b", rec: rec})

	mm := NewModuleManager(ModuleConfig{})
	loadAll(t, mm, a, b)
	if err := mm.StartModules(context.Background()); err != nil {
		t.Fatalf("StartModules: %v", err)
	}

	mm.Cleanup()

	want := []string{
		"start:test.lifecycle.cleanup.a",
		"start:test.lifecycle.cleanup.b",
		"stop:test.lifecycle.cleanup.b",
		"stop:test.lifecycle.cleanup.a",
		"cleanup:test.lifecycle.cleanup.b",
		"cleanup:test.lifecycle.cleanup.a",
	}
	got := rec.snapshot()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("cleanup order:\n got %v\nwant %v", got, want)
	}
}

func TestUnloadModuleStopsBeforeCleanup(t *testing.T) {
	rec := &recorder{}
	a := registerForTest(t, &lifecycleModule{id: "test.lifecycle.unload.a", rec: rec})

	mm := NewModuleManager(ModuleConfig{})
	loadAll(t, mm, a)
	if err := mm.StartModules(context.Background()); err != nil {
		t.Fatalf("StartModules: %v", err)
	}
	if err := mm.UnloadModule(a); err != nil {
		t.Fatalf("UnloadModule: %v", err)
	}

	want := []string{
		"start:test.lifecycle.unload.a",
		"stop:test.lifecycle.unload.a",
		"cleanup:test.lifecycle.unload.a",
	}
	if got := rec.snapshot(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("unload order:\n got %v\nwant %v", got, want)
	}
	if ids := mm.LoadedModuleIDs(); len(ids) != 0 {
		t.Fatalf("module must be removed from the load order, got %v", ids)
	}
}

func TestModulesWithoutLifecycleAreUnaffected(t *testing.T) {
	rec := &recorder{}
	var ids []ModuleID
	for i := 0; i < 3; i++ {
		ids = append(ids, registerForTest(t, &plainModule{
			id:  ModuleID(fmt.Sprintf("test.lifecycle.plain.%d", i)),
			rec: rec,
		}))
	}

	mm := NewModuleManager(ModuleConfig{})
	loadAll(t, mm, ids...)

	if err := mm.StartModules(context.Background()); err != nil {
		t.Fatalf("StartModules: %v", err)
	}
	if err := mm.StopModules(context.Background()); err != nil {
		t.Fatalf("StopModules: %v", err)
	}
	mm.Cleanup()

	if got := rec.snapshot(); len(got) != 0 {
		t.Fatalf("plain modules must produce no lifecycle events, got %v", got)
	}
}

func TestLoadedModuleIDsKeepsLoadOrder(t *testing.T) {
	rec := &recorder{}
	a := registerForTest(t, &plainModule{id: "test.lifecycle.ids.z", rec: rec})
	b := registerForTest(t, &plainModule{id: "test.lifecycle.ids.a", rec: rec})
	c := registerForTest(t, &plainModule{id: "test.lifecycle.ids.m", rec: rec})

	mm := NewModuleManager(ModuleConfig{})
	loadAll(t, mm, a, b, c)

	want := []ModuleID{a, b, c}
	got := mm.LoadedModuleIDs()
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("load order: got %v want %v", got, want)
		}
	}
}
