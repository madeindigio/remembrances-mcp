// Package storage provides code indexing storage operations for SurrealDB.
// This file contains project-related operations for code indexing.
package storage

import (
	"context"
	"fmt"

	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// ===== PROJECT OPERATIONS =====

// CreateCodeProject creates or updates a code project
func (s *SurrealDBStorage) CreateCodeProject(ctx context.Context, project *treesitter.CodeProject) error {
	query := `
		UPSERT code_projects SET
			project_id = $project_id,
			name = $name,
			root_path = $root_path,
			language_stats = $language_stats,
			last_indexed_at = $last_indexed_at,
			indexing_status = $indexing_status,
			updated_at = time::now()
		WHERE project_id = $project_id;
	`

	params := map[string]interface{}{
		"project_id":      project.ProjectID,
		"name":            project.Name,
		"root_path":       project.RootPath,
		"language_stats":  project.LanguageStats,
		"last_indexed_at": project.LastIndexedAt,
		"indexing_status": string(project.IndexingStatus),
	}

	_, err := s.query(ctx, query, params)
	return err
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
