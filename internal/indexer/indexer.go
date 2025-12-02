// Package indexer provides the main indexing service for code projects.
// This file contains the core Indexer struct and main indexing operations.
// Types are in indexer_types.go
// Embedding generation is in indexer_embeddings.go
// Progress tracking is in indexer_progress.go
// Chunking is in indexer_chunks.go
package indexer

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/embedder"
	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// Indexer is the main code indexing service
type Indexer struct {
	config   IndexerConfig
	storage  storage.FullStorage
	embedder embedder.Embedder
	parser   *treesitter.Parser
	walker   *treesitter.ASTWalker

	// For progress tracking
	mu       sync.RWMutex
	progress map[string]*IndexingProgress
}

// NewIndexer creates a new indexer instance
func NewIndexer(storage storage.FullStorage, embedder embedder.Embedder, config IndexerConfig) *Indexer {
	walkerConfig := treesitter.WalkerConfig{
		IncludeSourceCode: config.StoreSourceCode,
		MaxSymbolSize:     config.MaxSourceCodeLength,
	}

	return &Indexer{
		config:   config,
		storage:  storage,
		embedder: embedder,
		parser:   treesitter.NewParser(),
		walker:   treesitter.NewASTWalker(walkerConfig),
		progress: make(map[string]*IndexingProgress),
	}
}

// IndexProject indexes a code project
func (idx *Indexer) IndexProject(ctx context.Context, projectPath string, projectName string) (string, error) {
	// Normalize path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return "", fmt.Errorf("invalid project path: %w", err)
	}

	// Check path exists
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("cannot access project path: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project path is not a directory: %s", absPath)
	}

	// Generate project ID from path
	projectID := idx.generateProjectID(absPath)

	// Use directory name if no name provided
	if projectName == "" {
		projectName = filepath.Base(absPath)
	}

	// Initialize progress tracking
	idx.initProgress(projectID)

	// Create or update project record
	project := &treesitter.CodeProject{
		ProjectID:      projectID,
		Name:           projectName,
		RootPath:       absPath,
		LanguageStats:  make(map[treesitter.Language]int),
		IndexingStatus: treesitter.IndexingStatusInProgress,
	}

	if err := idx.storage.CreateCodeProject(ctx, project); err != nil {
		return "", fmt.Errorf("failed to create project: %w", err)
	}

	// Scan for files
	log.Printf("Scanning project: %s", absPath)
	scanResult, err := idx.config.Scanner.Scan(absPath)
	if err != nil {
		idx.setError(projectID, err)
		return projectID, fmt.Errorf("failed to scan project: %w", err)
	}

	log.Printf("Found %d files to index", scanResult.TotalFiles)
	idx.updateProgress(projectID, func(p *IndexingProgress) {
		p.FilesTotal = scanResult.TotalFiles
	})

	// Process files
	if err := idx.processFiles(ctx, projectID, absPath, scanResult.Files); err != nil {
		idx.setError(projectID, err)
		idx.storage.UpdateProjectStatus(ctx, projectID, treesitter.IndexingStatusFailed)
		return projectID, fmt.Errorf("indexing failed: %w", err)
	}

	// Update project with final stats
	now := time.Now()
	project.LastIndexedAt = &now
	project.IndexingStatus = treesitter.IndexingStatusCompleted
	project.LanguageStats = scanResult.GetLanguageStats()

	if err := idx.storage.CreateCodeProject(ctx, project); err != nil {
		log.Printf("Warning: failed to update project stats: %v", err)
	}

	// Explicitly update the project status to completed
	// This ensures the status is updated even if CreateCodeProject's upsert doesn't update it
	if err := idx.storage.UpdateProjectStatus(ctx, projectID, treesitter.IndexingStatusCompleted); err != nil {
		log.Printf("Warning: failed to update project status to completed: %v", err)
	}

	idx.updateProgress(projectID, func(p *IndexingProgress) {
		p.Status = treesitter.IndexingStatusCompleted
	})

	log.Printf("Indexing completed for project: %s", projectName)
	return projectID, nil
}

// processFiles processes all discovered files
func (idx *Indexer) processFiles(ctx context.Context, projectID, rootPath string, files []ScannedFile) error {
	// Create work channel
	fileChan := make(chan ScannedFile, len(files))
	errChan := make(chan error, idx.config.Concurrency)
	var wg sync.WaitGroup

	// Start workers - each with its own parser to avoid tree-sitter thread-safety issues
	for i := 0; i < idx.config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Create a parser per worker - tree-sitter parsers are NOT thread-safe
			workerParser := treesitter.NewParser()
			for file := range fileChan {
				if err := idx.processFileWithParser(ctx, projectID, rootPath, file, workerParser); err != nil {
					log.Printf("Error processing file %s: %v", file.RelPath, err)
					errChan <- err
				}
			}
		}()
	}

	// Send files to workers
	for _, file := range files {
		select {
		case <-ctx.Done():
			close(fileChan)
			return ctx.Err()
		case fileChan <- file:
		}
	}
	close(fileChan)

	// Wait for workers to finish
	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("encountered %d errors during indexing", len(errors))
	}

	return nil
}

// processFile processes a single source file using the shared parser (for single-threaded use)
func (idx *Indexer) processFile(ctx context.Context, projectID, rootPath string, file ScannedFile) error {
	return idx.processFileWithParser(ctx, projectID, rootPath, file, idx.parser)
}

// processFileWithParser processes a single source file with a specific parser instance
func (idx *Indexer) processFileWithParser(ctx context.Context, projectID, rootPath string, file ScannedFile, parser *treesitter.Parser) error {
	idx.updateProgress(projectID, func(p *IndexingProgress) {
		p.CurrentFile = file.RelPath
	})

	// Check if file has changed
	existingFile, err := idx.storage.GetCodeFile(ctx, projectID, file.RelPath)
	if err != nil {
		return fmt.Errorf("failed to check existing file: %w", err)
	}

	if existingFile != nil && existingFile.FileHash == file.Hash {
		// File hasn't changed, skip
		idx.updateProgress(projectID, func(p *IndexingProgress) {
			p.FilesIndexed++
		})
		return nil
	}

	// Read file content
	content, err := os.ReadFile(file.AbsPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse file using the provided parser instance
	tree, lang, err := parser.ParseFile(ctx, file.AbsPath)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Extract symbols
	symbols, err := idx.walker.ExtractSymbols(tree, content, lang, file.RelPath, projectID)
	if err != nil {
		return fmt.Errorf("failed to extract symbols: %w", err)
	}

	// Delete old symbols for this file
	if existingFile != nil {
		if err := idx.storage.DeleteSymbolsByFile(ctx, projectID, file.RelPath); err != nil {
			log.Printf("Warning: failed to delete old symbols: %v", err)
		}
	}

	// Generate embeddings for symbols
	if err := idx.generateEmbeddings(ctx, symbols); err != nil {
		log.Printf("Warning: failed to generate embeddings for %s: %v", file.RelPath, err)
		// Continue without embeddings
	}

	// Save symbols
	if err := idx.storage.SaveCodeSymbols(ctx, symbols); err != nil {
		return fmt.Errorf("failed to save symbols: %w", err)
	}

	// Save file record
	codeFile := &treesitter.CodeFile{
		ProjectID:    projectID,
		FilePath:     file.RelPath,
		Language:     file.Language,
		FileHash:     file.Hash,
		SymbolsCount: len(symbols),
		IndexedAt:    time.Now(),
	}

	if err := idx.storage.SaveCodeFile(ctx, codeFile); err != nil {
		return fmt.Errorf("failed to save file record: %w", err)
	}

	// Process large symbols for chunking
	if err := idx.processLargeSymbols(ctx, projectID, file.RelPath, symbols); err != nil {
		log.Printf("Warning: failed to process large symbols for chunking: %v", err)
		// Continue even if chunking fails
	}

	idx.updateProgress(projectID, func(p *IndexingProgress) {
		p.FilesIndexed++
		p.SymbolsFound += len(symbols)
	})

	return nil
}

// generateProjectID creates a stable project ID from the path
func (idx *Indexer) generateProjectID(absPath string) string {
	// Use sanitized path as ID
	id := strings.ReplaceAll(absPath, "/", "_")
	id = strings.ReplaceAll(id, "\\", "_")
	id = strings.ReplaceAll(id, ":", "_")
	id = strings.TrimPrefix(id, "_")

	// Limit length
	if len(id) > 100 {
		id = id[len(id)-100:]
	}

	return id
}

// ReindexFile re-indexes a single file
func (idx *Indexer) ReindexFile(ctx context.Context, projectID, filePath string) error {
	// Get project to find root path
	project, err := idx.storage.GetCodeProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return fmt.Errorf("project not found: %s", projectID)
	}

	absPath := filepath.Join(project.RootPath, filePath)

	// Check file exists
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	// Get language from extension
	ext := filepath.Ext(absPath)
	lang, ok := treesitter.GetLanguageByExtension(ext)
	if !ok {
		return fmt.Errorf("unsupported file extension: %s", ext)
	}

	// Calculate hash
	hash, err := idx.config.Scanner.calculateHash(absPath)
	if err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	file := ScannedFile{
		AbsPath:  absPath,
		RelPath:  filePath,
		Language: lang,
		Size:     info.Size(),
		Hash:     hash,
	}

	return idx.processFile(ctx, projectID, project.RootPath, file)
}

// DeleteProject removes a project and all its data
func (idx *Indexer) DeleteProject(ctx context.Context, projectID string) error {
	return idx.storage.DeleteCodeProject(ctx, projectID)
}
