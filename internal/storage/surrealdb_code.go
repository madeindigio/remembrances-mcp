// Package storage provides code indexing storage operations for SurrealDB.
package storage

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// CodeProject represents a stored code project
type CodeProject struct {
	ID             string                      `json:"id"`
	ProjectID      string                      `json:"project_id"`
	Name           string                      `json:"name"`
	RootPath       string                      `json:"root_path"`
	LanguageStats  map[treesitter.Language]int `json:"language_stats"`
	LastIndexedAt  *time.Time                  `json:"last_indexed_at"`
	IndexingStatus treesitter.IndexingStatus   `json:"indexing_status"`
	CreatedAt      time.Time                   `json:"created_at"`
	UpdatedAt      time.Time                   `json:"updated_at"`
}

// CodeFile represents a stored code file
type CodeFile struct {
	ID           string             `json:"id"`
	ProjectID    string             `json:"project_id"`
	FilePath     string             `json:"file_path"`
	Language     treesitter.Language `json:"language"`
	FileHash     string             `json:"file_hash"`
	SymbolsCount int                `json:"symbols_count"`
	IndexedAt    time.Time          `json:"indexed_at"`
}

// CodeSymbol represents a stored code symbol
type CodeSymbol struct {
	ID         string                 `json:"id"`
	ProjectID  string                 `json:"project_id"`
	FilePath   string                 `json:"file_path"`
	Language   treesitter.Language    `json:"language"`
	SymbolType treesitter.SymbolType  `json:"symbol_type"`
	Name       string                 `json:"name"`
	NamePath   string                 `json:"name_path"`
	StartLine  int                    `json:"start_line"`
	EndLine    int                    `json:"end_line"`
	StartByte  int                    `json:"start_byte"`
	EndByte    int                    `json:"end_byte"`
	SourceCode *string                `json:"source_code,omitempty"`
	Signature  *string                `json:"signature,omitempty"`
	DocString  *string                `json:"doc_string,omitempty"`
	Embedding  []float32              `json:"embedding,omitempty"`
	ParentID   *string                `json:"parent_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// CodeSymbolSearchResult represents a symbol search result with similarity
type CodeSymbolSearchResult struct {
	Symbol     *CodeSymbol `json:"symbol"`
	Similarity float64     `json:"similarity"`
}

// CodeIndexingJob represents a stored indexing job
type CodeIndexingJob struct {
	ID           string                     `json:"id"`
	ProjectID    string                     `json:"project_id"`
	ProjectPath  string                     `json:"project_path"`
	Status       treesitter.IndexingStatus  `json:"status"`
	Progress     float64                    `json:"progress"`
	FilesTotal   int                        `json:"files_total"`
	FilesIndexed int                        `json:"files_indexed"`
	StartedAt    time.Time                  `json:"started_at"`
	CompletedAt  *time.Time                 `json:"completed_at,omitempty"`
	Error        *string                    `json:"error,omitempty"`
}

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

// ===== SYMBOL OPERATIONS =====

// SaveCodeSymbol saves or updates a code symbol
func (s *SurrealDBStorage) SaveCodeSymbol(ctx context.Context, symbol *treesitter.CodeSymbol) error {
	query := `
		UPSERT code_symbols SET
			project_id = $project_id,
			file_path = $file_path,
			language = $language,
			symbol_type = $symbol_type,
			name = $name,
			name_path = $name_path,
			start_line = $start_line,
			end_line = $end_line,
			start_byte = $start_byte,
			end_byte = $end_byte,
			source_code = $source_code,
			signature = $signature,
			doc_string = $doc_string,
			embedding = $embedding,
			parent_id = $parent_id,
			metadata = $metadata,
			updated_at = time::now()
		WHERE project_id = $project_id AND name_path = $name_path;
	`

	var sourceCode, signature, docString *string
	if symbol.SourceCode != "" {
		sourceCode = &symbol.SourceCode
	}
	if symbol.Signature != "" {
		signature = &symbol.Signature
	}
	if symbol.DocString != "" {
		docString = &symbol.DocString
	}

	var embedding interface{}
	if len(symbol.Embedding) > 0 {
		embedding = symbol.Embedding
	}

	params := map[string]interface{}{
		"project_id":  symbol.ProjectID,
		"file_path":   symbol.FilePath,
		"language":    string(symbol.Language),
		"symbol_type": string(symbol.SymbolType),
		"name":        symbol.Name,
		"name_path":   symbol.NamePath,
		"start_line":  symbol.StartLine,
		"end_line":    symbol.EndLine,
		"start_byte":  symbol.StartByte,
		"end_byte":    symbol.EndByte,
		"source_code": sourceCode,
		"signature":   signature,
		"doc_string":  docString,
		"embedding":   embedding,
		"parent_id":   symbol.ParentID,
		"metadata":    symbol.Metadata,
	}

	_, err := s.query(ctx, query, params)
	return err
}

// SaveCodeSymbols saves multiple symbols in batch
func (s *SurrealDBStorage) SaveCodeSymbols(ctx context.Context, symbols []*treesitter.CodeSymbol) error {
	for _, symbol := range symbols {
		if err := s.SaveCodeSymbol(ctx, symbol); err != nil {
			return fmt.Errorf("failed to save symbol %s: %w", symbol.NamePath, err)
		}
	}
	return nil
}

// GetCodeSymbol retrieves a symbol by project and name path
func (s *SurrealDBStorage) GetCodeSymbol(ctx context.Context, projectID, namePath string) (*CodeSymbol, error) {
	query := `SELECT * FROM code_symbols WHERE project_id = $project_id AND name_path = $name_path LIMIT 1;`
	params := map[string]interface{}{
		"project_id": projectID,
		"name_path":  namePath,
	}

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get symbol: %w", err)
	}

	symbols, err := decodeResult[CodeSymbol](result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode symbol: %w", err)
	}

	if len(symbols) == 0 {
		return nil, nil
	}

	return &symbols[0], nil
}

// FindSymbolsByName finds symbols by name (partial or exact match)
func (s *SurrealDBStorage) FindSymbolsByName(ctx context.Context, projectID, name string, symbolTypes []treesitter.SymbolType, limit int) ([]CodeSymbol, error) {
	query := `
		SELECT * FROM code_symbols 
		WHERE project_id = $project_id 
		AND name CONTAINS $name
	`

	params := map[string]interface{}{
		"project_id": projectID,
		"name":       name,
	}

	if len(symbolTypes) > 0 {
		types := make([]string, len(symbolTypes))
		for i, t := range symbolTypes {
			types[i] = string(t)
		}
		query += ` AND symbol_type IN $types`
		params["types"] = types
	}

	query += fmt.Sprintf(` LIMIT %d;`, limit)

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find symbols: %w", err)
	}

	return decodeResult[CodeSymbol](result)
}

// FindSymbolsByFile retrieves all symbols in a file
func (s *SurrealDBStorage) FindSymbolsByFile(ctx context.Context, projectID, filePath string) ([]CodeSymbol, error) {
	query := `SELECT * FROM code_symbols WHERE project_id = $project_id AND file_path = $file_path ORDER BY start_line ASC;`
	params := map[string]interface{}{
		"project_id": projectID,
		"file_path":  filePath,
	}

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find symbols: %w", err)
	}

	return decodeResult[CodeSymbol](result)
}

// FindChildSymbols retrieves child symbols of a parent
func (s *SurrealDBStorage) FindChildSymbols(ctx context.Context, projectID, parentID string) ([]CodeSymbol, error) {
	query := `SELECT * FROM code_symbols WHERE project_id = $project_id AND parent_id = $parent_id ORDER BY start_line ASC;`
	params := map[string]interface{}{
		"project_id": projectID,
		"parent_id":  parentID,
	}

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find child symbols: %w", err)
	}

	return decodeResult[CodeSymbol](result)
}

// SearchSymbolsBySimilarity performs semantic search on code symbols
func (s *SurrealDBStorage) SearchSymbolsBySimilarity(ctx context.Context, projectID string, queryEmbedding []float32, symbolTypes []treesitter.SymbolType, limit int) ([]CodeSymbolSearchResult, error) {
	query := `
		SELECT *, vector::similarity::cosine(embedding, $embedding) AS similarity 
		FROM code_symbols 
		WHERE project_id = $project_id 
		AND embedding != NONE
	`

	params := map[string]interface{}{
		"project_id": projectID,
		"embedding":  queryEmbedding,
	}

	if len(symbolTypes) > 0 {
		types := make([]string, len(symbolTypes))
		for i, t := range symbolTypes {
			types[i] = string(t)
		}
		query += ` AND symbol_type IN $types`
		params["types"] = types
	}

	query += fmt.Sprintf(` ORDER BY similarity DESC LIMIT %d;`, limit)

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search symbols: %w", err)
	}

	// Decode results with similarity
	type symbolWithSimilarity struct {
		CodeSymbol
		Similarity float64 `json:"similarity"`
	}

	symbols, err := decodeResult[symbolWithSimilarity](result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	results := make([]CodeSymbolSearchResult, len(symbols))
	for i, s := range symbols {
		sym := s.CodeSymbol
		results[i] = CodeSymbolSearchResult{
			Symbol:     &sym,
			Similarity: s.Similarity,
		}
	}

	return results, nil
}

// DeleteSymbolsByFile deletes all symbols in a file
func (s *SurrealDBStorage) DeleteSymbolsByFile(ctx context.Context, projectID, filePath string) error {
	query := `DELETE FROM code_symbols WHERE project_id = $project_id AND file_path = $file_path;`
	params := map[string]interface{}{
		"project_id": projectID,
		"file_path":  filePath,
	}

	_, err := s.query(ctx, query, params)
	return err
}

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

// GetProjectStats returns statistics for a code project
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
