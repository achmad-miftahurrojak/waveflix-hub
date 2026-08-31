// Package main provides async handlers for WaveFlix Hub background processing.
//
// These handlers wrap existing synchronous operations to use the message queue
// system for better performance and scalability.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// handleSendCodeAsync sends email verification code using async processing.
func handleSendCodeAsync(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid request")
		return
	}

	if req.Email == "" {
		httpError(w, http.StatusBadRequest, "email required")
		return
	}

	// Check if message queue is available
	if messageQueue == nil || !messageQueue.enabled {
		// Fallback to synchronous processing
		handleSendCode(w, r)
		return
	}

	// Generate verification code and store in database (sync)
	code, err := generateAndStoreVerificationCode(req.Email)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to generate code")
		return
	}

	// Queue email sending job for async processing
	job := &Job{
		Type:     JobTypeEmail,
		Priority: 7, // High priority for user-facing emails
		Payload: map[string]interface{}{
			"to":      req.Email,
			"subject": "Kode Verifikasi WaveFlix",
			"body":    "Kode verifikasi Anda: " + code,
			"type":    "verification",
		},
		MaxRetries: 3,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := messageQueue.EnqueueJob(ctx, job); err != nil {
		// Log error but don't fail the request - code is already stored
		log.Printf("[async] Failed to queue email job: %v", err)
		
		// Fallback to synchronous email sending
		if err := sendVerificationEmail(req.Email, job.Payload["code"].(string)); err != nil {
			log.Printf("[async] Fallback email also failed: %v", err)
		}
	}

	// Respond immediately without waiting for email delivery
	writeJSON(w, `{"success": true, "message": "Kode verifikasi telah dikirim"}`)
}

// generateAndStoreVerificationCode generates and stores verification code synchronously.
func generateAndStoreVerificationCode(email string) (string, error) {
	// Use existing generateCode function from emailverify.go
	code, err := generateCode()
	if err != nil {
		return "", err
	}
	
	// Store in database with expiration
	_, err = db.Exec(`
		INSERT OR REPLACE INTO email_verifications (email, code, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, email, code, time.Now().Add(10*time.Minute), time.Now())
	
	return code, err
}

// Enhanced cache warming job
func enqueueCacheWarmJob(cacheKey string, priority int) {
	if messageQueue == nil || !messageQueue.enabled {
		return
	}

	job := &Job{
		Type:     JobTypeCacheWarm,
		Priority: priority,
		Payload: map[string]interface{}{
			"cache_key": cacheKey,
		},
		MaxRetries: 1, // Don't retry cache warming aggressively
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	messageQueue.EnqueueJob(ctx, job)
}

// Async analytics data collection
func enqueueAnalyticsJob(eventType string, data map[string]interface{}) {
	if messageQueue == nil || !messageQueue.enabled {
		return
	}

	job := &Job{
		Type:     JobTypeAnalytics,
		Priority: 2, // Low priority for analytics
		Payload: map[string]interface{}{
			"event_type": eventType,
			"data":       data,
			"timestamp":  time.Now().Unix(),
		},
		MaxRetries: 2,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	messageQueue.EnqueueJob(ctx, job)
}

// Async data synchronization (e.g., user watchlist sync)
func enqueueDataSyncJob(syncType string, userID int, data interface{}) {
	if messageQueue == nil || !messageQueue.enabled {
		return
	}

	job := &Job{
		Type:     JobTypeDataSync,
		Priority: 5, // Medium priority
		Payload: map[string]interface{}{
			"type":    syncType,
			"user_id": userID,
			"data":    data,
		},
		MaxRetries: 3,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	messageQueue.EnqueueJob(ctx, job)
}

// Schedule cache warming for popular content
func schedulePopularContentCacheWarm() {
	if messageQueue == nil || !messageQueue.enabled {
		return
	}

	// Schedule cache warming for trending content every hour
	scheduledTime := time.Now().Add(1 * time.Hour)
	
	job := &Job{
		Type:        JobTypeCacheWarm,
		Priority:    3,
		ScheduledAt: scheduledTime,
		Payload: map[string]interface{}{
			"cache_key": "trending:movies",
			"type":      "scheduled_warm",
		},
		MaxRetries: 1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	messageQueue.EnqueueJob(ctx, job)
}

// Add queue status to health check
func getQueueHealthStatus() map[string]interface{} {
	if messageQueue == nil {
		return map[string]interface{}{
			"queue_enabled": false,
			"status":        "disabled",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	metrics, err := messageQueue.GetMetrics(ctx)
	if err != nil {
		return map[string]interface{}{
			"queue_enabled": true,
			"status":        "error",
			"error":         err.Error(),
		}
	}

	return map[string]interface{}{
		"queue_enabled":   true,
		"status":          "healthy",
		"pending_jobs":    metrics.PendingJobs,
		"active_workers":  metrics.WorkersActive,
		"failed_jobs":     metrics.FailedJobs,
	}
}