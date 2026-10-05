package queue

import (
"container/heap"
"context"
"encoding/json"
"fmt"
"log"
"sync"
"time"

"github.com/go-redis/redis/v8"
)

type Priority int

const (
LowPriority Priority = iota
NormalPriority
HighPriority
CriticalPriority
)

type JobStatus string

const (
StatusPending   JobStatus = "pending"
StatusRunning   JobStatus = "running"
StatusCompleted JobStatus = "completed"
StatusFailed    JobStatus = "failed"
StatusRetrying  JobStatus = "retrying"
)

type JobType string

const (
JobTypeTranscode JobType = "transcode"
JobTypeSegment   JobType = "segment"
JobTypeThumbnail JobType = "thumbnail"
JobTypeAnalyze   JobType = "analyze"
)

type Job struct {
ID           string                 `json:"id"`
Type         JobType                `json:"type"`
Priority     Priority               `json:"priority"`
Payload      map[string]interface{} `json:"payload"`
Status       JobStatus              `json:"status"`
CreatedAt    time.Time              `json:"created_at"`
StartedAt    *time.Time             `json:"started_at,omitempty"`
CompletedAt  *time.Time             `json:"completed_at,omitempty"`
RetryCount   int                    `json:"retry_count"`
MaxRetries   int                    `json:"max_retries"`
RetryDelay   time.Duration          `json:"retry_delay"`
LastError    string                 `json:"last_error,omitempty"`
WorkerID     string                 `json:"worker_id,omitempty"`
Progress     float64                `json:"progress"`
Metadata     map[string]interface{} `json:"metadata"`
}

type JobResult struct {
JobID   string                 `json:"job_id"`
Success bool                   `json:"success"`
Error   string                 `json:"error,omitempty"`
Result  map[string]interface{} `json:"result,omitempty"`
Output  map[string]interface{} `json:"output,omitempty"`
}

type Worker interface {
ProcessJob(ctx context.Context, job *Job) (*JobResult, error)
CanHandle(jobType JobType) bool
GetWorkerID() string
}

type PriorityQueue struct {
items []*Job
mutex sync.RWMutex
}

func (pq *PriorityQueue) Len() int {
return len(pq.items)
}

func (pq *PriorityQueue) Less(i, j int) bool {
if pq.items[i].Priority != pq.items[j].Priority {
return pq.items[i].Priority > pq.items[j].Priority
}
return pq.items[i].CreatedAt.Before(pq.items[j].CreatedAt)
}

func (pq *PriorityQueue) Swap(i, j int) {
pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
pq.items = append(pq.items, x.(*Job))
}

func (pq *PriorityQueue) Pop() interface{} {
old := pq.items
n := len(old)
item := old[n-1]
pq.items = old[0 : n-1]
return item
}

type QueueManager struct {
priorityQueue *PriorityQueue
redisClient   *redis.Client
workers       map[string]Worker
runningJobs   map[string]*Job
jobStatuses   map[string]*Job
maxConcurrent int
retryConfig   RetryConfig
metrics       *QueueMetrics
stopChan      chan struct{}
mutex         sync.RWMutex
}

type RetryConfig struct {
MaxRetries    int
InitialDelay  time.Duration
MaxDelay      time.Duration
BackoffFactor float64
}

type QueueMetrics struct {
JobsQueued    int64
JobsRunning   int64
JobsCompleted int64
JobsFailed    int64
AvgDuration   time.Duration
mutex         sync.RWMutex
}

func NewQueueManager(redisClient *redis.Client, maxConcurrent int) *QueueManager {
return &QueueManager{
priorityQueue: &PriorityQueue{},
redisClient:   redisClient,
workers:       make(map[string]Worker),
runningJobs:   make(map[string]*Job),
jobStatuses:   make(map[string]*Job),
maxConcurrent: maxConcurrent,
retryConfig: RetryConfig{
MaxRetries:    3,
InitialDelay:  30 * time.Second,
MaxDelay:      300 * time.Second,
BackoffFactor: 2.0,
},
metrics:  &QueueMetrics{},
stopChan: make(chan struct{}),
}
}

func (qm *QueueManager) RegisterWorker(worker Worker) {
qm.mutex.Lock()
defer qm.mutex.Unlock()
qm.workers[worker.GetWorkerID()] = worker
log.Printf("[Queue] Registered worker: %s", worker.GetWorkerID())
}

func (qm *QueueManager) EnqueueJob(job *Job) error {
job.CreatedAt = time.Now()
job.Status = StatusPending
job.Progress = 0.0

if job.MaxRetries == 0 {
job.MaxRetries = qm.retryConfig.MaxRetries
}
if job.RetryDelay == 0 {
job.RetryDelay = qm.retryConfig.InitialDelay
}

qm.mutex.Lock()
qm.jobStatuses[job.ID] = job
heap.Push(qm.priorityQueue, job)
qm.mutex.Unlock()

qm.metrics.mutex.Lock()
qm.metrics.JobsQueued++
qm.metrics.mutex.Unlock()

err := qm.persistJob(job)
if err != nil {
log.Printf("[Queue] Failed to persist job %s: %v", job.ID, err)
}

log.Printf("[Queue] Enqueued job %s (type: %s, priority: %d)", job.ID, job.Type, job.Priority)
return nil
}

func (qm *QueueManager) Start(ctx context.Context) {
log.Println("[Queue] Starting queue manager")

go qm.processJobs(ctx)
go qm.retryFailedJobs(ctx)
go qm.cleanupCompletedJobs(ctx)
go qm.monitorWorkers(ctx)

<-ctx.Done()
close(qm.stopChan)
log.Println("[Queue] Queue manager stopped")
}

func (qm *QueueManager) processJobs(ctx context.Context) {
ticker := time.NewTicker(1 * time.Second)
defer ticker.Stop()

for {
select {
case <-ctx.Done():
return
case <-ticker.C:
qm.processNextJob(ctx)
}
}
}

func (qm *QueueManager) processNextJob(ctx context.Context) {
qm.mutex.Lock()

if len(qm.runningJobs) >= qm.maxConcurrent {
qm.mutex.Unlock()
return
}

if qm.priorityQueue.Len() == 0 {
qm.mutex.Unlock()
return
}

job := heap.Pop(qm.priorityQueue).(*Job)
qm.runningJobs[job.ID] = job
qm.mutex.Unlock()

go qm.executeJob(ctx, job)
}

func (qm *QueueManager) executeJob(ctx context.Context, job *Job) {
defer func() {
qm.mutex.Lock()
delete(qm.runningJobs, job.ID)
qm.mutex.Unlock()

qm.metrics.mutex.Lock()
qm.metrics.JobsRunning--
qm.metrics.mutex.Unlock()
}()

startTime := time.Now()
job.StartedAt = &startTime
job.Status = StatusRunning

qm.metrics.mutex.Lock()
qm.metrics.JobsRunning++
qm.metrics.mutex.Unlock()

worker := qm.findWorkerForJob(job)
if worker == nil {
qm.handleJobFailure(job, fmt.Errorf("no worker available for job type: %s", job.Type))
return
}

job.WorkerID = worker.GetWorkerID()
qm.updateJobStatus(job)

log.Printf("[Queue] Processing job %s with worker %s", job.ID, job.WorkerID)

result, err := worker.ProcessJob(ctx, job)
completedAt := time.Now()
job.CompletedAt = &completedAt

if err != nil {
qm.handleJobFailure(job, err)
return
}

if result.Success {
job.Status = StatusCompleted
job.Progress = 100.0
qm.metrics.mutex.Lock()
qm.metrics.JobsCompleted++
duration := completedAt.Sub(startTime)
if qm.metrics.AvgDuration == 0 {
qm.metrics.AvgDuration = duration
} else {
qm.metrics.AvgDuration = (qm.metrics.AvgDuration + duration) / 2
}
qm.metrics.mutex.Unlock()

log.Printf("[Queue] Job %s completed successfully in %v", job.ID, duration)
} else {
qm.handleJobFailure(job, fmt.Errorf(result.Error))
}

qm.updateJobStatus(job)
}

func (qm *QueueManager) findWorkerForJob(job *Job) Worker {
qm.mutex.RLock()
defer qm.mutex.RUnlock()

for _, worker := range qm.workers {
if worker.CanHandle(job.Type) {
return worker
}
}
return nil
}

func (qm *QueueManager) handleJobFailure(job *Job, err error) {
job.LastError = err.Error()
job.RetryCount++

log.Printf("[Queue] Job %s failed (attempt %d/%d): %v", job.ID, job.RetryCount, job.MaxRetries, err)

if job.RetryCount <= job.MaxRetries {
job.Status = StatusRetrying
delay := qm.calculateRetryDelay(job.RetryCount)

go func() {
time.Sleep(delay)
qm.mutex.Lock()
heap.Push(qm.priorityQueue, job)
qm.mutex.Unlock()
log.Printf("[Queue] Job %s scheduled for retry in %v", job.ID, delay)
}()
} else {
job.Status = StatusFailed
qm.metrics.mutex.Lock()
qm.metrics.JobsFailed++
qm.metrics.mutex.Unlock()
log.Printf("[Queue] Job %s permanently failed after %d attempts", job.ID, job.RetryCount)
}
}

func (qm *QueueManager) calculateRetryDelay(attempt int) time.Duration {
delay := time.Duration(float64(qm.retryConfig.InitialDelay) * 
float64(attempt) * qm.retryConfig.BackoffFactor)

if delay > qm.retryConfig.MaxDelay {
delay = qm.retryConfig.MaxDelay
}

return delay
}

func (qm *QueueManager) retryFailedJobs(ctx context.Context) {
ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()

for {
select {
case <-ctx.Done():
return
case <-ticker.C:
qm.processRetryableJobs()
}
}
}

func (qm *QueueManager) processRetryableJobs() {
// Implementation for processing retryable jobs from Redis
// This would load failed jobs from Redis and requeue them if appropriate
}

func (qm *QueueManager) cleanupCompletedJobs(ctx context.Context) {
ticker := time.NewTicker(5 * time.Minute)
defer ticker.Stop()

for {
select {
case <-ctx.Done():
return
case <-ticker.C:
qm.cleanupOldJobs()
}
}
}

func (qm *QueueManager) cleanupOldJobs() {
cutoff := time.Now().Add(-24 * time.Hour)

qm.mutex.Lock()
for id, job := range qm.jobStatuses {
if job.Status == StatusCompleted && job.CompletedAt != nil && job.CompletedAt.Before(cutoff) {
delete(qm.jobStatuses, id)
qm.deletePersistedJob(id)
}
}
qm.mutex.Unlock()
}

func (qm *QueueManager) monitorWorkers(ctx context.Context) {
ticker := time.NewTicker(60 * time.Second)
defer ticker.Stop()

for {
select {
case <-ctx.Done():
return
case <-ticker.C:
qm.logMetrics()
}
}
}

func (qm *QueueManager) logMetrics() {
qm.metrics.mutex.RLock()
qm.mutex.RLock()

log.Printf("[Queue] Metrics - Queued: %d, Running: %d, Completed: %d, Failed: %d, Workers: %d, Avg Duration: %v",
qm.metrics.JobsQueued-qm.metrics.JobsCompleted-qm.metrics.JobsFailed,
len(qm.runningJobs),
qm.metrics.JobsCompleted,
qm.metrics.JobsFailed,
len(qm.workers),
qm.metrics.AvgDuration)

qm.mutex.RUnlock()
qm.metrics.mutex.RUnlock()
}

func (qm *QueueManager) GetJobStatus(jobID string) (*Job, error) {
qm.mutex.RLock()
job, exists := qm.jobStatuses[jobID]
qm.mutex.RUnlock()

if !exists {
return qm.loadJobFromRedis(jobID)
}

return job, nil
}

func (qm *QueueManager) GetQueueMetrics() *QueueMetrics {
qm.metrics.mutex.RLock()
defer qm.metrics.mutex.RUnlock()

return &QueueMetrics{
JobsQueued:    qm.metrics.JobsQueued,
JobsRunning:   qm.metrics.JobsRunning,
JobsCompleted: qm.metrics.JobsCompleted,
JobsFailed:    qm.metrics.JobsFailed,
AvgDuration:   qm.metrics.AvgDuration,
}
}

func (qm *QueueManager) persistJob(job *Job) error {
if qm.redisClient == nil {
return nil
}

jobData, err := json.Marshal(job)
if err != nil {
return err
}

ctx := context.Background()
key := fmt.Sprintf("job:%s", job.ID)

return qm.redisClient.Set(ctx, key, jobData, 24*time.Hour).Err()
}

func (qm *QueueManager) updateJobStatus(job *Job) error {
qm.mutex.Lock()
qm.jobStatuses[job.ID] = job
qm.mutex.Unlock()

return qm.persistJob(job)
}

func (qm *QueueManager) loadJobFromRedis(jobID string) (*Job, error) {
if qm.redisClient == nil {
return nil, fmt.Errorf("job not found: %s", jobID)
}

ctx := context.Background()
key := fmt.Sprintf("job:%s", jobID)

data, err := qm.redisClient.Get(ctx, key).Result()
if err != nil {
return nil, fmt.Errorf("job not found: %s", jobID)
}

var job Job
err = json.Unmarshal([]byte(data), &job)
if err != nil {
return nil, err
}

return &job, nil
}

func (qm *QueueManager) deletePersistedJob(jobID string) {
if qm.redisClient == nil {
return
}

ctx := context.Background()
key := fmt.Sprintf("job:%s", jobID)
qm.redisClient.Del(ctx, key)
}

func (qm *QueueManager) BatchEnqueue(jobs []*Job) error {
qm.mutex.Lock()
defer qm.mutex.Unlock()

for _, job := range jobs {
job.CreatedAt = time.Now()
job.Status = StatusPending
job.Progress = 0.0

if job.MaxRetries == 0 {
job.MaxRetries = qm.retryConfig.MaxRetries
}
if job.RetryDelay == 0 {
job.RetryDelay = qm.retryConfig.InitialDelay
}

qm.jobStatuses[job.ID] = job
heap.Push(qm.priorityQueue, job)
qm.persistJob(job)
}

qm.metrics.mutex.Lock()
qm.metrics.JobsQueued += int64(len(jobs))
qm.metrics.mutex.Unlock()

log.Printf("[Queue] Batch enqueued %d jobs", len(jobs))
return nil
}

func (qm *QueueManager) GetQueueStatus() map[string]interface{} {
qm.mutex.RLock()
qm.metrics.mutex.RLock()

status := map[string]interface{}{
"queue_length":    qm.priorityQueue.Len(),
"running_jobs":    len(qm.runningJobs),
"total_jobs":      len(qm.jobStatuses),
"active_workers":  len(qm.workers),
"max_concurrent":  qm.maxConcurrent,
"jobs_queued":     qm.metrics.JobsQueued,
"jobs_completed":  qm.metrics.JobsCompleted,
"jobs_failed":     qm.metrics.JobsFailed,
"avg_duration_ms": qm.metrics.AvgDuration.Milliseconds(),
}

qm.metrics.mutex.RUnlock()
qm.mutex.RUnlock()

return status
}
