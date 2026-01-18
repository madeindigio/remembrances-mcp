package modules

import "context"

// ModuleID identifies a module uniquely (e.g., "tools.reasoning").
type ModuleID string

// ModuleInfo contains module metadata and factory function.
type ModuleInfo struct {
	ID          ModuleID
	Name        string
	Description string
	Version     string
	Author      string
	License     string
	New         func() Module
}

// Module is the base interface every module must implement.
type Module interface {
	ModuleInfo() ModuleInfo
}

// Provisioner provides post-load configuration.
type Provisioner interface {
	Provision(ctx context.Context, cfg ModuleConfig) error
}

// Validator validates module configuration.
type Validator interface {
	Validate() error
}

// CleanerUpper releases resources when a module is unloaded.
type CleanerUpper interface {
	Cleanup() error
}
