// Package storage provides code indexing storage operations for SurrealDB.
// This file contains indexing job and project stats operations.
package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// ===== INDEXING JOB OPERATIONS =====

// CreateIndexingJob creates a new indexing job
func (s *SurrealDBStorage) CreateIndexingJob(ctx context.Context, job *treesitter.IndexingJob) (string, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return "", err
	}

	query := fmt.Sprintf(`
		CREATE code_indexing_jobs CONTENT {
			%sproject_id: $project_id,
			project_path: $project_path,
			status: $status,
			progress: $progress,
			files_total: $files_total,
			files_indexed: $files_indexed,
			started_at: time::now()
		};
	`, t.contentFields())

	params := t.params(map[string]interface{}{
		"project_id":    job.ProjectID,
		"project_path":  job.ProjectPath,
		"status":        string(job.Status),
		"progress":      job.Progress,
		"files_total":   job.FilesTotal,
		"files_indexed": job.FilesIndexed,
	})

	result, err := s.query(ctx, query, params)
	if err != nil {
		return "", fmt.Errorf("failed to create job: %w", err)
	}

	jobs, err := decodeResult[CodeIndexingJob](result)
	if err != nil || len(jobs) == 0 {
		return "", fmt.Errorf("failed to get created job ID")
	}

	return jobs[0].ID, nil
}

// UpdateIndexingJob updates an indexing job's progress
func (s *SurrealDBStorage) UpdateIndexingJob(ctx context.Context, jobID string, status treesitter.IndexingStatus, progress float64, filesIndexed int, err *string) error {
	t, scopeErr := s.scope(ctx, "")
	if scopeErr != nil {
		return scopeErr
	}

	// The job id is cast rather than interpolated: a bare string parameter in
	// the FROM or UPDATE position is not a record link and matches nothing.
	assignments := `status = $status,
			progress = $progress,
			files_indexed = $files_indexed,
			error = $error`
	if status == treesitter.IndexingStatusCompleted || status == treesitter.IndexingStatusFailed {
		assignments += `,
			completed_at = time::now()`
	}

	query := fmt.Sprintf(`
		UPDATE code_indexing_jobs SET
			%s%s
		%s;
	`, t.updateSet(), assignments, t.where("id = type::record($job_id)"))

	params := t.params(map[string]interface{}{
		"job_id":        jobID,
		"status":        string(status),
		"progress":      progress,
		"files_indexed": filesIndexed,
		"error":         err,
	})

	_, queryErr := s.query(ctx, query, params)
	return queryErr
}

// GetIndexingJob retrieves an indexing job by ID
func (s *SurrealDBStorage) GetIndexingJob(ctx context.Context, jobID string) (*CodeIndexingJob, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM code_indexing_jobs %s LIMIT 1;",
		t.where("id = type::record($job_id)"))
	params := t.params(map[string]interface{}{"job_id": jobID})

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	jobs, err := decodeResult[CodeIndexingJob](result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode job: %w", err)
	}

	if len(jobs) == 0 {
		return nil, nil
	}

	return &jobs[0], nil
}

// ListActiveIndexingJobs lists all active indexing jobs
func (s *SurrealDBStorage) ListActiveIndexingJobs(ctx context.Context) ([]CodeIndexingJob, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM code_indexing_jobs %s ORDER BY started_at DESC;",
		t.where("status IN ['pending', 'in_progress']"))

	result, err := s.query(ctx, query, t.params(nil))
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	return decodeResult[CodeIndexingJob](result)
}

// GetCodeProjectStats returns statistics for a code project
func (s *SurrealDBStorage) GetCodeProjectStats(ctx context.Context, projectID string) (map[string]interface{}, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})
	params := t.params(map[string]interface{}{"project_id": projectID})
	where := t.where("project_id = $project_id")

	// Get file count
	fileQuery := fmt.Sprintf("SELECT count() AS count FROM code_files %s GROUP ALL;", where)
	result, err := s.query(ctx, fileQuery, params)
	if err == nil {
		if counts, _ := decodeResult[map[string]interface{}](result); len(counts) > 0 {
			stats["files_count"] = counts[0]["count"]
		}
	}

	// Get symbol count
	symbolQuery := fmt.Sprintf("SELECT count() AS count FROM code_symbols %s GROUP ALL;", where)
	result, err = s.query(ctx, symbolQuery, params)
	if err == nil {
		if counts, _ := decodeResult[map[string]interface{}](result); len(counts) > 0 {
			stats["symbols_count"] = counts[0]["count"]
		}
	}

	// Get symbols by type
	typeQuery := fmt.Sprintf("SELECT symbol_type, count() AS count FROM code_symbols %s GROUP BY symbol_type;", where)
	result, err = s.query(ctx, typeQuery, params)
	if err == nil {
		if typeCounts, _ := decodeResult[map[string]interface{}](result); len(typeCounts) > 0 {
			stats["symbols_by_type"] = typeCounts
		}
	}

	// Get languages
	langQuery := fmt.Sprintf("SELECT language, count() AS count FROM code_files %s GROUP BY language;", where)
	result, err = s.query(ctx, langQuery, params)
	if err == nil {
		if langCounts, _ := decodeResult[map[string]interface{}](result); len(langCounts) > 0 {
			stats["files_by_language"] = langCounts
		}
	}

	slog.Debug("Project stats retrieved", "project_id", projectID, "stats", stats)
	return stats, nil
}
