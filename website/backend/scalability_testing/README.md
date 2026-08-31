# WaveFlix Hub Scalability Testing & Monitoring

This system validates and monitors the scalability improvements implemented through PostgreSQL migration, Redis caching, TMDB API optimization, HTTP connection pooling, and rate limiting components.

## Features

- **Integration Testing Framework**: Validates that all scalability components work together seamlessly
- **Load Testing Engine**: Generates realistic user scenarios and validates performance under load
- **Performance Monitoring**: Real-time monitoring of all scalability components with alerting
- **Environment Management**: Configuration management across different deployment environments
- **Comprehensive Metrics Collection**: Detailed performance metrics and trend analysis

## Architecture

The system is built with a modular architecture:

- `internal/interfaces/` - Core interfaces and data models
- `internal/config/` - Configuration management and validation
- `internal/integration/` - Integration testing framework
- `internal/loadtest/` - Load testing engine
- `internal/monitoring/` - Performance monitoring system
- `internal/errors/` - Standardized error handling

## Quick Start

### Prerequisites

- Go 1.25.5 or later
- Redis server (for caching tests)
- PostgreSQL database (for database integration tests)
- TMDB API key (for API integration tests)

### Environment Variables

```bash
# Required
TMDB_API_KEY=your_tmdb_api_key_here
DATABASE_URL=postgres://localhost:5432/waveflix
REDIS_URL=localhost:6379

# Optional
ENVIRONMENT=development
REDIS_PASSWORD=your_redis_password
```

### Running the System

1. **Clone and build**:
   ```bash
   cd scalability_testing
   go mod download
   go build -o scalability-testing .
   ```

2. **Run integration tests**:
   ```bash
   ./scalability-testing --mode integration
   ```

3. **Run load tests**:
   ```bash
   ./scalability-testing --mode loadtest --scenario basic
   ```

4. **Start monitoring**:
   ```bash
   ./scalability-testing --mode monitor
   ```

## Configuration

Configuration is managed through YAML files and environment variables:

- `config/development.yaml` - Development environment configuration
- `config/staging.yaml` - Staging environment configuration (optional)
- `config/production.yaml` - Production environment configuration

Environment variables take precedence over configuration files.

## Integration with Existing Components

The testing system integrates with the following existing WaveFlix Hub components:

- **Redis Cache Manager** (`redis_cache.go`) - Tests caching performance and fallback mechanisms
- **TMDB Client** (`tmdb_client.go`) - Tests API rate limiting and response caching
- **HTTP Client** (`http_client.go`) - Tests connection pooling and performance optimization
- **API Protection** (`api_protection.go`) - Tests rate limiting and circuit breaker functionality

## Testing Framework

### Integration Tests

Integration tests validate that all scalability components work together:

```go
validator, _ := integration.NewValidator(configManager)
results, _ := validator.RunAllTests(ctx)

for component, result := range results {
    if result.Success {
        fmt.Printf("✓ %s: PASS\n", component)
    } else {
        fmt.Printf("✗ %s: FAIL\n", component)
    }
}
```

### Load Tests

Load tests generate realistic user scenarios:

```go
engine, _ := loadtest.NewEngine(configManager)
scenario := &loadtest.Scenario{
    Name:            "Basic Load Test",
    ConcurrentUsers: 100,
    Duration:        time.Minute * 5,
    RampUpTime:      time.Minute * 1,
}
results, _ := engine.ExecuteScenario(ctx, scenario)
```

### Performance Monitoring

Real-time monitoring with alerting:

```go
monitor, _ := monitoring.NewPerformanceMonitor(configManager)
go monitor.Start(ctx)

metrics, _ := monitor.GetRealTimeMetrics()
fmt.Printf("System Health: %s\n", metrics.SystemHealth)
```

## Metrics and Alerting

The system collects comprehensive performance metrics:

- **Redis Metrics**: Hit rates, response times, memory usage
- **TMDB Metrics**: Request rates, rate limiting status, circuit breaker state
- **HTTP Pool Metrics**: Connection utilization, reuse rates, latency
- **Database Metrics**: Connection pool health, query performance
- **System Resources**: CPU, memory, disk, network I/O

Alerting is configurable with multiple severity levels and delivery channels.

## Performance Benchmarks

Expected performance targets:

- **Response Time**: <200ms for cached content, <500ms for fresh API calls
- **Throughput**: >1000 requests/second sustained
- **Cache Efficiency**: >80% hit rate for movie/TV data
- **Error Rate**: <1% under normal load, <5% under stress

## Development

### Project Structure

```
scalability_testing/
├── main.go                    # Application entry point
├── logger.go                  # Logging framework
├── go.mod                     # Go module definition
├── README.md                  # This file
├── config/                    # Configuration files
│   ├── development.yaml
│   └── production.yaml
└── internal/                  # Internal packages
    ├── interfaces/            # Core interfaces
    ├── config/                # Configuration management
    ├── integration/           # Integration testing
    ├── loadtest/              # Load testing
    ├── monitoring/            # Performance monitoring
    └── errors/                # Error handling
```

### Adding New Tests

To add new integration tests:

1. Implement the test in `internal/integration/validator.go`
2. Add metrics collection in the test method
3. Include validation logic and success criteria
4. Update the `RunAllTests` method to include the new test

### Configuration Management

Configuration follows this precedence (highest to lowest):

1. Environment variables
2. Configuration files (`config/{environment}.yaml`)
3. Default values in code

All configuration is validated at startup with clear error messages for issues.

## Deployment

### Docker Support

```dockerfile
FROM golang:1.25.5-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o scalability-testing .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/scalability-testing .
COPY --from=builder /app/config ./config
CMD ["./scalability-testing", "--mode", "monitor"]
```

### Kubernetes Deployment

The system can be deployed as a Kubernetes deployment with ConfigMaps for environment-specific configuration and Secrets for sensitive data like API keys.

## Troubleshooting

### Common Issues

1. **Redis Connection Failed**: Check Redis server status and connection string
2. **TMDB API Errors**: Verify API key and rate limiting configuration
3. **Database Connection Issues**: Check PostgreSQL connection string and SSL settings
4. **Configuration Validation Errors**: Review configuration file syntax and required fields

### Logs and Debugging

The system provides structured logging with different levels:

```bash
# Enable debug logging
export LOG_LEVEL=debug
./scalability-testing --mode integration
```

### Health Checks

Monitor system health through the monitoring endpoints:

- `GET /health` - Overall system health
- `GET /metrics` - Prometheus-compatible metrics
- `GET /dashboard` - Web-based performance dashboard

## License

This project is part of WaveFlix Hub and follows the same licensing terms.