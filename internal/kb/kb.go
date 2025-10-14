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
	path     string
	storage  storage.Storage
	embedder embedder.Embedder
	watcher  *fsnotify.Watcher
	cancel   context.CancelFunc
	once     sync.Once
}

// StartWatcher starts a watcher if path is non-empty and exists. Returns nil if path is empty.
func StartWatcher(parentCtx context.Context, path string, st storage.Storage, emb embedder.Embedder) (*Watcher, error) {
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
	w := &Watcher{path: path, storage: st, embedder: emb, watcher: fw, cancel: cancel}

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

// initialScan processes all existing .md files.
func (w *Watcher) initialScan(ctx context.Context) {
	// Walk entire tree to capture subdirectories and files.
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
			w.processFile(ctx, path)
		}
		return nil
	})
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
	content, err := os.ReadFile(fullPath)
	if err != nil {
		slog.Warn("failed reading kb file", "file", rel, "error", err)
		return
	}
	embedding, err := w.embedder.EmbedQuery(ctx, string(content))
	if err != nil {
		slog.Warn("failed embedding kb file", "file", rel, "error", err)
		return
	}
	if err := w.storage.SaveDocument(ctx, rel, string(content), embedding, map[string]interface{}{"source": "watcher"}); err != nil {
		slog.Warn("failed saving kb document", "file", rel, "error", err)
		return
	}
	slog.Info("kb document synced", "file", rel, "bytes", len(content))
}

func (w *Watcher) relativePath(full string) string {
	rel, err := filepath.Rel(w.path, full)
	if err != nil {
		return filepath.Base(full)
	}
	// Normalize path separator to '/' for consistency with existing tools
	return filepath.ToSlash(rel)
}
