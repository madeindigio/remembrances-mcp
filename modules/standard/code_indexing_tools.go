package standard

import (
	"context"
	"fmt"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/madeindigio/remembrances-mcp/internal/indexer"
	"github.com/madeindigio/remembrances-mcp/pkg/mcp_tools"
	"github.com/madeindigio/remembrances-mcp/pkg/modules"
)

func init() {
	modules.RegisterModule(CodeIndexingToolsModule{})
}

// CodeIndexingToolsModule provides code indexing and watch tools.
type CodeIndexingToolsModule struct {
	jobManager     *indexer.JobManager
	watcherManager *indexer.WatcherManager
	toolManager    *mcp_tools.CodeToolManager
	tools          []modules.ToolDefinition
	disableWatch   bool
}

// ModuleInfo returns module metadata.
func (CodeIndexingToolsModule) ModuleInfo() modules.ModuleInfo {
	return modules.ModuleInfo{
		ID:          "tools.code_indexing",
		Name:        "Code Indexing Tools",
		Description: "Registers MCP tools for code indexing and project watching",
		Version:     "1.0.0",
		Author:      "Remembrances Team",
		License:     "MIT",
		New:         func() modules.Module { return new(CodeIndexingToolsModule) },
	}
}

// Provision configures the module.
func (m *CodeIndexingToolsModule) Provision(ctx context.Context, cfg modules.ModuleConfig) error {
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

	baseManager := mcp_tools.NewToolManagerWithCodeEmbedder(
		cfg.Storage,
		cfg.Embedder,
		codeEmbedder,
		cfg.KnowledgeBasePath,
	)
	baseManager.SetKBChunking(cfg.KBChunkSize, cfg.KBChunkOverlap)

	indexerConfig := cfg.IndexerConfig
	if indexerConfig == (indexer.IndexerConfig{}) {
		indexerConfig = indexer.DefaultIndexerConfig()
	}
	jobManagerConfig := cfg.JobManagerConfig
	if jobManagerConfig == (indexer.JobManagerConfig{}) {
		jobManagerConfig = indexer.DefaultJobManagerConfig()
	}

	m.jobManager = baseManager.CreateJobManager(cfg.Storage, indexerConfig, jobManagerConfig)
	m.watcherManager = baseManager.CreateWatcherManager(cfg.Storage, m.jobManager)
	m.disableWatch = cfg.DisableCodeWatch

	m.toolManager = mcp_tools.NewCodeToolManager(baseManager, m.jobManager, m.watcherManager)

	var tools []modules.ToolDefinition
	reg := func(name string, tool *protocol.Tool, handler func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error {
		if tool == nil {
			return fmt.Errorf("tool %s creation returned nil", name)
		}
		tools = append(tools, modules.ToolDefinition{Tool: tool, Handler: handler})
		return nil
	}

	if err := m.toolManager.RegisterCodeTools(reg); err != nil {
		return err
	}

	m.tools = tools

	if !m.disableWatch && m.watcherManager != nil {
		if err := m.watcherManager.AutoActivateOnStartup(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Cleanup stops background watchers and job manager.
func (m *CodeIndexingToolsModule) Cleanup() error {
	if m.watcherManager != nil {
		_ = m.watcherManager.Stop()
	}
	if m.jobManager != nil {
		m.jobManager.Stop()
	}
	return nil
}

// Tools returns the tool definitions.
func (m *CodeIndexingToolsModule) Tools() []modules.ToolDefinition {
	return m.tools
}
