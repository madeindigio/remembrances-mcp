// Package indexer provides the main indexing service for code projects.
package indexer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// CodeWatcher watches a code project directory for file changes and triggers reindexing.
type CodeWatcher struct {
	projectID string
	rootPath  string
	indexer   *Indexer
	storage   storage.FullStorage
	watcher   *fsnotify.Watcher
	cancel    context.CancelFunc
	once      sync.Once
}

// StartCodeWatcher creates and starts a new code watcher for a project.
// It returns immediately after starting background goroutines for the initial scan and event loop.
func StartCodeWatcher(parentCtx context.Context, project *storage.CodeProject, indexer *Indexer, st storage.FullStorage) (*CodeWatcher, error) {
	if project == nil {
		return nil, nil
	}

	info, err := os.Stat(project.RootPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, os.ErrNotExist
	}

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(parentCtx)
	w := &CodeWatcher{
		projectID: project.ProjectID,
		rootPath:  project.RootPath,
		indexer:   indexer,
		storage:   st,
		watcher:   fw,
		cancel:    cancel,
	}

	// Add the root directory (fsnotify is not recursive)
	// We will dynamically add subdirectories when Create events are detected
	if err := fw.Add(project.RootPath); err != nil {
		fw.Close()
		return nil, err
	}

	// Add all subdirectories recursively
	err = filepath.WalkDir(project.RootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if d.IsDir() && path != project.RootPath {
			// Skip excluded directories
			if w.isExcludedDir(d.Name()) {
				return filepath.SkipDir
			}
			if err := fw.Add(path); err != nil {
				slog.Warn("failed to watch subdirectory", "path", path, "error", err)
			}
		}
		return nil
	})
	if err != nil {
		fw.Close()
		return nil, err
	}

	// Start event loop
	go w.run(ctx)

	slog.Info("code watcher started", "project_id", project.ProjectID, "path", project.RootPath)
	return w, nil
}

// Stop stops the watcher (idempotent).
func (w *CodeWatcher) Stop() {
	if w == nil {
		return
	}
	w.once.Do(func() {
		w.cancel()
		_ = w.watcher.Close()
		slog.Info("code watcher stopped", "project_id", w.projectID, "path", w.rootPath)
	})
}

// GetProjectID returns the project ID being watched.
func (w *CodeWatcher) GetProjectID() string {
	if w == nil {
		return ""
	}
	return w.projectID
}

// run processes watcher events and debounces rapid successive writes.
func (w *CodeWatcher) run(ctx context.Context) {
	debounce := make(map[string]time.Time)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Handle new directories for recursive support
			if evt.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(evt.Name)
				if err == nil && info.IsDir() {
					dirName := filepath.Base(evt.Name)
					if !w.isExcludedDir(dirName) {
						if err := w.watcher.Add(evt.Name); err != nil {
							slog.Warn("failed to add new directory to watcher", "dir", evt.Name, "error", err)
						}
					}
					continue
				}
			}

			// Check if it's a code file
			if !w.isCodeFile(evt.Name) {
				continue
			}

			// Delete/Rename events -> remove from index
			if evt.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
				rel := w.relativePath(evt.Name)
				if err := w.storage.DeleteCodeFile(ctx, w.projectID, rel); err != nil {
					slog.Warn("failed to delete code file after removal", "file", rel, "error", err)
				} else {
					slog.Info("code file removed from index", "file", rel)
				}
				continue
			}

			// Create or Write => schedule for debounced processing
			if evt.Op&(fsnotify.Create|fsnotify.Write) != 0 {
				debounce[evt.Name] = time.Now()
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			slog.Warn("code watcher error", "error", err)

		case now := <-ticker.C:
			for file, t := range debounce {
				if now.Sub(t) > 300*time.Millisecond {
					w.processFile(ctx, file)
					delete(debounce, file)
				}
			}
		}
	}
}

// processFile reindexes a single file.
func (w *CodeWatcher) processFile(ctx context.Context, fullPath string) {
	rel := w.relativePath(fullPath)

	// Check file still exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		slog.Debug("file no longer exists, skipping", "file", rel)
		return
	}

	startTime := time.Now()
	slog.Debug("processing code file change", "file", rel)

	// Use the indexer to reindex the file
	if err := w.indexer.ReindexFile(ctx, w.projectID, rel); err != nil {
		slog.Warn("failed to reindex code file", "file", rel, "error", err)
		return
	}

	slog.Info("code file reindexed", "file", rel, "duration", time.Since(startTime))
}

// isCodeFile checks if the file is a supported code file based on extension.
func (w *CodeWatcher) isCodeFile(path string) bool {
	ext := filepath.Ext(path)
	if ext == "" {
		return false
	}

	// Use treesitter to determine if this extension is supported
	_, ok := treesitter.GetLanguageByExtension(ext)
	return ok
}

// isExcludedDir checks if a directory should be excluded from watching.
// It delegates to the FileScanner's exclusion logic so that user-configured
// exclude patterns (e.g., "Pods", ".venv") are respected.
func (w *CodeWatcher) isExcludedDir(name string) bool {
	scanner := w.indexer.GetScanner()
	if scanner == nil {
		return false
	}
	// Use the scanner's full exclusion logic (patterns + hidden-dir rules).
	return scanner.ShouldExclude(name, name, true)
}

// relativePath returns the path relative to the project root.
func (w *CodeWatcher) relativePath(full string) string {
	rel, err := filepath.Rel(w.rootPath, full)
	if err != nil {
		return filepath.Base(full)
	}
	// Normalize path separator to '/' for consistency
	return filepath.ToSlash(rel)
}

// OutdatedFile represents a file that needs reindexing.
type OutdatedFile struct {
	FilePath string // Relative path from project root
	Reason   string // "modified", "new", "deleted"
	AbsPath  string // Absolute path on disk
}

// ScanOutdatedFiles scans the project for files that need reindexing.
// It compares file hashes with stored hashes and detects new/deleted files.
func (w *CodeWatcher) ScanOutdatedFiles(ctx context.Context) ([]OutdatedFile, error) {
	var outdated []OutdatedFile

	// Get all indexed files from database
	indexedFiles, err := w.storage.ListCodeFiles(ctx, w.projectID)
	if err != nil {
		return nil, err
	}

	// Build a map of indexed files by path
	indexedMap := make(map[string]string) // path -> hash
	for _, f := range indexedFiles {
		indexedMap[f.FilePath] = f.FileHash
	}

	// Scan filesystem for code files
	filesOnDisk := make(map[string]bool)

	err = filepath.WalkDir(w.rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if d.IsDir() {
			if w.isExcludedDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a code file
		if !w.isCodeFile(path) {
			return nil
		}

		rel := w.relativePath(path)
		filesOnDisk[rel] = true

		// Calculate current hash
		currentHash, err := w.calculateHash(path)
		if err != nil {
			slog.Warn("failed to calculate hash", "file", rel, "error", err)
			return nil
		}

		// Check if file is indexed
		storedHash, exists := indexedMap[rel]
		if !exists {
			// New file
			outdated = append(outdated, OutdatedFile{
				FilePath: rel,
				Reason:   "new",
				AbsPath:  path,
			})
		} else if storedHash != currentHash {
			// Modified file
			outdated = append(outdated, OutdatedFile{
				FilePath: rel,
				Reason:   "modified",
				AbsPath:  path,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Check for deleted files (in index but not on disk)
	for indexedPath := range indexedMap {
		if !filesOnDisk[indexedPath] {
			outdated = append(outdated, OutdatedFile{
				FilePath: indexedPath,
				Reason:   "deleted",
				AbsPath:  filepath.Join(w.rootPath, indexedPath),
			})
		}
	}

	return outdated, nil
}

// ProcessOutdatedFiles processes a list of outdated files.
// It reindexes modified/new files and removes deleted files from the index.
func (w *CodeWatcher) ProcessOutdatedFiles(ctx context.Context, files []OutdatedFile) error {
	for _, f := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		switch f.Reason {
		case "deleted":
			if err := w.storage.DeleteCodeFile(ctx, w.projectID, f.FilePath); err != nil {
				slog.Warn("failed to delete file from index", "file", f.FilePath, "error", err)
			} else {
				slog.Info("deleted file removed from index", "file", f.FilePath)
			}
		case "new", "modified":
			if err := w.indexer.ReindexFile(ctx, w.projectID, f.FilePath); err != nil {
				slog.Warn("failed to reindex file", "file", f.FilePath, "reason", f.Reason, "error", err)
			} else {
				slog.Info("file reindexed", "file", f.FilePath, "reason", f.Reason)
			}
		}
	}
	return nil
}

// calculateHash computes SHA-256 hash of file content.
func (w *CodeWatcher) calculateHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
