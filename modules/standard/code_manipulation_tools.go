package standard

import (
	"context"
	"fmt"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/madeindigio/remembrances-mcp/pkg/mcp_tools"
	"github.com/madeindigio/remembrances-mcp/pkg/modules"
)

func init() {
	modules.RegisterModule(CodeManipulationToolsModule{})
}

// CodeManipulationToolsModule provides code manipulation tools.
type CodeManipulationToolsModule struct {
	toolManager *mcp_tools.CodeManipulationToolManager
	tools       []modules.ToolDefinition
}

// ModuleInfo returns module metadata.
func (CodeManipulationToolsModule) ModuleInfo() modules.ModuleInfo {
	return modules.ModuleInfo{
		ID:          "tools.code_manipulation",
		Name:        "Code Manipulation Tools",
		Description: "Registers MCP tools for code manipulation",
		Version:     "1.0.0",
		Author:      "Remembrances Team",
		License:     "MIT",
		New:         func() modules.Module { return new(CodeManipulationToolsModule) },
	}
}

// Provision configures the module.
func (m *CodeManipulationToolsModule) Provision(ctx context.Context, cfg modules.ModuleConfig) error {
	if cfg.Storage == nil {
		return fmt.Errorf("storage is required")
	}
	if cfg.Embedder == nil {
		return fmt.Errorf("embedder is required")
	}

	codeEmbedder := cfg.CodeEmbedder
	if codeEmbedder == nil {
		codeEmbedder = cfg.Embedder
	}

	m.toolManager = mcp_tools.NewCodeManipulationToolManager(cfg.Storage, codeEmbedder)

	var tools []modules.ToolDefinition
	reg := func(name string, tool *protocol.Tool, handler func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error {
		if tool == nil {
			return fmt.Errorf("tool %s creation returned nil", name)
		}
		tools = append(tools, modules.ToolDefinition{Tool: tool, Handler: handler})
		return nil
	}

	if err := m.toolManager.RegisterCodeManipulationTools(reg); err != nil {
		return err
	}

	m.tools = tools
	return nil
}

// Tools returns the tool definitions.
func (m *CodeManipulationToolsModule) Tools() []modules.ToolDefinition {
	return m.tools
}
