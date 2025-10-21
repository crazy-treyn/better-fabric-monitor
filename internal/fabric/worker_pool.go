package fabric

import (
	"context"
	"sync"
)

const (
	// Concurrency limits
	MaxWorkspaceConcurrency = 8  // Process 8 workspaces in parallel
	MaxItemConcurrency      = 5  // 5 items per workspace initially
	MaxTotalConcurrency     = 80 // Global max concurrent requests
)

// WorkerPool manages concurrent execution of jobs
type WorkerPool struct {
	maxWorkers int
	semaphore  chan struct{}
	wg         sync.WaitGroup
}

// NewWorkerPool creates a new worker pool with the specified max workers
func NewWorkerPool(maxWorkers int) *WorkerPool {
	return &WorkerPool{
		maxWorkers: maxWorkers,
		semaphore:  make(chan struct{}, maxWorkers),
	}
}

// Submit submits a job to the worker pool
func (wp *WorkerPool) Submit(ctx context.Context, job func() error) {
	wp.wg.Add(1)

	go func() {
		defer wp.wg.Done()

		// Acquire semaphore
		select {
		case wp.semaphore <- struct{}{}:
			defer func() { <-wp.semaphore }()

			// Execute job
			if err := job(); err != nil {
				// Errors are handled by the job function itself
				// We don't propagate them here as we want to continue with other jobs
			}

		case <-ctx.Done():
			return
		}
	}()
}

// Wait waits for all jobs to complete
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

// WorkspaceResult holds the result of processing a workspace
type WorkspaceResult struct {
	WorkspaceID   string
	WorkspaceName string
	Jobs          []map[string]interface{}
	Items         []Item
	Error         error
}

// ItemResult holds the result of processing an item
type ItemResult struct {
	WorkspaceID   string
	WorkspaceName string
	Item          Item
	Jobs          []map[string]interface{}
	Error         error
}
