// Package indexer provides the main indexing service for code projects.
package indexer

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
)

// WatcherManager manages a single active code watcher.
// Only ONE code project can be actively monitored at a time.
// This is a resource constraint to prevent system overload.
type WatcherManager struct {
	mu            sync.RWMutex
	activeWatcher *CodeWatcher
	activeProject string
	indexer       *Indexer
	storage       storage.FullStorage
}

// NewWatcherManager creates a new watcher manager.
func NewWatcherManager(indexer *Indexer, storage storage.FullStorage) *WatcherManager {
	return &WatcherManager{
		indexer: indexer,
		storage: storage,
	}
}

// ActivateProject starts monitoring a code project.
// If another project is already being monitored, it will be deactivated first.
// Returns the number of outdated files found during initial scan.
func (wm *WatcherManager) ActivateProject(ctx context.Context, projectID string) (int, string, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	var previousProject string

	// Check if the same project is already active
	if wm.activeProject == projectID && wm.activeWatcher != nil {
		return 0, "", nil // Already active
	}

	// Get the project from storage
	project, err := wm.storage.GetCodeProject(ctx, projectID)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return 0, "", fmt.Errorf("project not found: %s", projectID)
	}

	// Check if project is properly indexed (not pending or failed)
	if project.IndexingStatus == "pending" || project.IndexingStatus == "failed" {
		return 0, "", fmt.Errorf("project is not properly indexed (status: %s)", project.IndexingStatus)
	}

	// Deactivate current watcher if different project
	if wm.activeWatcher != nil && wm.activeProject != projectID {
		previousProject = wm.activeProject
		wm.activeWatcher.Stop()
		// Update the previous project's watcher status
		if err := wm.storage.UpdateProjectWatcher(ctx, wm.activeProject, false); err != nil {
			slog.Warn("failed to update previous project watcher status", "project_id", wm.activeProject, "error", err)
		}
		wm.activeWatcher = nil
		wm.activeProject = ""
	}

	// Start new watcher
	watcher, err := StartCodeWatcher(ctx, project, wm.indexer, wm.storage)
	if err != nil {
		return 0, previousProject, fmt.Errorf("failed to start watcher: %w", err)
	}

	wm.activeWatcher = watcher
	wm.activeProject = projectID

	// Update project watcher status in storage
	if err := wm.storage.UpdateProjectWatcher(ctx, projectID, true); err != nil {
		slog.Warn("failed to update project watcher status", "project_id", projectID, "error", err)
	}

	// Scan for outdated files
	outdatedFiles, err := watcher.ScanOutdatedFiles(ctx)
	if err != nil {
		slog.Warn("failed to scan for outdated files", "project_id", projectID, "error", err)
		return 0, previousProject, nil
	}

	// Process outdated files in background
	if len(outdatedFiles) > 0 {
		go func() {
			if err := watcher.ProcessOutdatedFiles(context.Background(), outdatedFiles); err != nil {
				slog.Warn("failed to process outdated files", "project_id", projectID, "error", err)
			}
		}()
	}

	slog.Info("project watch activated",
		"project_id", projectID,
		"previous_project", previousProject,
		"outdated_files", len(outdatedFiles))

	return len(outdatedFiles), previousProject, nil
}

// DeactivateProject stops monitoring a specific project.
// If projectID is empty, it deactivates the current active project.
func (wm *WatcherManager) DeactivateProject(ctx context.Context, projectID string) (string, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// If no project specified, deactivate current
	if projectID == "" {
		projectID = wm.activeProject
	}

	if projectID == "" {
		return "", nil // Nothing to deactivate
	}

	// Check if this is the active project
	if wm.activeProject != projectID {
		return "", fmt.Errorf("project %s is not being watched", projectID)
	}

	deactivatedProject := wm.activeProject

	// Stop the watcher
	if wm.activeWatcher != nil {
		wm.activeWatcher.Stop()
		wm.activeWatcher = nil
	}
	wm.activeProject = ""

	// Update project watcher status in storage
	if err := wm.storage.UpdateProjectWatcher(ctx, deactivatedProject, false); err != nil {
		slog.Warn("failed to update project watcher status", "project_id", deactivatedProject, "error", err)
	}

	slog.Info("project watch deactivated", "project_id", deactivatedProject)

	return deactivatedProject, nil
}

// DeactivateCurrent stops monitoring the current active project.
func (wm *WatcherManager) DeactivateCurrent(ctx context.Context) (string, error) {
	return wm.DeactivateProject(ctx, "")
}

// GetActiveProject returns the ID of the currently watched project.
func (wm *WatcherManager) GetActiveProject() string {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.activeProject
}

// IsProjectActive returns true if the specified project is currently being watched.
func (wm *WatcherManager) IsProjectActive(projectID string) bool {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.activeProject == projectID
}

// Stop stops any active watcher and cleans up resources.
// This should be called on application shutdown.
func (wm *WatcherManager) Stop() error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if wm.activeWatcher != nil {
		wm.activeWatcher.Stop()
		wm.activeWatcher = nil
	}
	wm.activeProject = ""

	slog.Info("watcher manager stopped")
	return nil
}

// AutoActivateOnStartup looks for any project with WatcherEnabled=true
// and activates monitoring for it. Should be called at application startup.
func (wm *WatcherManager) AutoActivateOnStartup(ctx context.Context) error {
	projects, err := wm.storage.ListCodeProjects(ctx)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	for _, project := range projects {
		if project.WatcherEnabled {
			slog.Info("auto-activating watcher for project", "project_id", project.ProjectID)
			_, _, err := wm.ActivateProject(ctx, project.ProjectID)
			if err != nil {
				slog.Warn("failed to auto-activate watcher",
					"project_id", project.ProjectID,
					"error", err)
				// Continue, don't fail startup for this
				continue
			}
			// Only activate one project (the first one found with WatcherEnabled)
			break
		}
	}

	return nil
}

// GetWatchStatus returns the watch status for a project or all projects.
type WatchStatus struct {
	ProjectID      string `json:"project_id"`
	WatcherEnabled bool   `json:"watcher_enabled"`
	IsActive       bool   `json:"is_active"`
}

// GetProjectWatchStatus returns the watch status for a specific project.
func (wm *WatcherManager) GetProjectWatchStatus(ctx context.Context, projectID string) (*WatchStatus, error) {
	project, err := wm.storage.GetCodeProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	wm.mu.RLock()
	isActive := wm.activeProject == projectID
	wm.mu.RUnlock()

	return &WatchStatus{
		ProjectID:      project.ProjectID,
		WatcherEnabled: project.WatcherEnabled,
		IsActive:       isActive,
	}, nil
}

// GetAllWatchStatus returns the watch status for all projects.
func (wm *WatcherManager) GetAllWatchStatus(ctx context.Context) ([]WatchStatus, error) {
	projects, err := wm.storage.ListCodeProjects(ctx)
	if err != nil {
		return nil, err
	}

	wm.mu.RLock()
	activeProject := wm.activeProject
	wm.mu.RUnlock()

	statuses := make([]WatchStatus, len(projects))
	for i, project := range projects {
		statuses[i] = WatchStatus{
			ProjectID:      project.ProjectID,
			WatcherEnabled: project.WatcherEnabled,
			IsActive:       project.ProjectID == activeProject,
		}
	}

	return statuses, nil
}
