# WaveFlix Hub Scalability Testing Configuration

This document describes the enhanced configuration system for the WaveFlix Hub scalability testing and monitoring framework.

## Overview

The configuration system provides robust, environment-aware configuration management with the following features:

- **Multi-format support**: YAML and JSON configuration files
- **Environment detection**: Automatic detection of deployment environments
- **Configuration validation**: Comprehensive validation with specific error messages  
- **Backward compatibility**: Integration with existing WaveFlix Hub components
- **Environment-specific defaults**: Optimized settings per environment
- **Configuration sources**: Support for environment variables, files, and defaults

## Quick Start

### Basic Usage

```go
// Initialize configuration manager
manager, err := config.NewManager()
if err != nil {
    log.Fatalf("Failed to initialize config: %v", err)
}

// Get current configuration
config := manager.GetCurrentConfig()

// Use Redis configuration with existing redis_cache.go
redisConfig := config.Redis
cache := NewRedisCacheManager(redisConfig)

// Use TMDB configuration with existing tmdb_client.go
tmdbConfig := config.TMDB
client := NewTMDBClient(tmdbConfig)
```

### Environment Detection

The system automatically detects the deployment environment:

- `ENVIRONMENT` environment variable (development/staging/production/testing)
- CI systems (`CI=true`, `GITHUB_ACTIONS=true`)
- Kubernetes deployment indicators
- Defaults to `development`

## Configuration Structure

### Core Components

```yaml
environment: development

redis:
  url: "localhost:6379"
  password: ""
  db: 0
  max_retries: 3
  pool_size: 20
  pool_timeout: "30s"
  idle_timeout: "300s"
  default_ttl: "900s"
  enabled: true

postgresql:
  url: "postgres://localhost:5432/waveflix_dev"
  max_connections: 25
  max_idle_conns: 5
  conn_max_lifetime: "3600s"
  conn_max_idle_time: "300s"
  ssl_mode: "prefer"

tmdb:
  api_key: ""  # Set via TMDB_API_KEY environment variable
  base_url: "https://api.themoviedb.org/3"
  rate_limit: 30
  rate_period: "10s"
  timeout: "10s"
  max_retries: 3
  backoff_base: "1000ms"

http:
  max_idle_conns: 100
  max_idle_conns_per_host: 25
  max_conns_per_host: 50
  idle_conn_timeout: "90s"
  tls_handshake_timeout: "10s"
  response_header_timeout: "10s"
  request_timeout: "30s"

monitoring:
  metrics_interval: "30s"
  history_retention: "24h"
  alerting_enabled: true
  dashboard_enabled: true
  dashboard_port: 8080
  metrics_port: 9090

load_testing:
  max_concurrent_users: 100
  default_duration: "300s"
  ramp_up_duration: "60s"
  metrics_interval: "5s"
  results_retention: "168h"
```

## Environment-Specific Configuration

### Development
- Smaller connection pools for local development
- Relaxed validation (warnings instead of errors)
- Optional Redis (fallback to memory cache)
- Dashboard and monitoring enabled for testing

### Staging
- Production-like settings with reduced resource usage
- Redis enabled for realistic testing
- Monitoring enabled for performance validation
- SSL preferred but not required

### Production
- Maximum performance and reliability settings
- Redis and TMDB API key required
- SSL required for PostgreSQL
- Alerting and monitoring fully enabled
- Higher connection pools and conservative rate limits

### Testing
- Minimal resource usage
- Fast timeouts for quick test execution
- Monitoring disabled to reduce overhead
- Small connection pools and cache settings

## Configuration Files

### File Locations

The system searches for configuration files in this order:

1. `config/{environment}.yaml`
2. `config/{environment}.yml`
3. `config/{environment}.json`
4. `./{environment}.yaml`
5. `./{environment}.yml`
6. `./{environment}.json`

### File Formats

**YAML Format** (recommended):
```yaml
environment: production
redis:
  url: "redis.example.com:6379"
  pool_size: 50
```

**JSON Format**:
```json
{
  "environment": "production",
  "redis": {
    "url": "redis.example.com:6379",
    "pool_size": 50
  }
}
```

## Environment Variables

All configuration values can be overridden with environment variables:

### Redis Configuration
- `REDIS_URL`: Redis connection string
- `REDIS_PASSWORD`: Redis password
- `REDIS_DB`: Database number (0-15)
- `REDIS_MAX_RETRIES`: Maximum retry attempts
- `REDIS_POOL_SIZE`: Connection pool size
- `REDIS_POOL_TIMEOUT`: Pool timeout duration
- `REDIS_IDLE_TIMEOUT`: Idle connection timeout
- `REDIS_DEFAULT_TTL`: Default cache TTL
- `REDIS_ENABLED`: Enable/disable Redis

### PostgreSQL Configuration
- `DATABASE_URL`: PostgreSQL connection string
- `DB_MAX_CONNECTIONS`: Maximum connections
- `DB_MAX_IDLE_CONNS`: Maximum idle connections
- `DB_CONN_MAX_LIFETIME`: Connection lifetime
- `DB_CONN_MAX_IDLE_TIME`: Maximum idle time
- `DB_SSL_MODE`: SSL mode (disable/prefer/require)

### TMDB API Configuration
- `TMDB_API_KEY`: TMDB API key (required)
- `TMDB_BASE_URL`: API base URL
- `TMDB_RATE_LIMIT`: Requests per period
- `TMDB_RATE_PERIOD`: Rate limiting period
- `TMDB_TIMEOUT`: Request timeout
- `TMDB_MAX_RETRIES`: Maximum retries
- `TMDB_BACKOFF_BASE`: Base backoff delay

### HTTP Configuration
- `HTTP_MAX_IDLE_CONNS`: Maximum idle connections
- `HTTP_MAX_IDLE_CONNS_PER_HOST`: Max idle per host
- `HTTP_MAX_CONNS_PER_HOST`: Max connections per host
- `HTTP_IDLE_CONN_TIMEOUT`: Idle connection timeout
- `HTTP_TLS_HANDSHAKE_TIMEOUT`: TLS handshake timeout
- `HTTP_RESPONSE_HEADER_TIMEOUT`: Response header timeout
- `HTTP_REQUEST_TIMEOUT`: Request timeout

### Monitoring Configuration
- `MONITORING_METRICS_INTERVAL`: Metrics collection interval
- `MONITORING_HISTORY_RETENTION`: History retention period
- `MONITORING_ALERTING_ENABLED`: Enable/disable alerting
- `MONITORING_DASHBOARD_ENABLED`: Enable/disable dashboard
- `MONITORING_DASHBOARD_PORT`: Dashboard port
- `MONITORING_METRICS_PORT`: Metrics endpoint port

### Load Testing Configuration
- `LOADTEST_MAX_CONCURRENT_USERS`: Maximum concurrent users
- `LOADTEST_DEFAULT_DURATION`: Default test duration
- `LOADTEST_RAMP_UP_DURATION`: Ramp up time
- `LOADTEST_METRICS_INTERVAL`: Metrics collection interval
- `LOADTEST_RESULTS_RETENTION`: Results retention period

## Configuration Validation

### Validation Features

- **Type validation**: Ensures correct data types and formats
- **Range validation**: Validates numeric ranges and limits  
- **Format validation**: Checks URLs, connection strings, API keys
- **Environment-specific rules**: Different requirements per environment
- **Dependency validation**: Checks for required combinations

### Common Validation Errors

1. **TMDB API Key**: Must be 32 character hexadecimal string
2. **Redis URL**: Must be valid host:port or redis:// URL
3. **PostgreSQL URL**: Must be valid postgres:// connection string
4. **Port numbers**: Must be 1-65535 and not conflicting
5. **Timeouts**: Must be positive and reasonable
6. **Pool sizes**: Must be positive and not excessive

### Environment-Specific Requirements

**Production Environment**:
- TMDB API key is required
- Redis must be enabled
- SSL should be enabled for PostgreSQL
- Alerting should be enabled

**Staging Environment**:
- TMDB API key recommended
- Redis recommended for realistic testing

**Development Environment**:
- More flexible validation (warnings vs errors)
- TMDB API key optional (limited functionality warning)

## WaveFlix Hub Integration

### Backward Compatibility

The configuration system provides backward compatibility with existing WaveFlix Hub components:

```go
// Get legacy-compatible configuration
manager, _ := config.NewManager()
legacy := manager.GetBackwardCompatibleConfig()

// Use with existing redis_cache.go
redisCache := &RedisCacheManager{
    URL:         legacy.Redis.URL,
    Password:    legacy.Redis.Password,
    PoolSize:    legacy.Redis.PoolSize,
    DefaultTTL:  legacy.Redis.DefaultTTL,
}

// Use with existing tmdb_client.go
tmdbClient := &TMDBClient{
    APIKey:      legacy.TMDB.APIKey,
    BaseURL:     legacy.TMDB.BaseURL,
    RateLimit:   legacy.TMDB.RateLimit,
    Timeout:     legacy.TMDB.Timeout,
}
```

### Integration Points

1. **Redis Cache Manager**: Uses Redis configuration for connection settings
2. **TMDB Client**: Uses TMDB configuration for API settings and rate limiting
3. **HTTP Clients**: Uses HTTP configuration for connection pooling
4. **PostgreSQL**: Uses database configuration for connection management

## Advanced Features

### Configuration Templates

Generate configuration templates for any environment:

```go
manager := &config.Manager{}
err := manager.GenerateConfigTemplate(config.EnvProduction, "config/production.yaml")
```

### Configuration Reloading

Reload configuration without restarting:

```go
err := manager.ReloadConfiguration()
```

### Connectivity Testing

Test connectivity to external services:

```go
ctx := context.Background()
check, err := manager.ValidateConnectivity(ctx)
if check.OverallStatus != config.HealthHealthy {
    log.Printf("Connectivity issues detected: %+v", check)
}
```

### Validation for All Environments

Validate configuration across all environments:

```go
results, err := manager.ValidateAllEnvironments()
for env, validation := range results {
    if !validation.Valid {
        log.Printf("%s environment has %d errors", env, len(validation.Errors))
    }
}
```

## Best Practices

### Security
- Store sensitive values (passwords, API keys) in environment variables
- Use SSL/TLS for production PostgreSQL connections
- Validate API key formats to prevent typos
- Use strong connection timeouts to prevent resource exhaustion

### Performance
- Set appropriate connection pool sizes for your workload
- Configure reasonable timeouts for external services
- Use Redis caching to reduce database load
- Monitor cache hit rates and adjust TTL accordingly

### Reliability
- Enable monitoring and alerting in production
- Configure retry logic with exponential backoff
- Use circuit breakers for external service calls
- Implement graceful degradation (Redis → memory cache)

### Development
- Use development-specific configuration files
- Test configuration validation in CI/CD pipelines
- Document environment-specific requirements
- Validate configuration before deployment

## Troubleshooting

### Common Issues

1. **Configuration not loading**:
   - Check file paths and permissions
   - Verify YAML/JSON syntax
   - Check environment variable names

2. **Validation errors**:
   - Review validation messages carefully
   - Check required vs optional fields per environment
   - Verify environment variable formats

3. **Performance issues**:
   - Check connection pool sizes
   - Verify timeout settings
   - Monitor cache hit rates
   - Review rate limiting configuration

4. **Connectivity problems**:
   - Use ValidateConnectivity to test connections
   - Check network configuration and firewalls
   - Verify service availability and credentials

### Debug Mode

Enable detailed logging for configuration debugging:

```bash
export LOG_LEVEL=debug
```

### Validation Only

Validate configuration without starting services:

```go
manager := &config.Manager{Validator: config.NewValidator()}
validation, err := manager.ValidateConfiguration(config.EnvProduction)
if !validation.Valid {
    for _, err := range validation.Errors {
        fmt.Printf("Error: %s\n", err)
    }
}
```

## API Reference

See `internal/interfaces/core.go` for complete API documentation and type definitions.

### Key Interfaces

- `EnvironmentManager`: Main configuration management interface
- `EnvironmentConfig`: Complete configuration structure
- `ConfigValidation`: Validation results and feedback
- `ConnectivityCheck`: Service connectivity status

### Key Functions

- `NewManager()`: Create configuration manager
- `LoadEnvironmentConfig()`: Load environment-specific configuration
- `ValidateConfiguration()`: Validate configuration settings
- `GetBackwardCompatibleConfig()`: Get legacy-compatible configuration