// Package mcp_tools provides code search MCP tools.
package mcp_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// ====== Input Types ======

// CodeGetSymbolsOverviewInput represents input for code_get_symbols_overview tool
type CodeGetSymbolsOverviewInput struct {
	ProjectID    string `json:"project_id" description:"The project ID to search in."`
	RelativePath string `json:"relative_path" description:"Relative path to the file within the project."`
	MaxResults   int    `json:"max_results,omitempty" description:"Maximum number of symbols to return. Default is 100."`
}

// CodeFindSymbolInput represents input for code_find_symbol tool
type CodeFindSymbolInput struct {
	ProjectID       string   `json:"project_id" description:"The project ID to search in."`
	NamePathPattern string   `json:"name_path_pattern" description:"Symbol name or path pattern. Use '/ClassName/method' for exact match, 'ClassName/method' for suffix match, or 'method' for simple name match."`
	RelativePath    string   `json:"relative_path,omitempty" description:"Restrict search to this file or directory."`
	Depth           int      `json:"depth,omitempty" description:"Include children up to this depth level (0=symbol only, 1=direct children, etc)."`
	IncludeBody     bool     `json:"include_body,omitempty" description:"Include source code in results."`
	IncludeKinds    []string `json:"include_kinds,omitempty" description:"Filter by symbol types (class, function, method, interface, etc)."`
	ExcludeKinds    []string `json:"exclude_kinds,omitempty" description:"Exclude these symbol types."`
	SubstringMatch  bool     `json:"substring_matching,omitempty" description:"Enable partial name matching."`
}

// CodeSearchSymbolsSemanticInput represents input for code_search_symbols_semantic tool
type CodeSearchSymbolsSemanticInput struct {
	ProjectID   string   `json:"project_id" description:"The project ID to search in."`
	Query       string   `json:"query" description:"Natural language query describing what you're looking for."`
	Limit       int      `json:"limit,omitempty" description:"Maximum number of results to return. Default is 10."`
	Languages   []string `json:"languages,omitempty" description:"Filter by programming languages (go, typescript, python, etc)."`
	SymbolTypes []string `json:"symbol_types,omitempty" description:"Filter by symbol types (class, function, method, etc)."`
}

// CodeSearchPatternInput represents input for code_search_pattern tool
type CodeSearchPatternInput struct {
	ProjectID     string   `json:"project_id" description:"The project ID to search in."`
	Pattern       string   `json:"pattern" description:"Text pattern or regex to search for in source code."`
	IsRegex       bool     `json:"is_regex,omitempty" description:"Treat pattern as regular expression."`
	Languages     []string `json:"languages,omitempty" description:"Filter by programming languages."`
	SymbolTypes   []string `json:"symbol_types,omitempty" description:"Filter by symbol types."`
	CaseSensitive bool     `json:"case_sensitive,omitempty" description:"Enable case-sensitive matching. Default is false."`
	Limit         int      `json:"limit,omitempty" description:"Maximum number of results. Default is 50."`
}

// CodeFindReferencesInput represents input for code_find_references tool
type CodeFindReferencesInput struct {
	ProjectID    string   `json:"project_id" description:"The project ID to search in."`
	SymbolID     string   `json:"symbol_id,omitempty" description:"ID of the symbol to find references for."`
	SymbolName   string   `json:"symbol_name,omitempty" description:"Name of the symbol (alternative to symbol_id)."`
	IncludeKinds []string `json:"include_kinds,omitempty" description:"Filter referencing symbols by type."`
	Limit        int      `json:"limit,omitempty" description:"Maximum number of references. Default is 50."`
}

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
	return nil
}

// ====== Tool Definitions ======

func (cstm *CodeSearchToolManager) codeGetSymbolsOverviewTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_get_symbols_overview", `Get a high-level overview of code symbols in a file.

Explanation: Lists top-level symbols (classes, functions, interfaces, etc.) in a specific file,
showing their names, types, line numbers, and signatures. Does not include source code bodies.
This should be your first tool when understanding a new file.

When to call: Use when you want to understand the structure of a file before diving into specific symbols.

Example arguments/values:
	project_id: "my-project"
	relative_path: "src/services/user.ts"
	max_results: 50
`, CodeGetSymbolsOverviewInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_get_symbols_overview", "err", err)
		return nil
	}
	return tool
}

func (cstm *CodeSearchToolManager) codeFindSymbolTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_find_symbol", `Find symbols by name or path pattern.

Explanation: Searches for symbols matching a name pattern. Supports three pattern types:
- Absolute: "/ClassName/methodName" - Exact match of full path
- Suffix: "ClassName/methodName" - Match by path suffix
- Simple: "methodName" - Match by name anywhere

Use depth > 0 to also retrieve children (e.g., methods of a class).
Use substring_matching for partial name matches.

When to call: Use when you know the approximate name of a symbol and want to find its definition.

Example arguments/values:
	project_id: "my-project"
	name_path_pattern: "UserService/createUser"
	include_body: true
	depth: 1
`, CodeFindSymbolInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_find_symbol", "err", err)
		return nil
	}
	return tool
}

func (cstm *CodeSearchToolManager) codeSearchSymbolsSemanticTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_search_symbols_semantic", `Search for code symbols using natural language.

Explanation: Uses AI embeddings to find code semantically related to your query.
The search understands meaning, so "function that validates email" will find 
validateEmail, checkEmailFormat, isValidEmail, etc.

When to call: Use when you don't know the exact name but can describe what you're looking for.

Example arguments/values:
	project_id: "my-project"
	query: "function that handles user authentication"
	limit: 10
	symbol_types: ["function", "method"]
`, CodeSearchSymbolsSemanticInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_search_symbols_semantic", "err", err)
		return nil
	}
	return tool
}

func (cstm *CodeSearchToolManager) codeSearchPatternTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_search_pattern", `Search for patterns in source code.

Explanation: Searches for text patterns or regex within symbol source code.
Useful for finding specific code constructs, API usages, or string literals.

When to call: Use when you need to find specific text patterns in code,
like TODO comments, specific function calls, or string values.

Example arguments/values:
	project_id: "my-project"
	pattern: "TODO|FIXME|HACK"
	is_regex: true
	case_sensitive: false
`, CodeSearchPatternInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_search_pattern", "err", err)
		return nil
	}
	return tool
}

func (cstm *CodeSearchToolManager) codeFindReferencesTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_find_references", `Find references to a symbol in the codebase.

Explanation: Searches for usages of a symbol by looking for its name in other symbols' source code.
Note: This is a text-based search, not a true LSP reference finder.

When to call: Use when you want to find where a function, class, or variable is used.

Example arguments/values:
	project_id: "my-project"
	symbol_name: "UserService"
	include_kinds: ["method", "function"]
`, CodeFindReferencesInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_find_references", "err", err)
		return nil
	}
	return tool
}

// ====== Tool Handlers ======

func (cstm *CodeSearchToolManager) codeGetSymbolsOverviewHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeGetSymbolsOverviewInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.ProjectID == "" || input.RelativePath == "" {
		return nil, fmt.Errorf("project_id and relative_path are required")
	}

	if input.MaxResults <= 0 {
		input.MaxResults = 100
	}

	// Get storage with code capabilities
	codeStorage, ok := cstm.storage.(interface {
		FindSymbolsByFile(ctx context.Context, projectID, filePath string) ([]storage.CodeSymbol, error)
		GetCodeFile(ctx context.Context, projectID, filePath string) (*storage.CodeFile, error)
	})
	if !ok {
		return nil, fmt.Errorf("storage does not support code operations")
	}

	// Get file info
	file, err := codeStorage.GetCodeFile(ctx, input.ProjectID, input.RelativePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	if file == nil {
		return nil, fmt.Errorf("file not found: %s", input.RelativePath)
	}

	// Get symbols
	symbols, err := codeStorage.FindSymbolsByFile(ctx, input.ProjectID, input.RelativePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get symbols: %w", err)
	}

	// Filter to top-level symbols only and limit
	topLevelSymbols := make([]map[string]interface{}, 0)
	for _, sym := range symbols {
		if sym.ParentID == nil && len(topLevelSymbols) < input.MaxResults {
			topLevelSymbols = append(topLevelSymbols, map[string]interface{}{
				"name":       sym.Name,
				"type":       sym.SymbolType,
				"name_path":  sym.NamePath,
				"start_line": sym.StartLine,
				"end_line":   sym.EndLine,
				"signature":  sym.Signature,
			})
		}
	}

	result := map[string]interface{}{
		"file_path": input.RelativePath,
		"language":  file.Language,
		"symbols":   topLevelSymbols,
		"count":     len(topLevelSymbols),
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

func (cstm *CodeSearchToolManager) codeFindSymbolHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeFindSymbolInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.ProjectID == "" || input.NamePathPattern == "" {
		return nil, fmt.Errorf("project_id and name_path_pattern are required")
	}

	// Get storage with code capabilities
	codeStorage, ok := cstm.storage.(interface {
		Query(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error)
		FindChildSymbols(ctx context.Context, projectID, parentID string) ([]storage.CodeSymbol, error)
	})
	if !ok {
		return nil, fmt.Errorf("storage does not support code operations")
	}

	// Build query based on pattern type
	var query string
	params := map[string]interface{}{
		"project_id": input.ProjectID,
	}

	pattern := input.NamePathPattern

	if strings.HasPrefix(pattern, "/") {
		// Absolute match
		query = `SELECT * FROM code_symbols WHERE project_id = $project_id AND name_path = $pattern`
		params["pattern"] = pattern
	} else if strings.Contains(pattern, "/") {
		// Suffix match
		query = `SELECT * FROM code_symbols WHERE project_id = $project_id AND name_path CONTAINS $pattern`
		params["pattern"] = pattern
	} else {
		// Simple name match
		if input.SubstringMatch {
			query = `SELECT * FROM code_symbols WHERE project_id = $project_id AND name CONTAINS $pattern`
		} else {
			query = `SELECT * FROM code_symbols WHERE project_id = $project_id AND name = $pattern`
		}
		params["pattern"] = pattern
	}

	// Add file/dir filter
	if input.RelativePath != "" {
		if strings.HasSuffix(input.RelativePath, "/") {
			query += ` AND file_path CONTAINS $path`
		} else {
			query += ` AND file_path = $path`
		}
		params["path"] = input.RelativePath
	}

	// Add kind filters
	if len(input.IncludeKinds) > 0 {
		query += ` AND symbol_type IN $include_kinds`
		params["include_kinds"] = input.IncludeKinds
	}
	if len(input.ExcludeKinds) > 0 {
		query += ` AND symbol_type NOT IN $exclude_kinds`
		params["exclude_kinds"] = input.ExcludeKinds
	}

	query += ` ORDER BY file_path, start_line LIMIT 50;`

	results, err := codeStorage.Query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search symbols: %w", err)
	}

	// Process results
	symbols := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		sym := map[string]interface{}{
			"name":       r["name"],
			"type":       r["symbol_type"],
			"name_path":  r["name_path"],
			"file_path":  r["file_path"],
			"language":   r["language"],
			"start_line": r["start_line"],
			"end_line":   r["end_line"],
			"signature":  r["signature"],
		}

		if input.IncludeBody {
			sym["source_code"] = r["source_code"]
		}

		// Get children if depth > 0
		if input.Depth > 0 {
			if id, ok := r["id"].(string); ok {
				children, _ := cstm.getSymbolChildren(ctx, codeStorage, input.ProjectID, id, input.Depth-1, input.IncludeBody)
				if len(children) > 0 {
					sym["children"] = children
				}
			}
		}

		symbols = append(symbols, sym)
	}

	result := map[string]interface{}{
		"pattern": input.NamePathPattern,
		"symbols": symbols,
		"count":   len(symbols),
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

// Helper to recursively get children
func (cstm *CodeSearchToolManager) getSymbolChildren(ctx context.Context, codeStorage interface {
	FindChildSymbols(ctx context.Context, projectID, parentID string) ([]storage.CodeSymbol, error)
}, projectID, parentID string, remainingDepth int, includeBody bool) ([]map[string]interface{}, error) {
	children, err := codeStorage.FindChildSymbols(ctx, projectID, parentID)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(children))
	for _, child := range children {
		sym := map[string]interface{}{
			"name":       child.Name,
			"type":       child.SymbolType,
			"name_path":  child.NamePath,
			"start_line": child.StartLine,
			"end_line":   child.EndLine,
			"signature":  child.Signature,
		}

		if includeBody && child.SourceCode != nil {
			sym["source_code"] = *child.SourceCode
		}

		// Recurse if more depth
		if remainingDepth > 0 {
			grandchildren, _ := cstm.getSymbolChildren(ctx, codeStorage, projectID, child.ID, remainingDepth-1, includeBody)
			if len(grandchildren) > 0 {
				sym["children"] = grandchildren
			}
		}

		result = append(result, sym)
	}

	return result, nil
}

func (cstm *CodeSearchToolManager) codeSearchSymbolsSemanticHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeSearchSymbolsSemanticInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.ProjectID == "" || input.Query == "" {
		return nil, fmt.Errorf("project_id and query are required")
	}

	if input.Limit <= 0 {
		input.Limit = 10
	}

	// Get storage with vector search capabilities
	codeStorage, ok := cstm.storage.(interface {
		SearchSymbolsBySimilarity(ctx context.Context, projectID string, embedding []float32, symbolTypes []treesitter.SymbolType, limit int) ([]storage.CodeSymbolSearchResult, error)
	})
	if !ok {
		return nil, fmt.Errorf("storage does not support semantic search")
	}

	// Generate embedding for query
	embedding, err := cstm.embedder.EmbedQuery(ctx, input.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Convert symbol types
	var symbolTypes []treesitter.SymbolType
	for _, t := range input.SymbolTypes {
		symbolTypes = append(symbolTypes, treesitter.SymbolType(t))
	}

	// Search
	results, err := codeStorage.SearchSymbolsBySimilarity(ctx, input.ProjectID, embedding, symbolTypes, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Format results
	symbols := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		sym := map[string]interface{}{
			"name":       r.Symbol.Name,
			"type":       r.Symbol.SymbolType,
			"name_path":  r.Symbol.NamePath,
			"file_path":  r.Symbol.FilePath,
			"language":   r.Symbol.Language,
			"start_line": r.Symbol.StartLine,
			"end_line":   r.Symbol.EndLine,
			"signature":  r.Symbol.Signature,
			"similarity": fmt.Sprintf("%.4f", r.Similarity),
		}
		symbols = append(symbols, sym)
	}

	result := map[string]interface{}{
		"query":   input.Query,
		"symbols": symbols,
		"count":   len(symbols),
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

func (cstm *CodeSearchToolManager) codeSearchPatternHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeSearchPatternInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.ProjectID == "" || input.Pattern == "" {
		return nil, fmt.Errorf("project_id and pattern are required")
	}

	if input.Limit <= 0 {
		input.Limit = 50
	}

	// Get storage
	codeStorage, ok := cstm.storage.(interface {
		Query(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error)
	})
	if !ok {
		return nil, fmt.Errorf("storage does not support query operations")
	}

	// Build query - we'll filter in Go for regex support
	query := `SELECT * FROM code_symbols WHERE project_id = $project_id AND source_code != NONE`
	params := map[string]interface{}{
		"project_id": input.ProjectID,
	}

	// Add language filter
	if len(input.Languages) > 0 {
		query += ` AND language IN $languages`
		params["languages"] = input.Languages
	}

	// Add type filter
	if len(input.SymbolTypes) > 0 {
		query += ` AND symbol_type IN $symbol_types`
		params["symbol_types"] = input.SymbolTypes
	}

	query += ` LIMIT 500;` // Get more to filter in Go

	results, err := codeStorage.Query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to query symbols: %w", err)
	}

	// Compile pattern if regex
	var re *regexp.Regexp
	if input.IsRegex {
		flags := ""
		if !input.CaseSensitive {
			flags = "(?i)"
		}
		re, err = regexp.Compile(flags + input.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	// Filter and format results
	matches := make([]map[string]interface{}, 0)
	for _, r := range results {
		sourceCode, ok := r["source_code"].(string)
		if !ok || sourceCode == "" {
			continue
		}

		var matched bool
		var matchLocations []string

		if input.IsRegex {
			allMatches := re.FindAllString(sourceCode, 5)
			if len(allMatches) > 0 {
				matched = true
				matchLocations = allMatches
			}
		} else {
			searchCode := sourceCode
			searchPattern := input.Pattern
			if !input.CaseSensitive {
				searchCode = strings.ToLower(sourceCode)
				searchPattern = strings.ToLower(input.Pattern)
			}
			if strings.Contains(searchCode, searchPattern) {
				matched = true
				matchLocations = []string{input.Pattern}
			}
		}

		if matched {
			matches = append(matches, map[string]interface{}{
				"name":       r["name"],
				"type":       r["symbol_type"],
				"name_path":  r["name_path"],
				"file_path":  r["file_path"],
				"language":   r["language"],
				"start_line": r["start_line"],
				"end_line":   r["end_line"],
				"matches":    matchLocations,
			})

			if len(matches) >= input.Limit {
				break
			}
		}
	}

	result := map[string]interface{}{
		"pattern":  input.Pattern,
		"is_regex": input.IsRegex,
		"matches":  matches,
		"count":    len(matches),
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

func (cstm *CodeSearchToolManager) codeFindReferencesHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeFindReferencesInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	if input.SymbolID == "" && input.SymbolName == "" {
		return nil, fmt.Errorf("either symbol_id or symbol_name is required")
	}

	if input.Limit <= 0 {
		input.Limit = 50
	}

	// Get storage
	codeStorage, ok := cstm.storage.(interface {
		Query(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error)
		GetCodeSymbol(ctx context.Context, projectID, namePath string) (*storage.CodeSymbol, error)
	})
	if !ok {
		return nil, fmt.Errorf("storage does not support code operations")
	}

	// Get the target symbol name
	targetName := input.SymbolName
	if input.SymbolID != "" {
		// Get symbol by ID to get its name
		query := `SELECT * FROM $symbol_id;`
		results, err := codeStorage.Query(ctx, query, map[string]interface{}{"symbol_id": input.SymbolID})
		if err != nil || len(results) == 0 {
			return nil, fmt.Errorf("symbol not found: %s", input.SymbolID)
		}
		if name, ok := results[0]["name"].(string); ok {
			targetName = name
		}
	}

	if targetName == "" {
		return nil, fmt.Errorf("could not determine symbol name")
	}

	// Search for references in source code
	query := `SELECT * FROM code_symbols WHERE project_id = $project_id AND source_code CONTAINS $name AND name != $name`
	params := map[string]interface{}{
		"project_id": input.ProjectID,
		"name":       targetName,
	}

	if len(input.IncludeKinds) > 0 {
		query += ` AND symbol_type IN $kinds`
		params["kinds"] = input.IncludeKinds
	}

	query += fmt.Sprintf(` LIMIT %d;`, input.Limit)

	results, err := codeStorage.Query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search references: %w", err)
	}

	// Format results
	references := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		ref := map[string]interface{}{
			"name":       r["name"],
			"type":       r["symbol_type"],
			"name_path":  r["name_path"],
			"file_path":  r["file_path"],
			"language":   r["language"],
			"start_line": r["start_line"],
			"end_line":   r["end_line"],
		}

		// Try to find the exact line(s) with the reference
		if sourceCode, ok := r["source_code"].(string); ok {
			lines := strings.Split(sourceCode, "\n")
			refLines := make([]int, 0)
			for i, line := range lines {
				if strings.Contains(line, targetName) {
					refLines = append(refLines, i+1)
				}
			}
			if len(refLines) > 0 {
				ref["reference_lines"] = refLines
			}
		}

		references = append(references, ref)
	}

	result := map[string]interface{}{
		"target_symbol": targetName,
		"references":    references,
		"count":         len(references),
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
