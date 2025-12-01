// Package mcp_tools provides code indexing MCP tools.
// This file contains input type definitions for code indexing tools.
package mcp_tools

// ====== Input Types for Code Indexing Tools ======

// CodeIndexProjectInput represents input for code_index_project tool
type CodeIndexProjectInput struct {
	ProjectPath string   `json:"project_path" description:"Absolute path to the project directory to index."`
	ProjectName string   `json:"project_name,omitempty" description:"Human-readable name for the project. If omitted, uses the directory name."`
	Languages   []string `json:"languages,omitempty" description:"List of programming languages to index (e.g., ['go', 'typescript']). If omitted, indexes all supported languages."`
}

// CodeIndexStatusInput represents input for code_index_status tool
type CodeIndexStatusInput struct {
	JobID string `json:"job_id,omitempty" description:"The job ID returned by code_index_project. If omitted, lists all active jobs."`
}

// CodeListProjectsInput represents input for code_list_projects tool (no inputs required)
type CodeListProjectsInput struct{}

// CodeDeleteProjectInput represents input for code_delete_project tool
type CodeDeleteProjectInput struct {
	ProjectID string `json:"project_id" description:"The project ID to delete."`
}

// CodeReindexFileInput represents input for code_reindex_file tool
type CodeReindexFileInput struct {
	ProjectID string `json:"project_id" description:"The project ID containing the file."`
	FilePath  string `json:"file_path" description:"Relative path to the file within the project."`
}

// CodeGetProjectStatsInput represents input for code_get_project_stats tool
type CodeGetProjectStatsInput struct {
	ProjectID string `json:"project_id" description:"The project ID to get statistics for."`
}

// CodeGetFileSymbolsInput represents input for code_get_file_symbols tool
type CodeGetFileSymbolsInput struct {
	ProjectID    string `json:"project_id" description:"The project ID containing the file."`
	RelativePath string `json:"relative_path" description:"Relative path to the file within the project."`
	IncludeBody  bool   `json:"include_body,omitempty" description:"Whether to include the source code body of each symbol."`
}

// CodeActivateProjectWatchInput represents input for code_activate_project_watch tool
type CodeActivateProjectWatchInput struct {
	ProjectID string `json:"project_id" description:"The project ID to start monitoring for file changes."`
}

// CodeDeactivateProjectWatchInput represents input for code_deactivate_project_watch tool
type CodeDeactivateProjectWatchInput struct {
	ProjectID string `json:"project_id,omitempty" description:"The project ID to stop monitoring. If omitted, deactivates the current watched project."`
}

// CodeGetWatchStatusInput represents input for code_get_watch_status tool
type CodeGetWatchStatusInput struct {
	ProjectID string `json:"project_id,omitempty" description:"Query status for a specific project. If omitted, returns status for all projects."`
}

// SymbolNode represents a symbol in the hierarchical tree for file symbols display
type SymbolNode struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	NamePath   string        `json:"name_path"`
	SymbolType string        `json:"symbol_type"`
	StartLine  int           `json:"start_line"`
	EndLine    int           `json:"end_line"`
	Signature  *string       `json:"signature,omitempty"`
	DocString  *string       `json:"doc_string,omitempty"`
	SourceCode *string       `json:"source_code,omitempty"`
	Children   []*SymbolNode `json:"children,omitempty"`
}
