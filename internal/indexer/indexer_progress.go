// Package indexer provides progress tracking functionality.
package indexer

import (
	"time"

	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// initProgress initializes progress tracking for a project
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

// updateProgress updates progress for a project using a function
func (idx *Indexer) updateProgress(projectID string, fn func(p *IndexingProgress)) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if p, ok := idx.progress[projectID]; ok {
		fn(p)
		p.UpdatedAt = time.Now()
	}
}

// setError sets an error for a project's progress
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
