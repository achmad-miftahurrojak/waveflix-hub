

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

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

	if messageQueue == nil || !messageQueue.enabled {

		handleSendCode(w, r)
		return
	}

	code, err := generateAndStoreVerificationCode(req.Email)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to generate code")
		return
	}

	job := &Job{
		Type:     JobTypeEmail,
		Priority: 7, 
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

		log.Printf("[async] Failed to queue email job: %v", err)

		if err := sendVerificationEmail(req.Email, job.Payload["code"].(string)); err != nil {
			log.Printf("[async] Fallback email also failed: %v", err)
		}
	}

	writeJSON(w, `{"success": true, "message": "Kode verifikasi telah dikirim"}`)
}

func generateAndStoreVerificationCode(email string) (string, error) {

	code, err := generateCode()
	if err != nil {
		return "", err
	}

	_, err = db.Exec(
		`INSERT INTO email_verifications(user_id, token, expires_at) VALUES(0, $1, $2)`,
		code+"|||"+email, time.Now().Add(10*time.Minute),
	)

	return code, err
}

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
		MaxRetries: 1, 
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	messageQueue.EnqueueJob(ctx, job)
}

func enqueueAnalyticsJob(eventType string, data map[string]interface{}) {
	if messageQueue == nil || !messageQueue.enabled {
		return
	}

	job := &Job{
		Type:     JobTypeAnalytics,
		Priority: 2, 
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

func enqueueDataSyncJob(syncType string, userID int, data interface{}) {
	if messageQueue == nil || !messageQueue.enabled {
		return
	}

	job := &Job{
		Type:     JobTypeDataSync,
		Priority: 5, 
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

func schedulePopularContentCacheWarm() {
	if messageQueue == nil || !messageQueue.enabled {
		return
	}

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