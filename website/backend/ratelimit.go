package main

import (
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func trustedProxy(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, entry := range strings.Split(os.Getenv("TRUSTED_PROXY_IPS"), ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			_, network, err := net.ParseCIDR(entry)
			if err == nil && network.Contains(ip) {
				return true
			}
			continue
		}
		if net.ParseIP(entry) != nil && net.ParseIP(entry).Equal(ip) {
			return true
		}
	}
	return false
}

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
	if trustedProxy(r.RemoteAddr) {
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

// rateLimit — max `limit` requests per `window` per IP, dengan `scope`
// sebagai pemisah bucket (login punya bucket sendiri, public API punya
// bucket sendiri — aktivitas browsing tidak menghabiskan kuota login).
func rateLimit(scope string, limit int, window time.Duration, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := scope + "|" + clientIP(r)

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
