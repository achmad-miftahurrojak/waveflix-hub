// Package interfaces defines the core interfaces for the scalability testing and monitoring system.
//
// These interfaces provide the contract for integration validation, load testing,
// performance monitoring, and environment management components.
package interfaces

import (
	"context"
	"time"
)

// IntegrationValidator defines the interface for validating scalability component integration.
//
// The validator ensures all scalability components (Redis, PostgreSQL, TMDB, HTTP pooling)
// work together seamlessly and deliver expected performance improvements.
type IntegrationValidator interface {
	// ValidateRedisIntegration tests Redis cache integration with database operations
	ValidateRedisIntegration(ctx context.Context) (*ValidationResult, error)

	// ValidateTMDBIntegration tests TMDB client integration with rate limiting
	ValidateTMDBIntegration(ctx context.Context) (*ValidationResult, error)

	// ValidateHTTPPooling tests HTTP connection pool integration with external APIs
	ValidateHTTPPooling(ctx context.Context) (*ValidationResult, error)

	// ValidateEndToEndPerformance measures end-to-end performance improvements
	ValidateEndToEndPerformance(ctx context.Context) (*ValidationResult, error)

	// RunAllTests executes all validation tests and returns comprehensive results
	RunAllTests(ctx context.Context) (map[string]*ValidationResult, error)

	// Shutdown gracefully shuts down the validator
	Shutdown(ctx context.Context) error
}

// LoadTester defines the interface for executing load testing scenarios.
//
// The load tester generates realistic user scenarios and validates performance
// under various load conditions to ensure scalability improvements are effective.
type LoadTester interface {
	// ExecuteLoadTest runs a load test scenario and returns performance results
	ExecuteLoadTest(ctx context.Context, scenario *LoadTestScenario) (*LoadTestResults, error)

	// GenerateUserScenarios creates realistic user behavior patterns
	GenerateUserScenarios(userCount int, duration time.Duration) ([]*UserScenario, error)

	// MeasureBaseline establishes performance baselines for comparison
	MeasureBaseline(ctx context.Context) (*BaselineMetrics, error)

	// ValidateCacheEfficiency tests cache performance against thresholds
	ValidateCacheEfficiency(ctx context.Context, threshold float64) (*CacheValidationResult, error)

	// Shutdown gracefully shuts down the load tester
	Shutdown(ctx context.Context) error
}

// PerformanceMonitor defines the interface for real-time performance monitoring.
//
// The monitor collects metrics from all scalability components and provides
// alerting capabilities for performance threshold violations.
type PerformanceMonitor interface {
	// CollectMetrics gathers current performance metrics from all components
	CollectMetrics(ctx context.Context) (*SystemMetrics, error)

	// RegisterAlert configures performance threshold alerts
	RegisterAlert(alert *AlertDefinition) error

	// GetRealTimeMetrics returns current real-time performance data
	GetRealTimeMetrics() (*RealTimeMetrics, error)

	// AnalyzeTrends performs trend analysis over specified time period
	AnalyzeTrends(period time.Duration) (*TrendAnalysis, error)

	// Start begins continuous monitoring
	Start(ctx context.Context) error

	// Shutdown gracefully shuts down the monitor
	Shutdown(ctx context.Context) error
}

// EnvironmentManager defines the interface for managing deployment environments.
//
// The manager handles configuration validation, environment detection, and
// connectivity verification across different deployment environments.
type EnvironmentManager interface {
	// ValidateConfiguration validates environment-specific configuration
	ValidateConfiguration(env Environment) (*ConfigValidation, error)

	// LoadEnvironmentConfig loads configuration for specified environment
	LoadEnvironmentConfig(env Environment) (*EnvironmentConfig, error)

	// DetectEnvironment automatically detects current deployment environment
	DetectEnvironment() (Environment, error)

	// ValidateConnectivity verifies connectivity to external services
	ValidateConnectivity(ctx context.Context) (*ConnectivityCheck, error)
}

// ValidationResult represents the result of an integration validation test.
type ValidationResult struct {
	Component   string                 `json:"component"`
	Success     bool                   `json:"success"`
	Metrics     map[string]float64     `json:"metrics"`
	Diagnostics []string               `json:"diagnostics"`
	Duration    time.Duration          `json:"duration"`
	Timestamp   time.Time              `json:"timestamp"`
	Details     map[string]interface{} `json:"details"`
}

// LoadTestScenario defines a load testing scenario configuration.
type LoadTestScenario struct {
	Name            string                 `json:"name"`
	ConcurrentUsers int                    `json:"concurrent_users"`
	Duration        time.Duration          `json:"duration"`
	RampUpTime      time.Duration          `json:"ramp_up_time"`
	RequestPatterns []*RequestPattern      `json:"request_patterns"`
	ExpectedMetrics map[string]float64     `json:"expected_metrics"`
	Environment     Environment            `json:"environment"`
	Configuration   map[string]interface{} `json:"configuration"`
}

// RequestPattern defines a pattern of requests for load testing.
type RequestPattern struct {
	Name        string        `json:"name"`
	Method      string        `json:"method"`
	Path        string        `json:"path"`
	Weight      float64       `json:"weight"`      // Probability weight (0.0-1.0)
	ThinkTime   time.Duration `json:"think_time"`  // Time between requests
	Parameters  map[string]interface{} `json:"parameters"`
}

// UserScenario defines a realistic user behavior pattern.
type UserScenario struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Actions     []*UserAction          `json:"actions"`
	Duration    time.Duration          `json:"duration"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// UserAction represents a single user action in a scenario.
type UserAction struct {
	Type        string                 `json:"type"`
	Target      string                 `json:"target"`
	Parameters  map[string]interface{} `json:"parameters"`
	ThinkTime   time.Duration          `json:"think_time"`
	Timeout     time.Duration          `json:"timeout"`
}

// LoadTestResults contains comprehensive load test results.
type LoadTestResults struct {
	TestID             string                 `json:"test_id"`
	Scenario           *LoadTestScenario      `json:"scenario"`
	StartTime          time.Time              `json:"start_time"`
	EndTime            time.Time              `json:"end_time"`
	Duration           time.Duration          `json:"duration"`
	TotalRequests      int64                  `json:"total_requests"`
	SuccessfulRequests int64                  `json:"successful_requests"`
	FailedRequests     int64                  `json:"failed_requests"`
	AverageLatency     time.Duration          `json:"average_latency"`
	P50Latency         time.Duration          `json:"p50_latency"`
	P95Latency         time.Duration          `json:"p95_latency"`
	P99Latency         time.Duration          `json:"p99_latency"`
	ThroughputRPS      float64                `json:"throughput_rps"`
	ErrorRate          float64                `json:"error_rate"`
	CacheEfficiency    float64                `json:"cache_efficiency"`
	ComponentMetrics   map[string]interface{} `json:"component_metrics"`
	ResourceUtilization *ResourceMetrics      `json:"resource_utilization"`
}

// BaselineMetrics establishes performance baselines for comparison.
type BaselineMetrics struct {
	Version            string                 `json:"version"`
	Environment        Environment            `json:"environment"`
	Timestamp          time.Time              `json:"timestamp"`
	ResponseTime       time.Duration          `json:"response_time"`
	ThroughputRPS      float64                `json:"throughput_rps"`
	CacheHitRate       float64                `json:"cache_hit_rate"`
	DatabaseQueryTime  time.Duration          `json:"database_query_time"`
	ComponentMetrics   map[string]float64     `json:"component_metrics"`
	SystemResources    *ResourceMetrics       `json:"system_resources"`
	Configuration      map[string]interface{} `json:"configuration"`
}

// CacheValidationResult contains cache performance validation results.
type CacheValidationResult struct {
	Threshold       float64   `json:"threshold"`
	ActualHitRate   float64   `json:"actual_hit_rate"`
	Passed          bool      `json:"passed"`
	CacheSize       int64     `json:"cache_size"`
	HitCount        int64     `json:"hit_count"`
	MissCount       int64     `json:"miss_count"`
	AverageHitTime  time.Duration `json:"average_hit_time"`
	AverageMissTime time.Duration `json:"average_miss_time"`
	Timestamp       time.Time `json:"timestamp"`
}

// SystemMetrics contains comprehensive system performance metrics.
type SystemMetrics struct {
	Timestamp       time.Time              `json:"timestamp"`
	RedisMetrics    *RedisMetrics          `json:"redis_metrics"`
	TMDBMetrics     *TMDBMetrics           `json:"tmdb_metrics"`
	HTTPPoolMetrics *HTTPPoolMetrics       `json:"http_pool_metrics"`
	DatabaseMetrics *DatabaseMetrics       `json:"database_metrics"`
	SystemResources *ResourceMetrics       `json:"system_resources"`
	CustomMetrics   map[string]interface{} `json:"custom_metrics"`
}

// RedisMetrics captures Redis cache performance data.
type RedisMetrics struct {
	HitRate           float64       `json:"hit_rate"`
	MissRate          float64       `json:"miss_rate"`
	ResponseTime      time.Duration `json:"response_time"`
	ConnectionsActive int           `json:"connections_active"`
	MemoryUsage       int64         `json:"memory_usage_bytes"`
	KeyCount          int           `json:"key_count"`
	CommandsProcessed int64         `json:"commands_processed"`
	NetworkIO         *NetworkIO    `json:"network_io"`
	Timestamp         time.Time     `json:"timestamp"`
}

// TMDBMetrics captures TMDB API client performance data.
type TMDBMetrics struct {
	RequestRate              float64       `json:"requests_per_second"`
	ResponseTime             time.Duration `json:"response_time"`
	RateLimitUtilization     float64       `json:"rate_limit_utilization"`
	CircuitBreakerState      string        `json:"circuit_breaker_state"`
	CacheHitRate             float64       `json:"cache_hit_rate"`
	ErrorRate                float64       `json:"error_rate"`
	QueuedRequests           int           `json:"queued_requests"`
	ActiveRequests           int           `json:"active_requests"`
	BackoffDelay             time.Duration `json:"backoff_delay"`
	APIQuotaRemaining        int           `json:"api_quota_remaining"`
	Timestamp                time.Time     `json:"timestamp"`
}

// HTTPPoolMetrics captures HTTP connection pool performance.
type HTTPPoolMetrics struct {
	ActiveConnections   int           `json:"active_connections"`
	IdleConnections     int           `json:"idle_connections"`
	TotalConnections    int           `json:"total_connections"`
	ConnectionReuse     float64       `json:"connection_reuse_rate"`
	AverageLatency      time.Duration `json:"average_latency"`
	ThroughputRPS       float64       `json:"throughput_rps"`
	ConnectionLifetime  time.Duration `json:"connection_lifetime"`
	HandshakeTime       time.Duration `json:"handshake_time"`
	DNSLookupTime       time.Duration `json:"dns_lookup_time"`
	ConnectionErrors    int64         `json:"connection_errors"`
	Timestamp           time.Time     `json:"timestamp"`
}

// DatabaseMetrics captures PostgreSQL database performance.
type DatabaseMetrics struct {
	ActiveConnections    int           `json:"active_connections"`
	IdleConnections      int           `json:"idle_connections"`
	QueryTime            time.Duration `json:"query_time"`
	TransactionTime      time.Duration `json:"transaction_time"`
	QueriesPerSecond     float64       `json:"queries_per_second"`
	CacheHitRatio        float64       `json:"cache_hit_ratio"`
	IndexHitRatio        float64       `json:"index_hit_ratio"`
	ConnectionPoolHealth string        `json:"connection_pool_health"`
	SlowQueries          int64         `json:"slow_queries"`
	LockWaits            int64         `json:"lock_waits"`
	Timestamp            time.Time     `json:"timestamp"`
}

// ResourceMetrics captures system resource utilization.
type ResourceMetrics struct {
	CPUUsage       float64     `json:"cpu_usage_percent"`
	MemoryUsage    int64       `json:"memory_usage_bytes"`
	MemoryTotal    int64       `json:"memory_total_bytes"`
	MemoryPercent  float64     `json:"memory_usage_percent"`
	DiskUsage      int64       `json:"disk_usage_bytes"`
	DiskTotal      int64       `json:"disk_total_bytes"`
	DiskPercent    float64     `json:"disk_usage_percent"`
	NetworkIO      *NetworkIO  `json:"network_io"`
	LoadAverage    [3]float64  `json:"load_average"`
	OpenFiles      int         `json:"open_files"`
	Goroutines     int         `json:"goroutines"`
	GCStats        *GCStats    `json:"gc_stats"`
	Timestamp      time.Time   `json:"timestamp"`
}

// NetworkIO captures network I/O statistics.
type NetworkIO struct {
	BytesIn   int64 `json:"bytes_in"`
	BytesOut  int64 `json:"bytes_out"`
	PacketsIn int64 `json:"packets_in"`
	PacketsOut int64 `json:"packets_out"`
}

// GCStats captures garbage collection statistics.
type GCStats struct {
	NumGC        uint32        `json:"num_gc"`
	PauseTotal   time.Duration `json:"pause_total_ns"`
	PauseAverage time.Duration `json:"pause_average_ns"`
	LastGC       time.Time     `json:"last_gc"`
}

// RealTimeMetrics contains current real-time performance data.
type RealTimeMetrics struct {
	Timestamp     time.Time              `json:"timestamp"`
	SystemHealth  HealthStatus           `json:"system_health"`
	Performance   *SystemMetrics         `json:"performance"`
	Alerts        []*ActiveAlert         `json:"alerts"`
	Trends        map[string]interface{} `json:"trends"`
}

// AlertDefinition defines performance threshold alerts.
type AlertDefinition struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	MetricPath  string                 `json:"metric_path"`  // e.g., "redis.hit_rate"
	Condition   AlertCondition         `json:"condition"`    // >, <, ==, etc.
	Threshold   float64                `json:"threshold"`
	Duration    time.Duration          `json:"duration"`     // How long threshold must be exceeded
	Severity    AlertSeverity          `json:"severity"`
	Description string                 `json:"description"`
	Actions     []AlertAction          `json:"actions"`
	Enabled     bool                   `json:"enabled"`
	Tags        map[string]string      `json:"tags"`
}

// ActiveAlert represents an currently active alert.
type ActiveAlert struct {
	Definition   *AlertDefinition `json:"definition"`
	StartTime    time.Time        `json:"start_time"`
	LastTriggered time.Time       `json:"last_triggered"`
	CurrentValue float64          `json:"current_value"`
	Count        int              `json:"count"`
	Status       AlertStatus      `json:"status"`
}

// TrendAnalysis contains performance trend analysis results.
type TrendAnalysis struct {
	Period         time.Duration          `json:"period"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	Trends         map[string]*MetricTrend `json:"trends"`
	Predictions    map[string]*Prediction `json:"predictions"`
	Recommendations []string             `json:"recommendations"`
	OverallHealth  HealthStatus           `json:"overall_health"`
}

// MetricTrend represents a trend for a specific metric.
type MetricTrend struct {
	MetricName    string    `json:"metric_name"`
	Direction     TrendDirection `json:"direction"`  // Increasing, Decreasing, Stable
	Slope         float64   `json:"slope"`
	Confidence    float64   `json:"confidence"`    // 0.0-1.0
	StartValue    float64   `json:"start_value"`
	EndValue      float64   `json:"end_value"`
	MinValue      float64   `json:"min_value"`
	MaxValue      float64   `json:"max_value"`
	Volatility    float64   `json:"volatility"`
}

// Prediction contains performance predictions.
type Prediction struct {
	MetricName     string        `json:"metric_name"`
	PredictedValue float64       `json:"predicted_value"`
	Confidence     float64       `json:"confidence"`
	TimeHorizon    time.Duration `json:"time_horizon"`
	Methodology    string        `json:"methodology"`
}

// Environment represents deployment environments.
type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvStaging     Environment = "staging" 
	EnvProduction  Environment = "production"
	EnvTesting     Environment = "testing"
)

// HealthStatus represents the health status of a component or system.
type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthDegraded  HealthStatus = "degraded" 
	HealthUnhealthy HealthStatus = "unhealthy"
	HealthUnknown   HealthStatus = "unknown"
)

// AlertCondition defines alert threshold conditions.
type AlertCondition string

const (
	AlertGreaterThan    AlertCondition = ">"
	AlertLessThan       AlertCondition = "<"
	AlertEquals         AlertCondition = "=="
	AlertNotEquals      AlertCondition = "!="
	AlertGreaterEquals  AlertCondition = ">="
	AlertLessEquals     AlertCondition = "<="
)

// AlertSeverity defines alert severity levels.
type AlertSeverity string

const (
	AlertCritical AlertSeverity = "critical"
	AlertHigh     AlertSeverity = "high" 
	AlertMedium   AlertSeverity = "medium"
	AlertLow      AlertSeverity = "low"
	AlertInfo     AlertSeverity = "info"
)

// AlertStatus represents the current status of an alert.
type AlertStatus string

const (
	AlertFiring    AlertStatus = "firing"
	AlertPending   AlertStatus = "pending"
	AlertResolved  AlertStatus = "resolved"
	AlertSilenced  AlertStatus = "silenced"
)

// AlertAction defines actions to take when an alert fires.
type AlertAction struct {
	Type       string                 `json:"type"`        // email, webhook, slack, etc.
	Target     string                 `json:"target"`      // email address, URL, etc.
	Parameters map[string]interface{} `json:"parameters"`
	Enabled    bool                   `json:"enabled"`
}

// TrendDirection indicates the direction of a metric trend.
type TrendDirection string

const (
	TrendIncreasing TrendDirection = "increasing"
	TrendDecreasing TrendDirection = "decreasing"
	TrendStable     TrendDirection = "stable"
	TrendVolatile   TrendDirection = "volatile"
)

// EnvironmentConfig holds environment-specific configuration.
type EnvironmentConfig struct {
	Environment   Environment   `json:"environment"`
	Redis         *RedisConfig  `json:"redis"`
	PostgreSQL    *PostgreSQLConfig `json:"postgresql"`
	TMDB          *TMDBConfig   `json:"tmdb"`
	HTTP          *HTTPConfig   `json:"http"`
	Monitoring    *MonitoringConfig `json:"monitoring"`
	LoadTesting   *LoadTestingConfig `json:"load_testing"`
}

// RedisConfig contains Redis-specific configuration.
type RedisConfig struct {
	URL         string        `json:"url"`
	Password    string        `json:"password"`
	DB          int           `json:"db"`
	MaxRetries  int           `json:"max_retries"`
	PoolSize    int           `json:"pool_size"`
	PoolTimeout time.Duration `json:"pool_timeout"`
	IdleTimeout time.Duration `json:"idle_timeout"`
	DefaultTTL  time.Duration `json:"default_ttl"`
	Enabled     bool          `json:"enabled"`
}

// PostgreSQLConfig contains PostgreSQL-specific configuration.
type PostgreSQLConfig struct {
	URL             string        `json:"url"`
	MaxConnections  int           `json:"max_connections"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
	SSLMode         string        `json:"ssl_mode"`
}

// TMDBConfig contains TMDB API configuration.
type TMDBConfig struct {
	APIKey      string        `json:"api_key"`
	BaseURL     string        `json:"base_url"`
	RateLimit   int           `json:"rate_limit"`      // requests per period
	RatePeriod  time.Duration `json:"rate_period"`
	Timeout     time.Duration `json:"timeout"`
	MaxRetries  int           `json:"max_retries"`
	BackoffBase time.Duration `json:"backoff_base"`
}

// HTTPConfig contains HTTP client configuration.
type HTTPConfig struct {
	MaxIdleConns        int           `json:"max_idle_conns"`
	MaxIdleConnsPerHost int           `json:"max_idle_conns_per_host"`
	MaxConnsPerHost     int           `json:"max_conns_per_host"`
	IdleConnTimeout     time.Duration `json:"idle_conn_timeout"`
	TLSHandshakeTimeout time.Duration `json:"tls_handshake_timeout"`
	ResponseHeaderTimeout time.Duration `json:"response_header_timeout"`
	RequestTimeout      time.Duration `json:"request_timeout"`
}

// MonitoringConfig contains monitoring configuration.
type MonitoringConfig struct {
	MetricsInterval   time.Duration `json:"metrics_interval"`
	HistoryRetention  time.Duration `json:"history_retention"`
	AlertingEnabled   bool          `json:"alerting_enabled"`
	DashboardEnabled  bool          `json:"dashboard_enabled"`
	DashboardPort     int           `json:"dashboard_port"`
	MetricsPort       int           `json:"metrics_port"`
}

// LoadTestingConfig contains load testing configuration.
type LoadTestingConfig struct {
	MaxConcurrentUsers int           `json:"max_concurrent_users"`
	DefaultDuration    time.Duration `json:"default_duration"`
	RampUpDuration     time.Duration `json:"ramp_up_duration"`
	MetricsInterval    time.Duration `json:"metrics_interval"`
	ResultsRetention   time.Duration `json:"results_retention"`
}

// ConfigValidation contains configuration validation results.
type ConfigValidation struct {
	Environment Environment `json:"environment"`
	Valid       bool        `json:"valid"`
	Errors      []string    `json:"errors"`
	Warnings    []string    `json:"warnings"`
	CheckedAt   time.Time   `json:"checked_at"`
}

// ConnectivityCheck contains connectivity verification results.
type ConnectivityCheck struct {
	Redis       ConnectivityResult `json:"redis"`
	PostgreSQL  ConnectivityResult `json:"postgresql"`
	TMDB        ConnectivityResult `json:"tmdb"`
	External    map[string]ConnectivityResult `json:"external"`
	OverallStatus HealthStatus     `json:"overall_status"`
	CheckedAt   time.Time          `json:"checked_at"`
}

// ConnectivityResult contains the result of a single connectivity check.
type ConnectivityResult struct {
	Service     string        `json:"service"`
	Status      HealthStatus  `json:"status"`
	ResponseTime time.Duration `json:"response_time"`
	Error       string        `json:"error,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}