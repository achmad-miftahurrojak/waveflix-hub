package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ReadinessChecker struct {
	db          DatabaseAdapter
	redisClient *RedisCacheManager
	checks      map[string]ReadinessCheck
	mu          sync.RWMutex
}

// ReadinessCheck represents a single readiness check
type ReadinessCheck struct {
	Name        string
	CheckFunc   func() error
	Timeout     time.Duration
	Required    bool // If true, failure will mark service as not ready
	LastCheck   time.Time
	LastResult  error
	CheckCount  int64
	FailCount   int64
}

// ReadinessResult represents the result of readiness checks
type ReadinessResult struct {
	Ready      bool                       `json:"ready"`
	Timestamp  time.Time                  `json:"timestamp"`
	Checks     map[string]CheckResult     `json:"checks"`
	Summary    ReadinessSummary           `json:"summary"`
}

// CheckResult represents the result of an individual check
type CheckResult struct {
	Status      string    `json:"status"`
	Error       string    `json:"error,omitempty"`
	Duration    string    `json:"duration"`
	LastChecked time.Time `json:"last_checked"`
	Required    bool      `json:"required"`
	CheckCount  int64     `json:"check_count"`
	FailCount   int64     `json:"fail_count"`
}

// ReadinessSummary provides overall readiness statistics
type ReadinessSummary struct {
	TotalChecks    int `json:"total_checks"`
	PassedChecks   int `json:"passed_checks"`
	FailedChecks   int `json:"failed_checks"`
	RequiredFailed int `json:"required_failed"`
}

func NewReadinessChecker(db DatabaseAdapter, redisClient *RedisCacheManager) *ReadinessChecker {
	rc := &ReadinessChecker{
		db:          db,
		redisClient: redisClient,
		checks:      make(map[string]ReadinessCheck),
	}
	
	// Add default checks
	rc.addDefaultChecks()
	
	return rc
}

// addDefaultChecks adds the standard readiness checks
func (rc *ReadinessChecker) addDefaultChecks() {
	// Database connectivity check
	rc.AddCheck("database", ReadinessCheck{
		Name:      "Database Connection",
		CheckFunc: rc.checkDatabase,
		Timeout:   5 * time.Second,
		Required:  true,
	})
	
	// Redis connectivity check
	rc.AddCheck("redis", ReadinessCheck{
		Name:      "Redis Connection",
		CheckFunc: rc.checkRedis,
		Timeout:   3 * time.Second,
		Required:  true,
	})
	
	// Database schema check
	rc.AddCheck("database_schema", ReadinessCheck{
		Name:      "Database Schema",
		CheckFunc: rc.checkDatabaseSchema,
		Timeout:   10 * time.Second,
		Required:  true,
	})
	
	// Essential services check
	rc.AddCheck("worker_pools", ReadinessCheck{
		Name:      "Worker Pools",
		CheckFunc: rc.checkWorkerPools,
		Timeout:   2 * time.Second,
		Required:  false,
	})
	
	// Message queue check
	rc.AddCheck("message_queue", ReadinessCheck{
		Name:      "Message Queue",
		CheckFunc: rc.checkMessageQueue,
		Timeout:   3 * time.Second,
		Required:  false,
	})
}

// AddCheck adds a custom readiness check
func (rc *ReadinessChecker) AddCheck(key string, check ReadinessCheck) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.checks[key] = check
}

// RemoveCheck removes a readiness check
func (rc *ReadinessChecker) RemoveCheck(key string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	delete(rc.checks, key)
}

// CheckReadiness performs all readiness checks and returns the result
func (rc *ReadinessChecker) CheckReadiness() ReadinessResult {
	result := ReadinessResult{
		Ready:     true,
		Timestamp: time.Now(),
		Checks:    make(map[string]CheckResult),
		Summary:   ReadinessSummary{},
	}
	
	rc.mu.RLock()
	checks := make(map[string]ReadinessCheck)
	for k, v := range rc.checks {
		checks[k] = v
	}
	rc.mu.RUnlock()
	
	// Run all checks concurrently
	checkChan := make(chan struct {
		key    string
		result CheckResult
	}, len(checks))
	
	for key, check := range checks {
		go func(k string, c ReadinessCheck) {
			checkResult := rc.runSingleCheck(k, c)
			checkChan <- struct {
				key    string
				result CheckResult
			}{k, checkResult}
		}(key, check)
	}
	
	// Collect results
	for i := 0; i < len(checks); i++ {
		checkRes := <-checkChan
		result.Checks[checkRes.key] = checkRes.result
		
		result.Summary.TotalChecks++
		
		if checkRes.result.Status == "pass" {
			result.Summary.PassedChecks++
		} else {
			result.Summary.FailedChecks++
			
			// If this is a required check and it failed, mark as not ready
			if checkRes.result.Required {
				result.Ready = false
				result.Summary.RequiredFailed++
			}
		}
	}
	
	return result
}

// runSingleCheck executes a single readiness check
func (rc *ReadinessChecker) runSingleCheck(key string, check ReadinessCheck) CheckResult {
	start := time.Now()
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), check.Timeout)
	defer cancel()
	
	// Run the check in a goroutine to respect timeout
	errChan := make(chan error, 1)
	go func() {
		errChan <- check.CheckFunc()
	}()
	
	var err error
	select {
	case err = <-errChan:
	case <-ctx.Done():
		err = ctx.Err()
	}
	
	duration := time.Since(start)
	
	// Update check statistics
	rc.mu.Lock()
	if existingCheck, exists := rc.checks[key]; exists {
		existingCheck.LastCheck = time.Now()
		existingCheck.LastResult = err
		existingCheck.CheckCount++
		if err != nil {
			existingCheck.FailCount++
		}
		rc.checks[key] = existingCheck
	}
	rc.mu.Unlock()
	
	// Build result
	result := CheckResult{
		Duration:    duration.String(),
		LastChecked: time.Now(),
		Required:    check.Required,
		CheckCount:  check.CheckCount + 1,
		FailCount:   check.FailCount,
	}
	
	if err != nil {
		result.Status = "fail"
		result.Error = err.Error()
		result.FailCount++
	} else {
		result.Status = "pass"
	}
	
	return result
}

// Individual check functions

func (rc *ReadinessChecker) checkDatabase() error {
	if rc.db == nil {
		return fmt.Errorf("database connection not initialized")
	}
	return rc.db.Ping()
}

func (rc *ReadinessChecker) checkRedis() error {
	if rc.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}
	
	stats := rc.redisClient.GetStats()
	if !stats.RedisConnected {
		return fmt.Errorf("redis disconnected")
	}
	return nil
}

func (rc *ReadinessChecker) checkDatabaseSchema() error {
	if rc.db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	// Check if essential tables exist
	essentialTables := []string{"users", "media_items", "user_favorites"}
	
	for _, table := range essentialTables {
		var exists bool
		query := `SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		)`
		
		err := rc.db.QueryRow(query, table).Scan(&exists)
		
		if err != nil {
			return fmt.Errorf("failed to check table %s: %w", table, err)
		}
		
		if !exists {
			return fmt.Errorf("essential table %s does not exist", table)
		}
	}
	
	return nil
}

func (rc *ReadinessChecker) checkWorkerPools() error {
	if workerPoolManager == nil {
		return fmt.Errorf("worker pool manager not initialized")
	}
	
	metrics := workerPoolManager.GetMetrics()
	if metrics.ActivePools == 0 {
		return fmt.Errorf("no active worker pools")
	}
	
	return nil
}

func (rc *ReadinessChecker) checkMessageQueue() error {
	if messageQueue == nil {
		return fmt.Errorf("message queue not initialized")
	}
	
	// Try to get queue status
	status := getQueueHealthStatus()
	if status == nil {
		return fmt.Errorf("failed to get queue status")
	}
	
	return nil
}

// GetCheckHistory returns the history of a specific check
func (rc *ReadinessChecker) GetCheckHistory(checkName string) *ReadinessCheck {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	
	if check, exists := rc.checks[checkName]; exists {
		return &check
	}
	
	return nil
}

// GetAllChecks returns all configured checks
func (rc *ReadinessChecker) GetAllChecks() map[string]ReadinessCheck {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	
	checks := make(map[string]ReadinessCheck)
	for k, v := range rc.checks {
		checks[k] = v
	}
	
	return checks
}