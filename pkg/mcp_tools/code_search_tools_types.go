// Package mcp_tools provides code search MCP tools.
// This file contains input type definitions for code search tools.
package mcp_tools

// ====== Input Types for Code Search Tools ======

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

// CodeHybridSearchInput represents input for code_hybrid_search tool
type CodeHybridSearchInput struct {
	ProjectID     string   `json:"project_id" description:"The project ID to search in."`
	Query         string   `json:"query" description:"Natural language query for semantic search."`
	Languages     []string `json:"languages,omitempty" description:"Filter by programming languages (go, typescript, python, etc)."`
	SymbolTypes   []string `json:"symbol_types,omitempty" description:"Filter by symbol types (class, function, method, interface, etc)."`
	PathPattern   string   `json:"path_pattern,omitempty" description:"Filter by file path pattern (e.g., 'src/auth/**')."`
	IncludeChunks bool     `json:"include_chunks,omitempty" description:"Search in code chunks for better large-symbol coverage."`
	Limit         int      `json:"limit,omitempty" description:"Maximum number of results. Default is 20."`
}
