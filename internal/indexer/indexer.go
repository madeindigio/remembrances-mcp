// Package indexer provides the main indexing service for code projects.
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

// IndexerConfig holds configuration for the indexer
type IndexerConfig struct {
	// Number of concurrent file processors
	Concurrency int

	// Batch size for embedding generation
	EmbeddingBatchSize int

	// Whether to store source code in symbols
	StoreSourceCode bool

	// Maximum source code length to store
	MaxSourceCodeLength int

	// Scanner configuration
	Scanner *FileScanner
}

// DefaultIndexerConfig returns sensible defaults
func DefaultIndexerConfig() IndexerConfig {
	return IndexerConfig{
		Concurrency:         4,
		EmbeddingBatchSize:  10,
		StoreSourceCode:     true,
		MaxSourceCodeLength: 10000,
		Scanner:             NewFileScanner(),
	}
}

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

// IndexingProgress tracks the progress of an indexing operation
type IndexingProgress struct {
	ProjectID    string
	Status       treesitter.IndexingStatus
	FilesTotal   int
	FilesIndexed int
	SymbolsFound int
	CurrentFile  string
	StartedAt    time.Time
	UpdatedAt    time.Time
	Error        *string
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

	// Start workers
	for i := 0; i < idx.config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				if err := idx.processFile(ctx, projectID, rootPath, file); err != nil {
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

// processFile processes a single source file
func (idx *Indexer) processFile(ctx context.Context, projectID, rootPath string, file ScannedFile) error {
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

	// Parse file
	tree, lang, err := idx.parser.ParseFile(ctx, file.AbsPath)
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

	idx.updateProgress(projectID, func(p *IndexingProgress) {
		p.FilesIndexed++
		p.SymbolsFound += len(symbols)
	})

	return nil
}

// generateEmbeddings generates embeddings for symbols in batches
func (idx *Indexer) generateEmbeddings(ctx context.Context, symbols []*treesitter.CodeSymbol) error {
	if len(symbols) == 0 {
		return nil
	}

	// Prepare texts for embedding
	texts := make([]string, 0, len(symbols))
	symbolIdxs := make([]int, 0, len(symbols))

	for i, sym := range symbols {
		text := idx.prepareSymbolText(sym)
		if text != "" {
			texts = append(texts, text)
			symbolIdxs = append(symbolIdxs, i)
		}
	}

	if len(texts) == 0 {
		return nil
	}

	// Generate embeddings in batches
	for i := 0; i < len(texts); i += idx.config.EmbeddingBatchSize {
		end := i + idx.config.EmbeddingBatchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embeddings, err := idx.embedder.EmbedDocuments(ctx, batch)
		if err != nil {
			return fmt.Errorf("failed to generate embeddings: %w", err)
		}

		// Assign embeddings to symbols
		for j, embedding := range embeddings {
			symIdx := symbolIdxs[i+j]
			symbols[symIdx].Embedding = embedding
		}
	}

	return nil
}

// prepareSymbolText creates the text representation for embedding
func (idx *Indexer) prepareSymbolText(sym *treesitter.CodeSymbol) string {
	var parts []string

	// Add symbol type and name
	parts = append(parts, fmt.Sprintf("%s %s", sym.SymbolType, sym.Name))

	// Add signature if available
	if sym.Signature != "" {
		parts = append(parts, sym.Signature)
	}

	// Add docstring if available
	if sym.DocString != "" {
		parts = append(parts, sym.DocString)
	}

	// Add source code if available and not too long
	if sym.SourceCode != "" && len(sym.SourceCode) < 500 {
		parts = append(parts, sym.SourceCode)
	}

	return strings.Join(parts, "\n")
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

// Progress tracking methods

func (idx *Indexer) initProgress(projectID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.progress[projectID] = &IndexingProgress{
		ProjectID: projectID,
		Status:    treesitter.IndexingStatusInProgress,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (idx *Indexer) updateProgress(projectID string, fn func(p *IndexingProgress)) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if p, ok := idx.progress[projectID]; ok {
		fn(p)
		p.UpdatedAt = time.Now()
	}
}

func (idx *Indexer) setError(projectID string, err error) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if p, ok := idx.progress[projectID]; ok {
		errStr := err.Error()
		p.Error = &errStr
		p.Status = treesitter.IndexingStatusFailed
		p.UpdatedAt = time.Now()
	}
}

// GetProgress returns the current progress for a project
func (idx *Indexer) GetProgress(projectID string) *IndexingProgress {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if p, ok := idx.progress[projectID]; ok {
		// Return a copy
		copy := *p
		return &copy
	}
	return nil
}

// GetAllProgress returns progress for all active indexing operations
func (idx *Indexer) GetAllProgress() map[string]*IndexingProgress {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	result := make(map[string]*IndexingProgress)
	for k, v := range idx.progress {
		copy := *v
		result[k] = &copy
	}
	return result
}

// ClearProgress removes progress tracking for completed projects
func (idx *Indexer) ClearProgress(projectID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	delete(idx.progress, projectID)
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
