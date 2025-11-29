// Package mcp_tools provides code indexing MCP tools.
package mcp_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/madeindigio/remembrances-mcp/internal/indexer"
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
