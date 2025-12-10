// Package mcp_tools provides code indexing MCP tools.
// This file contains file watcher tool definitions and handlers.
package mcp_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// ====== Watch Tool Definitions ======

func (ctm *CodeToolManager) codeActivateProjectWatchTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_activate_project_watch",
		`Activate file monitoring for a code project. Automatically deactivates any previously watched project. Only ONE project can be monitored at a time.`,
		CodeActivateProjectWatchInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_activate_project_watch", "err", err)
		return nil
	}
	return tool
}

func (ctm *CodeToolManager) codeDeactivateProjectWatchTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_deactivate_project_watch",
		`Stop file monitoring for a code project. If no project_id is specified, deactivates the currently watched project.`,
		CodeDeactivateProjectWatchInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_deactivate_project_watch", "err", err)
		return nil
	}
	return tool
}

func (ctm *CodeToolManager) codeGetWatchStatusTool() *protocol.Tool {
	tool, err := protocol.NewTool("code_get_watch_status",
		`Get the current file monitoring status. Returns the currently watched project and watcher_enabled status for all or a specific project.`,
		CodeGetWatchStatusInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "code_get_watch_status", "err", err)
		return nil
	}
	return tool
}

// ====== Watch Tool Handlers ======

func (ctm *CodeToolManager) codeActivateProjectWatchHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeActivateProjectWatchInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if input.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	if ctm.watcherManager == nil {
		return nil, fmt.Errorf("watcher manager not available")
	}

	outdatedCount, previousProject, err := ctm.watcherManager.ActivateProject(ctx, input.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to activate project watch: %w", err)
	}

	result := map[string]interface{}{
		"success":              true,
		"message":              fmt.Sprintf("File monitoring activated for project %s", input.ProjectID),
		"project_id":           input.ProjectID,
		"outdated_files_found": outdatedCount,
	}

	if previousProject != "" {
		result["previous_project"] = previousProject
		result["message"] = fmt.Sprintf("File monitoring activated for project %s (deactivated %s)", input.ProjectID, previousProject)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{Type: "text", Text: MarshalTOON(result)},
	}, false), nil
}

func (ctm *CodeToolManager) codeDeactivateProjectWatchHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeDeactivateProjectWatchInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if ctm.watcherManager == nil {
		return nil, fmt.Errorf("watcher manager not available")
	}

	deactivatedProject, err := ctm.watcherManager.DeactivateProject(ctx, input.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate project watch: %w", err)
	}

	var result map[string]interface{}
	if deactivatedProject == "" {
		result = map[string]interface{}{
			"success": true,
			"message": "No project was being watched",
		}
	} else {
		result = map[string]interface{}{
			"success":             true,
			"message":             fmt.Sprintf("File monitoring deactivated for project %s", deactivatedProject),
			"deactivated_project": deactivatedProject,
		}
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{Type: "text", Text: MarshalTOON(result)},
	}, false), nil
}

func (ctm *CodeToolManager) codeGetWatchStatusHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CodeGetWatchStatusInput
	if err := json.Unmarshal(req.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if ctm.watcherManager == nil {
		return nil, fmt.Errorf("watcher manager not available")
	}

	var result map[string]interface{}

	if input.ProjectID != "" {
		// Get status for specific project
		status, err := ctm.watcherManager.GetProjectWatchStatus(ctx, input.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("failed to get watch status: %w", err)
		}

		result = map[string]interface{}{
			"active_project": ctm.watcherManager.GetActiveProject(),
			"project_status": status,
		}
	} else {
		// Get status for all projects
		statuses, err := ctm.watcherManager.GetAllWatchStatus(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get watch status: %w", err)
		}

		result = map[string]interface{}{
			"active_project": ctm.watcherManager.GetActiveProject(),
			"projects":       statuses,
		}
	}

	if projects, ok := result["projects"].([]map[string]interface{}); ok && len(projects) == 0 {
		suggestions := ctm.FindProjectAlternatives(ctx, input.ProjectID)
		payload := CreateEmptyResultTOON("No watched projects", suggestions)
		return protocol.NewCallToolResult([]protocol.Content{
			&protocol.TextContent{Type: "text", Text: payload},
		}, false), nil
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{Type: "text", Text: MarshalTOON(result)},
	}, false), nil
}
