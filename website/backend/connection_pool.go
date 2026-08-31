// Package main provides connection pool management for PostgreSQL connections.
//
// The connection pool manager configures and monitors PostgreSQL connection pools
// with tunable parameters, health monitoring, and graceful shutdown capabilities.
// It provides production-ready defaults while allowing customization via environment variables.
package main

import (
"context"
"database/sql"
"log"
"time"
)

// ConnectionPoolConfig holds configuration parameters for database connection pooling.
//
// These parameters control how the database connection pool behaves under different
// load conditions and help optimize performance and resource usage.
//
// Configuration Priority:
//   1. Environment variables (PG_MAX_CONNS, PG_MAX_IDLE_CONNS, etc.)
//   2. Explicit configuration passed to NewConnectionPoolConfig
//   3. Production-ready defaults
//
// Example usage:
//
//config := DefaultPoolConfig()
//pool := NewPoolManager(db, config)
//pool.StartHealthMonitoring()
type ConnectionPoolConfig struct {
// MaxOpenConns is the maximum number of open connections to the database.
// Default: 25 (configurable via PG_MAX_CONNS)
//
// Recommended formula: CPU cores * 2 + disk spindles
// For cloud databases, start with 25 and adjust based on monitoring.
MaxOpenConns int

// MaxIdleConns is the maximum number of connections in the idle connection pool.
// Default: 25 (configurable via PG_MAX_IDLE_CONNS)
//
// Should typically match MaxOpenConns for consistent performance.
// Lower values save memory but may cause connection churn.
MaxIdleConns int

// ConnMaxLifetime is the maximum amount of time a connection may be reused.
// Default: 30 minutes
//
// Prevents accumulation of stale connections and ensures fresh connections.
// Should be shorter than database server's connection timeout.
ConnMaxLifetime time.Duration

// ConnMaxIdleTime is the maximum amount of time a connection may be idle.
// Default: 10 minutes
//
// Automatically closes idle connections to free resources.
// Balance between resource usage and connection establishment overhead.
ConnMaxIdleTime time.Duration

// HealthCheckInterval is how frequently to check connection pool health.
// Default: 1 minute
//
// Health checks ping the database to detect connectivity issues early.
// More frequent checks provide faster failure detection but increase load.
HealthCheckInterval time.Duration
}

// DefaultPoolConfig returns a production-ready connection pool configuration.
//
// The configuration uses conservative defaults suitable for most applications
// while allowing override via environment variables. The defaults balance
// performance, resource usage, and reliability.
//
// Environment variable overrides:
//   - PG_MAX_CONNS: Override MaxOpenConns (default: 25)
//   - PG_MAX_IDLE_CONNS: Override MaxIdleConns (default: 25)
//   - PG_CONN_MAX_LIFETIME: Override ConnMaxLifetime in minutes (default: 30)
//   - PG_CONN_MAX_IDLE_TIME: Override ConnMaxIdleTime in minutes (default: 10)
//   - PG_HEALTH_CHECK_INTERVAL: Override HealthCheckInterval in seconds (default: 60)
//
// Returns: ConnectionPoolConfig with environment-aware defaults
//
// Example:
//
//config := DefaultPoolConfig()
//log.Printf("Using max connections: %d", config.MaxOpenConns)
func DefaultPoolConfig() *ConnectionPoolConfig {
return &ConnectionPoolConfig{
MaxOpenConns:        getEnvInt("PG_MAX_CONNS", 25),
MaxIdleConns:        getEnvInt("PG_MAX_IDLE_CONNS", 25),
ConnMaxLifetime:     time.Duration(getEnvInt("PG_CONN_MAX_LIFETIME", 30)) * time.Minute,
ConnMaxIdleTime:     time.Duration(getEnvInt("PG_CONN_MAX_IDLE_TIME", 10)) * time.Minute,
HealthCheckInterval: time.Duration(getEnvInt("PG_HEALTH_CHECK_INTERVAL", 60)) * time.Second,
}
}

// PoolManager manages database connection pool lifecycle and health monitoring.
//
// The manager applies configuration to database connections, monitors pool health,
// and provides graceful shutdown capabilities. It runs health checks in the background
// and can detect connection issues before they affect application performance.
//
// Features:
//   - Configurable connection pool parameters
//   - Background health monitoring with ping checks
//   - Graceful shutdown with timeout
//   - Detailed logging of pool configuration and health events
//   - Context-aware operations for cancellation support
//
// Example usage:
//
//db, _ := sql.Open("pgx", connectionString)
//config := DefaultPoolConfig()
//manager := NewPoolManager(db, config)
//
//// Start health monitoring
//ctx, cancel := context.WithCancel(context.Background())
//manager.StartHealthMonitoring(ctx)
//
//// Later, gracefully shutdown
//cancel()
//manager.Shutdown(5 * time.Second)
type PoolManager struct {
db      *sql.DB
config  *ConnectionPoolConfig
healthTicker *time.Ticker
done    chan struct{}
}

// NewPoolManager creates a new connection pool manager with the specified configuration.
//
// The manager immediately applies the pool configuration to the database connection.
// Health monitoring can be started separately with StartHealthMonitoring().
//
// Parameters:
//   - db: Database connection to manage
//   - config: Pool configuration parameters
//
// Returns: Configured PoolManager ready for health monitoring
//
// Example:
//
//config := DefaultPoolConfig()
//config.MaxOpenConns = 50 // Override default
//manager := NewPoolManager(db, config)
func NewPoolManager(db *sql.DB, config *ConnectionPoolConfig) *PoolManager {
manager := &PoolManager{
db:     db,
config: config,
done:   make(chan struct{}),
}

// Apply configuration to the database connection
manager.applyConfiguration()

return manager
}

// applyConfiguration applies the pool configuration to the database connection.
//
// This method sets connection pool limits, timeouts, and lifecycle parameters
// on the database connection. The configuration is applied once during
// initialization and cannot be changed without creating a new manager.
//
// Configuration applied:
//   - Maximum open connections (prevents pool exhaustion)
//   - Maximum idle connections (balances resource usage and performance)
//   - Connection maximum lifetime (prevents stale connections)
//   - Connection maximum idle time (closes unused connections)
//
// All parameters are logged for operational visibility.
func (pm *PoolManager) applyConfiguration() {
pm.db.SetMaxOpenConns(pm.config.MaxOpenConns)
pm.db.SetMaxIdleConns(pm.config.MaxIdleConns)
pm.db.SetConnMaxLifetime(pm.config.ConnMaxLifetime)
pm.db.SetConnMaxIdleTime(pm.config.ConnMaxIdleTime)

log.Printf("[pool] Connection pool configured: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%v, MaxIdleTime=%v",
pm.config.MaxOpenConns,
pm.config.MaxIdleConns,
pm.config.ConnMaxLifetime,
pm.config.ConnMaxIdleTime)
}

// StartHealthMonitoring begins periodic health checks for the connection pool.
//
// The health monitoring runs in a background goroutine and performs the following:
//   - Ping the database at regular intervals (HealthCheckInterval)
//   - Log successful health checks for operational visibility
//   - Log and handle health check failures (connection issues)
//   - Support graceful shutdown via context cancellation
//
// Health checks help detect database connectivity issues early, before they
// impact user-facing operations. The monitoring can be stopped by cancelling
// the provided context or calling Shutdown().
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//
// Example:
//
//ctx, cancel := context.WithCancel(context.Background())
//defer cancel()
//manager.StartHealthMonitoring(ctx)
func (pm *PoolManager) StartHealthMonitoring(ctx context.Context) {
pm.healthTicker = time.NewTicker(pm.config.HealthCheckInterval)

log.Printf("[pool] Starting health monitoring (interval: %v)", pm.config.HealthCheckInterval)

go func() {
defer pm.healthTicker.Stop()

for {
select {
case <-ctx.Done():
log.Println("[pool] Health monitoring stopped (context cancelled)")
return

case <-pm.done:
log.Println("[pool] Health monitoring stopped (shutdown requested)")
return

case <-pm.healthTicker.C:
pm.performHealthCheck()
}
}
}()
}

// performHealthCheck executes a single health check against the database.
//
// The health check performs a simple ping operation to verify database connectivity.
// Success and failure events are logged with appropriate detail for monitoring
// and troubleshooting.
//
// Health check failures may indicate:
//   - Network connectivity issues
//   - Database server problems
//   - Connection pool exhaustion
//   - Authentication or permission problems
//
// The application continues running during health check failures, but operators
// should investigate persistent failures promptly.
func (pm *PoolManager) performHealthCheck() {
if err := pm.db.Ping(); err != nil {
log.Printf("[pool] Health check FAILED: %v", err)
return
}

// Get current pool statistics for health check logging
stats := pm.db.Stats()
log.Printf("[pool] Health check OK (connections: %d open, %d idle, %d in-use)",
stats.OpenConnections,
stats.Idle,
stats.InUse)
}

// GetStats returns current connection pool statistics.
//
// The statistics provide real-time insight into pool utilization and can be
// used for monitoring, alerting, and capacity planning.
//
// Key metrics include:
//   - OpenConnections: Total established connections (should be ≤ MaxOpenConns)
//   - Idle: Available connections waiting for reuse
//   - InUse: Connections currently executing queries
//   - WaitCount: Total number of times connections were waited for
//   - WaitDuration: Total time spent waiting for connections
//
// Returns: sql.DBStats with current pool statistics
//
// Example:
//
//stats := manager.GetStats()
//if stats.InUse > stats.MaxOpenConnections * 0.8 {
//    log.Warn("Connection pool utilization high")
//}
func (pm *PoolManager) GetStats() sql.DBStats {
return pm.db.Stats()
}

// Shutdown gracefully stops the connection pool manager.
//
// The shutdown process:
//   1. Stops health monitoring
//   2. Signals background goroutines to terminate
//   3. Waits for graceful shutdown with timeout
//   4. Does NOT close the database connection (caller's responsibility)
//
// The database connection remains open after shutdown - the caller should
// call db.Close() separately if needed. This design allows the manager to
// be stopped without affecting database connectivity.
//
// Parameters:
//   - timeout: Maximum time to wait for graceful shutdown
//
// Example:
//
//// Graceful shutdown with 5-second timeout
//manager.Shutdown(5 * time.Second)
//db.Close() // Caller closes database if needed
func (pm *PoolManager) Shutdown(timeout time.Duration) {
log.Println("[pool] Shutting down connection pool manager...")

// Signal shutdown to health monitoring goroutine
close(pm.done)

// Stop the health check ticker if running
if pm.healthTicker != nil {
pm.healthTicker.Stop()
}

// Wait briefly for goroutines to finish
select {
case <-time.After(timeout):
log.Println("[pool] Shutdown completed (timeout reached)")
case <-time.After(100 * time.Millisecond):
log.Println("[pool] Shutdown completed")
}
}



