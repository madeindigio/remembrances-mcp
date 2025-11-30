// Package mcp_tools provides code manipulation MCP tools.
package mcp_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// ====== Input Types ======

// CodeReplaceSymbolInput represents input for code_replace_symbol tool
type CodeReplaceSymbolInput struct {
	ProjectID    string `json:"project_id" description:"The project ID containing the symbol."`
	SymbolID     string `json:"symbol_id,omitempty" description:"ID of the symbol to replace (from previous search)."`
	NamePath     string `json:"name_path,omitempty" description:"Name path of symbol (alternative to symbol_id)."`
	RelativePath string `json:"relative_path,omitempty" description:"File path (required if using name_path)."`
	NewBody      string `json:"new_body" description:"New source code for the symbol, including its definition/signature."`
}

// CodeInsertAfterSymbolInput represents input for code_insert_after_symbol tool
type CodeInsertAfterSymbolInput struct {
	ProjectID    string `json:"project_id" description:"The project ID containing the symbol."`
	SymbolID     string `json:"symbol_id,omitempty" description:"ID of the symbol after which to insert."`
	NamePath     string `json:"name_path,omitempty" description:"Name path of symbol (alternative to symbol_id)."`
	RelativePath string `json:"relative_path,omitempty" description:"File path (required if using name_path)."`
	Body         string `json:"body" description:"Code to insert after the symbol."`
}

// CodeInsertBeforeSymbolInput represents input for code_insert_before_symbol tool
type CodeInsertBeforeSymbolInput struct {
	ProjectID    string `json:"project_id" description:"The project ID containing the symbol."`
	SymbolID     string `json:"symbol_id,omitempty" description:"ID of the symbol before which to insert."`
	NamePath     string `json:"name_path,omitempty" description:"Name path of symbol (alternative to symbol_id)."`
	RelativePath string `json:"relative_path,omitempty" description:"File path (required if using name_path)."`
	Body         string `json:"body" description:"Code to insert before the symbol."`
}

// CodeDeleteSymbolInput represents input for code_delete_symbol tool
type CodeDeleteSymbolInput struct {
	ProjectID    string `json:"project_id" description:"The project ID containing the symbol."`
	SymbolID     string `json:"symbol_id,omitempty" description:"ID of the symbol to delete."`
	NamePath     string `json:"name_path,omitempty" description:"Name path of symbol (alternative to symbol_id)."`
	RelativePath string `json:"relative_path,omitempty" description:"File path (required if using name_path)."`
}

// ====== CodeManipulationToolManager ======

// CodeManipulationToolManager manages code manipulation tools
type CodeManipulationToolManager struct {
	storage  storage.Storage
	parser   *treesitter.Parser
	walker   *treesitter.ASTWalker
	embedder interface {
		EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
	}
}

// NewCodeManipulationToolManager creates a new code manipulation tool manager
func NewCodeManipulationToolManager(s storage.Storage, embedder interface {
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
}) *CodeManipulationToolManager {
	walkerConfig := treesitter.WalkerConfig{
		IncludeSourceCode: true,
		MaxSymbolSize:     10000,
		ExtractDocStrings: true,
	}

	return &CodeManipulationToolManager{
		storage:  s,
		parser:   treesitter.NewParser(),
		walker:   treesitter.NewASTWalker(walkerConfig),
		embedder: embedder,
	}
}

// RegisterCodeManipulationTools registers all code manipulation tools
func (cmtm *CodeManipulationToolManager) RegisterCodeManipulationTools(reg func(string, *protocol.Tool, func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error) error {
	if err := reg("code_replace_symbol", cmtm.codeReplaceSymbolTool(), cmtm.codeReplaceSymbolHandler); err != nil {
		return err
	}
	if err := reg("code_insert_after_symbol", cmtm.codeInsertAfterSymbolTool(), cmtm.codeInsertAfterSymbolHandler); err != nil {
		return err
	}
	if err := reg("code_insert_before_symbol", cmtm.codeInsertBeforeSymbolTool(), cmtm.codeInsertBeforeSymbolHandler); err != nil {
		return err
	}
	if err := reg("code_delete_symbol", cmtm.codeDeleteSymbolTool(), cmtm.codeDeleteSymbolHandler); err != nil {
		return err
	}
	return nil
}

// ====== Tool Definitions ======

func (cmtm *CodeManipulationToolManager) codeReplaceSymbolTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_replace_symbol", `Replace a symbol's source code. Use how_to_use("code_replace_symbol") for details.`, CodeReplaceSymbolInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_replace_symbol", "err", err)
		return nil
	}
	return tool
}

func (cmtm *CodeManipulationToolManager) codeInsertAfterSymbolTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_insert_after_symbol", `Insert code after a symbol. Use how_to_use("code_insert_after_symbol") for details.`, CodeInsertAfterSymbolInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_insert_after_symbol", "err", err)
		return nil
	}
	return tool
}

func (cmtm *CodeManipulationToolManager) codeInsertBeforeSymbolTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_insert_before_symbol", `Insert code before a symbol. Use how_to_use("code_insert_before_symbol") for details.`, CodeInsertBeforeSymbolInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_insert_before_symbol", "err", err)
		return nil
	}
	return tool
}

func (cmtm *CodeManipulationToolManager) codeDeleteSymbolTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_delete_symbol", `Delete a symbol from the file. Use how_to_use("code_delete_symbol") for details.`, CodeDeleteSymbolInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_delete_symbol", "err", err)
		return nil
	}
	return tool
}

// ====== Helper Functions ======

// symbolInfo holds resolved symbol information
type symbolInfo struct {
	ID           string
	ProjectID    string
	FilePath     string
	NamePath     string
	StartByte    int
	EndByte      int
	StartLine    int
	EndLine      int
	Language     treesitter.Language
	AbsolutePath string
}

// resolveSymbol finds a symbol by ID or name_path
func (cmtm *CodeManipulationToolManager) resolveSymbol(ctx context.Context, projectID, symbolID, namePath, relativePath string) (*symbolInfo, error) {
	codeStorage, ok := cmtm.storage.(interface {
		Query(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error)
		GetCodeProject(ctx context.Context, projectID string) (*storage.CodeProject, error)
	})
	if !ok {
		return nil, fmt.Errorf("storage does not support code operations")
	}

	var query string
	params := map[string]interface{}{"project_id": projectID}

	if symbolID != "" {
		// Query by ID
		query = `SELECT * FROM $symbol_id;`
		params["symbol_id"] = symbolID
	} else if namePath != "" && relativePath != "" {
		// Query by name_path and file_path
		query = `SELECT * FROM code_symbols WHERE project_id = $project_id AND name_path = $name_path AND file_path = $file_path LIMIT 1;`
		params["name_path"] = namePath
		params["file_path"] = relativePath
	} else if namePath != "" {
		// Query by name_path only (may return multiple)
		query = `SELECT * FROM code_symbols WHERE project_id = $project_id AND name_path = $name_path LIMIT 1;`
		params["name_path"] = namePath
	} else {
		return nil, fmt.Errorf("either symbol_id or name_path is required")
	}

	results, err := codeStorage.Query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to query symbol: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("symbol not found")
	}

	r := results[0]

	// Get project to find root path
	project, err := codeStorage.GetCodeProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	filePath, _ := r["file_path"].(string)
	absPath := filepath.Join(project.RootPath, filePath)

	// Extract byte positions
	startByte, _ := r["start_byte"].(float64)
	endByte, _ := r["end_byte"].(float64)
	startLine, _ := r["start_line"].(float64)
	endLine, _ := r["end_line"].(float64)
	id, _ := r["id"].(string)
	namePathVal, _ := r["name_path"].(string)
	lang, _ := r["language"].(string)

	return &symbolInfo{
		ID:           id,
		ProjectID:    projectID,
		FilePath:     filePath,
		NamePath:     namePathVal,
		StartByte:    int(startByte),
		EndByte:      int(endByte),
		StartLine:    int(startLine),
		EndLine:      int(endLine),
		Language:     treesitter.Language(lang),
		AbsolutePath: absPath,
	}, nil
}

// modifyFile reads a file, applies a modification, and writes it back
func (cmtm *CodeManipulationToolManager) modifyFile(absPath string, modifier func(content []byte) ([]byte, error)) error {
	// Read file
	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Apply modification
	newContent, err := modifier(content)
	if err != nil {
		return err
	}

	// Write back
	if err := os.WriteFile(absPath, newContent, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// reindexFile re-parses a file and updates symbols in the database
func (cmtm *CodeManipulationToolManager) reindexFile(ctx context.Context, projectID, filePath, absPath string, lang treesitter.Language) error {
	codeStorage, ok := cmtm.storage.(interface {
		DeleteSymbolsByFile(ctx context.Context, projectID, filePath string) error
		SaveCodeSymbols(ctx context.Context, symbols []*treesitter.CodeSymbol) error
	})
	if !ok {
		return fmt.Errorf("storage does not support code operations")
	}

	// Read file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse file
	tree, parsedLang, err := cmtm.parser.ParseFile(ctx, absPath)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Extract symbols
	symbols, err := cmtm.walker.ExtractSymbols(tree, content, parsedLang, filePath, projectID)
	if err != nil {
		return fmt.Errorf("failed to extract symbols: %w", err)
	}

	// Generate embeddings for symbols
	if cmtm.embedder != nil && len(symbols) > 0 {
		texts := make([]string, len(symbols))
		for i, sym := range symbols {
			// Create a searchable text from symbol info
			texts[i] = fmt.Sprintf("%s %s %s", sym.Name, sym.SymbolType, sym.SourceCode)
		}

		embeddings, err := cmtm.embedder.EmbedDocuments(ctx, texts)
		if err == nil && len(embeddings) == len(symbols) {
			for i, emb := range embeddings {
				symbols[i].Embedding = emb
			}
		}
	}

	// Delete old symbols
	if err := codeStorage.DeleteSymbolsByFile(ctx, projectID, filePath); err != nil {
		return fmt.Errorf("failed to delete old symbols: %w", err)
	}

	// Save new symbols
	if err := codeStorage.SaveCodeSymbols(ctx, symbols); err != nil {
		return fmt.Errorf("failed to save new symbols: %w", err)
	}

	return nil
}

// ====== Tool Handlers ======

func (cmtm *CodeManipulationToolManager) codeReplaceSymbolHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeReplaceSymbolInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	if input.NewBody == "" {
		return nil, fmt.Errorf("new_body is required")
	}

	// Resolve symbol
	sym, err := cmtm.resolveSymbol(ctx, input.ProjectID, input.SymbolID, input.NamePath, input.RelativePath)
	if err != nil {
		return nil, err
	}

	// Modify file
	err = cmtm.modifyFile(sym.AbsolutePath, func(content []byte) ([]byte, error) {
		if sym.StartByte < 0 || sym.EndByte > len(content) || sym.StartByte > sym.EndByte {
			return nil, fmt.Errorf("invalid byte range: %d-%d (file size: %d)", sym.StartByte, sym.EndByte, len(content))
		}

		// Replace the symbol content
		newContent := make([]byte, 0, len(content)-sym.EndByte+sym.StartByte+len(input.NewBody))
		newContent = append(newContent, content[:sym.StartByte]...)
		newContent = append(newContent, []byte(input.NewBody)...)
		newContent = append(newContent, content[sym.EndByte:]...)

		return newContent, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to modify file: %w", err)
	}

	// Re-index file
	if err := cmtm.reindexFile(ctx, sym.ProjectID, sym.FilePath, sym.AbsolutePath, sym.Language); err != nil {
		slog.Warn("failed to reindex file after replacement", "file", sym.FilePath, "error", err)
	}

	result := map[string]interface{}{
		"message":   "Symbol replaced successfully",
		"file_path": sym.FilePath,
		"name_path": sym.NamePath,
		"old_range": map[string]int{
			"start_line": sym.StartLine,
			"end_line":   sym.EndLine,
		},
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: string(resultJSON),
		},
	}, false), nil
}

func (cmtm *CodeManipulationToolManager) codeInsertAfterSymbolHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeInsertAfterSymbolInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	if input.Body == "" {
		return nil, fmt.Errorf("body is required")
	}

	// Resolve symbol
	sym, err := cmtm.resolveSymbol(ctx, input.ProjectID, input.SymbolID, input.NamePath, input.RelativePath)
	if err != nil {
		return nil, err
	}

	// Modify file
	err = cmtm.modifyFile(sym.AbsolutePath, func(content []byte) ([]byte, error) {
		if sym.EndByte < 0 || sym.EndByte > len(content) {
			return nil, fmt.Errorf("invalid end byte: %d (file size: %d)", sym.EndByte, len(content))
		}

		// Insert after end of symbol
		newContent := make([]byte, 0, len(content)+len(input.Body))
		newContent = append(newContent, content[:sym.EndByte]...)
		newContent = append(newContent, []byte(input.Body)...)
		newContent = append(newContent, content[sym.EndByte:]...)

		return newContent, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to modify file: %w", err)
	}

	// Re-index file
	if err := cmtm.reindexFile(ctx, sym.ProjectID, sym.FilePath, sym.AbsolutePath, sym.Language); err != nil {
		slog.Warn("failed to reindex file after insertion", "file", sym.FilePath, "error", err)
	}

	result := map[string]interface{}{
		"message":         "Code inserted after symbol successfully",
		"file_path":       sym.FilePath,
		"reference_symbol": sym.NamePath,
		"inserted_after_line": sym.EndLine,
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: string(resultJSON),
		},
	}, false), nil
}

func (cmtm *CodeManipulationToolManager) codeInsertBeforeSymbolHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeInsertBeforeSymbolInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	if input.Body == "" {
		return nil, fmt.Errorf("body is required")
	}

	// Resolve symbol
	sym, err := cmtm.resolveSymbol(ctx, input.ProjectID, input.SymbolID, input.NamePath, input.RelativePath)
	if err != nil {
		return nil, err
	}

	// Modify file
	err = cmtm.modifyFile(sym.AbsolutePath, func(content []byte) ([]byte, error) {
		if sym.StartByte < 0 || sym.StartByte > len(content) {
			return nil, fmt.Errorf("invalid start byte: %d (file size: %d)", sym.StartByte, len(content))
		}

		// Insert before start of symbol
		newContent := make([]byte, 0, len(content)+len(input.Body))
		newContent = append(newContent, content[:sym.StartByte]...)
		newContent = append(newContent, []byte(input.Body)...)
		newContent = append(newContent, content[sym.StartByte:]...)

		return newContent, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to modify file: %w", err)
	}

	// Re-index file
	if err := cmtm.reindexFile(ctx, sym.ProjectID, sym.FilePath, sym.AbsolutePath, sym.Language); err != nil {
		slog.Warn("failed to reindex file after insertion", "file", sym.FilePath, "error", err)
	}

	result := map[string]interface{}{
		"message":          "Code inserted before symbol successfully",
		"file_path":        sym.FilePath,
		"reference_symbol": sym.NamePath,
		"inserted_before_line": sym.StartLine,
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: string(resultJSON),
		},
	}, false), nil
}

func (cmtm *CodeManipulationToolManager) codeDeleteSymbolHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeDeleteSymbolInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	// Resolve symbol
	sym, err := cmtm.resolveSymbol(ctx, input.ProjectID, input.SymbolID, input.NamePath, input.RelativePath)
	if err != nil {
		return nil, err
	}

	// Store info for response before deletion
	deletedInfo := map[string]interface{}{
		"name_path":  sym.NamePath,
		"start_line": sym.StartLine,
		"end_line":   sym.EndLine,
	}

	// Modify file
	err = cmtm.modifyFile(sym.AbsolutePath, func(content []byte) ([]byte, error) {
		if sym.StartByte < 0 || sym.EndByte > len(content) || sym.StartByte > sym.EndByte {
			return nil, fmt.Errorf("invalid byte range: %d-%d (file size: %d)", sym.StartByte, sym.EndByte, len(content))
		}

		// Find the start of the line containing the symbol
		lineStart := sym.StartByte
		for lineStart > 0 && content[lineStart-1] != '\n' {
			lineStart--
		}

		// Find the end of the line containing the symbol end
		lineEnd := sym.EndByte
		for lineEnd < len(content) && content[lineEnd] != '\n' {
			lineEnd++
		}
		// Include the newline
		if lineEnd < len(content) {
			lineEnd++
		}

		// Remove the symbol (including whitespace on the lines)
		newContent := make([]byte, 0, len(content)-(lineEnd-lineStart))
		newContent = append(newContent, content[:lineStart]...)
		newContent = append(newContent, content[lineEnd:]...)

		// Clean up multiple empty lines
		newContent = []byte(strings.ReplaceAll(string(newContent), "\n\n\n", "\n\n"))

		return newContent, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to modify file: %w", err)
	}

	// Re-index file
	if err := cmtm.reindexFile(ctx, sym.ProjectID, sym.FilePath, sym.AbsolutePath, sym.Language); err != nil {
		slog.Warn("failed to reindex file after deletion", "file", sym.FilePath, "error", err)
	}

	result := map[string]interface{}{
		"message":        "Symbol deleted successfully",
		"file_path":      sym.FilePath,
		"deleted_symbol": deletedInfo,
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: string(resultJSON),
		},
	}, false), nil
}
