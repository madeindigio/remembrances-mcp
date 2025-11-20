// Package kb provides utilities to automatically watch a knowledge base directory
// and synchronize markdown (.md) documents with the database generating embeddings on each change.
package kb

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/embedder"
)

// Watcher controls monitoring of the knowledge base directory.
type Watcher struct {
	path         string
	storage      storage.Storage
	embedder     embedder.Embedder
	watcher      *fsnotify.Watcher
	cancel       context.CancelFunc
	once         sync.Once
	chunkSize    int
	chunkOverlap int
}

// StartWatcher starts a watcher if path is non-empty and exists. Returns nil if path is empty.
func StartWatcher(parentCtx context.Context, path string, st storage.Storage, emb embedder.Embedder, chunkSize, chunkOverlap int) (*Watcher, error) {
	if path == "" {
		return nil, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("knowledge base path is not a directory")
	}

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(parentCtx)
	w := &Watcher{
		path:         path,
		storage:      st,
		embedder:     emb,
		watcher:      fw,
		cancel:       cancel,
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
	}

	// Add only the root directory (fsnotify is not recursive). We will dynamically add subdirectories
	// when Create events for new directories are detected.
	if err := fw.Add(path); err != nil {
		fw.Close()
		return nil, err
	}

	// Indexación inicial de ficheros ya presentes
	go w.initialScan(ctx)
	// Bucle de eventos
	go w.run(ctx)
	slog.Info("knowledge base watcher started", "path", path)
	return w, nil
}

// Stop stops the watcher (idempotent).
func (w *Watcher) Stop() {
	if w == nil {
		return
	}
	w.once.Do(func() {
		w.cancel()
		_ = w.watcher.Close()
		slog.Info("knowledge base watcher stopped", "path", w.path)
	})
}

// initialScan processes all existing .md files with concurrency control.
func (w *Watcher) initialScan(ctx context.Context) {
	slog.Info("starting initial knowledge base scan", "path", w.path)

	// Collect all files first
	var files []string
	filepath.WalkDir(w.path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			slog.Warn("initial scan error", "path", path, "error", err)
			return nil
		}
		if d.IsDir() {
			// Add each subdirectory to watcher for recursive behavior.
			if path != w.path {
				if err := w.watcher.Add(path); err != nil {
					slog.Warn("failed to watch subdirectory", "path", path, "error", err)
				}
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			files = append(files, path)
		}
		return nil
	})

	slog.Info("initial scan found files", "count", len(files))

	// Process files SEQUENTIALLY to avoid memory exhaustion with GGUF models
	// GGUF models can consume significant memory, especially with multiple concurrent operations
	for i, file := range files {
		select {
		case <-ctx.Done():
			slog.Info("initial scan cancelled", "processed", i, "total", len(files))
			return
		default:
		}

		slog.Debug("processing kb file during initial scan", "file", file, "progress", i+1, "total", len(files))
		w.processFile(ctx, file)
	}

	slog.Info("initial knowledge base scan completed", "files_processed", len(files))
}

// run processes watcher events and debounces rapid successive writes.
func (w *Watcher) run(ctx context.Context) {
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
			// Add new directories for recursive support
			if evt.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(evt.Name)
				if err == nil && info.IsDir() {
					if err := w.watcher.Add(evt.Name); err != nil {
						slog.Warn("failed to add new directory to watcher", "dir", evt.Name, "error", err)
					}
					continue
				}
			}
			if !strings.HasSuffix(strings.ToLower(evt.Name), ".md") {
				continue
			}
			// Delete / rename events -> remove from DB
			if evt.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
				rel := w.relativePath(evt.Name)
				if err := w.storage.DeleteDocument(ctx, rel); err != nil {
					slog.Warn("failed to delete document after file removal", "file", rel, "error", err)
				} else {
					slog.Info("document deleted after file removal", "file", rel)
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
			slog.Warn("watcher error", "error", err)
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

// processFile reads the file, generates an embedding and upserts the document.
func (w *Watcher) processFile(ctx context.Context, fullPath string) {
	rel := w.relativePath(fullPath)

	// Add timeout to prevent hanging on large files
	processingCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	startTime := time.Now()
	slog.Debug("processing kb file", "file", rel)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		slog.Warn("failed reading kb file", "file", rel, "error", err)
		return
	}

	contentSize := len(content)
	slog.Debug("file read", "file", rel, "bytes", contentSize)

	// Skip very large files (>500KB) to avoid memory/processing issues
	const maxFileSize = 500 * 1024 // 500KB limit
	if contentSize > maxFileSize {
		slog.Warn("skipping large file", "file", rel, "bytes", contentSize, "max", maxFileSize)
		return
	}

	// Skip empty files
	contentStr := string(content)
	if len(strings.TrimSpace(contentStr)) == 0 {
		slog.Debug("skipping empty file", "file", rel)
		return
	}

	// Chunk the text and generate individual embeddings for each chunk
	// This allows for more precise retrieval compared to averaged embeddings
	chunks, embeddings, err := embedder.EmbedTextChunksWithOverlap(processingCtx, w.embedder, contentStr, w.chunkSize, w.chunkOverlap)
	if err != nil {
		slog.Warn("failed embedding kb file", "file", rel, "error", err, "duration", time.Since(startTime))
		return
	}

	slog.Debug("chunks and embeddings generated", "file", rel, "chunks", len(chunks), "duration", time.Since(startTime))

	// Save each chunk as a separate document with its own embedding
	metadata := map[string]interface{}{
		"source":     "watcher",
		"total_size": contentSize,
	}

	if err := w.storage.SaveDocumentChunks(processingCtx, rel, chunks, embeddings, metadata); err != nil {
		slog.Warn("failed saving kb document chunks", "file", rel, "error", err)
		return
	}

	slog.Info("kb document synced", "file", rel, "bytes", contentSize, "chunks", len(chunks), "duration", time.Since(startTime))
}

func (w *Watcher) relativePath(full string) string {
	rel, err := filepath.Rel(w.path, full)
	if err != nil {
		return filepath.Base(full)
	}
	// Normalize path separator to '/' for consistency with existing tools
	return filepath.ToSlash(rel)
}
