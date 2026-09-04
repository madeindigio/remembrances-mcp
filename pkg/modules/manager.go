package modules

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/madeindigio/remembrances-mcp/internal/indexer"
	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/embedder"
)

// ModuleConfig is passed to Provision().
type ModuleConfig struct {
	Raw               map[string]any
	Storage           storage.FullStorage
	Embedder          embedder.Embedder
	CodeEmbedder      embedder.Embedder
	KnowledgeBasePath string
	KBChunkSize       int
	KBChunkOverlap    int
	DisableCodeWatch  bool
	IndexerConfig     indexer.IndexerConfig
	JobManagerConfig  indexer.JobManagerConfig
	Logger            *slog.Logger
}

// ModuleManager manages module lifecycle.
//
// The manager keeps the load order of the modules: providers are returned,
// modules are started and storage wrappers are applied in that order, and
// modules are stopped in the reverse one. Ordering matters because a module
// may depend on a module loaded before it.
type ModuleManager struct {
	instances map[ModuleID]Module
	// order holds the module ids in load order; instances is a map and does
	// not preserve it.
	order []ModuleID
	// started tracks which modules had Lifecycle.Start called successfully,
	// so stopping is idempotent.
	started map[ModuleID]bool
	config  ModuleConfig
	mu      sync.RWMutex
}

// NewModuleManager creates a new ModuleManager.
func NewModuleManager(cfg ModuleConfig) *ModuleManager {
	return &ModuleManager{
		instances: make(map[ModuleID]Module),
		started:   make(map[ModuleID]bool),
		config:    cfg,
	}
}

// logf logs through the configured logger when there is one. ModuleConfig
// may legitimately carry no logger, for example in tests.
func (mm *ModuleManager) logf(msg string, args ...any) {
	if mm.config.Logger == nil {
		return
	}
	mm.config.Logger.Info(msg, args...)
}

// LoadModule loads and initializes a module by ID.
func (mm *ModuleManager) LoadModule(ctx context.Context, id ModuleID, cfg map[string]any) (Module, error) {
	info, ok := GetModule(id)
	if !ok {
		return nil, fmt.Errorf("module not found: %s", id)
	}

	instance := info.New()

	if prov, ok := instance.(Provisioner); ok {
		modCfg := mm.config
		modCfg.Raw = cfg
		if err := prov.Provision(ctx, modCfg); err != nil {
			return nil, fmt.Errorf("provision failed for %s: %w", id, err)
		}
	}

	if val, ok := instance.(Validator); ok {
		if err := val.Validate(); err != nil {
			if cu, ok := instance.(CleanerUpper); ok {
				_ = cu.Cleanup()
			}
			return nil, fmt.Errorf("validation failed for %s: %w", id, err)
		}
	}

	mm.mu.Lock()
	if _, exists := mm.instances[id]; !exists {
		mm.order = append(mm.order, id)
	}
	mm.instances[id] = instance
	mm.mu.Unlock()

	return instance, nil
}

// WrapStorage checks all loaded modules for StorageWrapperProvider implementations
// and applies them in sequence, returning the final wrapped storage.
// This should be called after all modules are loaded.
func (mm *ModuleManager) WrapStorage(storage storage.FullStorage) storage.FullStorage {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	wrapped := storage
	for _, id := range mm.order {
		instance, ok := mm.instances[id]
		if !ok {
			continue
		}
		if wrapper, ok := instance.(StorageWrapperProvider); ok {
			mm.logf("Applying storage wrapper from module", "module_id", id)
			wrapped = wrapper.WrapStorage(wrapped)
		}
	}

	// Update the config storage reference for future modules
	mm.config.Storage = wrapped

	return wrapped
}

// UnloadModule unloads a module and releases resources.
// UnloadModule stops the module first when it implements Lifecycle and was
// started, then cleans it up.
func (mm *ModuleManager) UnloadModule(id ModuleID) error {
	mm.mu.Lock()
	instance, ok := mm.instances[id]
	var stopErr error
	if ok {
		stopErr = mm.stopLocked(context.Background(), []ModuleID{id})
		delete(mm.instances, id)
		for i, existing := range mm.order {
			if existing == id {
				mm.order = append(mm.order[:i], mm.order[i+1:]...)
				break
			}
		}
	}
	mm.mu.Unlock()

	if !ok {
		return fmt.Errorf("module not loaded: %s", id)
	}

	if cu, ok := instance.(CleanerUpper); ok {
		return errors.Join(stopErr, cu.Cleanup())
	}
	return stopErr
}

// Cleanup unloads all modules and releases resources. Modules that are still
// started are stopped first, in reverse load order, so Cleanup is safe to
// call without a preceding StopModules; cleanup then runs in the same reverse
// order.
func (mm *ModuleManager) Cleanup() {
	mm.mu.Lock()
	_ = mm.stopLocked(context.Background(), mm.startedOrderLocked())

	order := make([]ModuleID, len(mm.order))
	copy(order, mm.order)
	instances := make(map[ModuleID]Module, len(mm.instances))
	for id, instance := range mm.instances {
		instances[id] = instance
	}
	mm.instances = make(map[ModuleID]Module)
	mm.order = nil
	mm.started = make(map[ModuleID]bool)
	mm.mu.Unlock()

	for i := len(order) - 1; i >= 0; i-- {
		instance, ok := instances[order[i]]
		if !ok {
			continue
		}
		if cu, ok := instance.(CleanerUpper); ok {
			_ = cu.Cleanup()
		}
	}
}

// GetToolProviders returns all loaded ToolProvider modules.
func (mm *ModuleManager) GetToolProviders() []ToolProvider {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	var providers []ToolProvider
	for _, id := range mm.order {
		if tp, ok := mm.instances[id].(ToolProvider); ok {
			providers = append(providers, tp)
		}
	}
	return providers
}

// GetHTTPEndpointProviders returns all loaded HTTPEndpointProvider modules.
func (mm *ModuleManager) GetHTTPEndpointProviders() []HTTPEndpointProvider {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	var providers []HTTPEndpointProvider
	for _, id := range mm.order {
		if hp, ok := mm.instances[id].(HTTPEndpointProvider); ok {
			providers = append(providers, hp)
		}
	}
	return providers
}

// GetHTTPAuthProviders returns all loaded HTTPAuthProvider modules in load
// order. The HTTP transport turns them into the authentication layer that
// wraps every module route; see AuthMiddleware.
func (mm *ModuleManager) GetHTTPAuthProviders() []HTTPAuthProvider {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	var providers []HTTPAuthProvider
	for _, id := range mm.order {
		if ap, ok := mm.instances[id].(HTTPAuthProvider); ok {
			providers = append(providers, ap)
		}
	}
	return providers
}

// LoadedModuleIDs returns the ids of the loaded modules in load order.
func (mm *ModuleManager) LoadedModuleIDs() []ModuleID {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	ids := make([]ModuleID, len(mm.order))
	copy(ids, mm.order)
	return ids
}
