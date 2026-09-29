// Package indexer provides async job management for code indexing.
package indexer

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/madeindigio/remembrances-mcp/v2/internal/storage"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/embedder"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/principal"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/treesitter"
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
	ID           string
	ProjectID    string
	ProjectPath  string
	ProjectName  string
	Status       treesitter.IndexingStatus
	Progress     float64
	FilesTotal   int
	FilesIndexed int
	SymbolsFound int
	StartedAt    time.Time
	CompletedAt  *time.Time
	Error        *string
	CreatedAt    time.Time

	// owner is the principal the job runs as: the identity of whoever
	// submitted it, captured at submission because the submitting request is
	// long gone by the time a worker picks the job up. It is unexported so a
	// caller cannot set an identity the submission did not carry, and it is
	// never nil (SubmitJob falls back to the service principal).
	owner *principal.Principal
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

// SubmitJob submits a new indexing job.
//
// ctx is the submitting caller's context and is used for its identity only:
// the job runs on a worker goroutine long after this call returns, so the
// principal is captured here and reinstalled on the job's own context. A
// submission with no principal runs as the service principal rather than as
// nobody.
func (jm *JobManager) SubmitJob(ctx context.Context, projectPath, projectName string) (*Job, error) {
	// Generate job ID
	jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())

	owner, _ := principal.FromContext(storage.ServiceContext(ctx))

	job := &Job{
		ID:          jobID,
		ProjectPath: projectPath,
		ProjectName: projectName,
		Status:      treesitter.IndexingStatusPending,
		CreatedAt:   time.Now(),
		owner:       owner,
	}

	jm.mu.Lock()
	jm.jobs[jobID] = job
	jm.mu.Unlock()

	// Send to queue
	select {
	case jm.jobQueue <- job:
		slog.Info("Job queued for project", "job_id", jobID, "project", projectPath)
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

// processJob executes an indexing job.
//
// The job's context is fresh (a queued job must not inherit the deadline of
// the request that queued it) but never anonymous: it carries the principal
// SubmitJob captured, so every storage call the indexer makes is scoped to
// whoever asked for the indexing.
func (jm *JobManager) processJob(job *Job) {
	ctx, cancel := context.WithCancel(jm.jobContext(job))

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

	slog.Info("Starting job for project", "job_id", job.ID, "project", job.ProjectPath)

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
		slog.Error("Job failed", "job_id", job.ID, "error", err)
	} else {
		job.Status = treesitter.IndexingStatusCompleted

		// Get final stats from progress
		if progress := jm.indexer.GetProgress(projectID); progress != nil {
			job.FilesTotal = progress.FilesTotal
			job.FilesIndexed = progress.FilesIndexed
			job.SymbolsFound = progress.SymbolsFound
		}

		slog.Info("Job completed", "job_id", job.ID, "files", job.FilesIndexed, "symbols", job.SymbolsFound)
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
		slog.Warn("failed to persist job record", "error", err)
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

// GetJobStatus returns the current status of a job.
//
// ctx is the caller's, because a job that is no longer in the in-memory map is
// read back from storage and that read is scoped like any other: asking for
// someone else's job is a lookup that must find nothing, not a lookup that
// runs unscoped.
func (jm *JobManager) GetJobStatus(ctx context.Context, jobID string) (*Job, error) {
	jm.mu.RLock()
	job, exists := jm.jobs[jobID]
	jm.mu.RUnlock()

	if !exists {
		// Try to find in database
		dbJob, err := jm.storage.GetIndexingJob(ctx, jobID)
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

	slog.Info("Job cancelled", "job_id", jobID)
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
	slog.Info("Job manager stopped")
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
	return jm.SubmitJob(ctx, project.RootPath, project.Name)
}

// jobContext builds the context a queued job runs on: detached from every
// deadline and cancellation of the submission, carrying the principal the
// submission had.
//
// A job whose owner is missing (only possible for a Job value built outside
// SubmitJob, which the tests do) runs as the service principal, so there is no
// path through this function that produces an unscoped context.
func (jm *JobManager) jobContext(job *Job) context.Context {
	if job == nil || job.owner == nil {
		return storage.ServiceContext(context.Background())
	}
	return principal.NewContext(context.Background(), job.owner)
}
