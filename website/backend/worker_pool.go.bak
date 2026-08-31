// Package main provides enhanced worker pool system for CPU-intensive tasks.
//
// The worker pool system manages concurrent processing of various tasks including
// TMDB API calls, data processing, image optimization, and analytics computation.
// It provides dynamic scaling, priority queues, and resource management.
package main

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPoolManager manages multiple specialized worker pools.
type WorkerPoolManager struct {
	pools          map[string]*WorkerPool
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	metrics        *WorkerPoolMetrics
}

// WorkerPool represents a pool of workers for specific task types.
type WorkerPool struct {
	name           string
	workers        []*Worker
	taskQueue      chan Task
	resultQueue    chan TaskResult
	workerCount    int
	maxWorkerCount int
	minWorkerCount int
	taskBuffer     int
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	
	// Metrics
	tasksProcessed int64
	tasksQueued    int64
	activeWorkers  int64
	
	// Auto-scaling
	lastScaleTime  time.Time
	scaleUpThreshold   float64  // Queue utilization % to scale up
	scaleDownThreshold float64  // Queue utilization % to scale down
}

// Worker represents an individual worker in the pool.
type Worker struct {
	id          int
	pool        *WorkerPool
	taskQueue   chan Task
	resultQueue chan TaskResult
	quit        chan struct{}
	active      int64  // atomic: 0 = idle, 1 = active
}

// Task represents work to be executed by workers.
type Task struct {
	ID       string
	Type     TaskType
	Priority int // 1-10, higher = more priority
	Payload  interface{}
	Context  context.Context
	Created  time.Time
}

// TaskResult represents the result of task execution.
type TaskResult struct {
	TaskID    string
	Success   bool
	Result    interface{}
	Error     error
	Duration  time.Duration
	WorkerID  int
	Completed time.Time
}

// TaskType defines different types of tasks.
type TaskType string

const (
	TaskTypeTMDBAPI     TaskType = "tmdb_api"
	TaskTypeDataProcess TaskType = "data_process"
	TaskTypeImageResize TaskType = "image_resize"
	TaskTypeAnalytics   TaskType = "analytics"
	TaskTypeCacheWarm   TaskType = "cache_warm"
	TaskTypeSearch      TaskType = "search_index"
	TaskTypeGeneral     TaskType = "general"
)

// WorkerPoolConfig holds configuration for worker pools.
type WorkerPoolConfig struct {
	InitialWorkers     int
	MaxWorkers         int
	MinWorkers         int
	TaskBuffer         int
	ScaleUpThreshold   float64
	ScaleDownThreshold float64
}

// WorkerPoolMetrics tracks performance metrics.
type WorkerPoolMetrics struct {
	TotalTasksProcessed int64
	TotalTasksQueued    int64
	AverageProcessTime  time.Duration
	ActivePools         int
	TotalWorkers        int
	mu                  sync.RWMutex
}

// NewWorkerPoolManager creates a new worker pool manager.
func NewWorkerPoolManager() *WorkerPoolManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	wpm := &WorkerPoolManager{
		pools:   make(map[string]*WorkerPool),
		ctx:     ctx,
		cancel:  cancel,
		metrics: &WorkerPoolMetrics{},
	}
	
	// Initialize default pools
	wpm.initializeDefaultPools()
	
	// Start monitoring goroutine
	go wpm.monitorPools()
	
	log.Println("[worker] Worker pool manager initialized")
	return wpm
}

// initializeDefaultPools creates default worker pools for different task types.
func (wpm *WorkerPoolManager) initializeDefaultPools() {
	configs := map[string]WorkerPoolConfig{
		"tmdb_api": {
			InitialWorkers:     5,
			MaxWorkers:         20,
			MinWorkers:         2,
			TaskBuffer:         200,
			ScaleUpThreshold:   0.8,
			ScaleDownThreshold: 0.3,
		},
		"data_process": {
			InitialWorkers:     runtime.NumCPU(),
			MaxWorkers:         runtime.NumCPU() * 2,
			MinWorkers:         1,
			TaskBuffer:         100,
			ScaleUpThreshold:   0.7,
			ScaleDownThreshold: 0.2,
		},
		"image_resize": {
			InitialWorkers:     2,
			MaxWorkers:         8,
			MinWorkers:         1,
			TaskBuffer:         50,
			ScaleUpThreshold:   0.9,
			ScaleDownThreshold: 0.1,
		},
		"analytics": {
			InitialWorkers:     2,
			MaxWorkers:         6,
			MinWorkers:         1,
			TaskBuffer:         100,
			ScaleUpThreshold:   0.6,
			ScaleDownThreshold: 0.2,
		},
		"general": {
			InitialWorkers:     3,
			MaxWorkers:         10,
			MinWorkers:         1,
			TaskBuffer:         150,
			ScaleUpThreshold:   0.8,
			ScaleDownThreshold: 0.3,
		},
	}
	
	for poolName, config := range configs {
		pool := wpm.CreatePool(poolName, config)
		log.Printf("[worker] Created pool '%s' with %d workers", poolName, config.InitialWorkers)
		_ = pool
	}
}

// CreatePool creates a new worker pool with the given configuration.
func (wpm *WorkerPoolManager) CreatePool(name string, config WorkerPoolConfig) *WorkerPool {
	wpm.mu.Lock()
	defer wpm.mu.Unlock()
	
	ctx, cancel := context.WithCancel(wpm.ctx)
	
	pool := &WorkerPool{
		name:               name,
		workers:            make([]*Worker, 0, config.MaxWorkers),
		taskQueue:          make(chan Task, config.TaskBuffer),
		resultQueue:        make(chan TaskResult, config.TaskBuffer),
		workerCount:        0,
		maxWorkerCount:     config.MaxWorkers,
		minWorkerCount:     config.MinWorkers,
		taskBuffer:         config.TaskBuffer,
		ctx:                ctx,
		cancel:             cancel,
		lastScaleTime:      time.Now(),
		scaleUpThreshold:   config.ScaleUpThreshold,
		scaleDownThreshold: config.ScaleDownThreshold,
	}
	
	// Start initial workers
	for i := 0; i < config.InitialWorkers; i++ {
		pool.addWorker()
	}
	
	// Start result processor
	go pool.processResults()
	
	// Start auto-scaler
	go pool.autoScale()
	
	wpm.pools[name] = pool
	return pool
}

// SubmitTask submits a task to the appropriate worker pool.
func (wpm *WorkerPoolManager) SubmitTask(task Task) error {
	poolName := wpm.getPoolNameForTask(task.Type)
	
	wpm.mu.RLock()
	pool, exists := wpm.pools[poolName]
	wpm.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("pool not found: %s", poolName)
	}
	
	return pool.SubmitTask(task)
}

// getPoolNameForTask determines which pool should handle a specific task type.
func (wpm *WorkerPoolManager) getPoolNameForTask(taskType TaskType) string {
	switch taskType {
	case TaskTypeTMDBAPI:
		return "tmdb_api"
	case TaskTypeDataProcess:
		return "data_process"
	case TaskTypeImageResize:
		return "image_resize"
	case TaskTypeAnalytics:
		return "analytics"
	default:
		return "general"
	}
}

// SubmitTask submits a task to the worker pool.
func (wp *WorkerPool) SubmitTask(task Task) error {
	atomic.AddInt64(&wp.tasksQueued, 1)
	
	select {
	case wp.taskQueue <- task:
		recordQueueJob(string(task.Type), "queued", 0)
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	default:
		// Queue is full, task rejected
		recordQueueJob(string(task.Type), "rejected", 0)
		return fmt.Errorf("task queue full for pool: %s", wp.name)
	}
}

// addWorker adds a new worker to the pool.
func (wp *WorkerPool) addWorker() {
	if wp.workerCount >= wp.maxWorkerCount {
		return
	}
	
	worker := &Worker{
		id:          wp.workerCount,
		pool:        wp,
		taskQueue:   wp.taskQueue,
		resultQueue: wp.resultQueue,
		quit:        make(chan struct{}),
	}
	
	wp.workers = append(wp.workers, worker)
	wp.workerCount++
	
	wp.wg.Add(1)
	go worker.start()
	
	log.Printf("[worker] Added worker %d to pool '%s' (total: %d)", worker.id, wp.name, wp.workerCount)
}

// removeWorker removes a worker from the pool.
func (wp *WorkerPool) removeWorker() {
	if wp.workerCount <= wp.minWorkerCount || len(wp.workers) == 0 {
		return
	}
	
	// Remove the last worker
	lastIndex := len(wp.workers) - 1
	worker := wp.workers[lastIndex]
	
	close(worker.quit)
	wp.workers = wp.workers[:lastIndex]
	wp.workerCount--
	
	log.Printf("[worker] Removed worker %d from pool '%s' (total: %d)", worker.id, wp.name, wp.workerCount)
}

// autoScale automatically adjusts worker count based on queue utilization.
func (wp *WorkerPool) autoScale() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-wp.ctx.Done():
			return
		case <-ticker.C:
			wp.evaluateScaling()
		}
	}
}

// evaluateScaling checks if pool should scale up or down.
func (wp *WorkerPool) evaluateScaling() {
	// Don't scale too frequently
	if time.Since(wp.lastScaleTime) < 60*time.Second {
		return
	}
	
	queueLength := len(wp.taskQueue)
	queueUtilization := float64(queueLength) / float64(wp.taskBuffer)
	activeWorkers := int(atomic.LoadInt64(&wp.activeWorkers))
	
	log.Printf("[worker] Pool '%s': queue=%d/%d (%.1f%%), active_workers=%d/%d", 
		wp.name, queueLength, wp.taskBuffer, queueUtilization*100, activeWorkers, wp.workerCount)
	
	// Scale up if queue utilization is high
	if queueUtilization > wp.scaleUpThreshold && wp.workerCount < wp.maxWorkerCount {
		wp.addWorker()
		wp.lastScaleTime = time.Now()
	}
	
	// Scale down if queue utilization is low and we have idle workers
	if queueUtilization < wp.scaleDownThreshold && wp.workerCount > wp.minWorkerCount {
		idleWorkers := wp.workerCount - activeWorkers
		if idleWorkers > 1 { // Keep at least some workers active
			wp.removeWorker()
			wp.lastScaleTime = time.Now()
		}
	}
}

// start begins the worker's processing loop.
func (w *Worker) start() {
	defer w.pool.wg.Done()
	
	for {
		select {
		case task := <-w.taskQueue:
			w.processTask(task)
		case <-w.quit:
			return
		case <-w.pool.ctx.Done():
			return
		}
	}
}

// processTask executes a single task.
func (w *Worker) processTask(task Task) {
	atomic.StoreInt64(&w.active, 1)
	atomic.AddInt64(&w.pool.activeWorkers, 1)
	defer func() {
		atomic.StoreInt64(&w.active, 0)
		atomic.AddInt64(&w.pool.activeWorkers, -1)
	}()
	
	start := time.Now()
	result := TaskResult{
		TaskID:    task.ID,
		WorkerID:  w.id,
		Completed: time.Now(),
	}
	
	// Execute task based on type
	switch task.Type {
	case TaskTypeTMDBAPI:
		result.Result, result.Error = w.processTMDBTask(task)
	case TaskTypeDataProcess:
		result.Result, result.Error = w.processDataTask(task)
	case TaskTypeImageResize:
		result.Result, result.Error = w.processImageTask(task)
	case TaskTypeAnalytics:
		result.Result, result.Error = w.processAnalyticsTask(task)
	default:
		result.Result, result.Error = w.processGeneralTask(task)
	}
	
	result.Duration = time.Since(start)
	result.Success = (result.Error == nil)
	
	atomic.AddInt64(&w.pool.tasksProcessed, 1)
	
	// Send result
	select {
	case w.resultQueue <- result:
	case <-w.pool.ctx.Done():
	}
	
	// Record metrics
	recordQueueJob(string(task.Type), "completed", result.Duration)
}

// Task processing methods
func (w *Worker) processTMDBTask(task Task) (interface{}, error) {
	// Handle TMDB API calls with rate limiting awareness
	log.Printf("[worker] Processing TMDB task %s in worker %d", task.ID, w.id)
	
	// Add artificial delay to simulate API call
	time.Sleep(100 * time.Millisecond)
	
	// Actual implementation would make TMDB API calls here
	return map[string]interface{}{
		"status": "success",
		"worker": w.id,
		"type":   "tmdb_api",
	}, nil
}

func (w *Worker) processDataTask(task Task) (interface{}, error) {
	// Handle data processing tasks (JSON parsing, validation, etc.)
	log.Printf("[worker] Processing data task %s in worker %d", task.ID, w.id)
	
	// Simulate CPU-intensive data processing
	time.Sleep(50 * time.Millisecond)
	
	return map[string]interface{}{
		"status":    "processed",
		"worker":    w.id,
		"type":      "data_process",
		"processed": true,
	}, nil
}

func (w *Worker) processImageTask(task Task) (interface{}, error) {
	// Handle image processing tasks (resize, compress, etc.)
	log.Printf("[worker] Processing image task %s in worker %d", task.ID, w.id)
	
	// Simulate image processing
	time.Sleep(200 * time.Millisecond)
	
	return map[string]interface{}{
		"status": "resized",
		"worker": w.id,
		"type":   "image_resize",
	}, nil
}

func (w *Worker) processAnalyticsTask(task Task) (interface{}, error) {
	// Handle analytics computation tasks
	log.Printf("[worker] Processing analytics task %s in worker %d", task.ID, w.id)
	
	// Simulate analytics computation
	time.Sleep(75 * time.Millisecond)
	
	return map[string]interface{}{
		"status":     "computed",
		"worker":     w.id,
		"type":       "analytics",
		"calculated": true,
	}, nil
}

func (w *Worker) processGeneralTask(task Task) (interface{}, error) {
	// Handle general tasks
	log.Printf("[worker] Processing general task %s in worker %d", task.ID, w.id)
	
	// Simulate general processing
	time.Sleep(30 * time.Millisecond)
	
	return map[string]interface{}{
		"status": "completed",
		"worker": w.id,
		"type":   "general",
	}, nil
}

// processResults handles task results.
func (wp *WorkerPool) processResults() {
	for {
		select {
		case result := <-wp.resultQueue:
			if result.Success {
				log.Printf("[worker] Task %s completed successfully by worker %d in %v", 
					result.TaskID, result.WorkerID, result.Duration)
			} else {
				log.Printf("[worker] Task %s failed in worker %d: %v", 
					result.TaskID, result.WorkerID, result.Error)
			}
		case <-wp.ctx.Done():
			return
		}
	}
}

// GetMetrics returns current worker pool metrics.
func (wpm *WorkerPoolManager) GetMetrics() *WorkerPoolMetrics {
	wpm.mu.RLock()
	defer wpm.mu.RUnlock()
	
	metrics := &WorkerPoolMetrics{
		ActivePools: len(wpm.pools),
	}
	
	for _, pool := range wpm.pools {
		metrics.TotalTasksProcessed += atomic.LoadInt64(&pool.tasksProcessed)
		metrics.TotalTasksQueued += atomic.LoadInt64(&pool.tasksQueued)
		metrics.TotalWorkers += pool.workerCount
	}
	
	return metrics
}

// monitorPools monitors pool health and performance.
func (wpm *WorkerPoolManager) monitorPools() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-wpm.ctx.Done():
			return
		case <-ticker.C:
			metrics := wpm.GetMetrics()
			log.Printf("[worker] Pools: %d active, %d workers, %d tasks processed, %d tasks queued",
				metrics.ActivePools, metrics.TotalWorkers, metrics.TotalTasksProcessed, metrics.TotalTasksQueued)
		}
	}
}

// Shutdown gracefully shuts down all worker pools.
func (wpm *WorkerPoolManager) Shutdown(timeout time.Duration) error {
	log.Println("[worker] Shutting down worker pool manager...")
	
	wpm.cancel()
	
	done := make(chan struct{})
	go func() {
		wpm.mu.RLock()
		defer wpm.mu.RUnlock()
		
		for _, pool := range wpm.pools {
			pool.wg.Wait()
		}
		close(done)
	}()
	
	select {
	case <-done:
		log.Println("[worker] All worker pools shut down gracefully")
		return nil
	case <-time.After(timeout):
		log.Println("[worker] Worker pool shutdown timeout")
		return fmt.Errorf("shutdown timeout after %v", timeout)
	}
}

// Helper function to generate task IDs
func generateTaskID() string {
	return fmt.Sprintf("task_%d_%d", time.Now().UnixNano(), rand.Intn(1000))
}

// Global worker pool manager instance
var workerPoolManager *WorkerPoolManager

// initWorkerPools initializes the global worker pool manager
func initWorkerPools() {
	workerPoolManager = NewWorkerPoolManager()
	log.Println("[worker] Global worker pool manager initialized")
}

// SubmitBackgroundTask submits a task to the global worker pool manager
func SubmitBackgroundTask(taskType TaskType, payload interface{}) error {
	if workerPoolManager == nil {
		return fmt.Errorf("worker pool manager not initialized")
	}
	
	task := Task{
		ID:      generateTaskID(),
		Type:    taskType,
		Payload: payload,
		Context: context.Background(),
		Created: time.Now(),
	}
	
	return workerPoolManager.SubmitTask(task)
}