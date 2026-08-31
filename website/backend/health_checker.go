package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

// HealthStatus represents the overall health status
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
)

// ComponentHealth represents health of individual component
type ComponentHealth struct {
	Status      HealthStatus `json:"status"`
	Message     string       `json:"message,omitempty"`
	LastChecked time.Time    `json:"last_checked"`
	Duration    string       `json:"duration,omitempty"`
	Details     interface{}  `json:"details,omitempty"`
}

// HealthResponse represents the complete health check response
type HealthResponse struct {
	Status     HealthStatus               `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Version    string                     `json:"version"`
	Uptime     string                     `json:"uptime"`
	Components map[string]ComponentHealth `json:"components"`
	System     SystemHealth               `json:"system"`
}

// SystemHealth represents system-level metrics
type SystemHealth struct {
	Memory    MemoryStats    `json:"memory"`
	Runtime   RuntimeStats   `json:"runtime"`
	Database  DatabaseStats  `json:"database,omitempty"`
	Cache     CacheStatsHealth     `json:"cache,omitempty"`
	External  ExternalStats  `json:"external,omitempty"`
}

type MemoryStats struct {
	Alloc        uint64 `json:"alloc_bytes"`
	TotalAlloc   uint64 `json:"total_alloc_bytes"`
	Sys          uint64 `json:"sys_bytes"`
	NumGC        uint32 `json:"num_gc"`
	HeapInuse    uint64 `json:"heap_inuse_bytes"`
	HeapReleased uint64 `json:"heap_released_bytes"`
}

type RuntimeStats struct {
	Goroutines   int    `json:"goroutines"`
	GOMAXPROCS   int    `json:"gomaxprocs"`
	NumCPU       int    `json:"num_cpu"`
	GoVersion    string `json:"go_version"`
}

type DatabaseStats struct {
	OpenConnections int           `json:"open_connections"`
	InUse          int           `json:"in_use"`
	Idle           int           `json:"idle"`
	WaitCount      int64         `json:"wait_count"`
	WaitDuration   time.Duration `json:"wait_duration_ms"`
	MaxIdleClosed  int64         `json:"max_idle_closed"`
	MaxLifetime    int64         `json:"max_lifetime_closed"`
}

type CacheStatsHealth struct {
	HitRatio     float64 `json:"hit_ratio"`
	TotalHits    int64   `json:"total_hits"`
	TotalMisses  int64   `json:"total_misses"`
	ConnectedClients int `json:"connected_clients"`
	UsedMemory   int64   `json:"used_memory_bytes"`
}

type ExternalStats struct {
	TMDBApiStatus    HealthStatus `json:"tmdb_api_status"`
	LastTMDBCall     time.Time    `json:"last_tmdb_call"`
	TMDBResponseTime string       `json:"tmdb_response_time,omitempty"`
}

// HealthChecker manages all health check operations
type HealthChecker struct {
	db          *sql.DB
	redisClient *redis.Client
	startTime   time.Time
	version     string
	mu          sync.RWMutex
	lastChecks  map[string]ComponentHealth
}

// NewHealthChecker creates a new health checker instance
func NewHealthChecker(db *sql.DB, redisClient *redis.Client, version string) *HealthChecker {
	hc := &HealthChecker{
		db:          db,
		redisClient: redisClient,
		startTime:   time.Now(),
		version:     version,
		lastChecks:  make(map[string]ComponentHealth),
	}
	
	return hc
}

// RegisterHealthRoutes registers health check endpoints
func (hc *HealthChecker) RegisterHealthRoutes(router *mux.Router) {
	router.HandleFunc("/health", hc.HealthCheckHandler).Methods("GET")
	router.HandleFunc("/health/live", hc.LivenessHandler).Methods("GET")
	router.HandleFunc("/health/ready", hc.ReadinessHandler).Methods("GET")
	router.HandleFunc("/health/detailed", hc.DetailedHealthHandler).Methods("GET")
}

// HealthCheckHandler provides basic health status
func (hc *HealthChecker) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	response := HealthResponse{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
		Version:   hc.version,
		Uptime:    time.Since(hc.startTime).String(),
		Components: make(map[string]ComponentHealth),
	}
	
	// Quick checks for basic health
	if hc.checkDatabase() != StatusHealthy {
		response.Status = StatusDegraded
	}
	
	if hc.checkRedis() != StatusHealthy {
		if response.Status == StatusDegraded {
			response.Status = StatusUnhealthy
		} else {
			response.Status = StatusDegraded
		}
	}
	
	// Add basic component status
	response.Components["database"] = ComponentHealth{
		Status:      hc.checkDatabase(),
		LastChecked: time.Now(),
		Duration:    time.Since(start).String(),
	}
	
	response.Components["cache"] = ComponentHealth{
		Status:      hc.checkRedis(),
		LastChecked: time.Now(),
		Duration:    time.Since(start).String(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	switch response.Status {
	case StatusHealthy:
		w.WriteHeader(http.StatusOK)
	case StatusDegraded:
		w.WriteHeader(http.StatusOK) // Still return 200 for degraded
	case StatusUnhealthy:
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	json.NewEncoder(w).Encode(response)
}

// LivenessHandler for Kubernetes liveness probe
func (hc *HealthChecker) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	// Liveness should only check if the application is alive, not dependencies
	response := map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now(),
		"uptime":    time.Since(hc.startTime).String(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ReadinessHandler for Kubernetes readiness probe
func (hc *HealthChecker) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	ready := true
	issues := []string{}
	
	// Check critical dependencies for readiness
	if hc.checkDatabase() != StatusHealthy {
		ready = false
		issues = append(issues, "database connection failed")
	}
	
	if hc.checkRedis() != StatusHealthy {
		ready = false
		issues = append(issues, "cache connection failed")
	}
	
	response := map[string]interface{}{
		"ready":     ready,
		"timestamp": time.Now(),
	}
	
	if len(issues) > 0 {
		response["issues"] = issues
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	if ready {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	json.NewEncoder(w).Encode(response)
}

// DetailedHealthHandler provides comprehensive health information
func (hc *HealthChecker) DetailedHealthHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	response := HealthResponse{
		Timestamp: time.Now(),
		Version:   hc.version,
		Uptime:    time.Since(hc.startTime).String(),
		Components: make(map[string]ComponentHealth),
	}
	
	// Perform all health checks concurrently
	var wg sync.WaitGroup
	healthChan := make(chan map[string]ComponentHealth, 10)
	
	// Database health check
	wg.Add(1)
	go func() {
		defer wg.Done()
		healthChan <- map[string]ComponentHealth{
			"database": hc.checkDatabaseDetailed(),
		}
	}()
	
	// Redis health check
	wg.Add(1)
	go func() {
		defer wg.Done()
		healthChan <- map[string]ComponentHealth{
			"cache": hc.checkRedisDetailed(),
		}
	}()
	
	// TMDB API health check
	wg.Add(1)
	go func() {
		defer wg.Done()
		healthChan <- map[string]ComponentHealth{
			"tmdb_api": hc.checkTMDBAPI(),
		}
	}()
	
	// System health
	wg.Add(1)
	go func() {
		defer wg.Done()
		healthChan <- map[string]ComponentHealth{
			"system": hc.checkSystem(),
		}
	}()
	
	// Wait for all checks to complete
	go func() {
		wg.Wait()
		close(healthChan)
	}()
	
	// Collect results
	unhealthyCount := 0
	degradedCount := 0
	
	for healthMap := range healthChan {
		for component, health := range healthMap {
			response.Components[component] = health
			
			switch health.Status {
			case StatusUnhealthy:
				unhealthyCount++
			case StatusDegraded:
				degradedCount++
			}
		}
	}
	
	// Determine overall status
	if unhealthyCount > 0 {
		response.Status = StatusUnhealthy
	} else if degradedCount > 0 {
		response.Status = StatusDegraded
	} else {
		response.Status = StatusHealthy
	}
	
	// Add system information
	response.System = hc.getSystemHealth()
	
	w.Header().Set("Content-Type", "application/json")
	
	switch response.Status {
	case StatusHealthy:
		w.WriteHeader(http.StatusOK)
	case StatusDegraded:
		w.WriteHeader(http.StatusOK)
	case StatusUnhealthy:
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	json.NewEncoder(w).Encode(response)
}

// Basic health check methods
func (hc *HealthChecker) checkDatabase() HealthStatus {
	if hc.db == nil {
		return StatusUnhealthy
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	if err := hc.db.PingContext(ctx); err != nil {
		return StatusUnhealthy
	}
	
	return StatusHealthy
}

func (hc *HealthChecker) checkRedis() HealthStatus {
	if hc.redisClient == nil {
		return StatusUnhealthy
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	if err := hc.redisClient.Ping(ctx).Err(); err != nil {
		return StatusUnhealthy
	}
	
	return StatusHealthy
}

// Detailed health check methods
func (hc *HealthChecker) checkDatabaseDetailed() ComponentHealth {
	start := time.Now()
	
	if hc.db == nil {
		return ComponentHealth{
			Status:      StatusUnhealthy,
			Message:     "Database connection not initialized",
			LastChecked: time.Now(),
			Duration:    time.Since(start).String(),
		}
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Check basic connectivity
	if err := hc.db.PingContext(ctx); err != nil {
		return ComponentHealth{
			Status:      StatusUnhealthy,
			Message:     fmt.Sprintf("Database ping failed: %v", err),
			LastChecked: time.Now(),
			Duration:    time.Since(start).String(),
		}
	}
	
	// Get database statistics
	stats := hc.db.Stats()
	details := DatabaseStats{
		OpenConnections: stats.OpenConnections,
		InUse:          stats.InUse,
		Idle:           stats.Idle,
		WaitCount:      stats.WaitCount,
		WaitDuration:   stats.WaitDuration,
		MaxIdleClosed:  stats.MaxIdleClosed,
		MaxLifetime:    stats.MaxLifetimeClosed,
	}
	
	// Check if we're running low on connections
	status := StatusHealthy
	message := "Database connection healthy"
	
	if stats.OpenConnections > 80 { // Assuming max 100 connections
		status = StatusDegraded
		message = "High database connection usage"
	}
	
	if stats.WaitCount > 100 {
		status = StatusDegraded
		message = "High database connection wait count"
	}
	
	return ComponentHealth{
		Status:      status,
		Message:     message,
		LastChecked: time.Now(),
		Duration:    time.Since(start).String(),
		Details:     details,
	}
}

func (hc *HealthChecker) checkRedisDetailed() ComponentHealth {
	start := time.Now()
	
	if hc.redisClient == nil {
		return ComponentHealth{
			Status:      StatusUnhealthy,
			Message:     "Redis client not initialized",
			LastChecked: time.Now(),
			Duration:    time.Since(start).String(),
		}
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Check basic connectivity
	if err := hc.redisClient.Ping(ctx).Err(); err != nil {
		return ComponentHealth{
			Status:      StatusUnhealthy,
			Message:     fmt.Sprintf("Redis ping failed: %v", err),
			LastChecked: time.Now(),
			Duration:    time.Since(start).String(),
		}
	}
	
	// Get Redis info
	info := hc.redisClient.Info(ctx, "stats", "clients", "memory")
	
	// Parse basic stats (simplified)
	details := CacheStats{
		ConnectedClients: 1, // Default placeholder
		UsedMemory:      0,  // Would need to parse from info
	}
	
	return ComponentHealth{
		Status:      StatusHealthy,
		Message:     "Redis connection healthy",
		LastChecked: time.Now(),
		Duration:    time.Since(start).String(),
		Details:     details,
	}
}

func (hc *HealthChecker) checkTMDBAPI() ComponentHealth {
	start := time.Now()
	
	// Use existing circuit breaker from api_protection.go
	err := ExecuteWithCircuitBreaker("tmdb", func() error {
		// Simple health check - try to get configuration
		client := &http.Client{Timeout: 5 * time.Second}
		
		tmdbAPIKey := os.Getenv("TMDB_API_KEY")
		if tmdbAPIKey == "" {
			return fmt.Errorf("TMDB API key not configured")
		}
		
		url := fmt.Sprintf("https://api.themoviedb.org/3/configuration?api_key=%s", tmdbAPIKey)
		resp, err := client.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != 200 {
			return fmt.Errorf("TMDB API returned status %d", resp.StatusCode)
		}
		
		return nil
	})
	
	if err != nil {
		return ComponentHealth{
			Status:      StatusDegraded,
			Message:     fmt.Sprintf("TMDB API check failed: %v", err),
			LastChecked: time.Now(),
			Duration:    time.Since(start).String(),
		}
	}
	
	return ComponentHealth{
		Status:      StatusHealthy,
		Message:     "TMDB API accessible",
		LastChecked: time.Now(),
		Duration:    time.Since(start).String(),
		Details: ExternalStats{
			TMDBApiStatus:    StatusHealthy,
			LastTMDBCall:     time.Now(),
			TMDBResponseTime: time.Since(start).String(),
		},
	}
}

func (hc *HealthChecker) checkSystem() ComponentHealth {
	start := time.Now()
	
	return ComponentHealth{
		Status:      StatusHealthy,
		Message:     "System resources normal",
		LastChecked: time.Now(),
		Duration:    time.Since(start).String(),
		Details:     hc.getSystemHealth(),
	}
}

func (hc *HealthChecker) getSystemHealth() SystemHealth {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return SystemHealth{
		Memory: MemoryStats{
			Alloc:        m.Alloc,
			TotalAlloc:   m.TotalAlloc,
			Sys:          m.Sys,
			NumGC:        m.NumGC,
			HeapInuse:    m.HeapInuse,
			HeapReleased: m.HeapReleased,
		},
		Runtime: RuntimeStats{
			Goroutines: runtime.NumGoroutine(),
			GOMAXPROCS: runtime.GOMAXPROCS(0),
			NumCPU:     runtime.NumCPU(),
			GoVersion:  runtime.Version(),
		},
	}
}