// Package mcp_tools provides code search MCP tools.
// This file contains the CodeSearchToolManager and tool definitions.
// Input types are in code_search_tools_types.go
// Handler implementations are in code_search_tools_handlers.go
package mcp_tools

import (
	"context"
	"log/slog"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/madeindigio/remembrances-mcp/internal/storage"
)

// ====== CodeSearchToolManager ======

// CodeSearchToolManager manages code search tools
type CodeSearchToolManager struct {
	storage  storage.Storage
	embedder interface {
		EmbedQuery(ctx context.Context, text string) ([]float32, error)
	}
}

// NewCodeSearchToolManager creates a new code search tool manager
func NewCodeSearchToolManager(s storage.Storage, embedder interface {
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
}) *CodeSearchToolManager {
	return &CodeSearchToolManager{
		storage:  s,
		embedder: embedder,
	}
}

// RegisterCodeSearchTools registers all code search tools
func (cstm *CodeSearchToolManager) RegisterCodeSearchTools(reg func(string, *protocol.Tool, func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error) error {
	if err := reg("code_get_symbols_overview", cstm.codeGetSymbolsOverviewTool(), cstm.codeGetSymbolsOverviewHandler); err != nil {
		return err
	}
	if err := reg("code_find_symbol", cstm.codeFindSymbolTool(), cstm.codeFindSymbolHandler); err != nil {
		return err
	}
	if err := reg("code_search_symbols_semantic", cstm.codeSearchSymbolsSemanticTool(), cstm.codeSearchSymbolsSemanticHandler); err != nil {
		return err
	}
	if err := reg("code_search_pattern", cstm.codeSearchPatternTool(), cstm.codeSearchPatternHandler); err != nil {
		return err
	}
	if err := reg("code_find_references", cstm.codeFindReferencesTool(), cstm.codeFindReferencesHandler); err != nil {
		return err
	}
	if err := reg("code_hybrid_search", cstm.codeHybridSearchTool(), cstm.codeHybridSearchHandler); err != nil {
		return err
	}
	return nil
}

// ====== Tool Definitions ======

func (cstm *CodeSearchToolManager) codeGetSymbolsOverviewTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_get_symbols_overview", `Get high-level overview of symbols in a file. Use how_to_use("code_get_symbols_overview") for details.`, CodeGetSymbolsOverviewInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_get_symbols_overview", "err", err)
		return nil
	}
	return tool
}

func (cstm *CodeSearchToolManager) codeFindSymbolTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_find_symbol", `Find symbols by name or path pattern. Use how_to_use("code_find_symbol") for details.`, CodeFindSymbolInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_find_symbol", "err", err)
		return nil
	}
	return tool
}

func (cstm *CodeSearchToolManager) codeSearchSymbolsSemanticTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_search_symbols_semantic", `Search code using natural language. Use how_to_use("code_search_symbols_semantic") for details.`, CodeSearchSymbolsSemanticInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_search_symbols_semantic", "err", err)
		return nil
	}
	return tool
}

func (cstm *CodeSearchToolManager) codeSearchPatternTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_search_pattern", `Search for text patterns in code. Use how_to_use("code_search_pattern") for details.`, CodeSearchPatternInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_search_pattern", "err", err)
		return nil
	}
	return tool
}

func (cstm *CodeSearchToolManager) codeFindReferencesTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_find_references", `Find symbol usages in codebase. Use how_to_use("code_find_references") for details.`, CodeFindReferencesInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_find_references", "err", err)
		return nil
	}
	return tool
}

func (cstm *CodeSearchToolManager) codeHybridSearchTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_hybrid_search", `Combined semantic + filter search. Use how_to_use("code_hybrid_search") for details.`, CodeHybridSearchInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_hybrid_search", "err", err)
		return nil
	}
	return tool
}
