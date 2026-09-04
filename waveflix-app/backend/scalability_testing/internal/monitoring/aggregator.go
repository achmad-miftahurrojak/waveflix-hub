

package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hamin-baek/waveflix-hub/scalability-testing/internal/interfaces"
)

type MetricsAggregator interface {

	CollectAllMetrics(ctx context.Context) (*interfaces.SystemMetrics, error)

	CollectRedisMetrics(ctx context.Context) (*interfaces.RedisMetrics, error)

	CollectTMDBMetrics(ctx context.Context) (*interfaces.TMDBMetrics, error)

	CollectHTTPPoolMetrics(ctx context.Context) (*interfaces.HTTPPoolMetrics, error)

	CollectDatabaseMetrics(ctx context.Context) (*interfaces.DatabaseMetrics, error)

	CollectSystemResources(ctx context.Context) (*interfaces.ResourceMetrics, error)

	RegisterMetricsSource(name string, source MetricsSource) error

	UnregisterMetricsSource(name string) error

	SerializeMetrics(metrics *interfaces.SystemMetrics) ([]byte, error)

	DeserializeMetrics(data []byte) (*interfaces.SystemMetrics, error)

	ValidateTimestamps(metrics *interfaces.SystemMetrics) error

	GetCollectionInterval() time.Duration

	SetCollectionInterval(interval time.Duration)
}

type MetricsSource interface {

	GetMetrics(ctx context.Context) (map[string]interface{}, error)

	GetHealthStatus(ctx context.Context) (interfaces.HealthStatus, error)

	GetComponentName() string
}

type BasicMetricsAggregator struct {
	sources           map[string]MetricsSource
	collectionInterval time.Duration
	mutex             sync.RWMutex
	lastCollection    time.Time
}

func NewBasicMetricsAggregator(collectionInterval time.Duration) *BasicMetricsAggregator {
	return &BasicMetricsAggregator{
		sources:           make(map[string]MetricsSource),
		collectionInterval: collectionInterval,
		lastCollection:    time.Now(),
	}
}

func (bma *BasicMetricsAggregator) CollectAllMetrics(ctx context.Context) (*interfaces.SystemMetrics, error) {
	bma.mutex.Lock()
	defer bma.mutex.Unlock()

	timestamp := time.Now()
	bma.lastCollection = timestamp

	systemMetrics := &interfaces.SystemMetrics{
		Timestamp:     timestamp,
		CustomMetrics: make(map[string]interface{}),
	}

	redisMetrics, err := bma.CollectRedisMetrics(ctx)
	if err == nil {
		systemMetrics.RedisMetrics = redisMetrics
	} else {

		systemMetrics.CustomMetrics["redis_error"] = err.Error()
	}

	tmdbMetrics, err := bma.CollectTMDBMetrics(ctx)
	if err == nil {
		systemMetrics.TMDBMetrics = tmdbMetrics
	} else {
		systemMetrics.CustomMetrics["tmdb_error"] = err.Error()
	}

	httpPoolMetrics, err := bma.CollectHTTPPoolMetrics(ctx)
	if err == nil {
		systemMetrics.HTTPPoolMetrics = httpPoolMetrics
	} else {
		systemMetrics.CustomMetrics["http_pool_error"] = err.Error()
	}

	databaseMetrics, err := bma.CollectDatabaseMetrics(ctx)
	if err == nil {
		systemMetrics.DatabaseMetrics = databaseMetrics
	} else {
		systemMetrics.CustomMetrics["database_error"] = err.Error()
	}

	resourceMetrics, err := bma.CollectSystemResources(ctx)
	if err == nil {
		systemMetrics.SystemResources = resourceMetrics
	} else {
		systemMetrics.CustomMetrics["resources_error"] = err.Error()
	}

	if err := bma.ValidateTimestamps(systemMetrics); err != nil {
		return nil, fmt.Errorf("timestamp validation failed: %w", err)
	}

	for name, source := range bma.sources {
		if customMetrics, err := source.GetMetrics(ctx); err == nil {
			systemMetrics.CustomMetrics[name] = customMetrics
		} else {
			systemMetrics.CustomMetrics[fmt.Sprintf("%s_error", name)] = err.Error()
		}
	}

	return systemMetrics, nil
}

func (bma *BasicMetricsAggregator) CollectRedisMetrics(ctx context.Context) (*interfaces.RedisMetrics, error) {
	timestamp := time.Now()

	redisMetrics := &interfaces.RedisMetrics{
		HitRate:           0.85, 
		MissRate:          0.15, 
		ResponseTime:      time.Millisecond * 5, 
		ConnectionsActive: 10,   
		MemoryUsage:       1024 * 1024 * 100, 
		KeyCount:          5000, 
		CommandsProcessed: 10000, 
		NetworkIO: &interfaces.NetworkIO{
			BytesIn:    1024 * 1024 * 50, 
			BytesOut:   1024 * 1024 * 25, 
			PacketsIn:  50000,
			PacketsOut: 25000,
		},
		Timestamp: timestamp,
	}

	return redisMetrics, nil
}

func (bma *BasicMetricsAggregator) CollectTMDBMetrics(ctx context.Context) (*interfaces.TMDBMetrics, error) {
	timestamp := time.Now()

	tmdbMetrics := &interfaces.TMDBMetrics{
		RequestRate:              10.0, 
		ResponseTime:             time.Millisecond * 150, 
		RateLimitUtilization:     0.75, 
		CircuitBreakerState:      "closed", 
		CacheHitRate:             0.80, 
		ErrorRate:                0.02, 
		QueuedRequests:           5,    
		ActiveRequests:           3,    
		BackoffDelay:             time.Millisecond * 100, 
		APIQuotaRemaining:        1000, 
		Timestamp:                timestamp,
	}

	return tmdbMetrics, nil
}

func (bma *BasicMetricsAggregator) CollectHTTPPoolMetrics(ctx context.Context) (*interfaces.HTTPPoolMetrics, error) {
	timestamp := time.Now()

	httpPoolMetrics := &interfaces.HTTPPoolMetrics{
		ActiveConnections:  15,  
		IdleConnections:    35,  
		TotalConnections:   50,  
		ConnectionReuse:    0.90, 
		AverageLatency:     time.Millisecond * 50, 
		ThroughputRPS:      25.0, 
		ConnectionLifetime: time.Minute * 10, 
		HandshakeTime:      time.Millisecond * 100, 
		DNSLookupTime:      time.Millisecond * 20,  
		ConnectionErrors:   2,    
		Timestamp:          timestamp,
	}

	return httpPoolMetrics, nil
}

func (bma *BasicMetricsAggregator) CollectDatabaseMetrics(ctx context.Context) (*interfaces.DatabaseMetrics, error) {
	timestamp := time.Now()

	databaseMetrics := &interfaces.DatabaseMetrics{
		ActiveConnections:    8,   
		IdleConnections:      17,  
		QueryTime:            time.Millisecond * 25, 
		TransactionTime:      time.Millisecond * 40, 
		QueriesPerSecond:     50.0, 
		CacheHitRatio:        0.95, 
		IndexHitRatio:        0.98, 
		ConnectionPoolHealth: "healthy", 
		SlowQueries:          0,    
		LockWaits:            0,    
		Timestamp:            timestamp,
	}

	return databaseMetrics, nil
}

func (bma *BasicMetricsAggregator) CollectSystemResources(ctx context.Context) (*interfaces.ResourceMetrics, error) {
	timestamp := time.Now()

	resourceMetrics := &interfaces.ResourceMetrics{
		CPUUsage:      25.0, 
		MemoryUsage:   1024 * 1024 * 512,  
		MemoryTotal:   1024 * 1024 * 2048, 
		MemoryPercent: 25.0, 
		DiskUsage:     1024 * 1024 * 1024 * 10,  
		DiskTotal:     1024 * 1024 * 1024 * 100, 
		DiskPercent:   10.0, 
		NetworkIO: &interfaces.NetworkIO{
			BytesIn:    1024 * 1024 * 50, 
			BytesOut:   1024 * 1024 * 25, 
			PacketsIn:  50000,
			PacketsOut: 25000,
		},
		LoadAverage: [3]float64{1.2, 1.1, 1.0}, 
		OpenFiles:   150, 
		Goroutines:  25,  
		GCStats: &interfaces.GCStats{
			NumGC:        100, 
			PauseTotal:   time.Millisecond * 500, 
			PauseAverage: time.Millisecond * 5,   
			LastGC:       time.Now().Add(-time.Minute * 5), 
		},
		Timestamp: timestamp,
	}

	return resourceMetrics, nil
}

func (bma *BasicMetricsAggregator) RegisterMetricsSource(name string, source MetricsSource) error {
	bma.mutex.Lock()
	defer bma.mutex.Unlock()

	if _, exists := bma.sources[name]; exists {
		return fmt.Errorf("metrics source '%s' already registered", name)
	}

	bma.sources[name] = source
	return nil
}

func (bma *BasicMetricsAggregator) UnregisterMetricsSource(name string) error {
	bma.mutex.Lock()
	defer bma.mutex.Unlock()

	if _, exists := bma.sources[name]; !exists {
		return fmt.Errorf("metrics source '%s' not found", name)
	}

	delete(bma.sources, name)
	return nil
}

func (bma *BasicMetricsAggregator) SerializeMetrics(metrics *interfaces.SystemMetrics) ([]byte, error) {
	if metrics == nil {
		return nil, fmt.Errorf("metrics cannot be nil")
	}

	if err := bma.ValidateTimestamps(metrics); err != nil {
		return nil, fmt.Errorf("timestamp validation failed before serialization: %w", err)
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize metrics to JSON: %w", err)
	}

	return data, nil
}

func (bma *BasicMetricsAggregator) DeserializeMetrics(data []byte) (*interfaces.SystemMetrics, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}

	var metrics interfaces.SystemMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to deserialize metrics from JSON: %w", err)
	}

	if err := bma.ValidateTimestamps(&metrics); err != nil {
		return nil, fmt.Errorf("timestamp validation failed after deserialization: %w", err)
	}

	return &metrics, nil
}

func (bma *BasicMetricsAggregator) ValidateTimestamps(metrics *interfaces.SystemMetrics) error {
	if metrics == nil {
		return fmt.Errorf("metrics cannot be nil")
	}

	baseTime := metrics.Timestamp
	if baseTime.IsZero() {
		return fmt.Errorf("main timestamp cannot be zero")
	}

	maxSkew := time.Second * 5
	now := time.Now()

	if now.Sub(baseTime) > time.Hour {
		return fmt.Errorf("main timestamp %v is too old (current time: %v)", baseTime, now)
	}

	if metrics.RedisMetrics != nil {
		if metrics.RedisMetrics.Timestamp.IsZero() {
			return fmt.Errorf("redis metrics timestamp cannot be zero")
		}
		if abs(baseTime.Sub(metrics.RedisMetrics.Timestamp)) > maxSkew {
			return fmt.Errorf("redis metrics timestamp %v differs from base timestamp %v by more than %v", 
				metrics.RedisMetrics.Timestamp, baseTime, maxSkew)
		}
	}

	if metrics.TMDBMetrics != nil {
		if metrics.TMDBMetrics.Timestamp.IsZero() {
			return fmt.Errorf("tmdb metrics timestamp cannot be zero")
		}
		if abs(baseTime.Sub(metrics.TMDBMetrics.Timestamp)) > maxSkew {
			return fmt.Errorf("tmdb metrics timestamp %v differs from base timestamp %v by more than %v", 
				metrics.TMDBMetrics.Timestamp, baseTime, maxSkew)
		}
	}

	if metrics.HTTPPoolMetrics != nil {
		if metrics.HTTPPoolMetrics.Timestamp.IsZero() {
			return fmt.Errorf("http pool metrics timestamp cannot be zero")
		}
		if abs(baseTime.Sub(metrics.HTTPPoolMetrics.Timestamp)) > maxSkew {
			return fmt.Errorf("http pool metrics timestamp %v differs from base timestamp %v by more than %v", 
				metrics.HTTPPoolMetrics.Timestamp, baseTime, maxSkew)
		}
	}

	if metrics.DatabaseMetrics != nil {
		if metrics.DatabaseMetrics.Timestamp.IsZero() {
			return fmt.Errorf("database metrics timestamp cannot be zero")
		}
		if abs(baseTime.Sub(metrics.DatabaseMetrics.Timestamp)) > maxSkew {
			return fmt.Errorf("database metrics timestamp %v differs from base timestamp %v by more than %v", 
				metrics.DatabaseMetrics.Timestamp, baseTime, maxSkew)
		}
	}

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

func (bma *BasicMetricsAggregator) GetCollectionInterval() time.Duration {
	bma.mutex.RLock()
	defer bma.mutex.RUnlock()
	return bma.collectionInterval
}

func (bma *BasicMetricsAggregator) SetCollectionInterval(interval time.Duration) {
	bma.mutex.Lock()
	defer bma.mutex.Unlock()
	bma.collectionInterval = interval
}

func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func (bma *BasicMetricsAggregator) GetLastCollectionTime() time.Time {
	bma.mutex.RLock()
	defer bma.mutex.RUnlock()
	return bma.lastCollection
}

func (bma *BasicMetricsAggregator) GetRegisteredSources() []string {
	bma.mutex.RLock()
	defer bma.mutex.RUnlock()

	sources := make([]string, 0, len(bma.sources))
	for name := range bma.sources {
		sources = append(sources, name)
	}
	return sources
}