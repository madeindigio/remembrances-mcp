// Package mcp_tools provides code indexing MCP tools.
// This file contains the CodeToolManager, tool definitions, and core handlers.
// Input types are in code_indexing_tools_types.go
// Watch tools are in code_watch_tools.go
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

// ====== Tool Manager ======

// CodeToolManager extends ToolManager with code indexing capabilities
type CodeToolManager struct {
	*ToolManager
	jobManager     *indexer.JobManager
	watcherManager *indexer.WatcherManager
}

// NewCodeToolManager creates a new code tool manager
func NewCodeToolManager(tm *ToolManager, jm *indexer.JobManager, wm *indexer.WatcherManager) *CodeToolManager {
	return &CodeToolManager{
		ToolManager:    tm,
		jobManager:     jm,
		watcherManager: wm,
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
	// Register watch tools only if watcher manager is available
	if ctm.watcherManager != nil {
		if err := reg("code_activate_project_watch", ctm.codeActivateProjectWatchTool(), ctm.codeActivateProjectWatchHandler); err != nil {
			return err
		}
		if err := reg("code_deactivate_project_watch", ctm.codeDeactivateProjectWatchTool(), ctm.codeDeactivateProjectWatchHandler); err != nil {
			return err
		}
		if err := reg("code_get_watch_status", ctm.codeGetWatchStatusTool(), ctm.codeGetWatchStatusHandler); err != nil {
			return err
		}
	}
	return nil
}

// ====== Tool Definitions ======

func (ctm *CodeToolManager) codeIndexProjectTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_index_project", `Start indexing a code project for semantic search. Use how_to_use("code_index_project") for details.`, CodeIndexProjectInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_index_project", "err", err)
		return nil
	}
	return tool
}

func (ctm *CodeToolManager) codeIndexStatusTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_index_status", `Check indexing job status. Use how_to_use("code_index_status") for details.`, CodeIndexStatusInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_index_status", "err", err)
		return nil
	}
	return tool
}

func (ctm *CodeToolManager) codeListProjectsTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_list_projects", `List all indexed code projects. Use how_to_use("code_list_projects") for details.`, CodeListProjectsInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_list_projects", "err", err)
		return nil
	}
	return tool
}

func (ctm *CodeToolManager) codeDeleteProjectTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_delete_project", `Delete an indexed project. Use how_to_use("code_delete_project") for details.`, CodeDeleteProjectInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_delete_project", "err", err)
		return nil
	}
	return tool
}

func (ctm *CodeToolManager) codeReindexFileTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_reindex_file", `Re-index a single file. Use how_to_use("code_reindex_file") for details.`, CodeReindexFileInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_reindex_file", "err", err)
		return nil
	}
	return tool
}

func (ctm *CodeToolManager) codeGetProjectStatsTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_get_project_stats", `Get project statistics. Use how_to_use("code_get_project_stats") for details.`, CodeGetProjectStatsInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_get_project_stats", "err", err)
		return nil
	}
	return tool
}

func (ctm *CodeToolManager) codeGetFileSymbolsTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_get_file_symbols", `Get all symbols from a file. Use how_to_use("code_get_file_symbols") for details.`, CodeGetFileSymbolsInput{})
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

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{Type: "text", Text: MarshalTOON(result)},
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

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{Type: "text", Text: MarshalTOON(result)},
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

	if len(results) == 0 {
		suggestions := ctm.FindProjectAlternatives(ctx, "")
		payload := CreateEmptyResultTOON("No code projects indexed", suggestions)
		return protocol.NewCallToolResult([]protocol.Content{
			&protocol.TextContent{Type: "text", Text: payload},
		}, false), nil
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{Type: "text", Text: MarshalTOON(result)},
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

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{Type: "text", Text: MarshalTOON(result)},
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

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{Type: "text", Text: MarshalTOON(result)},
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
		"indexing_status": string(project.IndexingStatus),
		"language_stats":  project.LanguageStats,
	}
	if project.LastIndexedAt != nil {
		result["last_indexed_at"] = project.LastIndexedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	// Merge stats
	for k, v := range stats {
		result[k] = v
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{Type: "text", Text: MarshalTOON(result)},
	}, false), nil
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

	if len(symbols) == 0 {
		suggestions := ctm.FindProjectAlternatives(ctx, input.ProjectID)
		yamlText := CreateEmptyResultTOON(
			fmt.Sprintf("No symbols found in file '%s' for project '%s'", input.RelativePath, input.ProjectID),
			suggestions,
		)
		return protocol.NewCallToolResult([]protocol.Content{
			&protocol.TextContent{Type: "text", Text: yamlText},
		}, false), nil
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{Type: "text", Text: MarshalTOON(result)},
	}, false), nil
}
