package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var messageQueue *MessageQueue

var startTime = time.Now()

type cacheItem struct {
	data     []byte
	expireAt time.Time
}

func writeJSON(w http.ResponseWriter, jsonStr string) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, jsonStr)
}

var cache *RedisCacheManager

func normalizeMedia(s string) string {
	if s == "tv" {
		return "tv"
	}
	return "movie"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func getIDParam(q interface{ Get(string) string }) string {
	var raw string
	if v := q.Get("id"); v != "" {
		raw = v
	} else if v := q.Get("tmdb_id"); v != "" {
		raw = v
	}
	if raw == "" {
		return ""
	}
	for _, c := range raw {
		if c < '0' || c > '9' {
			return ""
		}
	}
	if len(raw) > 1 && raw[0] == '0' {
		return ""
	}
	return raw
}

const maxBatchIDs = 50

func parseBatchIDs(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		valid := true
		for _, c := range p {
			if c < '0' || c > '9' {
				valid = false
				break
			}
		}
		if valid && p != "" {
			result = append(result, p)
			if len(result) >= maxBatchIDs {
				break
			}
		}
	}
	return result
}

func SubmitBackgroundTask(taskType string, payload map[string]interface{}) error {
	if messageQueue == nil || !messageQueue.enabled {
		return fmt.Errorf("message queue disabled")
	}
	job := &Job{
		Type:    taskType,
		Payload: payload,
	}
	return messageQueue.EnqueueJob(context.Background(), job)
}

type WorkerPoolManagerMetrics struct {

	ActivePools int
}

type WorkerPoolManager struct{}

func (wpm *WorkerPoolManager) GetMetrics() WorkerPoolManagerMetrics {

	if messageQueue != nil && messageQueue.enabled {
		return WorkerPoolManagerMetrics{ActivePools: 1}
	}
	return WorkerPoolManagerMetrics{ActivePools: 0}
}

var workerPoolManager *WorkerPoolManager
