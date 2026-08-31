// Package monitoring provides a unified metrics collection system that integrates
// the aggregator and component integrator for comprehensive performance monitoring.
//
// This file implements the main MetricsCollector that orchestrates metrics collection
// from all WaveFlix Hub scalability components.
package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/hamin-baek/waveflix-hub/scalability-testing/internal/interfaces"
)

// MetricsCollector provides a unified interface for collecting performance metrics
// from all scalability components with integration capabilities.
type MetricsCollector struct {
	aggregator       MetricsAggregator
	integrator       *ComponentIntegrator
	collectionTicker *time.Ticker
	isRunning        bool
	stopChan         chan struct{}
	mutex            sync.RWMutex
	metricsHistory   []*interfaces.SystemMetrics
	maxHistorySize   int
}

// MetricsCollectorConfig contains configuration for the metrics collector.
type MetricsCollectorConfig struct {
	CollectionInterval time.Duration `json:"collection_interval"`
	HistorySize       int           `json:"history_size"`
	EnableIntegration bool          `json:"enable_integration"`
}

// DefaultMetricsCollectorConfig returns a default configuration for the metrics collector.
func DefaultMetricsCollectorConfig() *MetricsCollectorConfig {
	return &MetricsCollectorConfig{
		CollectionInterval: time.Second * 30, // Collect metrics every 30 seconds
		HistorySize:       100,               // Keep last 100 metric snapshots
		EnableIntegration: true,              // Enable component integration by default
	}
}

// NewMetricsCollector creates a new unified metrics collector instance.
func NewMetricsCollector(config *MetricsCollectorConfig) *MetricsCollector {
	if config == nil {
		config = DefaultMetricsCollectorConfig()
	}

	aggregator := NewBasicMetricsAggregator(config.CollectionInterval)
	
	// Create integrator with nil collectors initially - they will be set when available
	integrator := NewComponentIntegrator(nil, nil, nil, nil)

	return &MetricsCollector{
		aggregator:     aggregator,
		integrator:     integrator,
		stopChan:       make(chan struct{}),
		maxHistorySize: config.HistorySize,
		metricsHistory: make([]*interfaces.SystemMetrics, 0, config.HistorySize),
	}
}

// Start begins continuous metrics collection with the specified interval.
func (mc *MetricsCollector) Start(ctx context.Context) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if mc.isRunning {
		return fmt.Errorf("metrics collector is already running")
	}

	mc.collectionTicker = time.NewTicker(mc.aggregator.GetCollectionInterval())
	mc.isRunning = true

	go mc.collectionLoop(ctx)

	return nil
}

// Stop gracefully shuts down the metrics collector.
func (mc *MetricsCollector) Stop() error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if !mc.isRunning {
		return fmt.Errorf("metrics collector is not running")
	}

	if mc.collectionTicker != nil {
		mc.collectionTicker.Stop()
	}

	close(mc.stopChan)
	mc.isRunning = false

	return nil
}

// IsRunning returns whether the metrics collector is currently running.
func (mc *MetricsCollector) IsRunning() bool {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return mc.isRunning
}

// CollectMetrics performs a one-time metrics collection from all components.
func (mc *MetricsCollector) CollectMetrics(ctx context.Context) (*interfaces.SystemMetrics, error) {
	// Create enhanced system metrics with integration
	enhancedMetrics := &interfaces.SystemMetrics{
		Timestamp:     time.Now(),
		CustomMetrics: make(map[string]interface{}),
	}

	// Collect Redis metrics from integration if available
	if redisMetrics, err := mc.integrator.CollectRedisMetrics(ctx); err == nil {
		enhancedMetrics.RedisMetrics = redisMetrics
	} else {
		// Fallback to aggregator metrics
		if metrics, err := mc.aggregator.CollectRedisMetrics(ctx); err == nil {
			enhancedMetrics.RedisMetrics = metrics
		}
		enhancedMetrics.CustomMetrics["redis_integration_error"] = err.Error()
	}

	// Collect TMDB metrics from integration if available
	if tmdbMetrics, err := mc.integrator.CollectTMDBMetrics(ctx); err == nil {
		enhancedMetrics.TMDBMetrics = tmdbMetrics
	} else {
		// Fallback to aggregator metrics
		if metrics, err := mc.aggregator.CollectTMDBMetrics(ctx); err == nil {
			enhancedMetrics.TMDBMetrics = metrics
		}
		enhancedMetrics.CustomMetrics["tmdb_integration_error"] = err.Error()
	}

	// Collect HTTP Pool metrics from integration if available
	if httpPoolMetrics, err := mc.integrator.CollectHTTPPoolMetrics(ctx); err == nil {
		enhancedMetrics.HTTPPoolMetrics = httpPoolMetrics
	} else {
		// Fallback to aggregator metrics
		if metrics, err := mc.aggregator.CollectHTTPPoolMetrics(ctx); err == nil {
			enhancedMetrics.HTTPPoolMetrics = metrics
		}
		enhancedMetrics.CustomMetrics["http_pool_integration_error"] = err.Error()
	}

	// Collect Database metrics from integration if available
	if databaseMetrics, err := mc.integrator.CollectDatabaseMetrics(ctx); err == nil {
		enhancedMetrics.DatabaseMetrics = databaseMetrics
	} else {
		// Fallback to aggregator metrics
		if metrics, err := mc.aggregator.CollectDatabaseMetrics(ctx); err == nil {
			enhancedMetrics.DatabaseMetrics = metrics
		}
		enhancedMetrics.CustomMetrics["database_integration_error"] = err.Error()
	}

	// Always collect system resources through aggregator
	if resourceMetrics, err := mc.aggregator.CollectSystemResources(ctx); err == nil {
		enhancedMetrics.SystemResources = resourceMetrics
	} else {
		enhancedMetrics.CustomMetrics["resources_error"] = err.Error()
	}

	// Add component health information
	if healthy, componentHealth, err := mc.integrator.IsHealthy(ctx); err == nil {
		enhancedMetrics.CustomMetrics["overall_health"] = healthy
		enhancedMetrics.CustomMetrics["component_health"] = componentHealth
	}

	// Validate and synchronize timestamps
	if err := mc.aggregator.ValidateTimestamps(enhancedMetrics); err != nil {
		return nil, fmt.Errorf("metrics timestamp validation failed: %w", err)
	}

	// Add to history
	mc.addToHistory(enhancedMetrics)

	return enhancedMetrics, nil
}

// GetCurrentSystemMetrics returns the most recent system metrics.
func (mc *MetricsCollector) GetCurrentSystemMetrics() (*interfaces.SystemMetrics, error) {
	return mc.CollectMetrics(context.Background())
}

// GetMetricsHistory returns the historical metrics data.
func (mc *MetricsCollector) GetMetricsHistory() []*interfaces.SystemMetrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	// Return a copy to prevent external modification
	history := make([]*interfaces.SystemMetrics, len(mc.metricsHistory))
	copy(history, mc.metricsHistory)
	return history
}

// GetMetricsTrend analyzes trends in the metrics history.
func (mc *MetricsCollector) GetMetricsTrend(metricPath string, duration time.Duration) (*interfaces.MetricTrend, error) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	if len(mc.metricsHistory) < 2 {
		return nil, fmt.Errorf("insufficient metrics history for trend analysis")
	}

	cutoffTime := time.Now().Add(-duration)
	var relevantMetrics []*interfaces.SystemMetrics
	
	for _, metrics := range mc.metricsHistory {
		if metrics.Timestamp.After(cutoffTime) {
			relevantMetrics = append(relevantMetrics, metrics)
		}
	}

	if len(relevantMetrics) < 2 {
		return nil, fmt.Errorf("insufficient metrics in time period for trend analysis")
	}

	// Extract metric values for the specified path
	values, err := mc.extractMetricValues(relevantMetrics, metricPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metric values: %w", err)
	}

	// Calculate trend
	trend := mc.calculateTrend(metricPath, values)
	return trend, nil
}

// SerializeMetrics serializes system metrics to JSON.
func (mc *MetricsCollector) SerializeMetrics(metrics *interfaces.SystemMetrics) ([]byte, error) {
	return mc.aggregator.SerializeMetrics(metrics)
}

// DeserializeMetrics deserializes JSON back to system metrics.
func (mc *MetricsCollector) DeserializeMetrics(data []byte) (*interfaces.SystemMetrics, error) {
	return mc.aggregator.DeserializeMetrics(data)
}

// RegisterMetricsSource registers an additional custom metrics source.
func (mc *MetricsCollector) RegisterMetricsSource(name string, source MetricsSource) error {
	return mc.aggregator.RegisterMetricsSource(name, source)
}

// UnregisterMetricsSource removes a custom metrics source.
func (mc *MetricsCollector) UnregisterMetricsSource(name string) error {
	return mc.aggregator.UnregisterMetricsSource(name)
}

// UpdateIntegrationCollectors updates the component integration collectors.
func (mc *MetricsCollector) UpdateIntegrationCollectors(
	redisCollector RedisMetricsCollector,
	tmdbCollector TMDBMetricsCollector,
	httpPoolCollector HTTPPoolMetricsCollector,
	dbCollector DatabaseMetricsCollector,
) {
	mc.integrator.UpdateCollectors(redisCollector, tmdbCollector, httpPoolCollector, dbCollector)
}

// SetCollectionInterval updates the metrics collection interval.
func (mc *MetricsCollector) SetCollectionInterval(interval time.Duration) {
	mc.aggregator.SetCollectionInterval(interval)
	
	// Update ticker if running
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	if mc.isRunning && mc.collectionTicker != nil {
		mc.collectionTicker.Stop()
		mc.collectionTicker = time.NewTicker(interval)
	}
}

// GetCollectionInterval returns the current collection interval.
func (mc *MetricsCollector) GetCollectionInterval() time.Duration {
	return mc.aggregator.GetCollectionInterval()
}

// collectionLoop runs the continuous metrics collection in a separate goroutine.
func (mc *MetricsCollector) collectionLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-mc.stopChan:
			return
		case <-mc.collectionTicker.C:
			// Collect metrics and ignore errors (they're logged internally)
			_, _ = mc.CollectMetrics(ctx)
		}
	}
}

// addToHistory adds metrics to the history buffer with size management.
func (mc *MetricsCollector) addToHistory(metrics *interfaces.SystemMetrics) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	// Add to history
	mc.metricsHistory = append(mc.metricsHistory, metrics)

	// Trim history if it exceeds max size
	if len(mc.metricsHistory) > mc.maxHistorySize {
		// Remove oldest entries
		excess := len(mc.metricsHistory) - mc.maxHistorySize
		mc.metricsHistory = mc.metricsHistory[excess:]
	}
}

// extractMetricValues extracts numeric values for a specific metric path from history.
func (mc *MetricsCollector) extractMetricValues(metricsHistory []*interfaces.SystemMetrics, metricPath string) ([]float64, error) {
	values := make([]float64, 0, len(metricsHistory))

	for _, metrics := range metricsHistory {
		value, err := mc.getMetricValueByPath(metrics, metricPath)
		if err != nil {
			continue // Skip invalid values
		}
		values = append(values, value)
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("no valid values found for metric path: %s", metricPath)
	}

	return values, nil
}

// getMetricValueByPath extracts a numeric value from metrics using a dot-separated path.
func (mc *MetricsCollector) getMetricValueByPath(metrics *interfaces.SystemMetrics, path string) (float64, error) {
	// This is a simplified implementation. In a real system, you'd use reflection
	// or a more sophisticated path resolution system.
	switch path {
	case "redis.hit_rate":
		if metrics.RedisMetrics != nil {
			return metrics.RedisMetrics.HitRate, nil
		}
	case "redis.response_time":
		if metrics.RedisMetrics != nil {
			return float64(metrics.RedisMetrics.ResponseTime.Nanoseconds()), nil
		}
	case "tmdb.request_rate":
		if metrics.TMDBMetrics != nil {
			return metrics.TMDBMetrics.RequestRate, nil
		}
	case "http_pool.throughput":
		if metrics.HTTPPoolMetrics != nil {
			return metrics.HTTPPoolMetrics.ThroughputRPS, nil
		}
	case "database.queries_per_second":
		if metrics.DatabaseMetrics != nil {
			return metrics.DatabaseMetrics.QueriesPerSecond, nil
		}
	case "system.cpu_usage":
		if metrics.SystemResources != nil {
			return metrics.SystemResources.CPUUsage, nil
		}
	case "system.memory_percent":
		if metrics.SystemResources != nil {
			return metrics.SystemResources.MemoryPercent, nil
		}
	}

	return 0, fmt.Errorf("metric path not found: %s", path)
}

// calculateTrend performs simple trend analysis on a series of values.
func (mc *MetricsCollector) calculateTrend(metricName string, values []float64) *interfaces.MetricTrend {
	if len(values) < 2 {
		return &interfaces.MetricTrend{
			MetricName: metricName,
			Direction:  interfaces.TrendStable,
			Confidence: 0.0,
		}
	}

	// Calculate basic statistics
	startValue := values[0]
	endValue := values[len(values)-1]
	minValue := values[0]
	maxValue := values[0]
	sum := 0.0

	for _, v := range values {
		if v < minValue {
			minValue = v
		}
		if v > maxValue {
			maxValue = v
		}
		sum += v
	}

	// Calculate simple slope (change per observation)
	slope := (endValue - startValue) / float64(len(values)-1)
	
	// Calculate volatility (standard deviation)
	mean := sum / float64(len(values))
	varianceSum := 0.0
	for _, v := range values {
		varianceSum += (v - mean) * (v - mean)
	}
	volatility := varianceSum / float64(len(values))

	// Determine trend direction
	direction := interfaces.TrendStable
	if slope > 0.1 { // Threshold for significant increase
		direction = interfaces.TrendIncreasing
	} else if slope < -0.1 { // Threshold for significant decrease
		direction = interfaces.TrendDecreasing
	} else if volatility > mean*0.1 { // High volatility relative to mean
		direction = interfaces.TrendVolatile
	}

	// Calculate confidence based on consistency
	confidence := 1.0 - (volatility / (maxValue - minValue + 0.001)) // Avoid division by zero

	return &interfaces.MetricTrend{
		MetricName: metricName,
		Direction:  direction,
		Slope:      slope,
		Confidence: confidence,
		StartValue: startValue,
		EndValue:   endValue,
		MinValue:   minValue,
		MaxValue:   maxValue,
		Volatility: volatility,
	}
}

// GetSystemResourcesSnapshot returns a snapshot of current system resources.
func (mc *MetricsCollector) GetSystemResourcesSnapshot() *interfaces.ResourceMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &interfaces.ResourceMetrics{
		CPUUsage:      0.0, // TODO: Implement actual CPU monitoring
		MemoryUsage:   int64(m.Alloc),
		MemoryTotal:   int64(m.Sys),
		MemoryPercent: float64(m.Alloc) / float64(m.Sys) * 100,
		Goroutines:    runtime.NumGoroutine(),
		GCStats: &interfaces.GCStats{
			NumGC:        uint32(m.NumGC),
			PauseTotal:   time.Duration(m.PauseTotalNs),
			PauseAverage: time.Duration(m.PauseTotalNs / uint64(m.NumGC+1)),
			LastGC:       time.Unix(0, int64(m.LastGC)),
		},
		Timestamp: time.Now(),
	}
}