package tests

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/madeindigio/remembrances-mcp/internal/indexer"
	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

func TestIndexer_IndexProject_DoesNotProcessExcludedFiles(t *testing.T) {
	tempDir := t.TempDir()

	mustWriteFile(t, filepath.Join(tempDir, "main.go"), "package main\nfunc main() {}\n")
	mustWriteFile(t, filepath.Join(tempDir, "internal", "service", "service.go"), "package service\nfunc Run() {}\n")
	mustWriteFile(t, filepath.Join(tempDir, "pkg", "util", "helpers.go"), "package util\nfunc Help() {}\n")

	// Excluded by built-in defaults.
	mustWriteFile(t, filepath.Join(tempDir, "vendor", "dep", "ignored.go"), "package dep\nfunc VendorSkip() {}\n")
	mustWriteFile(t, filepath.Join(tempDir, "node_modules", "lib", "ignored.js"), "function nodeSkip() {}\n")
	mustWriteFile(t, filepath.Join(tempDir, ".cache", "ignored.py"), "def cache_skip():\n    pass\n")

	// Excluded by user-configured patterns.
	mustWriteFile(t, filepath.Join(tempDir, "var", "ignored.go"), "package ignored\nfunc SkipMe() {}\n")
	mustWriteFile(t, filepath.Join(tempDir, "custom_out", "ignored.go"), "package ignored\nfunc SkipCustomOut() {}\n")
	mustWriteFile(t, filepath.Join(tempDir, "auto.generated.go"), "package main\nfunc Generated() {}\n")
	mustWriteFile(t, filepath.Join(tempDir, "folder", "file.php"), "<?php echo 'skip';")
	mustWriteFile(t, filepath.Join(tempDir, "folder", "file.js"), "function keep() {}")

	scanner := indexer.NewFileScanner()
	scanner.MergeExcludePatterns([]string{"var", "custom_out", "*.generated.go", "folder/*.php"})

	cfg := indexer.DefaultIndexerConfig()
	cfg.Scanner = scanner
	cfg.Concurrency = 1

	spyStorage := &indexerSpyStorage{SurrealDBStorage: &storage.SurrealDBStorage{}}
	idx := indexer.NewIndexer(spyStorage, &stubEmbedder{}, cfg)

	projectID, err := idx.IndexProject(context.Background(), tempDir, "handoff-test")
	if err != nil {
		t.Fatalf("indexing failed: %v", err)
	}
	if projectID == "" {
		t.Fatalf("expected non-empty project ID")
	}

	includedFiles := []string{
		"main.go",
		filepath.Join("internal", "service", "service.go"),
		filepath.Join("pkg", "util", "helpers.go"),
		filepath.Join("folder", "file.js"),
	}
	excludedFiles := []string{
		filepath.Join("vendor", "dep", "ignored.go"),
		filepath.Join("node_modules", "lib", "ignored.js"),
		filepath.Join(".cache", "ignored.py"),
		filepath.Join("var", "ignored.go"),
		filepath.Join("custom_out", "ignored.go"),
		filepath.Join("folder", "file.php"),
		"auto.generated.go",
	}

	for _, relPath := range includedFiles {
		assertContainsPath(t, spyStorage.getCodeFileCalls, relPath, "expected included file to be checked for existing record")
		assertContainsPath(t, spyStorage.savedCodeFiles, relPath, "expected included file to be saved")
	}

	for _, relPath := range excludedFiles {
		assertNotContainsPath(t, spyStorage.getCodeFileCalls, relPath, "excluded file should not be checked in storage")
		assertNotContainsPath(t, spyStorage.savedCodeFiles, relPath, "excluded file should never be saved downstream")
	}

	if got, want := len(spyStorage.savedCodeFiles), len(includedFiles); got != want {
		t.Fatalf("expected exactly %d saved included files, got %d (%v)", want, got, spyStorage.savedCodeFiles)
	}
}

type indexerSpyStorage struct {
	*storage.SurrealDBStorage

	getCodeFileCalls []string
	savedCodeFiles   []string
	projects         map[string]*storage.CodeProject
}

func (s *indexerSpyStorage) CreateCodeProject(ctx context.Context, project *treesitter.CodeProject) error {
	if s.projects == nil {
		s.projects = make(map[string]*storage.CodeProject)
	}
	s.projects[project.ProjectID] = &storage.CodeProject{
		ProjectID: project.ProjectID,
		Name:      project.Name,
		RootPath:  project.RootPath,
	}
	return nil
}

func (s *indexerSpyStorage) GetCodeProject(ctx context.Context, projectID string) (*storage.CodeProject, error) {
	if s.projects != nil {
		if p, ok := s.projects[projectID]; ok {
			return p, nil
		}
	}
	return nil, nil
}

func (s *indexerSpyStorage) UpdateProjectStatus(ctx context.Context, projectID string, status treesitter.IndexingStatus) error {
	return nil
}

func (s *indexerSpyStorage) GetCodeFile(ctx context.Context, projectID, filePath string) (*storage.CodeFile, error) {
	s.getCodeFileCalls = append(s.getCodeFileCalls, filepath.Clean(filePath))
	return nil, nil
}

func (s *indexerSpyStorage) SaveCodeSymbols(ctx context.Context, symbols []*treesitter.CodeSymbol) error {
	return nil
}

func (s *indexerSpyStorage) SaveCodeFile(ctx context.Context, file *treesitter.CodeFile) error {
	s.savedCodeFiles = append(s.savedCodeFiles, filepath.Clean(file.FilePath))
	return nil
}

func (s *indexerSpyStorage) DeleteChunksByFile(ctx context.Context, projectID, filePath string) error {
	return nil
}

func (s *indexerSpyStorage) SaveCodeChunks(ctx context.Context, chunks []*storage.CodeChunk) error {
	return nil
}

type stubEmbedder struct{}

func (e *stubEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{0.1, 0.2, 0.3}
	}
	return out, nil
}

func (e *stubEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}

func (e *stubEmbedder) Dimension() int {
	return 3
}

func assertContainsPath(t *testing.T, paths []string, expected, msg string) {
	t.Helper()
	expected = filepath.Clean(expected)
	for _, p := range paths {
		if filepath.Clean(p) == expected {
			return
		}
	}
	t.Fatalf("%s: expected %q in %v", msg, expected, paths)
}

func assertNotContainsPath(t *testing.T, paths []string, forbidden, msg string) {
	t.Helper()
	forbidden = filepath.Clean(forbidden)
	for _, p := range paths {
		if filepath.Clean(p) == forbidden {
			t.Fatalf("%s: found forbidden path %q in %v", msg, forbidden, paths)
		}
	}
}

var _ storage.FullStorage = (*indexerSpyStorage)(nil)
