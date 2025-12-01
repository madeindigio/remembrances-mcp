// Package storage provides code indexing storage operations for SurrealDB.
// This file contains indexing job and project stats operations.
package storage

import (
	"context"
	"fmt"
	"log"

	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// ===== INDEXING JOB OPERATIONS =====

// CreateIndexingJob creates a new indexing job
func (s *SurrealDBStorage) CreateIndexingJob(ctx context.Context, job *treesitter.IndexingJob) (string, error) {
	query := `
		CREATE code_indexing_jobs SET
			project_id = $project_id,
			project_path = $project_path,
			status = $status,
			progress = $progress,
			files_total = $files_total,
			files_indexed = $files_indexed,
			started_at = time::now();
	`

	params := map[string]interface{}{
		"project_id":    job.ProjectID,
		"project_path":  job.ProjectPath,
		"status":        string(job.Status),
		"progress":      job.Progress,
		"files_total":   job.FilesTotal,
		"files_indexed": job.FilesIndexed,
	}

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
	query := `
		UPDATE $job_id SET
			status = $status,
			progress = $progress,
			files_indexed = $files_indexed,
			error = $error
	`

	params := map[string]interface{}{
		"job_id":        jobID,
		"status":        string(status),
		"progress":      progress,
		"files_indexed": filesIndexed,
		"error":         err,
	}

	if status == treesitter.IndexingStatusCompleted || status == treesitter.IndexingStatusFailed {
		query += `, completed_at = time::now()`
	}

	query += ";"

	_, queryErr := s.query(ctx, query, params)
	return queryErr
}

// GetIndexingJob retrieves an indexing job by ID
func (s *SurrealDBStorage) GetIndexingJob(ctx context.Context, jobID string) (*CodeIndexingJob, error) {
	query := `SELECT * FROM $job_id;`
	params := map[string]interface{}{"job_id": jobID}

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
	query := `SELECT * FROM code_indexing_jobs WHERE status IN ['pending', 'in_progress'] ORDER BY started_at DESC;`

	result, err := s.query(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	return decodeResult[CodeIndexingJob](result)
}

// GetCodeProjectStats returns statistics for a code project
func (s *SurrealDBStorage) GetCodeProjectStats(ctx context.Context, projectID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get file count
	fileQuery := `SELECT count() AS count FROM code_files WHERE project_id = $project_id GROUP ALL;`
	result, err := s.query(ctx, fileQuery, map[string]interface{}{"project_id": projectID})
	if err == nil {
		if counts, _ := decodeResult[map[string]interface{}](result); len(counts) > 0 {
			stats["files_count"] = counts[0]["count"]
		}
	}

	// Get symbol count
	symbolQuery := `SELECT count() AS count FROM code_symbols WHERE project_id = $project_id GROUP ALL;`
	result, err = s.query(ctx, symbolQuery, map[string]interface{}{"project_id": projectID})
	if err == nil {
		if counts, _ := decodeResult[map[string]interface{}](result); len(counts) > 0 {
			stats["symbols_count"] = counts[0]["count"]
		}
	}

	// Get symbols by type
	typeQuery := `SELECT symbol_type, count() AS count FROM code_symbols WHERE project_id = $project_id GROUP BY symbol_type;`
	result, err = s.query(ctx, typeQuery, map[string]interface{}{"project_id": projectID})
	if err == nil {
		if typeCounts, _ := decodeResult[map[string]interface{}](result); len(typeCounts) > 0 {
			stats["symbols_by_type"] = typeCounts
		}
	}

	// Get languages
	langQuery := `SELECT language, count() AS count FROM code_files WHERE project_id = $project_id GROUP BY language;`
	result, err = s.query(ctx, langQuery, map[string]interface{}{"project_id": projectID})
	if err == nil {
		if langCounts, _ := decodeResult[map[string]interface{}](result); len(langCounts) > 0 {
			stats["files_by_language"] = langCounts
		}
	}

	log.Printf("Project %s stats: %+v", projectID, stats)
	return stats, nil
}
