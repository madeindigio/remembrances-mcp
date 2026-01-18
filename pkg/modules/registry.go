package modules

import (
	"fmt"
	"strings"
	"sync"
)

var (
	modulesMu sync.RWMutex
	modules   = make(map[ModuleID]ModuleInfo)
)

// RegisterModule registers a module in the global registry.
func RegisterModule(instance Module) {
	info := instance.ModuleInfo()
	if info.ID == "" {
		panic("module ID is required")
	}
	if info.New == nil {
		panic("ModuleInfo.New is required")
	}

	modulesMu.Lock()
	defer modulesMu.Unlock()

	if _, exists := modules[info.ID]; exists {
		panic(fmt.Sprintf("module already registered: %s", info.ID))
	}

	modules[info.ID] = info
}

// GetModule returns a module by ID.
func GetModule(id ModuleID) (ModuleInfo, bool) {
	modulesMu.RLock()
	defer modulesMu.RUnlock()

	info, ok := modules[id]
	return info, ok
}

// GetModulesByNamespace returns modules matching a namespace prefix.
func GetModulesByNamespace(namespace string) []ModuleInfo {
	modulesMu.RLock()
	defer modulesMu.RUnlock()

	var result []ModuleInfo
	prefix := namespace + "."
	for id, info := range modules {
		if strings.HasPrefix(string(id), prefix) || string(id) == namespace {
			result = append(result, info)
		}
	}
	return result
}

// ListModules lists all registered modules.
func ListModules() []ModuleInfo {
	modulesMu.RLock()
	defer modulesMu.RUnlock()

	result := make([]ModuleInfo, 0, len(modules))
	for _, info := range modules {
		result = append(result, info)
	}
	return result
}
