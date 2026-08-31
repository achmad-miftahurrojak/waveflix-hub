package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// messageQueue adalah global instance untuk async job processing.
// Bernilai nil jika Redis tidak tersedia.
var messageQueue *MessageQueue

// startTime mencatat waktu server start untuk penghitungan uptime.
var startTime = time.Now()

// cacheItem menyimpan satu entry memory-cache fallback.
// Fields disesuaikan dengan redis_cache.go: data []byte dan expireAt time.Time.
type cacheItem struct {
	data     []byte
	expireAt time.Time
}

// writeJSON menulis string JSON langsung ke ResponseWriter.
// Dipakai oleh handler yang hanya perlu menulis raw JSON literal.
func writeJSON(w http.ResponseWriter, jsonStr string) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, jsonStr)
}

// cache adalah global RedisCacheManager instance.
// Bernilai nil jika Redis tidak tersedia saat startup.
var cache *RedisCacheManager

// normalizeMedia memastikan media_type selalu "movie" atau "tv".
func normalizeMedia(s string) string {
	if s == "tv" {
		return "tv"
	}
	return "movie"
}

// firstNonEmpty mengembalikan string pertama yang tidak kosong.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// getIDParam mengambil numeric ID dari URL query (id atau tmdb_id).
// Hanya mengizinkan angka positif tanpa leading zeros.
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

// parseBatchIDs memecah string comma-separated IDs menjadi slice string.
// Mengabaikan entri kosong atau non-numeric.
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

// SubmitBackgroundTask memasukkan task ke message queue.
// Jika queue tidak aktif, task di-skip (non-critical).
func SubmitBackgroundTask(taskType string, payload map[string]interface{}) error {
	if messageQueue == nil || !messageQueue.enabled {
		return nil
	}
	job := &Job{
		Type:    taskType,
		Payload: payload,
	}
	return messageQueue.EnqueueJob(context.Background(), job)
}

// ─── Worker Pool Manager ──────────────────────────────────────────────────────

// WorkerPoolManagerMetrics merupakan statistik worker pool manager.
type WorkerPoolManagerMetrics struct {
	// ActivePools adalah jumlah pool yang aktif saat ini.
	ActivePools int
}

// WorkerPoolManager mengelola beberapa worker pool untuk berbagai jenis task.
// Saat ini merupakan stub — production implementation akan mengelola pool nyata.
type WorkerPoolManager struct{}

// GetMetrics mengembalikan statistik worker pool saat ini.
func (wpm *WorkerPoolManager) GetMetrics() WorkerPoolManagerMetrics {
	// Jika messageQueue aktif, anggap ada 1 pool yang berjalan.
	if messageQueue != nil && messageQueue.enabled {
		return WorkerPoolManagerMetrics{ActivePools: 1}
	}
	return WorkerPoolManagerMetrics{ActivePools: 0}
}

// workerPoolManager adalah global instance worker pool manager.
var workerPoolManager *WorkerPoolManager
