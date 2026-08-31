// Package main provides the entry point for WaveFlix Hub Scalability Testing & Monitoring system.
//
// This system validates and monitors the scalability improvements implemented through
// PostgreSQL migration, Redis caching, TMDB API optimization, HTTP connection pooling,
// and rate limiting components.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hamin-baek/waveflix-hub/scalability-testing/internal/config"
	"github.com/hamin-baek/waveflix-hub/scalability-testing/internal/integration"
	"github.com/hamin-baek/waveflix-hub/scalability-testing/internal/interfaces"
	"github.com/hamin-baek/waveflix-hub/scalability-testing/internal/loadtest"
	"github.com/hamin-baek/waveflix-hub/scalability-testing/internal/monitoring"
)

// Application holds the main components of the scalability testing system.
type Application struct {
	config              *config.Manager
	integrationValidator *integration.Validator
	loadTester          *loadtest.Engine
	performanceMonitor  *monitoring.PerformanceMonitor
	logger              *Logger
	shutdownCh          chan os.Signal
	wg                  sync.WaitGroup
}

// NewApplication creates a new scalability testing application instance.
func NewApplication() (*Application, error) {
	logger := NewLogger()

	// Initialize configuration manager
	configManager, err := config.NewManager()
	if err != nil {
		return nil, err
	}

	// Initialize components
	integrationValidator, err := integration.NewValidator(configManager)
	if err != nil {
		return nil, err
	}

	loadTester, err := loadtest.NewEngine(configManager)
	if err != nil {
		return nil, err
	}

	performanceMonitor, err := monitoring.NewPerformanceMonitor(configManager)
	if err != nil {
		return nil, err
	}

	app := &Application{
		config:              configManager,
		integrationValidator: integrationValidator,
		loadTester:          loadTester,
		performanceMonitor:  performanceMonitor,
		logger:              logger,
		shutdownCh:          make(chan os.Signal, 1),
	}

	// Setup signal handling
	signal.Notify(app.shutdownCh, os.Interrupt, syscall.SIGTERM)

	return app, nil
}

// Run starts the scalability testing and monitoring system.
func (app *Application) Run(ctx context.Context) error {
	app.logger.Info("Starting WaveFlix Hub Scalability Testing & Monitoring System")

	// Start performance monitoring in background
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		if err := app.performanceMonitor.Start(ctx); err != nil {
			app.logger.Error("Performance monitor error: %v", err)
		}
	}()

	// Wait for shutdown signal or context cancellation
	select {
	case <-app.shutdownCh:
		app.logger.Info("Received shutdown signal")
	case <-ctx.Done():
		app.logger.Info("Context cancelled")
	}

	return app.shutdown()
}

// RunIntegrationTests executes comprehensive integration validation.
func (app *Application) RunIntegrationTests(ctx context.Context) error {
	app.logger.Info("Starting integration tests")

	results, err := app.integrationValidator.RunAllTests(ctx)
	if err != nil {
		return err
	}

	app.logger.Info("Integration test results:")
	for component, result := range results {
		if result.Success {
			app.logger.Info("✓ %s: PASS (duration: %v)", component, result.Duration)
		} else {
			app.logger.Error("✗ %s: FAIL (duration: %v)", component, result.Duration)
			for _, diagnostic := range result.Diagnostics {
				app.logger.Error("  - %s", diagnostic)
			}
		}
	}

	return nil
}

// RunLoadTests executes load testing scenarios.
func (app *Application) RunLoadTests(ctx context.Context, scenario *interfaces.LoadTestScenario) error {
	app.logger.Info("Starting load test: %s", scenario.Name)

	results, err := app.loadTester.ExecuteLoadTest(ctx, scenario)
	if err != nil {
		return err
	}

	app.logger.Info("Load test completed:")
	app.logger.Info("  Duration: %v", results.Duration)
	app.logger.Info("  Total Requests: %d", results.TotalRequests)
	app.logger.Info("  Success Rate: %.2f%%", float64(results.SuccessfulRequests)/float64(results.TotalRequests)*100)
	app.logger.Info("  Average Latency: %v", results.AverageLatency)
	app.logger.Info("  P95 Latency: %v", results.P95Latency)
	app.logger.Info("  Throughput: %.2f RPS", results.ThroughputRPS)

	return nil
}

// GetSystemMetrics returns current system performance metrics.
func (app *Application) GetSystemMetrics() (*interfaces.SystemMetrics, error) {
	return app.performanceMonitor.GetCurrentMetrics()
}

// shutdown gracefully shuts down all components.
func (app *Application) shutdown() error {
	app.logger.Info("Shutting down scalability testing system")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown components
	if err := app.performanceMonitor.Shutdown(shutdownCtx); err != nil {
		app.logger.Error("Error shutting down performance monitor: %v", err)
	}

	if err := app.loadTester.Shutdown(shutdownCtx); err != nil {
		app.logger.Error("Error shutting down load tester: %v", err)
	}

	if err := app.integrationValidator.Shutdown(shutdownCtx); err != nil {
		app.logger.Error("Error shutting down integration validator: %v", err)
	}

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		app.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		app.logger.Info("All components shutdown successfully")
	case <-shutdownCtx.Done():
		app.logger.Error("Shutdown timeout exceeded")
	}

	return nil
}

func main() {
	// Create CLI
	cli, err := NewCLI()
	if err != nil {
		log.Fatalf("Failed to create CLI: %v", err)
	}

	// Run CLI
	if err := cli.Run(); err != nil {
		log.Fatalf("CLI error: %v", err)
	}
}

// CLI provides command-line interface for the scalability testing system.
type CLI struct {
	app *Application
}

// NewCLI creates a new CLI instance.
func NewCLI() (*CLI, error) {
	app, err := NewApplication()
	if err != nil {
		return nil, err
	}

	return &CLI{app: app}, nil
}

// Run executes the CLI with command-line arguments.
func (cli *CLI) Run() error {
	var (
		mode     = flag.String("mode", "help", "Operation mode: integration, loadtest, monitor, validate, help")
		scenario = flag.String("scenario", "basic", "Load test scenario name")
		duration = flag.Duration("duration", time.Minute*5, "Test duration")
		users    = flag.Int("users", 10, "Concurrent users for load test")
	)
	flag.Parse()

	ctx := context.Background()

	switch *mode {
	case "integration":
		return cli.runIntegrationTests(ctx)
	case "loadtest":
		return cli.runLoadTest(ctx, *scenario, *duration, *users)
	case "monitor":
		return cli.runMonitoring(ctx)
	case "validate":
		return cli.runValidation(ctx)
	case "help", "":
		return cli.showHelp()
	default:
		fmt.Printf("Unknown mode: %s\n", *mode)
		return cli.showHelp()
	}
}

// runIntegrationTests executes integration validation tests.
func (cli *CLI) runIntegrationTests(ctx context.Context) error {
	fmt.Println("Starting WaveFlix Hub Scalability Integration Tests...")
	fmt.Println("==================================================")

	if err := cli.app.RunIntegrationTests(ctx); err != nil {
		return fmt.Errorf("integration tests failed: %w", err)
	}

	fmt.Println("==================================================")
	fmt.Println("Integration tests completed successfully!")
	return nil
}

// runLoadTest executes load testing scenarios.
func (cli *CLI) runLoadTest(ctx context.Context, scenarioName string, duration time.Duration, users int) error {
	fmt.Printf("Starting Load Test: %s\n", scenarioName)
	fmt.Printf("Duration: %v, Concurrent Users: %d\n", duration, users)
	fmt.Println("==================================================")

	scenario := &interfaces.LoadTestScenario{
		Name:            scenarioName,
		ConcurrentUsers: users,
		Duration:        duration,
		RampUpTime:      duration / 10, // 10% of test duration for ramp-up
		RequestPatterns: cli.getDefaultRequestPatterns(),
		Environment:     cli.app.config.GetCurrentEnvironment(),
	}

	if err := cli.app.RunLoadTests(ctx, scenario); err != nil {
		return fmt.Errorf("load test failed: %w", err)
	}

	fmt.Println("==================================================")
	fmt.Println("Load test completed successfully!")
	return nil
}

// runMonitoring starts continuous performance monitoring.
func (cli *CLI) runMonitoring(ctx context.Context) error {
	fmt.Println("Starting WaveFlix Hub Performance Monitoring...")
	fmt.Printf("Environment: %s\n", cli.app.config.GetCurrentEnvironment())
	fmt.Println("==================================================")

	// Show initial metrics
	metrics, err := cli.app.GetSystemMetrics()
	if err != nil {
		return fmt.Errorf("failed to get system metrics: %w", err)
	}

	cli.displayMetrics(metrics)

	fmt.Println("\nStarting continuous monitoring (Ctrl+C to stop)...")
	return cli.app.Run(ctx)
}

// runValidation validates system configuration.
func (cli *CLI) runValidation(ctx context.Context) error {
	fmt.Println("Validating WaveFlix Hub Scalability Configuration...")
	fmt.Println("==================================================")

	env := cli.app.config.GetCurrentEnvironment()
	validation, err := cli.app.config.ValidateConfiguration(env)
	if err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	fmt.Printf("Environment: %s\n", validation.Environment)
	fmt.Printf("Valid: %t\n", validation.Valid)

	if len(validation.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range validation.Errors {
			fmt.Printf("  ❌ %s\n", err)
		}
	}

	if len(validation.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, warning := range validation.Warnings {
			fmt.Printf("  ⚠️  %s\n", warning)
		}
	}

	if validation.Valid {
		fmt.Println("\n✅ Configuration is valid!")
	} else {
		fmt.Println("\n❌ Configuration has errors!")
		return fmt.Errorf("configuration validation failed")
	}

	// Test connectivity
	fmt.Println("\nTesting connectivity...")
	connectivity, err := cli.app.config.ValidateConnectivity(ctx)
	if err != nil {
		return fmt.Errorf("connectivity check failed: %w", err)
	}

	fmt.Printf("Overall Status: %s\n", connectivity.OverallStatus)
	fmt.Printf("Redis: %s (%v)\n", connectivity.Redis.Status, connectivity.Redis.ResponseTime)
	fmt.Printf("PostgreSQL: %s (%v)\n", connectivity.PostgreSQL.Status, connectivity.PostgreSQL.ResponseTime)
	fmt.Printf("TMDB: %s (%v)\n", connectivity.TMDB.Status, connectivity.TMDB.ResponseTime)

	return nil
}

// showHelp displays usage information.
func (cli *CLI) showHelp() error {
	fmt.Println("WaveFlix Hub Scalability Testing & Monitoring System")
	fmt.Println("==================================================")
	fmt.Println("Usage: scalability-testing [options]")
	fmt.Println()
	fmt.Println("Modes:")
	fmt.Println("  --mode integration  Run integration tests for all components")
	fmt.Println("  --mode loadtest     Execute load testing scenarios")
	fmt.Println("  --mode monitor      Start continuous performance monitoring")
	fmt.Println("  --mode validate     Validate configuration and connectivity")
	fmt.Println("  --mode help         Show this help message")
	fmt.Println()
	fmt.Println("Load Test Options:")
	fmt.Println("  --scenario NAME     Load test scenario (default: basic)")
	fmt.Println("  --duration TIME     Test duration (default: 5m)")
	fmt.Println("  --users COUNT       Concurrent users (default: 10)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  scalability-testing --mode integration")
	fmt.Println("  scalability-testing --mode loadtest --users 100 --duration 10m")
	fmt.Println("  scalability-testing --mode monitor")
	fmt.Println("  scalability-testing --mode validate")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  ENVIRONMENT         deployment environment (development, staging, production)")
	fmt.Println("  TMDB_API_KEY        TMDB API key for integration tests")
	fmt.Println("  DATABASE_URL        PostgreSQL connection string")
	fmt.Println("  REDIS_URL           Redis connection string")
	fmt.Println("  LOG_LEVEL           logging level (debug, info, warn, error)")

	return nil
}

// displayMetrics shows current system metrics in a formatted way.
func (cli *CLI) displayMetrics(metrics *interfaces.SystemMetrics) {
	fmt.Printf("System Metrics (Updated: %s)\n", metrics.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println()

	if metrics.RedisMetrics != nil {
		fmt.Printf("Redis Cache:\n")
		fmt.Printf("  Hit Rate: %.1f%%\n", metrics.RedisMetrics.HitRate*100)
		fmt.Printf("  Response Time: %v\n", metrics.RedisMetrics.ResponseTime)
		fmt.Printf("  Active Connections: %d\n", metrics.RedisMetrics.ConnectionsActive)
		fmt.Printf("  Memory Usage: %.1f MB\n", float64(metrics.RedisMetrics.MemoryUsage)/1024/1024)
		fmt.Println()
	}

	if metrics.TMDBMetrics != nil {
		fmt.Printf("TMDB API:\n")
		fmt.Printf("  Request Rate: %.1f RPS\n", metrics.TMDBMetrics.RequestRate)
		fmt.Printf("  Response Time: %v\n", metrics.TMDBMetrics.ResponseTime)
		fmt.Printf("  Rate Limit Usage: %.1f%%\n", metrics.TMDBMetrics.RateLimitUtilization*100)
		fmt.Printf("  Circuit Breaker: %s\n", metrics.TMDBMetrics.CircuitBreakerState)
		fmt.Printf("  Cache Hit Rate: %.1f%%\n", metrics.TMDBMetrics.CacheHitRate*100)
		fmt.Println()
	}

	if metrics.HTTPPoolMetrics != nil {
		fmt.Printf("HTTP Connection Pool:\n")
		fmt.Printf("  Active/Idle Connections: %d/%d\n", metrics.HTTPPoolMetrics.ActiveConnections, metrics.HTTPPoolMetrics.IdleConnections)
		fmt.Printf("  Connection Reuse: %.1f%%\n", metrics.HTTPPoolMetrics.ConnectionReuse*100)
		fmt.Printf("  Average Latency: %v\n", metrics.HTTPPoolMetrics.AverageLatency)
		fmt.Printf("  Throughput: %.1f RPS\n", metrics.HTTPPoolMetrics.ThroughputRPS)
		fmt.Println()
	}

	if metrics.DatabaseMetrics != nil {
		fmt.Printf("PostgreSQL Database:\n")
		fmt.Printf("  Active/Idle Connections: %d/%d\n", metrics.DatabaseMetrics.ActiveConnections, metrics.DatabaseMetrics.IdleConnections)
		fmt.Printf("  Query Time: %v\n", metrics.DatabaseMetrics.QueryTime)
		fmt.Printf("  Queries per Second: %.1f\n", metrics.DatabaseMetrics.QueriesPerSecond)
		fmt.Printf("  Cache Hit Ratio: %.1f%%\n", metrics.DatabaseMetrics.CacheHitRatio*100)
		fmt.Println()
	}

	if metrics.SystemResources != nil {
		fmt.Printf("System Resources:\n")
		fmt.Printf("  CPU Usage: %.1f%%\n", metrics.SystemResources.CPUUsage)
		fmt.Printf("  Memory Usage: %.1f%% (%.1f MB / %.1f MB)\n", 
			metrics.SystemResources.MemoryPercent,
			float64(metrics.SystemResources.MemoryUsage)/1024/1024,
			float64(metrics.SystemResources.MemoryTotal)/1024/1024)
		fmt.Printf("  Goroutines: %d\n", metrics.SystemResources.Goroutines)
		fmt.Println()
	}
}

// getDefaultRequestPatterns returns default request patterns for load testing.
func (cli *CLI) getDefaultRequestPatterns() []*interfaces.RequestPattern {
	return []*interfaces.RequestPattern{
		{
			Name:       "browse_movies",
			Method:     "GET",
			Path:       "/api/movies/trending",
			Weight:     0.4,
			ThinkTime:  time.Second * 2,
			Parameters: map[string]interface{}{"language": "en-US"},
		},
		{
			Name:       "search_movies",
			Method:     "GET",
			Path:       "/api/movies/search",
			Weight:     0.3,
			ThinkTime:  time.Second * 3,
			Parameters: map[string]interface{}{"query": "action"},
		},
		{
			Name:       "movie_details",
			Method:     "GET",
			Path:       "/api/movies/{id}",
			Weight:     0.2,
			ThinkTime:  time.Second * 1,
			Parameters: map[string]interface{}{"append_to_response": "credits,videos"},
		},
		{
			Name:       "watch_providers",
			Method:     "GET",
			Path:       "/api/movies/{id}/watch/providers",
			Weight:     0.1,
			ThinkTime:  time.Millisecond * 500,
			Parameters: map[string]interface{}{},
		},
	}
}