// Package indexer provides async job management for code indexing.
package indexer

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/embedder"
	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// JobManager manages asynchronous indexing jobs
type JobManager struct {
	indexer *Indexer
	storage storage.FullStorage

	// Active jobs
	mu      sync.RWMutex
	jobs    map[string]*Job
	running map[string]context.CancelFunc

	// Job queue
	jobQueue chan *Job
	quit     chan struct{}
	wg       sync.WaitGroup

	// Configuration
	maxConcurrentJobs int
}

// Job represents an indexing job
type Job struct {
	ID          string
	ProjectID   string
	ProjectPath string
	ProjectName string
	Status      treesitter.IndexingStatus
	Progress    float64
	FilesTotal  int
	FilesIndexed int
	SymbolsFound int
	StartedAt   time.Time
	CompletedAt *time.Time
	Error       *string
	CreatedAt   time.Time
}

// JobManagerConfig holds configuration for the job manager
type JobManagerConfig struct {
	MaxConcurrentJobs int
	QueueSize         int
}

// DefaultJobManagerConfig returns sensible defaults
func DefaultJobManagerConfig() JobManagerConfig {
	return JobManagerConfig{
		MaxConcurrentJobs: 2,
		QueueSize:         100,
	}
}

// NewJobManager creates a new job manager
func NewJobManager(storage storage.FullStorage, embedder embedder.Embedder, indexerConfig IndexerConfig, config JobManagerConfig) *JobManager {
	indexer := NewIndexer(storage, embedder, indexerConfig)

	jm := &JobManager{
		indexer:           indexer,
		storage:           storage,
		jobs:              make(map[string]*Job),
		running:           make(map[string]context.CancelFunc),
		jobQueue:          make(chan *Job, config.QueueSize),
		quit:              make(chan struct{}),
		maxConcurrentJobs: config.MaxConcurrentJobs,
	}

	// Start worker goroutines
	jm.wg.Add(config.MaxConcurrentJobs)
	for i := 0; i < config.MaxConcurrentJobs; i++ {
		go jm.worker()
	}

	return jm
}

// SubmitJob submits a new indexing job
func (jm *JobManager) SubmitJob(projectPath, projectName string) (*Job, error) {
	// Generate job ID
	jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())

	job := &Job{
		ID:          jobID,
		ProjectPath: projectPath,
		ProjectName: projectName,
		Status:      treesitter.IndexingStatusPending,
		CreatedAt:   time.Now(),
	}

	jm.mu.Lock()
	jm.jobs[jobID] = job
	jm.mu.Unlock()

	// Send to queue
	select {
	case jm.jobQueue <- job:
		log.Printf("Job %s queued for project: %s", jobID, projectPath)
		return job, nil
	default:
		jm.mu.Lock()
		delete(jm.jobs, jobID)
		jm.mu.Unlock()
		return nil, fmt.Errorf("job queue is full")
	}
}

// worker processes jobs from the queue
func (jm *JobManager) worker() {
	defer jm.wg.Done()

	for {
		select {
		case <-jm.quit:
			return
		case job := <-jm.jobQueue:
			jm.processJob(job)
		}
	}
}

// processJob executes an indexing job
func (jm *JobManager) processJob(job *Job) {
	ctx, cancel := context.WithCancel(context.Background())

	jm.mu.Lock()
	jm.running[job.ID] = cancel
	job.Status = treesitter.IndexingStatusInProgress
	job.StartedAt = time.Now()
	jm.mu.Unlock()

	defer func() {
		jm.mu.Lock()
		delete(jm.running, job.ID)
		jm.mu.Unlock()
		cancel()
	}()

	log.Printf("Starting job %s for project: %s", job.ID, job.ProjectPath)

	// Run indexing
	projectID, err := jm.indexer.IndexProject(ctx, job.ProjectPath, job.ProjectName)

	jm.mu.Lock()
	defer jm.mu.Unlock()

	job.ProjectID = projectID
	now := time.Now()
	job.CompletedAt = &now

	if err != nil {
		errStr := err.Error()
		job.Error = &errStr
		job.Status = treesitter.IndexingStatusFailed
		log.Printf("Job %s failed: %v", job.ID, err)
	} else {
		job.Status = treesitter.IndexingStatusCompleted

		// Get final stats from progress
		if progress := jm.indexer.GetProgress(projectID); progress != nil {
			job.FilesTotal = progress.FilesTotal
			job.FilesIndexed = progress.FilesIndexed
			job.SymbolsFound = progress.SymbolsFound
		}

		log.Printf("Job %s completed: %d files, %d symbols", job.ID, job.FilesIndexed, job.SymbolsFound)
	}

	// Persist job to database
	indexingJob := &treesitter.IndexingJob{
		ID:           job.ID,
		ProjectID:    job.ProjectID,
		ProjectPath:  job.ProjectPath,
		Status:       job.Status,
		Progress:     100.0,
		FilesTotal:   job.FilesTotal,
		FilesIndexed: job.FilesIndexed,
		StartedAt:    job.StartedAt,
		CompletedAt:  job.CompletedAt,
		Error:        job.Error,
	}

	if _, err := jm.storage.CreateIndexingJob(ctx, indexingJob); err != nil {
		log.Printf("Warning: failed to persist job record: %v", err)
	}
}

// GetJob returns a job by ID
func (jm *JobManager) GetJob(jobID string) *Job {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	if job, ok := jm.jobs[jobID]; ok {
		copy := *job
		return &copy
	}
	return nil
}

// GetJobStatus returns the current status of a job
func (jm *JobManager) GetJobStatus(jobID string) (*Job, error) {
	jm.mu.RLock()
	job, exists := jm.jobs[jobID]
	jm.mu.RUnlock()

	if !exists {
		// Try to find in database
		dbJob, err := jm.storage.GetIndexingJob(context.Background(), jobID)
		if err != nil {
			return nil, fmt.Errorf("failed to get job: %w", err)
		}
		if dbJob == nil {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}

		return &Job{
			ID:           dbJob.ID,
			ProjectID:    dbJob.ProjectID,
			ProjectPath:  dbJob.ProjectPath,
			Status:       dbJob.Status,
			Progress:     dbJob.Progress,
			FilesTotal:   dbJob.FilesTotal,
			FilesIndexed: dbJob.FilesIndexed,
			StartedAt:    dbJob.StartedAt,
			CompletedAt:  dbJob.CompletedAt,
			Error:        dbJob.Error,
		}, nil
	}

	// Update progress from indexer
	if job.Status == treesitter.IndexingStatusInProgress {
		if progress := jm.indexer.GetProgress(job.ProjectID); progress != nil {
			job.FilesTotal = progress.FilesTotal
			job.FilesIndexed = progress.FilesIndexed
			job.SymbolsFound = progress.SymbolsFound
			if progress.FilesTotal > 0 {
				job.Progress = float64(progress.FilesIndexed) / float64(progress.FilesTotal) * 100
			}
		}
	}

	copy := *job
	return &copy, nil
}

// ListActiveJobs returns all currently active jobs
func (jm *JobManager) ListActiveJobs() []*Job {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	jobs := make([]*Job, 0)
	for _, job := range jm.jobs {
		if job.Status == treesitter.IndexingStatusPending ||
			job.Status == treesitter.IndexingStatusInProgress {
			copy := *job
			jobs = append(jobs, &copy)
		}
	}
	return jobs
}

// ListAllJobs returns all jobs (active and completed)
func (jm *JobManager) ListAllJobs() []*Job {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	jobs := make([]*Job, 0, len(jm.jobs))
	for _, job := range jm.jobs {
		copy := *job
		jobs = append(jobs, &copy)
	}
	return jobs
}

// CancelJob attempts to cancel a running job
func (jm *JobManager) CancelJob(jobID string) error {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	cancel, ok := jm.running[jobID]
	if !ok {
		return fmt.Errorf("job not running: %s", jobID)
	}

	cancel()

	if job, ok := jm.jobs[jobID]; ok {
		job.Status = treesitter.IndexingStatusCancelled
		errStr := "cancelled by user"
		job.Error = &errStr
		now := time.Now()
		job.CompletedAt = &now
	}

	log.Printf("Job %s cancelled", jobID)
	return nil
}

// Stop gracefully stops the job manager
func (jm *JobManager) Stop() {
	close(jm.quit)

	// Cancel all running jobs
	jm.mu.Lock()
	for _, cancel := range jm.running {
		cancel()
	}
	jm.mu.Unlock()

	// Wait for workers to finish
	jm.wg.Wait()
	log.Println("Job manager stopped")
}

// CleanupOldJobs removes completed jobs older than the specified duration
func (jm *JobManager) CleanupOldJobs(olderThan time.Duration) int {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	removed := 0

	for id, job := range jm.jobs {
		if job.Status == treesitter.IndexingStatusCompleted ||
			job.Status == treesitter.IndexingStatusFailed ||
			job.Status == treesitter.IndexingStatusCancelled {
			if job.CompletedAt != nil && job.CompletedAt.Before(cutoff) {
				delete(jm.jobs, id)
				removed++
			}
		}
	}

	return removed
}

// GetIndexer returns the underlying indexer
func (jm *JobManager) GetIndexer() *Indexer {
	return jm.indexer
}

// ReindexProject queues a project for re-indexing
func (jm *JobManager) ReindexProject(ctx context.Context, projectID string) (*Job, error) {
	// Get project info
	project, err := jm.storage.GetCodeProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	// Submit new job
	return jm.SubmitJob(project.RootPath, project.Name)
}
