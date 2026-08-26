package main

import (
	"net"
	"net/http"
	"os"
	"strings"
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

// clientIP — ambil IP klien. Header X-Forwarded-For/X-Real-IP hanya dipercaya
// jika env TRUST_PROXY diset (backend berada di belakang reverse proxy).
// Tanpa itu, header tersebut bisa di-spoof client untuk bypass rate limit.
func clientIP(r *http.Request) string {
	if os.Getenv("TRUST_PROXY") != "" {
		if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
			ips := strings.Split(ip, ",")
			return strings.TrimSpace(ips[0])
		}
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return strings.TrimSpace(ip)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// rateLimit — max `limit` requests per `window` per IP
func rateLimit(limit int, window time.Duration, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

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
