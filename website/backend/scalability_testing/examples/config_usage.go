// Package main demonstrates how to use the enhanced configuration system with WaveFlix Hub components.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hamin-baek/waveflix-hub/scalability-testing/internal/config"
)

// MockRedisCache represents the existing redis_cache.go component
type MockRedisCache struct {
	URL         string
	Password    string
	PoolSize    int
	DefaultTTL  time.Duration
	Enabled     bool
}

// MockTMDBClient represents the existing tmdb_client.go component
type MockTMDBClient struct {
	APIKey      string
	BaseURL     string
	RateLimit   int
	Timeout     time.Duration
}

// MockHTTPClient represents HTTP connection pool configuration
type MockHTTPClient struct {
	MaxIdleConns    int
	RequestTimeout  time.Duration
}

func main() {
	fmt.Println("WaveFlix Hub Configuration Integration Example")
	fmt.Println("===========================================")

	// Initialize configuration manager
	fmt.Println("\n1. Initializing configuration manager...")
	manager, err := config.NewManager()
	if err != nil {
		log.Fatalf("Failed to initialize configuration manager: %v", err)
	}

	// Display detected environment
	env := manager.GetCurrentEnvironment()
	fmt.Printf("✓ Detected environment: %s\n", env)

	// Validate configuration
	fmt.Println("\n2. Validating configuration...")
	validation, err := manager.ValidateConfiguration(env)
	if err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	fmt.Printf("✓ Configuration valid: %v\n", validation.Valid)
	if len(validation.Errors) > 0 {
		fmt.Printf("⚠ Validation errors: %d\n", len(validation.Errors))
		for _, errMsg := range validation.Errors {
			fmt.Printf("  - %s\n", errMsg)
		}
	}
	if len(validation.Warnings) > 0 {
		fmt.Printf("⚠ Validation warnings: %d\n", len(validation.Warnings))
		for _, warnMsg := range validation.Warnings {
			fmt.Printf("  - %s\n", warnMsg)
		}
	}

	// Get configuration
	fmt.Println("\n3. Loading component configurations...")
	cfg := manager.GetCurrentConfig()

	// Initialize Redis cache (existing component integration)
	if cfg.Redis != nil {
		redisCache := &MockRedisCache{
			URL:         cfg.Redis.URL,
			Password:    cfg.Redis.Password,
			PoolSize:    cfg.Redis.PoolSize,
			DefaultTTL:  cfg.Redis.DefaultTTL,
			Enabled:     cfg.Redis.Enabled,
		}
		fmt.Printf("✓ Redis Cache configured: URL=%s, PoolSize=%d, TTL=%v\n", 
			redisCache.URL, redisCache.PoolSize, redisCache.DefaultTTL)
	}

	// Initialize TMDB client (existing component integration)
	if cfg.TMDB != nil {
		tmdbClient := &MockTMDBClient{
			APIKey:    cfg.TMDB.APIKey,
			BaseURL:   cfg.TMDB.BaseURL,
			RateLimit: cfg.TMDB.RateLimit,
			Timeout:   cfg.TMDB.Timeout,
		}
		apiKeyDisplay := cfg.TMDB.APIKey
		if len(apiKeyDisplay) > 8 {
			apiKeyDisplay = apiKeyDisplay[:8] + "..."
		}
		fmt.Printf("✓ TMDB Client configured: BaseURL=%s, RateLimit=%d, APIKey=%s\n", 
			tmdbClient.BaseURL, tmdbClient.RateLimit, apiKeyDisplay)
	}

	// Initialize HTTP client (connection pooling)
	if cfg.HTTP != nil {
		httpClient := &MockHTTPClient{
			MaxIdleConns:   cfg.HTTP.MaxIdleConns,
			RequestTimeout: cfg.HTTP.RequestTimeout,
		}
		fmt.Printf("✓ HTTP Client configured: MaxIdleConns=%d, Timeout=%v\n", 
			httpClient.MaxIdleConns, httpClient.RequestTimeout)
	}

	// Test connectivity
	fmt.Println("\n4. Testing connectivity...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connectivityCheck, err := manager.ValidateConnectivity(ctx)
	if err != nil {
		fmt.Printf("⚠ Connectivity check failed: %v\n", err)
	} else {
		fmt.Printf("✓ Overall connectivity status: %s\n", connectivityCheck.OverallStatus)
		fmt.Printf("  - Redis: %s (%v)\n", 
			connectivityCheck.Redis.Status, connectivityCheck.Redis.ResponseTime)
		fmt.Printf("  - PostgreSQL: %s (%v)\n", 
			connectivityCheck.PostgreSQL.Status, connectivityCheck.PostgreSQL.ResponseTime)
		fmt.Printf("  - TMDB: %s (%v)\n", 
			connectivityCheck.TMDB.Status, connectivityCheck.TMDB.ResponseTime)
	}

	// Demonstrate backward compatibility
	fmt.Println("\n5. Backward compatibility example...")
	legacyConfig := manager.GetBackwardCompatibleConfig()
	fmt.Printf("✓ Legacy Redis URL: %s\n", legacyConfig.Redis.URL)
	fmt.Printf("✓ Legacy TMDB BaseURL: %s\n", legacyConfig.TMDB.BaseURL)
	fmt.Printf("✓ Legacy HTTP MaxIdleConns: %d\n", legacyConfig.HTTP.MaxIdleConns)

	// Demonstrate configuration for different environments
	fmt.Println("\n6. Environment-specific configuration examples...")
	demonstrateEnvironmentConfigs(manager)

	// Show monitoring configuration
	fmt.Println("\n7. Monitoring and load testing configuration...")
	if cfg.Monitoring != nil {
		fmt.Printf("✓ Monitoring: MetricsInterval=%v, DashboardPort=%d\n", 
			cfg.Monitoring.MetricsInterval, cfg.Monitoring.DashboardPort)
	}
	if cfg.LoadTesting != nil {
		fmt.Printf("✓ Load Testing: MaxUsers=%d, Duration=%v\n", 
			cfg.LoadTesting.MaxConcurrentUsers, cfg.LoadTesting.DefaultDuration)
	}

	fmt.Println("\n✓ Configuration integration demonstration complete!")
}

// demonstrateEnvironmentConfigs shows configuration for different environments
func demonstrateEnvironmentConfigs(manager *config.Manager) {
	environments := []struct{
		name string
		env  string
	}{
		{"Development", "development"},
		{"Staging", "staging"}, 
		{"Production", "production"},
		{"Testing", "testing"},
	}

	for _, envInfo := range environments {
		fmt.Printf("\n  %s Environment:\n", envInfo.name)
		
		// For demonstration, we'll show some key differences
		switch envInfo.name {
		case "Development":
			fmt.Println("    - Redis: Optional (fallback to memory)")
			fmt.Println("    - Pool sizes: Small (development workload)")
			fmt.Println("    - Monitoring: Enabled for testing")
			fmt.Println("    - SSL: Optional")
			
		case "Staging":
			fmt.Println("    - Redis: Recommended (realistic testing)")
			fmt.Println("    - Pool sizes: Medium (staging workload)")
			fmt.Println("    - Monitoring: Enabled")
			fmt.Println("    - SSL: Preferred")
			
		case "Production":
			fmt.Println("    - Redis: Required (performance)")
			fmt.Println("    - Pool sizes: Large (production workload)")
			fmt.Println("    - Monitoring: Required with alerting")
			fmt.Println("    - SSL: Required")
			
		case "Testing":
			fmt.Println("    - Redis: Enabled (fast timeouts)")
			fmt.Println("    - Pool sizes: Minimal (test efficiency)")
			fmt.Println("    - Monitoring: Disabled (reduce overhead)")
			fmt.Println("    - SSL: Disabled for speed")
		}
	}
}

// Example of how existing WaveFlix Hub components would be initialized
func initializeWaveFlixComponents() {
	// This demonstrates how the existing components would use the configuration
	
	fmt.Println("\nInitializing WaveFlix Hub components:")
	
	// Redis Cache Manager (from redis_cache.go)
	fmt.Println("✓ Initializing RedisCacheManager with enhanced config...")
	// In real code: cache := NewRedisCacheManager(cfg.Redis)
	
	// TMDB Client (from tmdb_client.go)  
	fmt.Println("✓ Initializing TMDBClient with rate limiting config...")
	// In real code: client := NewTMDBClient(cfg.TMDB)
	
	// HTTP Connection Pool
	fmt.Println("✓ Configuring HTTP connection pooling...")
	// In real code: httpClient := NewOptimizedHTTPClient(cfg.HTTP)
	
	// PostgreSQL Database
	fmt.Println("✓ Setting up PostgreSQL connection pool...")
	// In real code: db := NewDatabasePool(cfg.PostgreSQL)
	
	// Performance Monitor
	fmt.Println("✓ Starting performance monitoring...")
	// In real code: monitor := NewPerformanceMonitor(cfg.Monitoring)
	
	// Load Testing Engine  
	fmt.Println("✓ Configuring load testing engine...")
	// In real code: loadTester := NewLoadTester(cfg.LoadTesting)
}