// Package config provides integration tests showing backward compatibility with WaveFlix Hub components.
package config

import (
	"os"
	"testing"
	"time"

	"github.com/hamin-baek/waveflix-hub/scalability-testing/internal/interfaces"
)

// TestWaveFlixHubIntegration demonstrates how the enhanced configuration system
// integrates with existing WaveFlix Hub components like redis_cache.go and tmdb_client.go
func TestWaveFlixHubIntegration(t *testing.T) {
	// Initialize configuration manager
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create configuration manager: %v", err)
	}

	// Test Redis cache configuration compatibility
	t.Run("Redis Cache Integration", func(t *testing.T) {
		config := manager.GetCurrentConfig()
		if config.Redis == nil {
			t.Fatal("Redis configuration should not be nil")
		}

		// Verify Redis configuration has all fields needed by redis_cache.go
		redisConfig := config.Redis
		
		// These fields are used by the existing redis_cache.go
		if redisConfig.URL == "" {
			t.Error("Redis URL should not be empty")
		}
		if redisConfig.PoolSize <= 0 {
			t.Error("Redis pool size should be positive")
		}
		if redisConfig.DefaultTTL <= 0 {
			t.Error("Redis default TTL should be positive")
		}

		// Test backward compatibility mapping
		legacy := manager.GetBackwardCompatibleConfig()
		if legacy.Redis.URL != redisConfig.URL {
			t.Error("Legacy Redis URL mapping failed")
		}
		if legacy.Redis.PoolSize != redisConfig.PoolSize {
			t.Error("Legacy Redis pool size mapping failed")
		}
		if legacy.Redis.DefaultTTL != redisConfig.DefaultTTL {
			t.Error("Legacy Redis TTL mapping failed")
		}

		t.Logf("Redis configuration: URL=%s, PoolSize=%d, TTL=%v", 
			redisConfig.URL, redisConfig.PoolSize, redisConfig.DefaultTTL)
	})

	// Test TMDB client configuration compatibility
	t.Run("TMDB Client Integration", func(t *testing.T) {
		config := manager.GetCurrentConfig()
		if config.TMDB == nil {
			t.Fatal("TMDB configuration should not be nil")
		}

		// Verify TMDB configuration has all fields needed by tmdb_client.go
		tmdbConfig := config.TMDB
		
		if tmdbConfig.BaseURL == "" {
			t.Error("TMDB base URL should not be empty")
		}
		if tmdbConfig.RateLimit <= 0 {
			t.Error("TMDB rate limit should be positive")
		}
		if tmdbConfig.Timeout <= 0 {
			t.Error("TMDB timeout should be positive")
		}

		// Test backward compatibility mapping
		legacy := manager.GetBackwardCompatibleConfig()
		if legacy.TMDB.BaseURL != tmdbConfig.BaseURL {
			t.Error("Legacy TMDB base URL mapping failed")
		}
		if legacy.TMDB.RateLimit != tmdbConfig.RateLimit {
			t.Error("Legacy TMDB rate limit mapping failed")
		}

		t.Logf("TMDB configuration: BaseURL=%s, RateLimit=%d, Timeout=%v", 
			tmdbConfig.BaseURL, tmdbConfig.RateLimit, tmdbConfig.Timeout)
	})

	// Test HTTP client configuration compatibility
	t.Run("HTTP Client Integration", func(t *testing.T) {
		config := manager.GetCurrentConfig()
		if config.HTTP == nil {
			t.Fatal("HTTP configuration should not be nil")
		}

		// Verify HTTP configuration has all fields needed for optimized HTTP clients
		httpConfig := config.HTTP
		
		if httpConfig.MaxIdleConns <= 0 {
			t.Error("HTTP max idle connections should be positive")
		}
		if httpConfig.RequestTimeout <= 0 {
			t.Error("HTTP request timeout should be positive")
		}

		// Test backward compatibility mapping
		legacy := manager.GetBackwardCompatibleConfig()
		if legacy.HTTP.MaxIdleConns != httpConfig.MaxIdleConns {
			t.Error("Legacy HTTP max idle connections mapping failed")
		}
		if legacy.HTTP.RequestTimeout != httpConfig.RequestTimeout {
			t.Error("Legacy HTTP request timeout mapping failed")
		}

		t.Logf("HTTP configuration: MaxIdleConns=%d, RequestTimeout=%v", 
			httpConfig.MaxIdleConns, httpConfig.RequestTimeout)
	})
}

// TestEnvironmentSpecificIntegration tests environment-specific configuration behavior
func TestEnvironmentSpecificIntegration(t *testing.T) {
	environments := []interfaces.Environment{
		interfaces.EnvDevelopment,
		interfaces.EnvStaging, 
		interfaces.EnvProduction,
		interfaces.EnvTesting,
	}

	for _, env := range environments {
		t.Run(string(env), func(t *testing.T) {
			manager := &Manager{validator: NewValidator()}
			
			config, err := manager.LoadEnvironmentConfig(env)
			if err != nil {
				t.Fatalf("Failed to load config for %s: %v", env, err)
			}

			// Test environment-specific Redis settings
			redisConfig := config.Redis
			switch env {
			case interfaces.EnvProduction:
				// Production should have higher pool sizes and stricter settings
				if redisConfig.PoolSize < 20 {
					t.Errorf("Production Redis pool size too small: %d", redisConfig.PoolSize)
				}
			case interfaces.EnvDevelopment:
				// Development can have smaller pool sizes
				if redisConfig.PoolSize > 25 {
					t.Logf("Development Redis pool size: %d (reasonable)", redisConfig.PoolSize)
				}
			case interfaces.EnvTesting:
				// Testing should have minimal resources
				if redisConfig.PoolSize > 15 {
					t.Logf("Testing Redis pool size: %d (could be smaller)", redisConfig.PoolSize)
				}
			}

			// Test environment-specific TMDB rate limits
			tmdbConfig := config.TMDB
			switch env {
			case interfaces.EnvProduction:
				// Production should be conservative with rate limits
				if tmdbConfig.RateLimit > 35 {
					t.Errorf("Production TMDB rate limit too high: %d", tmdbConfig.RateLimit)
				}
			case interfaces.EnvTesting:
				// Testing can have lower limits
				if tmdbConfig.RateLimit > 25 {
					t.Logf("Testing TMDB rate limit: %d (appropriate)", tmdbConfig.RateLimit)
				}
			}

			// Test environment-specific monitoring settings
			monitoringConfig := config.Monitoring
			switch env {
			case interfaces.EnvProduction:
				// Production should have monitoring enabled
				if !monitoringConfig.AlertingEnabled {
					t.Error("Production should have alerting enabled")
				}
			case interfaces.EnvTesting:
				// Testing might not need alerting
				if monitoringConfig.AlertingEnabled {
					t.Logf("Testing has alerting enabled (may not be necessary)")
				}
			}

			t.Logf("%s environment configured successfully", env)
		})
	}
}

// TestConfigurationOverrides tests how environment variables override configuration files
func TestConfigurationOverrides(t *testing.T) {
	// Set environment variables that should override file configuration
	os.Setenv("REDIS_URL", "override.redis.com:6379")
	os.Setenv("TMDB_API_KEY", "override_api_key")
	os.Setenv("DATABASE_URL", "postgres://override:5432/db")
	defer func() {
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("TMDB_API_KEY") 
		os.Unsetenv("DATABASE_URL")
	}()

	manager := &Manager{}
	config, err := manager.LoadEnvironmentConfig(interfaces.EnvDevelopment)
	if err != nil {
		t.Fatalf("Failed to load config with overrides: %v", err)
	}

	// Verify environment variables override file settings
	if config.Redis.URL != "override.redis.com:6379" {
		t.Errorf("Redis URL not overridden by env var: got %s", config.Redis.URL)
	}
	if config.TMDB.APIKey != "override_api_key" {
		t.Errorf("TMDB API key not overridden by env var: got %s", config.TMDB.APIKey)
	}
	if config.PostgreSQL.URL != "postgres://override:5432/db" {
		t.Errorf("PostgreSQL URL not overridden by env var: got %s", config.PostgreSQL.URL)
	}

	t.Log("Environment variable overrides working correctly")
}

// TestConfigurationValidationIntegration tests validation specific to WaveFlix Hub requirements
func TestConfigurationValidationIntegration(t *testing.T) {
	manager := &Manager{validator: NewValidator()}

	// Test production-specific validation requirements
	t.Run("Production Validation", func(t *testing.T) {
		// Set minimal environment for production validation
		os.Setenv("TMDB_API_KEY", "a1b2c3d4e5f6789012345678901234567890abcd")
		os.Setenv("REDIS_ENABLED", "true")
		defer func() {
			os.Unsetenv("TMDB_API_KEY")
			os.Unsetenv("REDIS_ENABLED")
		}()

		validation, err := manager.ValidateConfiguration(interfaces.EnvProduction)
		if err != nil {
			t.Fatalf("Production validation failed: %v", err)
		}

		// Production should have specific requirements
		for _, errMsg := range validation.Errors {
			t.Logf("Production validation error: %s", errMsg)
		}
		for _, warnMsg := range validation.Warnings {
			t.Logf("Production validation warning: %s", warnMsg)
		}

		t.Logf("Production validation completed with %d errors, %d warnings", 
			len(validation.Errors), len(validation.Warnings))
	})

	// Test development flexibility
	t.Run("Development Validation", func(t *testing.T) {
		validation, err := manager.ValidateConfiguration(interfaces.EnvDevelopment) 
		if err != nil {
			t.Fatalf("Development validation failed: %v", err)
		}

		// Development should be more flexible (warnings instead of errors)
		hasWarnings := len(validation.Warnings) > 0
		if !hasWarnings {
			t.Log("Development validation has no warnings (good configuration)")
		} else {
			t.Logf("Development validation has %d warnings (expected)", len(validation.Warnings))
		}
	})
}

// TestPerformanceConfiguration tests performance-related configuration aspects
func TestPerformanceConfiguration(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	config := manager.GetCurrentConfig()

	// Test Redis performance settings
	t.Run("Redis Performance", func(t *testing.T) {
		redis := config.Redis
		
		// Pool timeout should be reasonable for performance
		if redis.PoolTimeout > time.Minute {
			t.Errorf("Redis pool timeout too high: %v", redis.PoolTimeout)
		}
		
		// Idle timeout should prevent resource leaks
		if redis.IdleTimeout < time.Minute {
			t.Errorf("Redis idle timeout too low: %v", redis.IdleTimeout)  
		}

		// TTL should balance freshness with performance
		if redis.DefaultTTL > time.Hour {
			t.Errorf("Redis default TTL too high: %v", redis.DefaultTTL)
		}

		t.Logf("Redis performance settings: PoolTimeout=%v, IdleTimeout=%v, TTL=%v",
			redis.PoolTimeout, redis.IdleTimeout, redis.DefaultTTL)
	})

	// Test HTTP performance settings
	t.Run("HTTP Performance", func(t *testing.T) {
		http := config.HTTP

		// Connection limits should enable concurrency
		if http.MaxIdleConns < 10 {
			t.Errorf("HTTP max idle connections too low: %d", http.MaxIdleConns)
		}

		// Timeouts should be reasonable for external APIs
		if http.RequestTimeout > time.Minute {
			t.Errorf("HTTP request timeout too high: %v", http.RequestTimeout)
		}

		t.Logf("HTTP performance settings: MaxIdleConns=%d, RequestTimeout=%v",
			http.MaxIdleConns, http.RequestTimeout)
	})

	// Test monitoring performance impact
	t.Run("Monitoring Performance", func(t *testing.T) {
		monitoring := config.Monitoring

		// Metrics interval should balance accuracy with overhead
		if monitoring.MetricsInterval < time.Second*5 {
			t.Errorf("Monitoring metrics interval too frequent: %v", monitoring.MetricsInterval)
		}
		if monitoring.MetricsInterval > time.Minute*5 {
			t.Errorf("Monitoring metrics interval too sparse: %v", monitoring.MetricsInterval)
		}

		t.Logf("Monitoring performance settings: MetricsInterval=%v", monitoring.MetricsInterval)
	})
}

// BenchmarkConfigurationLoading benchmarks configuration loading performance
func BenchmarkConfigurationLoading(b *testing.B) {
	environments := []interfaces.Environment{
		interfaces.EnvDevelopment,
		interfaces.EnvStaging,
		interfaces.EnvProduction,
		interfaces.EnvTesting,
	}

	for _, env := range environments {
		b.Run(string(env), func(b *testing.B) {
			manager := &Manager{}
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				config, err := manager.LoadEnvironmentConfig(env)
				if err != nil {
					b.Fatalf("Failed to load config: %v", err)
				}
				_ = config
			}
		})
	}
}