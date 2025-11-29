// Package mcp_tools provides code indexing MCP tools.
package mcp_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/madeindigio/remembrances-mcp/internal/indexer"
	"github.com/madeindigio/remembrances-mcp/internal/storage"
)

// ====== Input Types ======

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

// ====== Tool Manager Extension ======

// CodeToolManager extends ToolManager with code indexing capabilities
type CodeToolManager struct {
	*ToolManager
	jobManager *indexer.JobManager
}

// NewCodeToolManager creates a new code tool manager
func NewCodeToolManager(tm *ToolManager, jm *indexer.JobManager) *CodeToolManager {
	return &CodeToolManager{
		ToolManager: tm,
		jobManager:  jm,
	}
}

// RegisterCodeTools registers all code indexing tools
func (ctm *CodeToolManager) RegisterCodeTools(reg func(string, *protocol.Tool, func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error) error {
	if err := reg("code_index_project", ctm.codeIndexProjectTool(), ctm.codeIndexProjectHandler); err != nil {
		return err
	}
	if err := reg("code_index_status", ctm.codeIndexStatusTool(), ctm.codeIndexStatusHandler); err != nil {
		return err
	}
	if err := reg("code_list_projects", ctm.codeListProjectsTool(), ctm.codeListProjectsHandler); err != nil {
		return err
	}
	if err := reg("code_delete_project", ctm.codeDeleteProjectTool(), ctm.codeDeleteProjectHandler); err != nil {
		return err
	}
	if err := reg("code_reindex_file", ctm.codeReindexFileTool(), ctm.codeReindexFileHandler); err != nil {
		return err
	}
	if err := reg("code_get_project_stats", ctm.codeGetProjectStatsTool(), ctm.codeGetProjectStatsHandler); err != nil {
		return err
	}
	if err := reg("code_get_file_symbols", ctm.codeGetFileSymbolsTool(), ctm.codeGetFileSymbolsHandler); err != nil {
		return err
	}
	return nil
}

// ====== Tool Definitions ======

func (ctm *CodeToolManager) codeIndexProjectTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_index_project", `Start indexing a code project for semantic search capabilities.

Explanation: Scans the specified directory for source code files, parses them using tree-sitter to extract
symbols (classes, functions, methods, etc.), generates embeddings for semantic search, and stores everything
in the database. The indexing runs asynchronously in the background.

When to call: Use when you want to enable semantic code search on a new project or codebase.
The project path should be an absolute path to a directory containing source code.

Example arguments/values:
	project_path: "/home/user/projects/my-app"
	project_name: "My Application"
	languages: ["go", "typescript", "python"]
`, CodeIndexProjectInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_index_project", "err", err)
		return nil
	}
	return tool
}

func (ctm *CodeToolManager) codeIndexStatusTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_index_status", `Check the status of an indexing job or list all active jobs.

Explanation: Returns information about indexing job progress, including files processed,
symbols found, and any errors encountered.

When to call: Use after starting an indexing job with code_index_project to monitor progress,
or call without job_id to see all active indexing jobs.

Example arguments/values:
	job_id: "job_1234567890"
`, CodeIndexStatusInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_index_status", "err", err)
		return nil
	}
	return tool
}

func (ctm *CodeToolManager) codeListProjectsTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_list_projects", `List all indexed code projects.

Explanation: Returns a list of all projects that have been indexed for code search,
including their names, paths, and indexing status.

When to call: Use to see what projects are available for code search queries.
`, CodeListProjectsInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_list_projects", "err", err)
		return nil
	}
	return tool
}

func (ctm *CodeToolManager) codeDeleteProjectTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_delete_project", `Delete an indexed project and all its data.

Explanation: Removes a project and all its indexed files and symbols from the database.
This cannot be undone.

When to call: Use when you no longer need a project indexed or want to free up space.

Example arguments/values:
	project_id: "www_projects_my-app"
`, CodeDeleteProjectInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_delete_project", "err", err)
		return nil
	}
	return tool
}

func (ctm *CodeToolManager) codeReindexFileTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_reindex_file", `Re-index a single file in a project.

Explanation: Updates the index for a specific file that may have changed.
Useful for keeping the index up to date after file modifications.

When to call: Use after modifying a source file to update its symbols in the index.

Example arguments/values:
	project_id: "www_projects_my-app"
	file_path: "src/main.go"
`, CodeReindexFileInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_reindex_file", "err", err)
		return nil
	}
	return tool
}

func (ctm *CodeToolManager) codeGetProjectStatsTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_get_project_stats", `Get detailed statistics for an indexed code project.

Explanation: Returns comprehensive statistics about an indexed project including file counts,
symbol counts, language breakdown, and symbol type distribution.

When to call: Use to get an overview of a project's codebase structure and size,
or to verify indexing completeness.

Example arguments/values:
	project_id: "www_projects_my-app"

Returns statistics like:
- Total files and symbols
- Files by programming language
- Symbols by type (class, function, method, etc.)
- Project indexing status
`, CodeGetProjectStatsInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_get_project_stats", "err", err)
		return nil
	}
	return tool
}

func (ctm *CodeToolManager) codeGetFileSymbolsTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_get_file_symbols", `Get all symbols from a specific file with hierarchical structure.

Explanation: Returns all code symbols (classes, functions, methods, etc.) found in a specific file,
organized in a hierarchical tree structure showing parent-child relationships.

When to call: Use to get a complete overview of a file's structure, similar to an IDE's outline view.
Set include_body to true if you need the actual source code for each symbol.

Example arguments/values:
	project_id: "www_projects_my-app"
	relative_path: "src/services/user_service.go"
	include_body: false
`, CodeGetFileSymbolsInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_get_file_symbols", "err", err)
		return nil
	}
	return tool
}

// ====== Tool Handlers ======

func (ctm *CodeToolManager) codeIndexProjectHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeIndexProjectInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.ProjectPath == "" {
		return nil, fmt.Errorf("project_path is required")
	}

	// TODO: Handle languages filter when implemented in JobManager

	job, err := ctm.jobManager.SubmitJob(input.ProjectPath, input.ProjectName)
	if err != nil {
		return nil, fmt.Errorf("failed to start indexing: %w", err)
	}

	result := map[string]interface{}{
		"message":    fmt.Sprintf("Indexing started for project: %s", input.ProjectPath),
		"job_id":     job.ID,
		"status":     string(job.Status),
		"created_at": job.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	resultJSON, err := json.Marshal(result)
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

func (ctm *CodeToolManager) codeIndexStatusHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeIndexStatusInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	var result interface{}

	if input.JobID != "" {
		// Get specific job status
		job, err := ctm.jobManager.GetJobStatus(input.JobID)
		if err != nil {
			return nil, err
		}

		result = map[string]interface{}{
			"job_id":        job.ID,
			"project_id":    job.ProjectID,
			"project_path":  job.ProjectPath,
			"status":        string(job.Status),
			"progress":      job.Progress,
			"files_total":   job.FilesTotal,
			"files_indexed": job.FilesIndexed,
			"symbols_found": job.SymbolsFound,
			"started_at":    job.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
			"error":         job.Error,
		}
		if job.CompletedAt != nil {
			result.(map[string]interface{})["completed_at"] = job.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
		}
	} else {
		// List all active jobs
		jobs := ctm.jobManager.ListActiveJobs()
		jobList := make([]map[string]interface{}, 0, len(jobs))

		for _, job := range jobs {
			jobInfo := map[string]interface{}{
				"job_id":        job.ID,
				"project_path":  job.ProjectPath,
				"status":        string(job.Status),
				"progress":      job.Progress,
				"files_indexed": job.FilesIndexed,
				"files_total":   job.FilesTotal,
			}
			jobList = append(jobList, jobInfo)
		}

		result = map[string]interface{}{
			"active_jobs": jobList,
			"count":       len(jobList),
		}
	}

	resultJSON, err := json.Marshal(result)
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

func (ctm *CodeToolManager) codeListProjectsHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	// Query code projects directly from storage
	results, err := ctm.storage.Query(ctx, "SELECT * FROM code_projects ORDER BY name ASC;", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	result := map[string]interface{}{
		"projects": results,
		"count":    len(results),
	}

	resultJSON, err := json.Marshal(result)
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

func (ctm *CodeToolManager) codeDeleteProjectHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeDeleteProjectInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	// Use indexer to delete project
	if err := ctm.jobManager.GetIndexer().DeleteProject(ctx, input.ProjectID); err != nil {
		return nil, fmt.Errorf("failed to delete project: %w", err)
	}

	result := map[string]interface{}{
		"message":    fmt.Sprintf("Project %s deleted successfully", input.ProjectID),
		"project_id": input.ProjectID,
	}

	resultJSON, err := json.Marshal(result)
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

func (ctm *CodeToolManager) codeReindexFileHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeReindexFileInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.ProjectID == "" || input.FilePath == "" {
		return nil, fmt.Errorf("project_id and file_path are required")
	}

	if err := ctm.jobManager.GetIndexer().ReindexFile(ctx, input.ProjectID, input.FilePath); err != nil {
		return nil, fmt.Errorf("failed to reindex file: %w", err)
	}

	result := map[string]interface{}{
		"message":    fmt.Sprintf("File %s reindexed successfully", input.FilePath),
		"project_id": input.ProjectID,
		"file_path":  input.FilePath,
	}

	resultJSON, err := json.Marshal(result)
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

func (ctm *CodeToolManager) codeGetProjectStatsHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeGetProjectStatsInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	// Get the code storage interface
	codeStorage, ok := ctm.storage.(interface {
		GetCodeProjectStats(ctx context.Context, projectID string) (map[string]interface{}, error)
		GetCodeProject(ctx context.Context, projectID string) (*storage.CodeProject, error)
	})
	if !ok {
		return nil, fmt.Errorf("storage does not support code operations")
	}

	// Get project info
	project, err := codeStorage.GetCodeProject(ctx, input.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return nil, fmt.Errorf("project not found: %s", input.ProjectID)
	}

	// Get stats
	stats, err := codeStorage.GetCodeProjectStats(ctx, input.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project stats: %w", err)
	}

	// Combine project info with stats
	result := map[string]interface{}{
		"project_id":      project.ProjectID,
		"name":            project.Name,
		"root_path":       project.RootPath,
		"indexing_status": project.IndexingStatus,
		"language_stats":  project.LanguageStats,
	}
	if project.LastIndexedAt != nil {
		result["last_indexed_at"] = project.LastIndexedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	// Merge stats
	for k, v := range stats {
		result[k] = v
	}

	resultJSON, err := json.Marshal(result)
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

// SymbolNode represents a symbol in the hierarchical tree
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

func (ctm *CodeToolManager) codeGetFileSymbolsHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeGetFileSymbolsInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.ProjectID == "" || input.RelativePath == "" {
		return nil, fmt.Errorf("project_id and relative_path are required")
	}

	// Get the code storage interface
	codeStorage, ok := ctm.storage.(interface {
		FindSymbolsByFile(ctx context.Context, projectID, filePath string) ([]storage.CodeSymbol, error)
	})
	if !ok {
		return nil, fmt.Errorf("storage does not support code operations")
	}

	// Get all symbols in the file
	symbols, err := codeStorage.FindSymbolsByFile(ctx, input.ProjectID, input.RelativePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file symbols: %w", err)
	}

	// Build hierarchical structure
	symbolMap := make(map[string]*SymbolNode)
	var roots []*SymbolNode

	// First pass: create all nodes
	for _, sym := range symbols {
		node := &SymbolNode{
			ID:         sym.ID,
			Name:       sym.Name,
			NamePath:   sym.NamePath,
			SymbolType: string(sym.SymbolType),
			StartLine:  sym.StartLine,
			EndLine:    sym.EndLine,
			Signature:  sym.Signature,
			DocString:  sym.DocString,
			Children:   []*SymbolNode{},
		}
		if input.IncludeBody {
			node.SourceCode = sym.SourceCode
		}
		symbolMap[sym.ID] = node
	}

	// Second pass: build hierarchy
	for _, sym := range symbols {
		node := symbolMap[sym.ID]
		if sym.ParentID != nil && *sym.ParentID != "" {
			if parent, ok := symbolMap[*sym.ParentID]; ok {
				parent.Children = append(parent.Children, node)
			} else {
				// Parent not found, treat as root
				roots = append(roots, node)
			}
		} else {
			roots = append(roots, node)
		}
	}

	result := map[string]interface{}{
		"project_id":    input.ProjectID,
		"relative_path": input.RelativePath,
		"symbols":       roots,
		"total_count":   len(symbols),
	}

	resultJSON, err := json.Marshal(result)
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
