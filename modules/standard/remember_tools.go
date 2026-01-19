package standard

import (
	"context"
	"fmt"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/madeindigio/remembrances-mcp/pkg/mcp_tools"
	"github.com/madeindigio/remembrances-mcp/pkg/modules"
)

func init() {
	modules.RegisterModule(RememberToolsModule{})
}

// RememberToolsModule provides to_remember and last_to_remember tools.
type RememberToolsModule struct {
	toolManager *mcp_tools.ToolManager
	tools       []modules.ToolDefinition
}

// ModuleInfo returns module metadata.
func (RememberToolsModule) ModuleInfo() modules.ModuleInfo {
	return modules.ModuleInfo{
		ID:          "tools.remember",
		Name:        "Remember Tools",
		Description: "Registers MCP tools for remembering and recalling context",
		Version:     "1.0.0",
		Author:      "Remembrances Team",
		License:     "MIT",
		New:         func() modules.Module { return new(RememberToolsModule) },
	}
}

// Provision configures the module.
func (m *RememberToolsModule) Provision(ctx context.Context, cfg modules.ModuleConfig) error {
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

	if err := m.toolManager.RegisterRememberToolsWith(reg); err != nil {
		return err
	}

	m.tools = tools
	return nil
}

// Tools returns the tool definitions.
func (m *RememberToolsModule) Tools() []modules.ToolDefinition {
	return m.tools
}
