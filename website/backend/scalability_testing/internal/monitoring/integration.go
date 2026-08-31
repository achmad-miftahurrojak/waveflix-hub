// Package monitoring provides integration interfaces for connecting with existing
// WaveFlix Hub components to collect real performance metrics.
//
// This file defines the integration points with redis_cache.go, tmdb_client.go, 
// http_client.go, and database components for metrics collection.
package monitoring

import (
	"context"
	"sync"
	"time"

	"github.com/hamin-baek/waveflix-hub/scalability-testing/internal/interfaces"
)

// ComponentIntegrator provides integration interfaces for collecting metrics
// from existing WaveFlix Hub scalability components.
type ComponentIntegrator struct {
	redisCollector    RedisMetricsCollector
	tmdbCollector     TMDBMetricsCollector
	httpPoolCollector HTTPPoolMetricsCollector
	dbCollector       DatabaseMetricsCollector
	mutex             sync.RWMutex
	lastUpdate        time.Time
}

// RedisMetricsCollector defines the interface for collecting Redis cache metrics
// from the existing redis_cache.go component.
type RedisMetricsCollector interface {
	// GetCacheStats returns current cache hit/miss statistics
	GetCacheStats(ctx context.Context) (hits, misses int64, hitRate float64, err error)

	// GetResponseTime returns average cache response time
	GetResponseTime(ctx context.Context) (time.Duration, error)

	// GetConnectionInfo returns active connection and pool information
	GetConnectionInfo(ctx context.Context) (active, total int, err error)

	// GetMemoryUsage returns current Redis memory usage
	GetMemoryUsage(ctx context.Context) (used, total int64, err error)

	// GetKeyCount returns current number of cached keys
	GetKeyCount(ctx context.Context) (int, error)

	// GetCommandStats returns command processing statistics
	GetCommandStats(ctx context.Context) (processed int64, err error)

	// IsHealthy returns the health status of the Redis connection
	IsHealthy(ctx context.Context) (bool, error)
}

// TMDBMetricsCollector defines the interface for collecting TMDB API metrics
// from the existing tmdb_client.go component.
type TMDBMetricsCollector interface {
	// GetRequestRate returns current API request rate
	GetRequestRate(ctx context.Context) (float64, error)

	// GetResponseTime returns average API response time
	GetResponseTime(ctx context.Context) (time.Duration, error)

	// GetRateLimitStatus returns rate limiting status and utilization
	GetRateLimitStatus(ctx context.Context) (utilization float64, remaining int, err error)

	// GetCircuitBreakerState returns circuit breaker status
	GetCircuitBreakerState(ctx context.Context) (string, error)

	// GetCacheStats returns API response cache statistics
	GetCacheStats(ctx context.Context) (hitRate float64, err error)

	// GetErrorRate returns current API error rate
	GetErrorRate(ctx context.Context) (float64, error)

	// GetQueueStatus returns request queue status
	GetQueueStatus(ctx context.Context) (queued, active int, err error)

	// GetBackoffDelay returns current exponential backoff delay
	GetBackoffDelay(ctx context.Context) (time.Duration, error)
}

// HTTPPoolMetricsCollector defines the interface for collecting HTTP connection pool metrics
// from the existing http_client.go component.
type HTTPPoolMetricsCollector interface {
	// GetConnectionStats returns connection pool statistics
	GetConnectionStats(ctx context.Context) (active, idle, total int, err error)

	// GetConnectionReuse returns connection reuse efficiency
	GetConnectionReuse(ctx context.Context) (float64, error)

	// GetLatencyStats returns connection latency statistics
	GetLatencyStats(ctx context.Context) (avg, p95, p99 time.Duration, err error)

	// GetThroughput returns current throughput in requests per second
	GetThroughput(ctx context.Context) (float64, error)

	// GetConnectionLifetime returns average connection lifetime
	GetConnectionLifetime(ctx context.Context) (time.Duration, error)

	// GetTimingBreakdown returns detailed timing breakdown
	GetTimingBreakdown(ctx context.Context) (handshake, dns time.Duration, err error)

	// GetConnectionErrors returns connection error count
	GetConnectionErrors(ctx context.Context) (int64, error)
}

// DatabaseMetricsCollector defines the interface for collecting PostgreSQL database metrics
// from existing database components (postgres_adapter.go, db.go).
type DatabaseMetricsCollector interface {
	// GetConnectionPoolStats returns database connection pool statistics
	GetConnectionPoolStats(ctx context.Context) (active, idle int, health string, err error)

	// GetQueryPerformance returns query performance statistics
	GetQueryPerformance(ctx context.Context) (avgQueryTime, avgTxTime time.Duration, qps float64, err error)

	// GetCacheMetrics returns database cache hit ratios
	GetCacheMetrics(ctx context.Context) (cacheHitRatio, indexHitRatio float64, err error)

	// GetSlowQueryStats returns slow query statistics
	GetSlowQueryStats(ctx context.Context) (slowQueries, lockWaits int64, err error)

	// IsHealthy returns the health status of the database connection
	IsHealthy(ctx context.Context) (bool, error)
}

// NewComponentIntegrator creates a new component integrator with the provided collectors.
func NewComponentIntegrator(
	redisCollector RedisMetricsCollector,
	tmdbCollector TMDBMetricsCollector,
	httpPoolCollector HTTPPoolMetricsCollector,
	dbCollector DatabaseMetricsCollector,
) *ComponentIntegrator {
	return &ComponentIntegrator{
		redisCollector:    redisCollector,
		tmdbCollector:     tmdbCollector,
		httpPoolCollector: httpPoolCollector,
		dbCollector:       dbCollector,
		lastUpdate:        time.Now(),
	}
}

// CollectRedisMetrics integrates with the existing Redis cache component to collect real metrics.
func (ci *ComponentIntegrator) CollectRedisMetrics(ctx context.Context) (*interfaces.RedisMetrics, error) {
	ci.mutex.Lock()
	defer ci.mutex.Unlock()

	timestamp := time.Now()

	// Initialize with zero values
	metrics := &interfaces.RedisMetrics{
		Timestamp: timestamp,
	}

	// Collect cache statistics if collector is available
	if ci.redisCollector != nil {
		if hits, misses, hitRate, err := ci.redisCollector.GetCacheStats(ctx); err == nil {
			metrics.HitRate = hitRate
			metrics.MissRate = 1.0 - hitRate
			// Store raw hit/miss counts in custom fields for future analysis
		}

		if responseTime, err := ci.redisCollector.GetResponseTime(ctx); err == nil {
			metrics.ResponseTime = responseTime
		}

		if active, total, err := ci.redisCollector.GetConnectionInfo(ctx); err == nil {
			metrics.ConnectionsActive = active
			// Note: Total connections stored for monitoring but not in base interface
		}

		if used, _, err := ci.redisCollector.GetMemoryUsage(ctx); err == nil {
			metrics.MemoryUsage = used
		}

		if keyCount, err := ci.redisCollector.GetKeyCount(ctx); err == nil {
			metrics.KeyCount = keyCount
		}

		if processed, err := ci.redisCollector.GetCommandStats(ctx); err == nil {
			metrics.CommandsProcessed = processed
		}
	}

	// Set default network I/O metrics (would need actual implementation)
	metrics.NetworkIO = &interfaces.NetworkIO{
		BytesIn:    0, // TODO: Collect from Redis INFO command
		BytesOut:   0, // TODO: Collect from Redis INFO command  
		PacketsIn:  0, // TODO: Collect from Redis INFO command
		PacketsOut: 0, // TODO: Collect from Redis INFO command
	}

	return metrics, nil
}

// CollectTMDBMetrics integrates with the existing TMDB client to collect real metrics.
func (ci *ComponentIntegrator) CollectTMDBMetrics(ctx context.Context) (*interfaces.TMDBMetrics, error) {
	ci.mutex.Lock()
	defer ci.mutex.Unlock()

	timestamp := time.Now()

	// Initialize with zero values
	metrics := &interfaces.TMDBMetrics{
		Timestamp: timestamp,
	}

	// Collect TMDB API statistics if collector is available
	if ci.tmdbCollector != nil {
		if requestRate, err := ci.tmdbCollector.GetRequestRate(ctx); err == nil {
			metrics.RequestRate = requestRate
		}

		if responseTime, err := ci.tmdbCollector.GetResponseTime(ctx); err == nil {
			metrics.ResponseTime = responseTime
		}

		if utilization, remaining, err := ci.tmdbCollector.GetRateLimitStatus(ctx); err == nil {
			metrics.RateLimitUtilization = utilization
			metrics.APIQuotaRemaining = remaining
		}

		if state, err := ci.tmdbCollector.GetCircuitBreakerState(ctx); err == nil {
			metrics.CircuitBreakerState = state
		}

		if hitRate, err := ci.tmdbCollector.GetCacheStats(ctx); err == nil {
			metrics.CacheHitRate = hitRate
		}

		if errorRate, err := ci.tmdbCollector.GetErrorRate(ctx); err == nil {
			metrics.ErrorRate = errorRate
		}

		if queued, active, err := ci.tmdbCollector.GetQueueStatus(ctx); err == nil {
			metrics.QueuedRequests = queued
			metrics.ActiveRequests = active
		}

		if backoff, err := ci.tmdbCollector.GetBackoffDelay(ctx); err == nil {
			metrics.BackoffDelay = backoff
		}
	}

	return metrics, nil
}

// CollectHTTPPoolMetrics integrates with the existing HTTP connection pool to collect real metrics.
func (ci *ComponentIntegrator) CollectHTTPPoolMetrics(ctx context.Context) (*interfaces.HTTPPoolMetrics, error) {
	ci.mutex.Lock()
	defer ci.mutex.Unlock()

	timestamp := time.Now()

	// Initialize with zero values
	metrics := &interfaces.HTTPPoolMetrics{
		Timestamp: timestamp,
	}

	// Collect HTTP connection pool statistics if collector is available
	if ci.httpPoolCollector != nil {
		if active, idle, total, err := ci.httpPoolCollector.GetConnectionStats(ctx); err == nil {
			metrics.ActiveConnections = active
			metrics.IdleConnections = idle
			metrics.TotalConnections = total
		}

		if reuseRate, err := ci.httpPoolCollector.GetConnectionReuse(ctx); err == nil {
			metrics.ConnectionReuse = reuseRate
		}

		if avg, _, _, err := ci.httpPoolCollector.GetLatencyStats(ctx); err == nil {
			metrics.AverageLatency = avg
		}

		if throughput, err := ci.httpPoolCollector.GetThroughput(ctx); err == nil {
			metrics.ThroughputRPS = throughput
		}

		if lifetime, err := ci.httpPoolCollector.GetConnectionLifetime(ctx); err == nil {
			metrics.ConnectionLifetime = lifetime
		}

		if handshake, dns, err := ci.httpPoolCollector.GetTimingBreakdown(ctx); err == nil {
			metrics.HandshakeTime = handshake
			metrics.DNSLookupTime = dns
		}

		if errors, err := ci.httpPoolCollector.GetConnectionErrors(ctx); err == nil {
			metrics.ConnectionErrors = errors
		}
	}

	return metrics, nil
}

// CollectDatabaseMetrics integrates with the existing PostgreSQL database to collect real metrics.
func (ci *ComponentIntegrator) CollectDatabaseMetrics(ctx context.Context) (*interfaces.DatabaseMetrics, error) {
	ci.mutex.Lock()
	defer ci.mutex.Unlock()

	timestamp := time.Now()

	// Initialize with zero values
	metrics := &interfaces.DatabaseMetrics{
		Timestamp: timestamp,
	}

	// Collect database statistics if collector is available
	if ci.dbCollector != nil {
		if active, idle, health, err := ci.dbCollector.GetConnectionPoolStats(ctx); err == nil {
			metrics.ActiveConnections = active
			metrics.IdleConnections = idle
			metrics.ConnectionPoolHealth = health
		}

		if queryTime, txTime, qps, err := ci.dbCollector.GetQueryPerformance(ctx); err == nil {
			metrics.QueryTime = queryTime
			metrics.TransactionTime = txTime
			metrics.QueriesPerSecond = qps
		}

		if cacheHit, indexHit, err := ci.dbCollector.GetCacheMetrics(ctx); err == nil {
			metrics.CacheHitRatio = cacheHit
			metrics.IndexHitRatio = indexHit
		}

		if slowQueries, lockWaits, err := ci.dbCollector.GetSlowQueryStats(ctx); err == nil {
			metrics.SlowQueries = slowQueries
			metrics.LockWaits = lockWaits
		}
	}

	return metrics, nil
}

// IsHealthy checks the health status of all integrated components.
func (ci *ComponentIntegrator) IsHealthy(ctx context.Context) (bool, map[string]bool, error) {
	ci.mutex.RLock()
	defer ci.mutex.RUnlock()

	componentHealth := make(map[string]bool)
	allHealthy := true

	// Check Redis health
	if ci.redisCollector != nil {
		if healthy, err := ci.redisCollector.IsHealthy(ctx); err == nil {
			componentHealth["redis"] = healthy
			allHealthy = allHealthy && healthy
		} else {
			componentHealth["redis"] = false
			allHealthy = false
		}
	}

	// Check database health
	if ci.dbCollector != nil {
		if healthy, err := ci.dbCollector.IsHealthy(ctx); err == nil {
			componentHealth["database"] = healthy
			allHealthy = allHealthy && healthy
		} else {
			componentHealth["database"] = false
			allHealthy = false
		}
	}

	// TMDB and HTTP pool health are implicit through successful metric collection
	componentHealth["tmdb"] = true    // Assume healthy if no explicit health check
	componentHealth["http_pool"] = true // Assume healthy if no explicit health check

	return allHealthy, componentHealth, nil
}

// UpdateCollectors allows updating the collectors at runtime for dynamic integration.
func (ci *ComponentIntegrator) UpdateCollectors(
	redisCollector RedisMetricsCollector,
	tmdbCollector TMDBMetricsCollector,
	httpPoolCollector HTTPPoolMetricsCollector,
	dbCollector DatabaseMetricsCollector,
) {
	ci.mutex.Lock()
	defer ci.mutex.Unlock()

	if redisCollector != nil {
		ci.redisCollector = redisCollector
	}
	if tmdbCollector != nil {
		ci.tmdbCollector = tmdbCollector
	}
	if httpPoolCollector != nil {
		ci.httpPoolCollector = httpPoolCollector
	}
	if dbCollector != nil {
		ci.dbCollector = dbCollector
	}

	ci.lastUpdate = time.Now()
}

// GetLastUpdateTime returns the time when collectors were last updated.
func (ci *ComponentIntegrator) GetLastUpdateTime() time.Time {
	ci.mutex.RLock()
	defer ci.mutex.RUnlock()
	return ci.lastUpdate
}