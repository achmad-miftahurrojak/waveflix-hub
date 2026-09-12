

package main

import (
"context"
"database/sql"
"log"
"time"
)

type ConnectionPoolConfig struct {

MaxOpenConns int

MaxIdleConns int

ConnMaxLifetime time.Duration

ConnMaxIdleTime time.Duration

HealthCheckInterval time.Duration
}

func DefaultPoolConfig() *ConnectionPoolConfig {
return &ConnectionPoolConfig{
MaxOpenConns:        getEnvInt("PG_MAX_CONNS", 25),
MaxIdleConns:        getEnvInt("PG_MAX_IDLE_CONNS", 25),
ConnMaxLifetime:     time.Duration(getEnvInt("PG_CONN_MAX_LIFETIME", 30)) * time.Minute,
ConnMaxIdleTime:     time.Duration(getEnvInt("PG_CONN_MAX_IDLE_TIME", 10)) * time.Minute,
HealthCheckInterval: time.Duration(getEnvInt("PG_HEALTH_CHECK_INTERVAL", 60)) * time.Second,
}
}

type PoolManager struct {
db      *sql.DB
config  *ConnectionPoolConfig
healthTicker *time.Ticker
done    chan struct{}
}

func NewPoolManager(db *sql.DB, config *ConnectionPoolConfig) *PoolManager {
manager := &PoolManager{
db:     db,
config: config,
done:   make(chan struct{}),
}

manager.applyConfiguration()

return manager
}

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

func (pm *PoolManager) performHealthCheck() {
if err := pm.db.Ping(); err != nil {
log.Printf("[pool] Health check FAILED: %v", err)
return
}

stats := pm.db.Stats()
log.Printf("[pool] Health check OK (connections: %d open, %d idle, %d in-use)",
stats.OpenConnections,
stats.Idle,
stats.InUse)
}

func (pm *PoolManager) GetStats() sql.DBStats {
return pm.db.Stats()
}

func (pm *PoolManager) Shutdown(timeout time.Duration) {
log.Println("[pool] Shutting down connection pool manager...")

close(pm.done)

if pm.healthTicker != nil {
pm.healthTicker.Stop()
}

select {
case <-time.After(timeout):
log.Println("[pool] Shutdown completed (timeout reached)")
case <-time.After(100 * time.Millisecond):
log.Println("[pool] Shutdown completed")
}
}

