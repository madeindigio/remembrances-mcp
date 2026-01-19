package standard

import (
	"context"
	"fmt"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/madeindigio/remembrances-mcp/pkg/mcp_tools"
	"github.com/madeindigio/remembrances-mcp/pkg/modules"
)

func init() {
	modules.RegisterModule(FactToolsModule{})
}

// FactToolsModule provides fact tools.
type FactToolsModule struct {
	toolManager *mcp_tools.ToolManager
	tools       []modules.ToolDefinition
}

// ModuleInfo returns module metadata.
func (FactToolsModule) ModuleInfo() modules.ModuleInfo {
	return modules.ModuleInfo{
		ID:          "tools.facts",
		Name:        "Fact Tools",
		Description: "Registers MCP tools for facts",
		Version:     "1.0.0",
		Author:      "Remembrances Team",
		License:     "MIT",
		New:         func() modules.Module { return new(FactToolsModule) },
	}
}

// Provision configures the module.
func (m *FactToolsModule) Provision(ctx context.Context, cfg modules.ModuleConfig) error {
	if cfg.Storage == nil {
		return fmt.Errorf("storage is required")
	}
	if cfg.Embedder == nil {
		return fmt.Errorf("embedder is required")
	}

	m.toolManager = mcp_tools.NewToolManagerWithCodeEmbedder(
		cfg.Storage,
		cfg.Embedder,
		cfg.CodeEmbedder,
		cfg.KnowledgeBasePath,
	)
	m.toolManager.SetKBChunking(cfg.KBChunkSize, cfg.KBChunkOverlap)

	var tools []modules.ToolDefinition
	reg := func(name string, tool *protocol.Tool, handler func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error {
		if tool == nil {
			return fmt.Errorf("tool %s creation returned nil", name)
		}
		tools = append(tools, modules.ToolDefinition{Tool: tool, Handler: handler})
		return nil
	}

	if err := m.toolManager.RegisterFactToolsWith(reg); err != nil {
		return err
	}

	m.tools = tools
	return nil
}

// Tools returns the tool definitions.
func (m *FactToolsModule) Tools() []modules.ToolDefinition {
	return m.tools
}
