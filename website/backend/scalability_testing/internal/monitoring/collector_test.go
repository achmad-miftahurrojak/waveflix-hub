// Package monitoring provides comprehensive tests for the metrics collection system.
//
// This file tests the MetricsAggregator interface, component integration,
// and metrics serialization functionality.
package monitoring

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hamin-baek/waveflix-hub/scalability-testing/internal/interfaces"
)

// TestBasicMetricsAggregator tests the basic metrics aggregation functionality.
func TestBasicMetricsAggregator(t *testing.T) {
	aggregator := NewBasicMetricsAggregator(time.Second * 30)

	// Test collecting all metrics
	ctx := context.Background()
	metrics, err := aggregator.CollectAllMetrics(ctx)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// Validate that metrics were collected
	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}

	if metrics.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	// Test individual component metrics
	redisMetrics, err := aggregator.CollectRedisMetrics(ctx)
	if err != nil {
		t.Errorf("Failed to collect Redis metrics: %v", err)
	}

	if redisMetrics == nil {
		t.Error("Redis metrics should not be nil")
	}

	// Validate Redis metrics requirements (hit rate > 80%)
	if redisMetrics.HitRate < 0.8 {
		t.Errorf("Redis hit rate %f is below required 80%%", redisMetrics.HitRate)
	}

	// Test TMDB metrics
	tmdbMetrics, err := aggregator.CollectTMDBMetrics(ctx)
	if err != nil {
		t.Errorf("Failed to collect TMDB metrics: %v", err)
	}

	if tmdbMetrics == nil {
		t.Error("TMDB metrics should not be nil")
	}

	// Validate TMDB rate limiting is working
	if tmdbMetrics.RateLimitUtilization < 0 || tmdbMetrics.RateLimitUtilization > 1 {
		t.Errorf("TMDB rate limit utilization %f is not in valid range [0,1]", tmdbMetrics.RateLimitUtilization)
	}

	// Test HTTP pool metrics
	httpMetrics, err := aggregator.CollectHTTPPoolMetrics(ctx)
	if err != nil {
		t.Errorf("Failed to collect HTTP pool metrics: %v", err)
	}

	if httpMetrics == nil {
		t.Error("HTTP pool metrics should not be nil")
	}

	// Validate connection reuse efficiency
	if httpMetrics.ConnectionReuse < 0 || httpMetrics.ConnectionReuse > 1 {
		t.Errorf("Connection reuse rate %f is not in valid range [0,1]", httpMetrics.ConnectionReuse)
	}

	// Test database metrics
	dbMetrics, err := aggregator.CollectDatabaseMetrics(ctx)
	if err != nil {
		t.Errorf("Failed to collect database metrics: %v", err)
	}

	if dbMetrics == nil {
		t.Error("Database metrics should not be nil")
	}

	// Validate database cache hit ratio is high
	if dbMetrics.CacheHitRatio < 0.9 {
		t.Errorf("Database cache hit ratio %f is below expected 90%%", dbMetrics.CacheHitRatio)
	}
}

// TestMetricsTimestampValidation tests the timestamp validation functionality.
func TestMetricsTimestampValidation(t *testing.T) {
	aggregator := NewBasicMetricsAggregator(time.Second * 30)

	// Test valid timestamps
	now := time.Now()
	validMetrics := &interfaces.SystemMetrics{
		Timestamp: now,
		RedisMetrics: &interfaces.RedisMetrics{
			Timestamp: now,
		},
		TMDBMetrics: &interfaces.TMDBMetrics{
			Timestamp: now,
		},
	}

	if err := aggregator.ValidateTimestamps(validMetrics); err != nil {
		t.Errorf("Valid timestamps should not produce error: %v", err)
	}

	// Test invalid timestamps (too much skew)
	skewedTime := now.Add(time.Second * 10) // 10 seconds skew
	invalidMetrics := &interfaces.SystemMetrics{
		Timestamp: now,
		RedisMetrics: &interfaces.RedisMetrics{
			Timestamp: skewedTime,
		},
	}

	if err := aggregator.ValidateTimestamps(invalidMetrics); err == nil {
		t.Error("Invalid timestamps should produce error")
	}

	// Test nil metrics
	if err := aggregator.ValidateTimestamps(nil); err == nil {
		t.Error("Nil metrics should produce error")
	}

	// Test zero timestamps
	zeroMetrics := &interfaces.SystemMetrics{
		Timestamp: time.Time{},
	}

	if err := aggregator.ValidateTimestamps(zeroMetrics); err == nil {
		t.Error("Zero timestamp should produce error")
	}
}

// TestMetricsSerialization tests metrics serialization and deserialization.
func TestMetricsSerialization(t *testing.T) {
	aggregator := NewBasicMetricsAggregator(time.Second * 30)

	// Create test metrics
	originalMetrics := &interfaces.SystemMetrics{
		Timestamp: time.Now(),
		RedisMetrics: &interfaces.RedisMetrics{
			HitRate:      0.85,
			MissRate:     0.15,
			ResponseTime: time.Millisecond * 5,
			Timestamp:    time.Now(),
		},
		TMDBMetrics: &interfaces.TMDBMetrics{
			RequestRate:              10.0,
			RateLimitUtilization:     0.75,
			CircuitBreakerState:      "closed",
			Timestamp:                time.Now(),
		},
		CustomMetrics: map[string]interface{}{
			"test_metric": 42.0,
		},
	}

	// Test serialization
	serializedData, err := aggregator.SerializeMetrics(originalMetrics)
	if err != nil {
		t.Fatalf("Failed to serialize metrics: %v", err)
	}

	if len(serializedData) == 0 {
		t.Error("Serialized data should not be empty")
	}

	// Test that serialized data is valid JSON
	var testJSON map[string]interface{}
	if err := json.Unmarshal(serializedData, &testJSON); err != nil {
		t.Errorf("Serialized data is not valid JSON: %v", err)
	}

	// Test deserialization
	deserializedMetrics, err := aggregator.DeserializeMetrics(serializedData)
	if err != nil {
		t.Fatalf("Failed to deserialize metrics: %v", err)
	}

	if deserializedMetrics == nil {
		t.Fatal("Deserialized metrics should not be nil")
	}

	// Validate deserialized data matches original
	if deserializedMetrics.RedisMetrics.HitRate != originalMetrics.RedisMetrics.HitRate {
		t.Errorf("Redis hit rate mismatch: expected %f, got %f", 
			originalMetrics.RedisMetrics.HitRate, deserializedMetrics.RedisMetrics.HitRate)
	}

	if deserializedMetrics.TMDBMetrics.RequestRate != originalMetrics.TMDBMetrics.RequestRate {
		t.Errorf("TMDB request rate mismatch: expected %f, got %f",
			originalMetrics.TMDBMetrics.RequestRate, deserializedMetrics.TMDBMetrics.RequestRate)
	}

	// Test serialization of nil metrics
	if _, err := aggregator.SerializeMetrics(nil); err == nil {
		t.Error("Serializing nil metrics should produce error")
	}

	// Test deserialization of empty data
	if _, err := aggregator.DeserializeMetrics([]byte{}); err == nil {
		t.Error("Deserializing empty data should produce error")
	}

	// Test deserialization of invalid JSON
	if _, err := aggregator.DeserializeMetrics([]byte("invalid json")); err == nil {
		t.Error("Deserializing invalid JSON should produce error")
	}
}

// TestMetricsSourceRegistration tests the metrics source registration functionality.
func TestMetricsSourceRegistration(t *testing.T) {
	aggregator := NewBasicMetricsAggregator(time.Second * 30)

	// Create a mock metrics source
	mockSource := &mockMetricsSource{
		name: "test_source",
		metrics: map[string]interface{}{
			"test_value": 100.0,
		},
	}

	// Test registration
	if err := aggregator.RegisterMetricsSource("test_source", mockSource); err != nil {
		t.Errorf("Failed to register metrics source: %v", err)
	}

	// Test duplicate registration
	if err := aggregator.RegisterMetricsSource("test_source", mockSource); err == nil {
		t.Error("Duplicate registration should produce error")
	}

	// Test unregistration
	if err := aggregator.UnregisterMetricsSource("test_source"); err != nil {
		t.Errorf("Failed to unregister metrics source: %v", err)
	}

	// Test unregistration of non-existent source
	if err := aggregator.UnregisterMetricsSource("non_existent"); err == nil {
		t.Error("Unregistering non-existent source should produce error")
	}
}

// TestMetricsCollector tests the unified metrics collector.
func TestMetricsCollector(t *testing.T) {
	config := DefaultMetricsCollectorConfig()
	config.CollectionInterval = time.Millisecond * 100 // Fast collection for testing
	config.HistorySize = 5

	collector := NewMetricsCollector(config)

	// Test basic collection
	ctx := context.Background()
	metrics, err := collector.CollectMetrics(ctx)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}

	// Test starting and stopping collection
	if err := collector.Start(ctx); err != nil {
		t.Errorf("Failed to start collector: %v", err)
	}

	if !collector.IsRunning() {
		t.Error("Collector should be running after start")
	}

	// Wait for a few collections
	time.Sleep(time.Millisecond * 300)

	// Check that history was populated
	history := collector.GetMetricsHistory()
	if len(history) == 0 {
		t.Error("History should have some metrics after running")
	}

	if err := collector.Stop(); err != nil {
		t.Errorf("Failed to stop collector: %v", err)
	}

	if collector.IsRunning() {
		t.Error("Collector should not be running after stop")
	}

	// Test collection interval changes
	newInterval := time.Second * 60
	collector.SetCollectionInterval(newInterval)
	if collector.GetCollectionInterval() != newInterval {
		t.Errorf("Collection interval not updated: expected %v, got %v", 
			newInterval, collector.GetCollectionInterval())
	}
}

// TestComponentIntegrator tests the component integration functionality.
func TestComponentIntegrator(t *testing.T) {
	// Create mock collectors
	redisCollector := &mockRedisCollector{
		healthy: true,
		hitRate: 0.9,
	}

	tmdbCollector := &mockTMDBCollector{
		requestRate: 15.0,
		responseTime: time.Millisecond * 120,
	}

	integrator := NewComponentIntegrator(redisCollector, tmdbCollector, nil, nil)

	ctx := context.Background()

	// Test Redis metrics collection
	redisMetrics, err := integrator.CollectRedisMetrics(ctx)
	if err != nil {
		t.Errorf("Failed to collect Redis metrics: %v", err)
	}

	if redisMetrics.HitRate != 0.9 {
		t.Errorf("Expected hit rate 0.9, got %f", redisMetrics.HitRate)
	}

	// Test TMDB metrics collection
	tmdbMetrics, err := integrator.CollectTMDBMetrics(ctx)
	if err != nil {
		t.Errorf("Failed to collect TMDB metrics: %v", err)
	}

	if tmdbMetrics.RequestRate != 15.0 {
		t.Errorf("Expected request rate 15.0, got %f", tmdbMetrics.RequestRate)
	}

	// Test health check
	healthy, componentHealth, err := integrator.IsHealthy(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	if !healthy {
		t.Error("Overall health should be true")
	}

	if !componentHealth["redis"] {
		t.Error("Redis should be healthy")
	}
}

// Mock implementations for testing

type mockMetricsSource struct {
	name    string
	metrics map[string]interface{}
}

func (m *mockMetricsSource) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	return m.metrics, nil
}

func (m *mockMetricsSource) GetHealthStatus(ctx context.Context) (interfaces.HealthStatus, error) {
	return interfaces.HealthHealthy, nil
}

func (m *mockMetricsSource) GetComponentName() string {
	return m.name
}

type mockRedisCollector struct {
	healthy bool
	hitRate float64
}

func (m *mockRedisCollector) GetCacheStats(ctx context.Context) (hits, misses int64, hitRate float64, err error) {
	return 900, 100, m.hitRate, nil
}

func (m *mockRedisCollector) GetResponseTime(ctx context.Context) (time.Duration, error) {
	return time.Millisecond * 5, nil
}

func (m *mockRedisCollector) GetConnectionInfo(ctx context.Context) (active, total int, err error) {
	return 10, 20, nil
}

func (m *mockRedisCollector) GetMemoryUsage(ctx context.Context) (used, total int64, err error) {
	return 1024*1024*100, 1024*1024*500, nil
}

func (m *mockRedisCollector) GetKeyCount(ctx context.Context) (int, error) {
	return 5000, nil
}

func (m *mockRedisCollector) GetCommandStats(ctx context.Context) (processed int64, err error) {
	return 10000, nil
}

func (m *mockRedisCollector) IsHealthy(ctx context.Context) (bool, error) {
	return m.healthy, nil
}

type mockTMDBCollector struct {
	requestRate  float64
	responseTime time.Duration
}

func (m *mockTMDBCollector) GetRequestRate(ctx context.Context) (float64, error) {
	return m.requestRate, nil
}

func (m *mockTMDBCollector) GetResponseTime(ctx context.Context) (time.Duration, error) {
	return m.responseTime, nil
}

func (m *mockTMDBCollector) GetRateLimitStatus(ctx context.Context) (utilization float64, remaining int, err error) {
	return 0.75, 1000, nil
}

func (m *mockTMDBCollector) GetCircuitBreakerState(ctx context.Context) (string, error) {
	return "closed", nil
}

func (m *mockTMDBCollector) GetCacheStats(ctx context.Context) (hitRate float64, err error) {
	return 0.8, nil
}

func (m *mockTMDBCollector) GetErrorRate(ctx context.Context) (float64, error) {
	return 0.02, nil
}

func (m *mockTMDBCollector) GetQueueStatus(ctx context.Context) (queued, active int, err error) {
	return 5, 3, nil
}

func (m *mockTMDBCollector) GetBackoffDelay(ctx context.Context) (time.Duration, error) {
	return time.Millisecond * 100, nil
}