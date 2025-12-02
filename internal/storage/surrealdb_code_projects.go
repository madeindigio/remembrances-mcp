// Package storage provides code indexing storage operations for SurrealDB.
// This file contains project-related operations for code indexing.
package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// ===== PROJECT OPERATIONS =====

// CreateCodeProject creates or updates a code project
// Uses INSERT ON DUPLICATE KEY UPDATE for atomic upsert operation
func (s *SurrealDBStorage) CreateCodeProject(ctx context.Context, project *treesitter.CodeProject) error {
	slog.Debug("CreateCodeProject called", "project_id", project.ProjectID, "status", project.IndexingStatus)

	// Convert last_indexed_at to ISO string format for SurrealDB compatibility
	var lastIndexedAtStr interface{}
	if project.LastIndexedAt != nil {
		lastIndexedAtStr = project.LastIndexedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	// Convert language_stats map to a plain map[string]interface{} for SurrealDB
	var langStats interface{}
	if len(project.LanguageStats) > 0 {
		ls := make(map[string]interface{})
		for lang, count := range project.LanguageStats {
			ls[string(lang)] = count
		}
		langStats = ls
	} else {
		langStats = nil
	}

	// Check if project exists to get watcher_enabled value
	existingProject, _ := s.GetCodeProject(ctx, project.ProjectID)
	watcherEnabled := false
	if existingProject != nil {
		watcherEnabled = existingProject.WatcherEnabled
	}

	// Use INSERT with ON DUPLICATE KEY UPDATE for atomic upsert
	// This handles the unique index on project_id properly
	slog.Debug("INSERT ON DUPLICATE KEY UPDATE", "project_id", project.ProjectID, "status", project.IndexingStatus)
	query := `
		INSERT INTO code_projects {
			project_id: $project_id,
			name: $name,
			root_path: $root_path,
			language_stats: $language_stats,
			last_indexed_at: $last_indexed_at,
			indexing_status: $indexing_status,
			watcher_enabled: $watcher_enabled
		}
		ON DUPLICATE KEY UPDATE
			name = $input.name,
			root_path = $input.root_path,
			language_stats = $input.language_stats,
			last_indexed_at = $input.last_indexed_at,
			indexing_status = $input.indexing_status,
			updated_at = time::now()
	`
	params := map[string]interface{}{
		"project_id":      project.ProjectID,
		"name":            project.Name,
		"root_path":       project.RootPath,
		"language_stats":  langStats,
		"last_indexed_at": lastIndexedAtStr,
		"indexing_status": string(project.IndexingStatus),
		"watcher_enabled": watcherEnabled,
	}

	result, err := s.query(ctx, query, params)
	if err != nil {
		slog.Error("INSERT ON DUPLICATE KEY UPDATE failed", "error", err)
		return fmt.Errorf("failed to upsert project: %w", err)
	}

	// Log result for debugging
	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		slog.Debug("INSERT result", "status", queryResult.Status, "result_count", len(queryResult.Result))
	}

	return nil
}

// GetCodeProject retrieves a code project by ID
func (s *SurrealDBStorage) GetCodeProject(ctx context.Context, projectID string) (*CodeProject, error) {
	query := `SELECT * FROM code_projects WHERE project_id = $project_id LIMIT 1;`
	params := map[string]interface{}{"project_id": projectID}

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	projects, err := decodeResult[CodeProject](result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode project: %w", err)
	}

	if len(projects) == 0 {
		return nil, nil
	}

	return &projects[0], nil
}

// ListCodeProjects lists all code projects
func (s *SurrealDBStorage) ListCodeProjects(ctx context.Context) ([]CodeProject, error) {
	query := `SELECT * FROM code_projects ORDER BY name ASC;`

	result, err := s.query(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return decodeResult[CodeProject](result)
}

// UpdateProjectStatus updates the indexing status of a project
func (s *SurrealDBStorage) UpdateProjectStatus(ctx context.Context, projectID string, status treesitter.IndexingStatus) error {
	query := `
		UPDATE code_projects SET 
			indexing_status = $status,
			updated_at = time::now()
		WHERE project_id = $project_id;
	`
	params := map[string]interface{}{
		"project_id": projectID,
		"status":     string(status),
	}

	_, err := s.query(ctx, query, params)
	return err
}

// UpdateProjectWatcher updates the watcher_enabled field of a project
func (s *SurrealDBStorage) UpdateProjectWatcher(ctx context.Context, projectID string, enabled bool) error {
	query := `
		UPDATE code_projects SET 
			watcher_enabled = $enabled,
			updated_at = time::now()
		WHERE project_id = $project_id;
	`
	params := map[string]interface{}{
		"project_id": projectID,
		"enabled":    enabled,
	}

	_, err := s.query(ctx, query, params)
	return err
}

// DeleteCodeProject deletes a project and all its files and symbols
func (s *SurrealDBStorage) DeleteCodeProject(ctx context.Context, projectID string) error {
	// Delete in order: symbols, files, project
	queries := []string{
		`DELETE FROM code_symbols WHERE project_id = $project_id;`,
		`DELETE FROM code_files WHERE project_id = $project_id;`,
		`DELETE FROM code_indexing_jobs WHERE project_id = $project_id;`,
		`DELETE FROM code_projects WHERE project_id = $project_id;`,
	}
	params := map[string]interface{}{"project_id": projectID}

	for _, query := range queries {
		if _, err := s.query(ctx, query, params); err != nil {
			return fmt.Errorf("failed to delete project resources: %w", err)
		}
	}

	return nil
}
