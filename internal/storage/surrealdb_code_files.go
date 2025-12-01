// Package storage provides code indexing storage operations for SurrealDB.
// This file contains file-related operations for code indexing.
package storage

import (
	"context"
	"fmt"

	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// ===== FILE OPERATIONS =====

// SaveCodeFile saves or updates a code file
func (s *SurrealDBStorage) SaveCodeFile(ctx context.Context, file *treesitter.CodeFile) error {
	query := `
		UPSERT code_files SET
			project_id = $project_id,
			file_path = $file_path,
			language = $language,
			file_hash = $file_hash,
			symbols_count = $symbols_count,
			indexed_at = time::now()
		WHERE project_id = $project_id AND file_path = $file_path;
	`

	params := map[string]interface{}{
		"project_id":    file.ProjectID,
		"file_path":     file.FilePath,
		"language":      string(file.Language),
		"file_hash":     file.FileHash,
		"symbols_count": file.SymbolsCount,
	}

	_, err := s.query(ctx, query, params)
	return err
}

// GetCodeFile retrieves a code file by project and path
func (s *SurrealDBStorage) GetCodeFile(ctx context.Context, projectID, filePath string) (*CodeFile, error) {
	query := `SELECT * FROM code_files WHERE project_id = $project_id AND file_path = $file_path LIMIT 1;`
	params := map[string]interface{}{
		"project_id": projectID,
		"file_path":  filePath,
	}

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	files, err := decodeResult[CodeFile](result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode file: %w", err)
	}

	if len(files) == 0 {
		return nil, nil
	}

	return &files[0], nil
}

// ListCodeFiles lists all files in a project
func (s *SurrealDBStorage) ListCodeFiles(ctx context.Context, projectID string) ([]CodeFile, error) {
	query := `SELECT * FROM code_files WHERE project_id = $project_id ORDER BY file_path ASC;`
	params := map[string]interface{}{"project_id": projectID}

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return decodeResult[CodeFile](result)
}

// DeleteCodeFile deletes a file and all its symbols
func (s *SurrealDBStorage) DeleteCodeFile(ctx context.Context, projectID, filePath string) error {
	// Delete symbols first, then file
	queries := []string{
		`DELETE FROM code_symbols WHERE project_id = $project_id AND file_path = $file_path;`,
		`DELETE FROM code_files WHERE project_id = $project_id AND file_path = $file_path;`,
	}
	params := map[string]interface{}{
		"project_id": projectID,
		"file_path":  filePath,
	}

	for _, query := range queries {
		if _, err := s.query(ctx, query, params); err != nil {
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}

	return nil
}
