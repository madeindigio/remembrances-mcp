// Package mcp_tools provides code search MCP tools.
// This file contains handler implementations for code search tools.
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

// ====== Handler Implementations ======

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

// getSymbolChildren recursively gets children of a symbol
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

func (cstm *CodeSearchToolManager) codeHybridSearchHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeHybridSearchInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.ProjectID == "" || input.Query == "" {
		return nil, fmt.Errorf("project_id and query are required")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}

	// Generate query embedding
	queryEmbedding, err := cstm.embedder.EmbedQuery(ctx, input.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Get storage with code operations
	codeStorage, ok := cstm.storage.(interface {
		SearchSymbolsBySimilarity(ctx context.Context, projectID string, queryEmbedding []float32, symbolTypes []treesitter.SymbolType, limit int) ([]storage.CodeSymbolSearchResult, error)
		SearchChunksBySimilarity(ctx context.Context, projectID string, queryEmbedding []float32, limit int) ([]storage.CodeChunkSearchResult, error)
	})
	if !ok {
		return nil, fmt.Errorf("storage does not support code search operations")
	}

	// Convert symbol types
	var symbolTypes []treesitter.SymbolType
	for _, t := range input.SymbolTypes {
		symbolTypes = append(symbolTypes, treesitter.SymbolType(t))
	}

	// Search symbols
	symbolResults, err := codeStorage.SearchSymbolsBySimilarity(ctx, input.ProjectID, queryEmbedding, symbolTypes, limit*2)
	if err != nil {
		return nil, fmt.Errorf("failed to search symbols: %w", err)
	}

	// Optionally search chunks
	var chunkResults []storage.CodeChunkSearchResult
	if input.IncludeChunks {
		chunkResults, err = codeStorage.SearchChunksBySimilarity(ctx, input.ProjectID, queryEmbedding, limit)
		if err != nil {
			slog.Warn("failed to search chunks", "error", err)
		}
	}

	// Compile path pattern if provided
	var pathRegex *regexp.Regexp
	if input.PathPattern != "" {
		// Convert glob-like pattern to regex
		pattern := strings.ReplaceAll(input.PathPattern, "**", ".*")
		pattern = strings.ReplaceAll(pattern, "*", "[^/]*")
		pattern = "^" + pattern
		pathRegex, err = regexp.Compile(pattern)
		if err != nil {
			slog.Warn("invalid path pattern, ignoring", "pattern", input.PathPattern, "error", err)
			pathRegex = nil
		}
	}

	// Filter and format results
	type hybridResult struct {
		Source     string  `json:"source"` // "symbol" or "chunk"
		Name       string  `json:"name"`
		NamePath   string  `json:"name_path,omitempty"`
		SymbolType string  `json:"symbol_type"`
		FilePath   string  `json:"file_path"`
		Language   string  `json:"language"`
		StartLine  int     `json:"start_line"`
		EndLine    int     `json:"end_line"`
		Similarity float64 `json:"similarity"`
		ChunkIndex *int    `json:"chunk_index,omitempty"`
		Preview    string  `json:"preview,omitempty"`
	}

	var results []hybridResult

	// Process symbol results
	for _, sr := range symbolResults {
		sym := sr.Symbol
		if sym == nil {
			continue
		}

		// Apply language filter
		if len(input.Languages) > 0 {
			match := false
			for _, lang := range input.Languages {
				if string(sym.Language) == lang {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		// Apply path filter
		if pathRegex != nil && !pathRegex.MatchString(sym.FilePath) {
			continue
		}

		// Create preview
		preview := ""
		if sym.Signature != nil {
			preview = *sym.Signature
		} else if sym.SourceCode != nil && len(*sym.SourceCode) < 200 {
			preview = *sym.SourceCode
		}

		results = append(results, hybridResult{
			Source:     "symbol",
			Name:       sym.Name,
			NamePath:   sym.NamePath,
			SymbolType: string(sym.SymbolType),
			FilePath:   sym.FilePath,
			Language:   string(sym.Language),
			StartLine:  sym.StartLine,
			EndLine:    sym.EndLine,
			Similarity: sr.Similarity,
			Preview:    preview,
		})
	}

	// Process chunk results
	for _, cr := range chunkResults {
		chunk := cr.Chunk
		if chunk == nil {
			continue
		}

		// Apply language filter
		if len(input.Languages) > 0 {
			match := false
			for _, lang := range input.Languages {
				if chunk.Language == lang {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		// Apply symbol type filter
		if len(input.SymbolTypes) > 0 {
			match := false
			for _, t := range input.SymbolTypes {
				if chunk.SymbolType == t {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		// Apply path filter
		if pathRegex != nil && !pathRegex.MatchString(chunk.FilePath) {
			continue
		}

		// Create preview (truncate chunk content)
		preview := chunk.Content
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}

		chunkIdx := chunk.ChunkIndex
		results = append(results, hybridResult{
			Source:     "chunk",
			Name:       chunk.SymbolName,
			SymbolType: chunk.SymbolType,
			FilePath:   chunk.FilePath,
			Language:   chunk.Language,
			StartLine:  chunk.StartOffset, // Approximate
			EndLine:    chunk.EndOffset,   // Approximate
			Similarity: cr.Similarity,
			ChunkIndex: &chunkIdx,
			Preview:    preview,
		})
	}

	// Sort by similarity (already sorted from DB, but chunks may need re-sorting)
	// For simplicity, we'll just limit
	if len(results) > limit {
		results = results[:limit]
	}

	output := map[string]interface{}{
		"query":   input.Query,
		"results": results,
		"count":   len(results),
		"filters": map[string]interface{}{
			"languages":      input.Languages,
			"symbol_types":   input.SymbolTypes,
			"path_pattern":   input.PathPattern,
			"include_chunks": input.IncludeChunks,
		},
	}

	resultJSON, err := json.MarshalIndent(output, "", "  ")
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
