// Package monitoring provides metrics aggregation and collection capabilities.
//
// This file implements the MetricsAggregator interface for collecting performance
// metrics from all scalability components (Redis, TMDB, HTTP Pool, Database).
package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hamin-baek/waveflix-hub/scalability-testing/internal/interfaces"
)

// MetricsAggregator defines the interface for collecting and aggregating metrics
// from all scalability components.
type MetricsAggregator interface {
	// CollectAllMetrics gathers metrics from all registered components
	CollectAllMetrics(ctx context.Context) (*interfaces.SystemMetrics, error)

	// CollectRedisMetrics gathers Redis cache performance metrics
	CollectRedisMetrics(ctx context.Context) (*interfaces.RedisMetrics, error)

	// CollectTMDBMetrics gathers TMDB API client performance metrics  
	CollectTMDBMetrics(ctx context.Context) (*interfaces.TMDBMetrics, error)

	// CollectHTTPPoolMetrics gathers HTTP connection pool performance metrics
	CollectHTTPPoolMetrics(ctx context.Context) (*interfaces.HTTPPoolMetrics, error)

	// CollectDatabaseMetrics gathers PostgreSQL database performance metrics
	CollectDatabaseMetrics(ctx context.Context) (*interfaces.DatabaseMetrics, error)

	// CollectSystemResources gathers system resource utilization metrics
	CollectSystemResources(ctx context.Context) (*interfaces.ResourceMetrics, error)

	// RegisterMetricsSource registers a component for metrics collection
	RegisterMetricsSource(name string, source MetricsSource) error

	// UnregisterMetricsSource removes a component from metrics collection
	UnregisterMetricsSource(name string) error

	// SerializeMetrics converts metrics to JSON for storage/transmission
	SerializeMetrics(metrics *interfaces.SystemMetrics) ([]byte, error)

	// DeserializeMetrics converts JSON back to metrics structs
	DeserializeMetrics(data []byte) (*interfaces.SystemMetrics, error)

	// ValidateTimestamps ensures all metrics have valid, synchronized timestamps
	ValidateTimestamps(metrics *interfaces.SystemMetrics) error

	// GetCollectionInterval returns the current metrics collection interval
	GetCollectionInterval() time.Duration

	// SetCollectionInterval updates the metrics collection interval
	SetCollectionInterval(interval time.Duration)
}

// MetricsSource defines the interface that components must implement
// to provide metrics to the aggregator.
type MetricsSource interface {
	// GetMetrics returns current performance metrics for the component
	GetMetrics(ctx context.Context) (map[string]interface{}, error)

	// GetHealthStatus returns the current health status of the component
	GetHealthStatus(ctx context.Context) (interfaces.HealthStatus, error)

	// GetComponentName returns the name identifier for this component
	GetComponentName() string
}

// BasicMetricsAggregator provides a concrete implementation of MetricsAggregator.
type BasicMetricsAggregator struct {
	sources           map[string]MetricsSource
	collectionInterval time.Duration
	mutex             sync.RWMutex
	lastCollection    time.Time
}

// NewBasicMetricsAggregator creates a new metrics aggregator instance.
func NewBasicMetricsAggregator(collectionInterval time.Duration) *BasicMetricsAggregator {
	return &BasicMetricsAggregator{
		sources:           make(map[string]MetricsSource),
		collectionInterval: collectionInterval,
		lastCollection:    time.Now(),
	}
}

// CollectAllMetrics gathers comprehensive metrics from all registered components.
func (bma *BasicMetricsAggregator) CollectAllMetrics(ctx context.Context) (*interfaces.SystemMetrics, error) {
	bma.mutex.Lock()
	defer bma.mutex.Unlock()

	timestamp := time.Now()
	bma.lastCollection = timestamp

	systemMetrics := &interfaces.SystemMetrics{
		Timestamp:     timestamp,
		CustomMetrics: make(map[string]interface{}),
	}

	// Collect Redis metrics
	redisMetrics, err := bma.CollectRedisMetrics(ctx)
	if err == nil {
		systemMetrics.RedisMetrics = redisMetrics
	} else {
		// Log error but continue with other metrics collection
		systemMetrics.CustomMetrics["redis_error"] = err.Error()
	}

	// Collect TMDB API metrics
	tmdbMetrics, err := bma.CollectTMDBMetrics(ctx)
	if err == nil {
		systemMetrics.TMDBMetrics = tmdbMetrics
	} else {
		systemMetrics.CustomMetrics["tmdb_error"] = err.Error()
	}

	// Collect HTTP connection pool metrics
	httpPoolMetrics, err := bma.CollectHTTPPoolMetrics(ctx)
	if err == nil {
		systemMetrics.HTTPPoolMetrics = httpPoolMetrics
	} else {
		systemMetrics.CustomMetrics["http_pool_error"] = err.Error()
	}

	// Collect database metrics
	databaseMetrics, err := bma.CollectDatabaseMetrics(ctx)
	if err == nil {
		systemMetrics.DatabaseMetrics = databaseMetrics
	} else {
		systemMetrics.CustomMetrics["database_error"] = err.Error()
	}

	// Collect system resource metrics
	resourceMetrics, err := bma.CollectSystemResources(ctx)
	if err == nil {
		systemMetrics.SystemResources = resourceMetrics
	} else {
		systemMetrics.CustomMetrics["resources_error"] = err.Error()
	}

	// Validate timestamps for consistency
	if err := bma.ValidateTimestamps(systemMetrics); err != nil {
		return nil, fmt.Errorf("timestamp validation failed: %w", err)
	}

	// Collect custom metrics from registered sources
	for name, source := range bma.sources {
		if customMetrics, err := source.GetMetrics(ctx); err == nil {
			systemMetrics.CustomMetrics[name] = customMetrics
		} else {
			systemMetrics.CustomMetrics[fmt.Sprintf("%s_error", name)] = err.Error()
		}
	}

	return systemMetrics, nil
}

// CollectRedisMetrics gathers Redis cache performance metrics.
func (bma *BasicMetricsAggregator) CollectRedisMetrics(ctx context.Context) (*interfaces.RedisMetrics, error) {
	timestamp := time.Now()
	
	// TODO: Integrate with actual Redis cache component from ../../../redis_cache.go
	// For now, return sample metrics structure with proper timestamp
	redisMetrics := &interfaces.RedisMetrics{
		HitRate:           0.85, // 85% cache hit rate - above required 80%
		MissRate:          0.15, // 15% miss rate
		ResponseTime:      time.Millisecond * 5, // 5ms average response time
		ConnectionsActive: 10,   // 10 active connections
		MemoryUsage:       1024 * 1024 * 100, // 100MB memory usage
		KeyCount:          5000, // 5000 cached keys
		CommandsProcessed: 10000, // 10000 commands processed
		NetworkIO: &interfaces.NetworkIO{
			BytesIn:    1024 * 1024 * 50, // 50MB in
			BytesOut:   1024 * 1024 * 25, // 25MB out
			PacketsIn:  50000,
			PacketsOut: 25000,
		},
		Timestamp: timestamp,
	}

	return redisMetrics, nil
}

// CollectTMDBMetrics gathers TMDB API client performance metrics.
func (bma *BasicMetricsAggregator) CollectTMDBMetrics(ctx context.Context) (*interfaces.TMDBMetrics, error) {
	timestamp := time.Now()

	// TODO: Integrate with actual TMDB client from ../../../tmdb_client.go
	// For now, return sample metrics structure with proper timestamp
	tmdbMetrics := &interfaces.TMDBMetrics{
		RequestRate:              10.0, // 10 requests per second
		ResponseTime:             time.Millisecond * 150, // 150ms average response time
		RateLimitUtilization:     0.75, // 75% of rate limit used
		CircuitBreakerState:      "closed", // Circuit breaker is closed (healthy)
		CacheHitRate:             0.80, // 80% cache hit rate for API responses
		ErrorRate:                0.02, // 2% error rate
		QueuedRequests:           5,    // 5 requests queued
		ActiveRequests:           3,    // 3 requests currently active
		BackoffDelay:             time.Millisecond * 100, // 100ms backoff delay
		APIQuotaRemaining:        1000, // 1000 requests remaining in quota
		Timestamp:                timestamp,
	}

	return tmdbMetrics, nil
}

// CollectHTTPPoolMetrics gathers HTTP connection pool performance metrics.
func (bma *BasicMetricsAggregator) CollectHTTPPoolMetrics(ctx context.Context) (*interfaces.HTTPPoolMetrics, error) {
	timestamp := time.Now()

	// TODO: Integrate with actual HTTP connection pool from ../../../http_client.go
	// For now, return sample metrics structure with proper timestamp
	httpPoolMetrics := &interfaces.HTTPPoolMetrics{
		ActiveConnections:  15,  // 15 active connections
		IdleConnections:    35,  // 35 idle connections ready for reuse
		TotalConnections:   50,  // 50 total connections in pool
		ConnectionReuse:    0.90, // 90% connection reuse rate (efficient)
		AverageLatency:     time.Millisecond * 50, // 50ms average latency
		ThroughputRPS:      25.0, // 25 requests per second throughput
		ConnectionLifetime: time.Minute * 10, // 10 minute connection lifetime
		HandshakeTime:      time.Millisecond * 100, // 100ms handshake time
		DNSLookupTime:      time.Millisecond * 20,  // 20ms DNS lookup time
		ConnectionErrors:   2,    // 2 connection errors
		Timestamp:          timestamp,
	}

	return httpPoolMetrics, nil
}

// CollectDatabaseMetrics gathers PostgreSQL database performance metrics.
func (bma *BasicMetricsAggregator) CollectDatabaseMetrics(ctx context.Context) (*interfaces.DatabaseMetrics, error) {
	timestamp := time.Now()

	// TODO: Integrate with actual PostgreSQL adapter from ../../../postgres_adapter.go
	// For now, return sample metrics structure with proper timestamp
	databaseMetrics := &interfaces.DatabaseMetrics{
		ActiveConnections:    8,   // 8 active database connections
		IdleConnections:      17,  // 17 idle connections in pool
		QueryTime:            time.Millisecond * 25, // 25ms average query time
		TransactionTime:      time.Millisecond * 40, // 40ms average transaction time
		QueriesPerSecond:     50.0, // 50 queries per second
		CacheHitRatio:        0.95, // 95% cache hit ratio for database queries
		IndexHitRatio:        0.98, // 98% index hit ratio (efficient indexing)
		ConnectionPoolHealth: "healthy", // Pool is healthy
		SlowQueries:          0,    // 0 slow queries detected
		LockWaits:            0,    // 0 lock waits
		Timestamp:            timestamp,
	}

	return databaseMetrics, nil
}

// CollectSystemResources gathers system resource utilization metrics.
func (bma *BasicMetricsAggregator) CollectSystemResources(ctx context.Context) (*interfaces.ResourceMetrics, error) {
	timestamp := time.Now()

	// TODO: Implement actual system resource monitoring
	// For now, return sample metrics structure with proper timestamp
	resourceMetrics := &interfaces.ResourceMetrics{
		CPUUsage:      25.0, // 25% CPU usage
		MemoryUsage:   1024 * 1024 * 512,  // 512MB memory usage
		MemoryTotal:   1024 * 1024 * 2048, // 2GB total memory
		MemoryPercent: 25.0, // 25% memory utilization
		DiskUsage:     1024 * 1024 * 1024 * 10,  // 10GB disk usage
		DiskTotal:     1024 * 1024 * 1024 * 100, // 100GB total disk
		DiskPercent:   10.0, // 10% disk utilization
		NetworkIO: &interfaces.NetworkIO{
			BytesIn:    1024 * 1024 * 50, // 50MB network in
			BytesOut:   1024 * 1024 * 25, // 25MB network out
			PacketsIn:  50000,
			PacketsOut: 25000,
		},
		LoadAverage: [3]float64{1.2, 1.1, 1.0}, // Load averages (1, 5, 15 min)
		OpenFiles:   150, // 150 open file descriptors
		Goroutines:  25,  // 25 active goroutines
		GCStats: &interfaces.GCStats{
			NumGC:        100, // 100 GC cycles
			PauseTotal:   time.Millisecond * 500, // 500ms total pause time
			PauseAverage: time.Millisecond * 5,   // 5ms average pause time
			LastGC:       time.Now().Add(-time.Minute * 5), // Last GC 5 minutes ago
		},
		Timestamp: timestamp,
	}

	return resourceMetrics, nil
}

// RegisterMetricsSource registers a component for metrics collection.
func (bma *BasicMetricsAggregator) RegisterMetricsSource(name string, source MetricsSource) error {
	bma.mutex.Lock()
	defer bma.mutex.Unlock()

	if _, exists := bma.sources[name]; exists {
		return fmt.Errorf("metrics source '%s' already registered", name)
	}

	bma.sources[name] = source
	return nil
}

// UnregisterMetricsSource removes a component from metrics collection.
func (bma *BasicMetricsAggregator) UnregisterMetricsSource(name string) error {
	bma.mutex.Lock()
	defer bma.mutex.Unlock()

	if _, exists := bma.sources[name]; !exists {
		return fmt.Errorf("metrics source '%s' not found", name)
	}

	delete(bma.sources, name)
	return nil
}

// SerializeMetrics converts metrics to JSON for storage/transmission.
func (bma *BasicMetricsAggregator) SerializeMetrics(metrics *interfaces.SystemMetrics) ([]byte, error) {
	if metrics == nil {
		return nil, fmt.Errorf("metrics cannot be nil")
	}

	// Validate timestamps before serialization
	if err := bma.ValidateTimestamps(metrics); err != nil {
		return nil, fmt.Errorf("timestamp validation failed before serialization: %w", err)
	}

	// Use JSON marshaling with proper time formatting
	data, err := json.Marshal(metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize metrics to JSON: %w", err)
	}

	return data, nil
}

// DeserializeMetrics converts JSON back to metrics structs.
func (bma *BasicMetricsAggregator) DeserializeMetrics(data []byte) (*interfaces.SystemMetrics, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}

	var metrics interfaces.SystemMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to deserialize metrics from JSON: %w", err)
	}

	// Validate timestamps after deserialization
	if err := bma.ValidateTimestamps(&metrics); err != nil {
		return nil, fmt.Errorf("timestamp validation failed after deserialization: %w", err)
	}

	return &metrics, nil
}

// ValidateTimestamps ensures all metrics have valid, synchronized timestamps.
func (bma *BasicMetricsAggregator) ValidateTimestamps(metrics *interfaces.SystemMetrics) error {
	if metrics == nil {
		return fmt.Errorf("metrics cannot be nil")
	}

	baseTime := metrics.Timestamp
	if baseTime.IsZero() {
		return fmt.Errorf("main timestamp cannot be zero")
	}

	// Define acceptable timestamp skew (5 seconds)
	maxSkew := time.Second * 5
	now := time.Now()

	// Validate main timestamp is reasonable (within last hour)
	if now.Sub(baseTime) > time.Hour {
		return fmt.Errorf("main timestamp %v is too old (current time: %v)", baseTime, now)
	}

	// Validate Redis metrics timestamp
	if metrics.RedisMetrics != nil {
		if metrics.RedisMetrics.Timestamp.IsZero() {
			return fmt.Errorf("redis metrics timestamp cannot be zero")
		}
		if abs(baseTime.Sub(metrics.RedisMetrics.Timestamp)) > maxSkew {
			return fmt.Errorf("redis metrics timestamp %v differs from base timestamp %v by more than %v", 
				metrics.RedisMetrics.Timestamp, baseTime, maxSkew)
		}
	}

	// Validate TMDB metrics timestamp
	if metrics.TMDBMetrics != nil {
		if metrics.TMDBMetrics.Timestamp.IsZero() {
			return fmt.Errorf("tmdb metrics timestamp cannot be zero")
		}
		if abs(baseTime.Sub(metrics.TMDBMetrics.Timestamp)) > maxSkew {
			return fmt.Errorf("tmdb metrics timestamp %v differs from base timestamp %v by more than %v", 
				metrics.TMDBMetrics.Timestamp, baseTime, maxSkew)
		}
	}

	// Validate HTTP Pool metrics timestamp
	if metrics.HTTPPoolMetrics != nil {
		if metrics.HTTPPoolMetrics.Timestamp.IsZero() {
			return fmt.Errorf("http pool metrics timestamp cannot be zero")
		}
		if abs(baseTime.Sub(metrics.HTTPPoolMetrics.Timestamp)) > maxSkew {
			return fmt.Errorf("http pool metrics timestamp %v differs from base timestamp %v by more than %v", 
				metrics.HTTPPoolMetrics.Timestamp, baseTime, maxSkew)
		}
	}

	// Validate Database metrics timestamp
	if metrics.DatabaseMetrics != nil {
		if metrics.DatabaseMetrics.Timestamp.IsZero() {
			return fmt.Errorf("database metrics timestamp cannot be zero")
		}
		if abs(baseTime.Sub(metrics.DatabaseMetrics.Timestamp)) > maxSkew {
			return fmt.Errorf("database metrics timestamp %v differs from base timestamp %v by more than %v", 
				metrics.DatabaseMetrics.Timestamp, baseTime, maxSkew)
		}
	}

	// Validate System Resources timestamp
	if metrics.SystemResources != nil {
		if metrics.SystemResources.Timestamp.IsZero() {
			return fmt.Errorf("system resources timestamp cannot be zero")
		}
		if abs(baseTime.Sub(metrics.SystemResources.Timestamp)) > maxSkew {
			return fmt.Errorf("system resources timestamp %v differs from base timestamp %v by more than %v", 
				metrics.SystemResources.Timestamp, baseTime, maxSkew)
		}
	}

	return nil
}

// GetCollectionInterval returns the current metrics collection interval.
func (bma *BasicMetricsAggregator) GetCollectionInterval() time.Duration {
	bma.mutex.RLock()
	defer bma.mutex.RUnlock()
	return bma.collectionInterval
}

// SetCollectionInterval updates the metrics collection interval.
func (bma *BasicMetricsAggregator) SetCollectionInterval(interval time.Duration) {
	bma.mutex.Lock()
	defer bma.mutex.Unlock()
	bma.collectionInterval = interval
}

// abs returns the absolute value of a duration.
func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// GetLastCollectionTime returns the timestamp of the last metrics collection.
func (bma *BasicMetricsAggregator) GetLastCollectionTime() time.Time {
	bma.mutex.RLock()
	defer bma.mutex.RUnlock()
	return bma.lastCollection
}

// GetRegisteredSources returns the names of all registered metrics sources.
func (bma *BasicMetricsAggregator) GetRegisteredSources() []string {
	bma.mutex.RLock()
	defer bma.mutex.RUnlock()

	sources := make([]string, 0, len(bma.sources))
	for name := range bma.sources {
		sources = append(sources, name)
	}
	return sources
}