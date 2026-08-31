

package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type MessageQueue struct {
	client       *redis.Client
	enabled      bool
	workerPool   *WorkerPool
	jobHandlers  map[string]JobHandler
	mu           sync.RWMutex
}

type Job struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Payload     map[string]interface{} `json:"payload"`
	Priority    int                    `json:"priority"` 
	MaxRetries  int                    `json:"max_retries"`
	RetryCount  int                    `json:"retry_count"`
	ScheduledAt time.Time              `json:"scheduled_at"`
	CreatedAt   time.Time              `json:"created_at"`
}

type JobHandler func(ctx context.Context, job *Job) error

type JobResult struct {
	JobID      string        `json:"job_id"`
	Success    bool          `json:"success"`
	Error      string        `json:"error,omitempty"`
	Duration   time.Duration `json:"duration"`
	CompletedAt time.Time    `json:"completed_at"`
}

type QueueMetrics struct {
	PendingJobs    int64     `json:"pending_jobs"`
	ActiveJobs     int64     `json:"active_jobs"`
	CompletedJobs  int64     `json:"completed_jobs"`
	FailedJobs     int64     `json:"failed_jobs"`
	WorkersActive  int       `json:"workers_active"`
	LastProcessed  time.Time `json:"last_processed"`
}

type WorkerPool struct {
	workerCount int
	jobChan     chan *Job
	resultChan  chan *JobResult
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

const (

	QueueDefault    = "waveflix:queue:default"
	QueueHighPrio   = "waveflix:queue:high"
	QueueLowPrio    = "waveflix:queue:low"
	QueueScheduled  = "waveflix:queue:scheduled"
	QueueDeadLetter = "waveflix:queue:failed"

	JobTypeEmail          = "email_notification"
	JobTypeDataSync       = "data_sync"
	JobTypeImageProcess   = "image_process"
	JobTypeAnalytics      = "analytics_batch"
	JobTypeCacheWarm      = "cache_warm"
)

func NewMessageQueue() *MessageQueue {
	config := loadQueueConfig()

	rdb := redis.NewClient(&redis.Options{
		Addr:         config.RedisURL,
		Password:     config.Password,
		DB:           config.DB + 1, 
		MaxRetries:   3,
		PoolSize:     config.PoolSize,
		PoolTimeout:  30 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	enabled := true
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("[queue] Redis connection failed: %v", err)
		log.Printf("[queue] Message queue disabled - jobs will run synchronously")
		enabled = false
	} else {
		log.Printf("[queue] Message queue connected to Redis: %s", config.RedisURL)
	}

	mq := &MessageQueue{
		client:      rdb,
		enabled:     enabled,
		jobHandlers: make(map[string]JobHandler),
	}

	if enabled {
		mq.workerPool = NewWorkerPool(config.WorkerCount, mq)
		mq.registerDefaultHandlers()
		log.Printf("[queue] Started %d workers", config.WorkerCount)
	}

	return mq
}

type QueueConfig struct {
	RedisURL     string
	Password     string
	DB           int
	PoolSize     int
	WorkerCount  int
}

func loadQueueConfig() *QueueConfig {
	return &QueueConfig{
		RedisURL:    getEnvDefault("REDIS_URL", "localhost:6379"),
		Password:    os.Getenv("REDIS_PASSWORD"),
		DB:          getEnvInt("REDIS_DB", 0),
		PoolSize:    getEnvInt("REDIS_POOL_SIZE", 20),
		WorkerCount: getEnvInt("QUEUE_WORKERS", 5),
	}
}

func (mq *MessageQueue) EnqueueJob(ctx context.Context, job *Job) error {
	if !mq.enabled {

		log.Printf("[queue] Executing job %s synchronously (queue disabled)", job.Type)
		return mq.executeJobSync(ctx, job)
	}

	if job.ID == "" {
		job.ID = generateJobID()
	}

	if job.MaxRetries == 0 {
		job.MaxRetries = 3
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	if job.ScheduledAt.IsZero() {
		job.ScheduledAt = time.Now()
	}

	jobData, err := json.Marshal(job)
	if err != nil {
		return err
	}

	queueName := mq.selectQueue(job)

	if job.ScheduledAt.After(time.Now()) {

		score := float64(job.ScheduledAt.Unix())
		return mq.client.ZAdd(ctx, QueueScheduled, redis.Z{
			Score:  score,
			Member: jobData,
		}).Err()
	}

	return mq.client.LPush(ctx, queueName, jobData).Err()
}

func (mq *MessageQueue) selectQueue(job *Job) string {
	switch {
	case job.Priority >= 8:
		return QueueHighPrio
	case job.Priority <= 3:
		return QueueLowPrio
	default:
		return QueueDefault
	}
}

func (mq *MessageQueue) RegisterHandler(jobType string, handler JobHandler) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.jobHandlers[jobType] = handler
	log.Printf("[queue] Registered handler for job type: %s", jobType)
}

func (mq *MessageQueue) executeJobSync(ctx context.Context, job *Job) error {
	mq.mu.RLock()
	handler, exists := mq.jobHandlers[job.Type]
	mq.mu.RUnlock()

	if !exists {
		log.Printf("[queue] No handler registered for job type: %s", job.Type)
		return nil 
	}

	return handler(ctx, job)
}

func (mq *MessageQueue) processJob(ctx context.Context, job *Job) *JobResult {
	start := time.Now()
	result := &JobResult{
		JobID:       job.ID,
		CompletedAt: time.Now(),
	}

	mq.mu.RLock()
	handler, exists := mq.jobHandlers[job.Type]
	mq.mu.RUnlock()

	if !exists {
		result.Error = "no handler registered for job type: " + job.Type
		result.Success = false
		result.Duration = time.Since(start)
		return result
	}

	jobCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	err := handler(jobCtx, job)
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = err.Error()
		result.Success = false
		log.Printf("[queue] Job %s (%s) failed: %v", job.ID, job.Type, err)
	} else {
		result.Success = true
		log.Printf("[queue] Job %s (%s) completed in %v", job.ID, job.Type, result.Duration)
	}

	return result
}

func (mq *MessageQueue) GetMetrics(ctx context.Context) (*QueueMetrics, error) {
	if !mq.enabled {
		return &QueueMetrics{WorkersActive: 0}, nil
	}

	metrics := &QueueMetrics{}

	pipe := mq.client.Pipeline()
	defaultLen := pipe.LLen(ctx, QueueDefault)
	highLen := pipe.LLen(ctx, QueueHighPrio)
	lowLen := pipe.LLen(ctx, QueueLowPrio)
	_ = pipe.ZCard(ctx, QueueScheduled)
	failedLen := pipe.LLen(ctx, QueueDeadLetter)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	metrics.PendingJobs = defaultLen.Val() + highLen.Val() + lowLen.Val()
	metrics.FailedJobs = failedLen.Val()

	if mq.workerPool != nil {
		metrics.WorkersActive = mq.workerPool.workerCount
	}

	return metrics, nil
}

func (mq *MessageQueue) Close() error {
	if mq.workerPool != nil {
		mq.workerPool.Stop()
	}

	if mq.client != nil {
		return mq.client.Close()
	}

	return nil
}

func NewWorkerPool(workerCount int, mq *MessageQueue) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	wp := &WorkerPool{
		workerCount: workerCount,
		jobChan:     make(chan *Job, 100),
		resultChan:  make(chan *JobResult, 100),
		ctx:         ctx,
		cancel:      cancel,
	}

	for i := 0; i < workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(i, mq)
	}

	go wp.fetchJobs(mq)

	go wp.processResults()

	return wp
}

func (wp *WorkerPool) worker(id int, mq *MessageQueue) {
	defer wp.wg.Done()

	for {
		select {
		case <-wp.ctx.Done():
			return
		case job := <-wp.jobChan:
			result := mq.processJob(wp.ctx, job)

			select {
			case wp.resultChan <- result:
			case <-wp.ctx.Done():
				return
			}
		}
	}
}

func (wp *WorkerPool) fetchJobs(mq *MessageQueue) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-wp.ctx.Done():
			return
		case <-ticker.C:
			wp.fetchFromQueues(mq)
		}
	}
}

func (wp *WorkerPool) fetchFromQueues(mq *MessageQueue) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wp.processScheduledJobs(ctx, mq)

	queues := []string{QueueHighPrio, QueueDefault, QueueLowPrio}

	for _, queueName := range queues {
		result := mq.client.BRPop(ctx, 100*time.Millisecond, queueName)
		if result.Err() != nil {
			continue 
		}

		var job Job
		if err := json.Unmarshal([]byte(result.Val()[1]), &job); err != nil {
			log.Printf("[queue] Failed to unmarshal job: %v", err)
			continue
		}

		select {
		case wp.jobChan <- &job:
		case <-wp.ctx.Done():
			return
		default:

			mq.client.LPush(ctx, queueName, result.Val()[1])
		}
	}
}

func (wp *WorkerPool) processScheduledJobs(ctx context.Context, mq *MessageQueue) {
	now := time.Now().Unix()

	result := mq.client.ZRangeByScore(ctx, QueueScheduled, &redis.ZRangeBy{
		Min: "0",
		Max: strconv.FormatInt(now, 10),
	})

	for _, jobData := range result.Val() {
		var job Job
		if err := json.Unmarshal([]byte(jobData), &job); err != nil {
			continue
		}

		queueName := mq.selectQueue(&job)
		mq.client.LPush(ctx, queueName, jobData)
		mq.client.ZRem(ctx, QueueScheduled, jobData)
	}
}

func (wp *WorkerPool) processResults() {
	for {
		select {
		case <-wp.ctx.Done():
			return
		case result := <-wp.resultChan:

			if result.Success {
				log.Printf("[queue] Job %s completed successfully", result.JobID)
			} else {
				log.Printf("[queue] Job %s failed: %s", result.JobID, result.Error)

			}
		}
	}
}

func (wp *WorkerPool) Stop() {
	wp.cancel()
	wp.wg.Wait()
	close(wp.jobChan)
	close(wp.resultChan)
}

func generateJobID() string {
	return "job_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func (mq *MessageQueue) registerDefaultHandlers() {
	mq.RegisterHandler(JobTypeEmail, mq.handleEmailJob)
	mq.RegisterHandler(JobTypeDataSync, mq.handleDataSyncJob)
	mq.RegisterHandler(JobTypeImageProcess, mq.handleImageProcessJob)
	mq.RegisterHandler(JobTypeAnalytics, mq.handleAnalyticsJob)
	mq.RegisterHandler(JobTypeCacheWarm, mq.handleCacheWarmJob)
}

func (mq *MessageQueue) handleEmailJob(ctx context.Context, job *Job) error {

	to, _ := job.Payload["to"].(string)
	subject, _ := job.Payload["subject"].(string)
	body, _ := job.Payload["body"].(string)

	log.Printf("[queue] Processing email job: to=%s, subject=%s", to, subject)

	if to != "" && subject != "" && body != "" {
		return sendVerificationEmail(to, body)
	}

	return nil
}

func (mq *MessageQueue) handleDataSyncJob(ctx context.Context, job *Job) error {
	syncType, _ := job.Payload["type"].(string)
	log.Printf("[queue] Processing data sync job: type=%s", syncType)

	return nil
}

func (mq *MessageQueue) handleImageProcessJob(ctx context.Context, job *Job) error {
	imageURL, _ := job.Payload["url"].(string)
	operation, _ := job.Payload["operation"].(string)

	log.Printf("[queue] Processing image job: url=%s, operation=%s", imageURL, operation)

	return nil
}

func (mq *MessageQueue) handleAnalyticsJob(ctx context.Context, job *Job) error {
	eventType, _ := job.Payload["event_type"].(string)
	log.Printf("[queue] Processing analytics job: event_type=%s", eventType)

	return nil
}

func (mq *MessageQueue) handleCacheWarmJob(ctx context.Context, job *Job) error {
	cacheKey, _ := job.Payload["cache_key"].(string)
	log.Printf("[queue] Processing cache warm job: key=%s", cacheKey)

	return nil
}

func (mq *MessageQueue) Shutdown(ctx context.Context) error {
	if mq == nil || !mq.enabled {
		return nil
	}
	log.Println("[queue] Shutting down message queue...")
	if mq.workerPool != nil {
		if err := mq.workerPool.Shutdown(ctx); err != nil {
			return err
		}
	}
	if mq.client != nil {
		return mq.client.Close()
	}
	return nil
}

func (wp *WorkerPool) Shutdown(ctx context.Context) error {
	if wp == nil {
		return nil
	}
	if wp.cancel != nil {
		wp.cancel()
	}

	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (wp *WorkerPool) GetActiveJobCount() int {
	if wp == nil {
		return 0
	}

	return len(wp.jobChan)
}