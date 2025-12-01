// Package mcp_tools provides input types for code manipulation tools.
package mcp_tools

// ====== Input Types for Code Manipulation Tools ======

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
