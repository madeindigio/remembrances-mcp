// Package mcp_tools provides the tool definitions for the MCP server.
package mcp_tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/madeindigio/remembrances-mcp/internal/indexer"
	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/embedder"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	mcpserver "github.com/ThinkInAIXYZ/go-mcp/server"
)

// ToolManager manages all MCP tools for the remembrances server
type ToolManager struct {
	storage           storage.StorageWithStats
	embedder          embedder.Embedder
	codeEmbedder      embedder.Embedder // Embedder for code indexing (may be same as default)
	knowledgeBasePath string            // Path to knowledge base directory for markdown files
}

// NewToolManager creates a new tool manager
func NewToolManager(storage storage.StorageWithStats, embedder embedder.Embedder, knowledgeBasePath string) *ToolManager {
	return &ToolManager{
		storage:           storage,
		embedder:          embedder,
		codeEmbedder:      embedder, // Default to same embedder for backwards compatibility
		knowledgeBasePath: knowledgeBasePath,
	}
}

// NewToolManagerWithCodeEmbedder creates a new tool manager with a separate code embedder
func NewToolManagerWithCodeEmbedder(storage storage.StorageWithStats, embedder embedder.Embedder, codeEmbedder embedder.Embedder, knowledgeBasePath string) *ToolManager {
	return &ToolManager{
		storage:           storage,
		embedder:          embedder,
		codeEmbedder:      codeEmbedder,
		knowledgeBasePath: knowledgeBasePath,
	}
}

// GetCodeEmbedder returns the embedder used for code indexing
func (tm *ToolManager) GetCodeEmbedder() embedder.Embedder {
	return tm.codeEmbedder
}

// RegisterTools registers all MCP tools with the server
func (tm *ToolManager) RegisterTools(srv *mcpserver.Server) error {
	// Helper to register a tool and return error if creation returned nil
	reg := func(name string, tool *protocol.Tool, handler func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error {
		if tool == nil {
			return fmt.Errorf("tool %s creation returned nil", name)
		}
		srv.RegisterTool(tool, handler)
		return nil
	}

	// Delegate to smaller registration groups
	if err := tm.registerRemembranceTools(reg); err != nil {
		return err
	}
	if err := tm.registerVectorTools(reg); err != nil {
		return err
	}
	if err := tm.registerGraphTools(reg); err != nil {
		return err
	}
	if err := tm.registerKBTools(reg); err != nil {
		return err
	}
	if err := tm.registerRememberTools(reg); err != nil {
		return err
	}
	if err := tm.registerMiscTools(reg); err != nil {
		return err
	}
	if err := tm.registerEventTools(reg); err != nil {
		return err
	}

	slog.Info("Successfully registered all MCP tools")
	return nil
}

// registration helper groups keep RegisterTools small and readable
func (tm *ToolManager) registerRemembranceTools(reg func(string, *protocol.Tool, func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error) error {
	if err := reg("remembrance_save_fact", tm.saveFactTool(), tm.saveFactHandler); err != nil {
		return err
	}
	if err := reg("remembrance_get_fact", tm.getFactTool(), tm.getFactHandler); err != nil {
		return err
	}
	if err := reg("remembrance_list_facts", tm.listFactsTool(), tm.listFactsHandler); err != nil {
		return err
	}
	if err := reg("remembrance_delete_fact", tm.deleteFactTool(), tm.deleteFactHandler); err != nil {
		return err
	}
	return nil
}

func (tm *ToolManager) registerVectorTools(reg func(string, *protocol.Tool, func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error) error {
	if err := reg("remembrance_add_vector", tm.addVectorTool(), tm.addVectorHandler); err != nil {
		return err
	}
	if err := reg("remembrance_search_vectors", tm.searchVectorsTool(), tm.searchVectorsHandler); err != nil {
		return err
	}
	if err := reg("remembrance_update_vector", tm.updateVectorTool(), tm.updateVectorHandler); err != nil {
		return err
	}
	if err := reg("remembrance_delete_vector", tm.deleteVectorTool(), tm.deleteVectorHandler); err != nil {
		return err
	}
	return nil
}

func (tm *ToolManager) registerGraphTools(reg func(string, *protocol.Tool, func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error) error {
	if err := reg("remembrance_create_entity", tm.createEntityTool(), tm.createEntityHandler); err != nil {
		return err
	}
	if err := reg("remembrance_create_relationship", tm.createRelationshipTool(), tm.createRelationshipHandler); err != nil {
		return err
	}
	if err := reg("remembrance_traverse_graph", tm.traverseGraphTool(), tm.traverseGraphHandler); err != nil {
		return err
	}
	if err := reg("remembrance_get_entity", tm.getEntityTool(), tm.getEntityHandler); err != nil {
		return err
	}
	return nil
}

func (tm *ToolManager) registerKBTools(reg func(string, *protocol.Tool, func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error) error {
	if err := reg("kb_add_document", tm.addDocumentTool(), tm.addDocumentHandler); err != nil {
		return err
	}
	if err := reg("kb_search_documents", tm.searchDocumentsTool(), tm.searchDocumentsHandler); err != nil {
		return err
	}
	if err := reg("kb_get_document", tm.getDocumentTool(), tm.getDocumentHandler); err != nil {
		return err
	}
	if err := reg("kb_delete_document", tm.deleteDocumentTool(), tm.deleteDocumentHandler); err != nil {
		return err
	}
	return nil
}

func (tm *ToolManager) registerRememberTools(reg func(string, *protocol.Tool, func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error) error {
	if err := reg("to_remember", tm.toRememberTool(), tm.toRememberHandler); err != nil {
		return err
	}
	if err := reg("last_to_remember", tm.lastToRememberTool(), tm.lastToRememberHandler); err != nil {
		return err
	}
	return nil
}

func (tm *ToolManager) registerMiscTools(reg func(string, *protocol.Tool, func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error) error {
	if err := reg("remembrance_hybrid_search", tm.hybridSearchTool(), tm.hybridSearchHandler); err != nil {
		return err
	}
	if err := reg("remembrance_get_stats", tm.getStatsTool(), tm.getStatsHandler); err != nil {
		return err
	}
	if err := reg("how_to_use", tm.howToUseTool(), tm.howToUseHandler); err != nil {
		return err
	}
	return nil
}

func (tm *ToolManager) registerEventTools(reg func(string, *protocol.Tool, func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error) error {
	if err := reg("save_event", tm.saveEventTool(), tm.saveEventHandler); err != nil {
		return err
	}
	if err := reg("search_events", tm.searchEventsTool(), tm.searchEventsHandler); err != nil {
		return err
	}
	return nil
}

// CreateJobManager creates a JobManager for code indexing using the code embedder.
// This uses the code-specific embedder (if configured) for generating embeddings
// of code symbols, allowing specialized code embedding models to be used.
//
// The returned JobManager should be used with CodeToolManager to provide
// code indexing capabilities.
func (tm *ToolManager) CreateJobManager(fullStorage storage.FullStorage, indexerConfig indexer.IndexerConfig, jmConfig indexer.JobManagerConfig) *indexer.JobManager {
	return indexer.NewJobManager(fullStorage, tm.codeEmbedder, indexerConfig, jmConfig)
}
