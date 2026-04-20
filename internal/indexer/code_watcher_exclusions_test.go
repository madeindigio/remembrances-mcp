package indexer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
)

func TestCodeWatcher_shouldExcludePath_UsesRelativePatterns(t *testing.T) {
	tempDir := t.TempDir()

	cfg := DefaultIndexerConfig()
	cfg.Scanner.MergeExcludePatterns([]string{"folder/*.php", "**/generated/**"})

	w := &CodeWatcher{
		rootPath: tempDir,
		indexer:  &Indexer{config: cfg},
	}

	if !w.shouldExcludePath(filepath.Join(tempDir, "folder", "file.php"), false) {
		t.Fatalf("expected folder/*.php exclusion to apply to watcher file events")
	}
	if w.shouldExcludePath(filepath.Join(tempDir, "folder", "file.js"), false) {
		t.Fatalf("expected non-matching file in watched folder to remain included")
	}
	if !w.shouldExcludePath(filepath.Join(tempDir, "deep", "generated", "code.go"), false) {
		t.Fatalf("expected **/generated/** exclusion to apply to nested file paths")
	}
	if !w.shouldExcludePath(filepath.Join(tempDir, "deep", "generated"), true) {
		t.Fatalf("expected **/generated/** exclusion to apply to nested directories")
	}
}

func TestStartCodeWatcher_ExcludesPathMatchedDirectoriesFromWatchList(t *testing.T) {
	tempDir := t.TempDir()

	mustWriteWatcherFile(t, filepath.Join(tempDir, "src", "main.go"), "package main\nfunc main() {}\n")
	mustWriteWatcherFile(t, filepath.Join(tempDir, "docs", "_build", "page.py"), "print('skip')\n")
	mustWriteWatcherFile(t, filepath.Join(tempDir, "internal", "generated", "code.go"), "package generated\nfunc Skip() {}\n")

	cfg := DefaultIndexerConfig()
	cfg.Scanner.MergeExcludePatterns([]string{"docs/_build", "**/generated/**"})
	idx := NewIndexer(&storage.SurrealDBStorage{}, nil, cfg)

	project := &storage.CodeProject{
		ProjectID: "watch-list-test",
		RootPath:  tempDir,
	}

	w, err := StartCodeWatcher(context.Background(), project, idx, &storage.SurrealDBStorage{})
	if err != nil {
		t.Fatalf("StartCodeWatcher failed: %v", err)
	}
	t.Cleanup(w.Stop)

	watched := make(map[string]bool)
	for _, path := range w.watcher.WatchList() {
		watched[path] = true
	}

	if !watched[tempDir] {
		t.Fatalf("expected watcher to include project root")
	}
	if !watched[filepath.Join(tempDir, "src")] {
		t.Fatalf("expected watcher to include non-excluded subdirectory")
	}
	if !watched[filepath.Join(tempDir, "docs")] {
		t.Fatalf("expected watcher to include parent directory of excluded nested path")
	}
	if watched[filepath.Join(tempDir, "docs", "_build")] {
		t.Fatalf("did not expect watcher to include docs/_build when excluded by path pattern")
	}
	if watched[filepath.Join(tempDir, "internal", "generated")] {
		t.Fatalf("did not expect watcher to include internal/generated when excluded by globstar pattern")
	}
}

func TestCodeWatcher_ScanOutdatedFiles_HonorsPathPatternExclusions(t *testing.T) {
	tempDir := t.TempDir()

	mainPath := filepath.Join(tempDir, "keep", "main.go")
	newVisiblePath := filepath.Join(tempDir, "keep", "new.go")
	excludedPHPPath := filepath.Join(tempDir, "folder", "file.php")
	excludedGeneratedPath := filepath.Join(tempDir, "deep", "generated", "code.go")

	mustWriteWatcherFile(t, mainPath, "package keep\nfunc Main() {}\n")
	mustWriteWatcherFile(t, newVisiblePath, "package keep\nfunc New() {}\n")
	mustWriteWatcherFile(t, excludedPHPPath, "<?php echo 'skip';\n")
	mustWriteWatcherFile(t, excludedGeneratedPath, "package generated\nfunc Skip() {}\n")

	cfg := DefaultIndexerConfig()
	cfg.Scanner.MergeExcludePatterns([]string{"folder/*.php", "**/generated/**"})
	w := &CodeWatcher{
		projectID: "scan-outdated-test",
		rootPath:  tempDir,
		indexer:   &Indexer{config: cfg},
	}

	mainHash, err := w.calculateHash(mainPath)
	if err != nil {
		t.Fatalf("calculateHash(main.go) failed: %v", err)
	}

	w.storage = &watcherTestStorage{
		SurrealDBStorage: &storage.SurrealDBStorage{},
		files: []storage.CodeFile{
			{ProjectID: w.projectID, FilePath: "keep/main.go", FileHash: mainHash},
			{ProjectID: w.projectID, FilePath: "folder/file.php", FileHash: "old-hash"},
			{ProjectID: w.projectID, FilePath: "deep/generated/code.go", FileHash: "old-hash"},
		},
	}

	outdated, err := w.ScanOutdatedFiles(context.Background())
	if err != nil {
		t.Fatalf("ScanOutdatedFiles failed: %v", err)
	}

	assertOutdatedReason(t, outdated, "keep/new.go", "new")
	assertOutdatedReason(t, outdated, "folder/file.php", "deleted")
	assertOutdatedReason(t, outdated, "deep/generated/code.go", "deleted")
	assertNoOutdatedReason(t, outdated, "folder/file.php", "new")
	assertNoOutdatedReason(t, outdated, "deep/generated/code.go", "new")
	assertNoOutdatedEntry(t, outdated, "keep/main.go")
}

type watcherTestStorage struct {
	*storage.SurrealDBStorage
	files []storage.CodeFile
}

func (s *watcherTestStorage) ListCodeFiles(ctx context.Context, projectID string) ([]storage.CodeFile, error) {
	return s.files, nil
}

func mustWriteWatcherFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create parent directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

func assertOutdatedReason(t *testing.T, outdated []OutdatedFile, filePath, reason string) {
	t.Helper()
	for _, file := range outdated {
		if file.FilePath == filePath && file.Reason == reason {
			return
		}
	}
	t.Fatalf("expected outdated entry %q with reason %q, got %#v", filePath, reason, outdated)
}

func assertNoOutdatedReason(t *testing.T, outdated []OutdatedFile, filePath, reason string) {
	t.Helper()
	for _, file := range outdated {
		if file.FilePath == filePath && file.Reason == reason {
			t.Fatalf("unexpected outdated entry %q with reason %q", filePath, reason)
		}
	}
}

func assertNoOutdatedEntry(t *testing.T, outdated []OutdatedFile, filePath string) {
	t.Helper()
	for _, file := range outdated {
		if file.FilePath == filePath {
			t.Fatalf("unexpected outdated entry for %q: %#v", filePath, file)
		}
	}
}
