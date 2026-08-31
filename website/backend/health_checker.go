package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"
)

type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
)

type ComponentHealth struct {
	Status      HealthStatus `json:"status"`
	Message     string       `json:"message,omitempty"`
	LastChecked time.Time    `json:"last_checked"`
	Duration    string       `json:"duration,omitempty"`
	Details     interface{}  `json:"details,omitempty"`
}

type HealthResponse struct {
	Status     HealthStatus               `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Version    string                     `json:"version"`
	Uptime     string                     `json:"uptime"`
	Components map[string]ComponentHealth `json:"components"`
	System     SystemHealth               `json:"system"`
}

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
	OpenConnections int `json:"open_connections"`
	InUse          int `json:"in_use"`
	IdleConnections int `json:"idle_connections"`
	MaxOpen        int `json:"max_open"`
	MaxIdle        int `json:"max_idle"`
}

type CacheStatsHealth struct {
	Enabled         bool `json:"enabled"`
	RedisConnected  bool `json:"redis_connected"`
	MemoryCacheSize int  `json:"memory_cache_size"`
}

type ExternalStats struct {
	TMDBApiStatus    HealthStatus `json:"tmdb_api_status"`
	LastTMDBCall     time.Time    `json:"last_tmdb_call"`
	TMDBResponseTime string       `json:"tmdb_response_time,omitempty"`
}

type HealthChecker struct {
	db          DatabaseAdapter
	redisClient *RedisCacheManager
	startTime   time.Time
	version     string
	mu          sync.RWMutex
	lastChecks  map[string]ComponentHealth
}

func NewHealthChecker(db DatabaseAdapter, redisClient *RedisCacheManager, version string) *HealthChecker {
	hc := &HealthChecker{
		db:          db,
		redisClient: redisClient,
		startTime:   time.Now(),
		version:     version,
		lastChecks:  make(map[string]ComponentHealth),
	}

	return hc
}

func (hc *HealthChecker) RegisterHealthRoutes(router *http.ServeMux) {
	router.HandleFunc("/health", hc.HealthCheckHandler)
	router.HandleFunc("/health/live", hc.LivenessHandler)
	router.HandleFunc("/health/ready", hc.ReadinessHandler)
	router.HandleFunc("/health/detailed", hc.DetailedHealthHandler)
}

func (hc *HealthChecker) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	response := HealthResponse{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
		Version:   hc.version,
		Uptime:    time.Since(hc.startTime).String(),
		Components: make(map[string]ComponentHealth),
	}

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
		w.WriteHeader(http.StatusOK) 
	case StatusUnhealthy:
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}

func (hc *HealthChecker) LivenessHandler(w http.ResponseWriter, r *http.Request) {

	response := map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now(),
		"uptime":    time.Since(hc.startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (hc *HealthChecker) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	ready := true
	issues := []string{}

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

func (hc *HealthChecker) DetailedHealthHandler(w http.ResponseWriter, r *http.Request) {
	_ = time.Now() 

	response := HealthResponse{
		Timestamp: time.Now(),
		Version:   hc.version,
		Uptime:    time.Since(hc.startTime).String(),
		Components: make(map[string]ComponentHealth),
	}

	var wg sync.WaitGroup
	healthChan := make(chan map[string]ComponentHealth, 10)

	wg.Add(1)
	go func() {
		defer wg.Done()
		healthChan <- map[string]ComponentHealth{
			"database": hc.checkDatabaseDetailed(),
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		healthChan <- map[string]ComponentHealth{
			"cache": hc.checkRedisDetailed(),
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		healthChan <- map[string]ComponentHealth{
			"tmdb_api": hc.checkTMDBAPI(),
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		healthChan <- map[string]ComponentHealth{
			"system": hc.checkSystem(),
		}
	}()

	go func() {
		wg.Wait()
		close(healthChan)
	}()

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

	if unhealthyCount > 0 {
		response.Status = StatusUnhealthy
	} else if degradedCount > 0 {
		response.Status = StatusDegraded
	} else {
		response.Status = StatusHealthy
	}

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

func (hc *HealthChecker) checkDatabase() HealthStatus {
	if hc.db == nil {
		return StatusUnhealthy
	}

	if err := hc.db.Ping(); err != nil {
		return StatusUnhealthy
	}
	return StatusHealthy
}

func (hc *HealthChecker) checkRedis() HealthStatus {
	if hc.redisClient == nil {
		return StatusUnhealthy
	}

	stats := hc.redisClient.GetStats()
	if !stats.RedisConnected {
		return StatusUnhealthy
	}

	return StatusHealthy
}

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

	if err := hc.db.Ping(); err != nil {
		return ComponentHealth{
			Status:      StatusUnhealthy,
			Message:     fmt.Sprintf("Database ping failed: %v", err),
			LastChecked: time.Now(),
			Duration:    time.Since(start).String(),
		}
	}

	stats := hc.db.Stats()
	var details DatabaseStats
	if stats != nil {
		details = DatabaseStats{
			OpenConnections: stats.OpenConnections,
			InUse:          stats.InUse,
			IdleConnections: stats.IdleConnections,
			MaxOpen:        stats.MaxOpen,
			MaxIdle:        stats.MaxIdle,
		}
	}

	status := StatusHealthy
	message := "Database connection healthy"

	if stats != nil && stats.OpenConnections > 80 { 
		status = StatusDegraded
		message = "High database connection usage"
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

	cacheStats := hc.redisClient.GetStats()
	if !cacheStats.RedisConnected {
		return ComponentHealth{
			Status:      StatusUnhealthy,
			Message:     "Redis ping failed or disconnected",
			LastChecked: time.Now(),
			Duration:    time.Since(start).String(),
			Details:     cacheStats,
		}
	}

	return ComponentHealth{
		Status:      StatusHealthy,
		Message:     "Redis connection healthy",
		LastChecked: time.Now(),
		Duration:    time.Since(start).String(),
		Details:     cacheStats,
	}
}

func (hc *HealthChecker) checkTMDBAPI() ComponentHealth {
	start := time.Now()

	var checkErr error
	func() {

		client := &http.Client{Timeout: 5 * time.Second}

		tmdbAPIKey := os.Getenv("TMDB_API_KEY")
		if tmdbAPIKey == "" {
			checkErr = fmt.Errorf("TMDB API key not configured")
			return
		}
		url := fmt.Sprintf("https://api.themoviedb.org/3/configuration?api_key=%s", tmdbAPIKey)
		resp, err := client.Get(url)
		if err != nil {
			checkErr = err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			checkErr = fmt.Errorf("TMDB API returned status %d", resp.StatusCode)
		}
	}()

	if checkErr != nil {
		return ComponentHealth{
			Status:      StatusDegraded,
			Message:     fmt.Sprintf("TMDB API check failed: %v", checkErr),
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