package modules

import (
	"context"
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
type ModuleManager struct {
	instances map[ModuleID]Module
	config    ModuleConfig
	mu        sync.RWMutex
}

// NewModuleManager creates a new ModuleManager.
func NewModuleManager(cfg ModuleConfig) *ModuleManager {
	return &ModuleManager{
		instances: make(map[ModuleID]Module),
		config:    cfg,
	}
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
	mm.instances[id] = instance
	mm.mu.Unlock()

	return instance, nil
}

// UnloadModule unloads a module and releases resources.
func (mm *ModuleManager) UnloadModule(id ModuleID) error {
	mm.mu.Lock()
	instance, ok := mm.instances[id]
	if ok {
		delete(mm.instances, id)
	}
	mm.mu.Unlock()

	if !ok {
		return fmt.Errorf("module not loaded: %s", id)
	}

	if cu, ok := instance.(CleanerUpper); ok {
		return cu.Cleanup()
	}
	return nil
}

// Cleanup unloads all modules and releases resources.
func (mm *ModuleManager) Cleanup() {
	mm.mu.Lock()
	instances := make(map[ModuleID]Module, len(mm.instances))
	for id, instance := range mm.instances {
		instances[id] = instance
	}
	mm.instances = make(map[ModuleID]Module)
	mm.mu.Unlock()

	for _, instance := range instances {
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
	for _, instance := range mm.instances {
		if tp, ok := instance.(ToolProvider); ok {
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
	for _, instance := range mm.instances {
		if hp, ok := instance.(HTTPEndpointProvider); ok {
			providers = append(providers, hp)
		}
	}
	return providers
}
