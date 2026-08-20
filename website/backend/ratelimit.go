package main

import (
	"net/http"
	"sync"
	"time"
)

type visitor struct {
	count    int
	lastSeen time.Time
}

var (
	visitors = make(map[string]*visitor)
	mu       sync.Mutex
)

// cleanupVisitors — goroutine untuk bersihkan visitor lama
func init() {
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()
}

// rateLimit — max `limit` requests per `window` per IP
func rateLimit(limit int, window time.Duration, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		mu.Lock()
		v, exists := visitors[ip]
		if !exists || time.Since(v.lastSeen) > window {
			visitors[ip] = &visitor{count: 1, lastSeen: time.Now()}
			mu.Unlock()
			next(w, r)
			return
		}
		v.count++
		v.lastSeen = time.Now()
		if v.count > limit {
			mu.Unlock()
			w.Header().Set("Retry-After", "60")
			httpError(w, http.StatusTooManyRequests, "terlalu banyak percobaan, coba lagi nanti")
			return
		}
		mu.Unlock()
		next(w, r)
	}
}
