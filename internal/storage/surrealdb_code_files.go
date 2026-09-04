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
	t, err := s.scope(ctx, "")
	if err != nil {
		return err
	}

	params := t.params(map[string]interface{}{
		"project_id":    file.ProjectID,
		"file_path":     file.FilePath,
		"language":      string(file.Language),
		"file_hash":     file.FileHash,
		"symbols_count": file.SymbolsCount,
	})

	// Check if the file is already indexed in one of the caller's scopes.
	existsQuery := fmt.Sprintf("SELECT id FROM code_files %s",
		t.where("project_id = $project_id", "file_path = $file_path"))
	existsResult, err := s.query(ctx, existsQuery, params)
	if err != nil {
		return fmt.Errorf("failed to check existing file: %w", err)
	}

	isNewFile := true
	if existsResult != nil && len(*existsResult) > 0 {
		queryResult := (*existsResult)[0]
		if queryResult.Status == "OK" && len(queryResult.Result) > 0 {
			isNewFile = false
		}
	}

	if isNewFile {
		query := fmt.Sprintf(`
			CREATE code_files CONTENT {
				%sproject_id: $project_id,
				file_path: $file_path,
				language: $language,
				file_hash: $file_hash,
				symbols_count: $symbols_count,
				indexed_at: time::now()
			}
		`, t.contentFields())
		if _, err := s.query(ctx, query, params); err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
	} else {
		query := fmt.Sprintf(`
			UPDATE code_files SET
				%slanguage = $language,
				file_hash = $file_hash,
				symbols_count = $symbols_count,
				indexed_at = time::now()
			%s
		`, t.updateSet(), t.where("project_id = $project_id", "file_path = $file_path"))
		if _, err := s.query(ctx, query, params); err != nil {
			return fmt.Errorf("failed to update file: %w", err)
		}
	}

	return nil
}

// GetCodeFile retrieves a code file by project and path
func (s *SurrealDBStorage) GetCodeFile(ctx context.Context, projectID, filePath string) (*CodeFile, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM code_files %s LIMIT 1;",
		t.where("project_id = $project_id", "file_path = $file_path"))
	params := t.params(map[string]interface{}{
		"project_id": projectID,
		"file_path":  filePath,
	})

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
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM code_files %s ORDER BY file_path ASC;",
		t.where("project_id = $project_id"))
	params := t.params(map[string]interface{}{"project_id": projectID})

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return decodeResult[CodeFile](result)
}

// DeleteCodeFile deletes a file and all its symbols
func (s *SurrealDBStorage) DeleteCodeFile(ctx context.Context, projectID, filePath string) error {
	t, err := s.scope(ctx, "")
	if err != nil {
		return err
	}

	// Delete chunks and symbols first, then the file. See DeleteCodeProject
	// for why the code index deletes rather than invalidates.
	queries := []string{
		fmt.Sprintf("DELETE FROM code_chunks %s;", t.whereAny("project_id = $project_id", "file_path = $file_path")),
		fmt.Sprintf("DELETE FROM code_symbols %s;", t.whereAny("project_id = $project_id", "file_path = $file_path")),
		fmt.Sprintf("DELETE FROM code_files %s;", t.whereAny("project_id = $project_id", "file_path = $file_path")),
	}
	params := t.params(map[string]interface{}{
		"project_id": projectID,
		"file_path":  filePath,
	})

	for _, query := range queries {
		if _, err := s.query(ctx, query, params); err != nil {
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}

	return nil
}
