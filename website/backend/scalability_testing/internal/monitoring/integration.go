

package monitoring

import (
"context"
"sync"
"time"

"github.com/hamin-baek/waveflix-hub/scalability-testing/internal/interfaces"
)

type ComponentIntegrator struct {
redisCollector    RedisMetricsCollector
tmdbCollector     TMDBMetricsCollector
httpPoolCollector HTTPPoolMetricsCollector
dbCollector       DatabaseMetricsCollector
mutex             sync.RWMutex
lastUpdate        time.Time
}

type RedisMetricsCollector interface {

GetCacheStats(ctx context.Context) (hits, misses int64, hitRate float64, err error)

GetResponseTime(ctx context.Context) (time.Duration, error)

GetConnectionInfo(ctx context.Context) (active, total int, err error)

GetMemoryUsage(ctx context.Context) (used, total int64, err error)

GetKeyCount(ctx context.Context) (int, error)

GetCommandStats(ctx context.Context) (processed int64, err error)

IsHealthy(ctx context.Context) (bool, error)
}

type TMDBMetricsCollector interface {

GetRequestRate(ctx context.Context) (float64, error)

GetResponseTime(ctx context.Context) (time.Duration, error)

GetRateLimitStatus(ctx context.Context) (utilization float64, remaining int, err error)

GetCircuitBreakerState(ctx context.Context) (string, error)

GetCacheStats(ctx context.Context) (hitRate float64, err error)

GetErrorRate(ctx context.Context) (float64, error)

GetQueueStatus(ctx context.Context) (queued, active int, err error)

GetBackoffDelay(ctx context.Context) (time.Duration, error)
}

type HTTPPoolMetricsCollector interface {

GetConnectionStats(ctx context.Context) (active, idle, total int, err error)

GetConnectionReuse(ctx context.Context) (float64, error)

GetLatencyStats(ctx context.Context) (avg, p95, p99 time.Duration, err error)

GetThroughput(ctx context.Context) (float64, error)

GetConnectionLifetime(ctx context.Context) (time.Duration, error)

GetTimingBreakdown(ctx context.Context) (handshake, dns time.Duration, err error)

GetConnectionErrors(ctx context.Context) (int64, error)
}

type DatabaseMetricsCollector interface {

GetConnectionPoolStats(ctx context.Context) (active, idle int, health string, err error)

GetQueryPerformance(ctx context.Context) (avgQueryTime, avgTxTime time.Duration, qps float64, err error)

GetCacheMetrics(ctx context.Context) (cacheHitRatio, indexHitRatio float64, err error)

GetSlowQueryStats(ctx context.Context) (slowQueries, lockWaits int64, err error)

IsHealthy(ctx context.Context) (bool, error)
}

func NewComponentIntegrator(
redisCollector RedisMetricsCollector,
tmdbCollector TMDBMetricsCollector,
httpPoolCollector HTTPPoolMetricsCollector,
dbCollector DatabaseMetricsCollector,
) *ComponentIntegrator {
return &ComponentIntegrator{
redisCollector:    redisCollector,
tmdbCollector:     tmdbCollector,
httpPoolCollector: httpPoolCollector,
dbCollector:       dbCollector,
lastUpdate:        time.Now(),
}
}

func (ci *ComponentIntegrator) CollectRedisMetrics(ctx context.Context) (*interfaces.RedisMetrics, error) {
ci.mutex.Lock()
defer ci.mutex.Unlock()

timestamp := time.Now()

metrics := &interfaces.RedisMetrics{
Timestamp: timestamp,
}

if ci.redisCollector != nil {
if _, _, hitRate, err := ci.redisCollector.GetCacheStats(ctx); err == nil {
metrics.HitRate = hitRate
metrics.MissRate = 1.0 - hitRate
}

if responseTime, err := ci.redisCollector.GetResponseTime(ctx); err == nil {
metrics.ResponseTime = responseTime
}

if active, _, err := ci.redisCollector.GetConnectionInfo(ctx); err == nil {
metrics.ConnectionsActive = active
}

if used, _, err := ci.redisCollector.GetMemoryUsage(ctx); err == nil {
metrics.MemoryUsage = used
}

if keyCount, err := ci.redisCollector.GetKeyCount(ctx); err == nil {
metrics.KeyCount = keyCount
}

if processed, err := ci.redisCollector.GetCommandStats(ctx); err == nil {
metrics.CommandsProcessed = processed
}
}

metrics.NetworkIO = &interfaces.NetworkIO{
BytesIn:    0, 
BytesOut:   0, 
PacketsIn:  0, 
PacketsOut: 0, 
}

return metrics, nil
}

func (ci *ComponentIntegrator) CollectTMDBMetrics(ctx context.Context) (*interfaces.TMDBMetrics, error) {
ci.mutex.Lock()
defer ci.mutex.Unlock()

timestamp := time.Now()

metrics := &interfaces.TMDBMetrics{
Timestamp: timestamp,
}

if ci.tmdbCollector != nil {
if requestRate, err := ci.tmdbCollector.GetRequestRate(ctx); err == nil {
metrics.RequestRate = requestRate
}

if responseTime, err := ci.tmdbCollector.GetResponseTime(ctx); err == nil {
metrics.ResponseTime = responseTime
}

if utilization, remaining, err := ci.tmdbCollector.GetRateLimitStatus(ctx); err == nil {
metrics.RateLimitUtilization = utilization
metrics.APIQuotaRemaining = remaining
}

if state, err := ci.tmdbCollector.GetCircuitBreakerState(ctx); err == nil {
metrics.CircuitBreakerState = state
}

if hitRate, err := ci.tmdbCollector.GetCacheStats(ctx); err == nil {
metrics.CacheHitRate = hitRate
}

if errorRate, err := ci.tmdbCollector.GetErrorRate(ctx); err == nil {
metrics.ErrorRate = errorRate
}

if queued, active, err := ci.tmdbCollector.GetQueueStatus(ctx); err == nil {
metrics.QueuedRequests = queued
metrics.ActiveRequests = active
}

if backoff, err := ci.tmdbCollector.GetBackoffDelay(ctx); err == nil {
metrics.BackoffDelay = backoff
}
}

return metrics, nil
}

func (ci *ComponentIntegrator) CollectHTTPPoolMetrics(ctx context.Context) (*interfaces.HTTPPoolMetrics, error) {
ci.mutex.Lock()
defer ci.mutex.Unlock()

timestamp := time.Now()

metrics := &interfaces.HTTPPoolMetrics{
Timestamp: timestamp,
}

if ci.httpPoolCollector != nil {
if active, idle, total, err := ci.httpPoolCollector.GetConnectionStats(ctx); err == nil {
metrics.ActiveConnections = active
metrics.IdleConnections = idle
metrics.TotalConnections = total
}

if reuseRate, err := ci.httpPoolCollector.GetConnectionReuse(ctx); err == nil {
metrics.ConnectionReuse = reuseRate
}

if avg, _, _, err := ci.httpPoolCollector.GetLatencyStats(ctx); err == nil {
metrics.AverageLatency = avg
}

if throughput, err := ci.httpPoolCollector.GetThroughput(ctx); err == nil {
metrics.ThroughputRPS = throughput
}

if lifetime, err := ci.httpPoolCollector.GetConnectionLifetime(ctx); err == nil {
metrics.ConnectionLifetime = lifetime
}

if handshake, dns, err := ci.httpPoolCollector.GetTimingBreakdown(ctx); err == nil {
metrics.HandshakeTime = handshake
metrics.DNSLookupTime = dns
}

if errors, err := ci.httpPoolCollector.GetConnectionErrors(ctx); err == nil {
metrics.ConnectionErrors = errors
}
}

return metrics, nil
}

func (ci *ComponentIntegrator) CollectDatabaseMetrics(ctx context.Context) (*interfaces.DatabaseMetrics, error) {
ci.mutex.Lock()
defer ci.mutex.Unlock()

timestamp := time.Now()

metrics := &interfaces.DatabaseMetrics{
Timestamp: timestamp,
}

if ci.dbCollector != nil {
if active, idle, health, err := ci.dbCollector.GetConnectionPoolStats(ctx); err == nil {
metrics.ActiveConnections = active
metrics.IdleConnections = idle
metrics.ConnectionPoolHealth = health
}

if queryTime, txTime, qps, err := ci.dbCollector.GetQueryPerformance(ctx); err == nil {
metrics.QueryTime = queryTime
metrics.TransactionTime = txTime
metrics.QueriesPerSecond = qps
}

if cacheHit, indexHit, err := ci.dbCollector.GetCacheMetrics(ctx); err == nil {
metrics.CacheHitRatio = cacheHit
metrics.IndexHitRatio = indexHit
}

if slowQueries, lockWaits, err := ci.dbCollector.GetSlowQueryStats(ctx); err == nil {
metrics.SlowQueries = slowQueries
metrics.LockWaits = lockWaits
}
}

return metrics, nil
}

func (ci *ComponentIntegrator) IsHealthy(ctx context.Context) (bool, map[string]bool, error) {
ci.mutex.RLock()
defer ci.mutex.RUnlock()

componentHealth := make(map[string]bool)
allHealthy := true

if ci.redisCollector != nil {
if healthy, err := ci.redisCollector.IsHealthy(ctx); err == nil {
componentHealth["redis"] = healthy
allHealthy = allHealthy && healthy
} else {
componentHealth["redis"] = false
allHealthy = false
}
}

if ci.dbCollector != nil {
if healthy, err := ci.dbCollector.IsHealthy(ctx); err == nil {
componentHealth["database"] = healthy
allHealthy = allHealthy && healthy
} else {
componentHealth["database"] = false
allHealthy = false
}
}

componentHealth["tmdb"] = true    
componentHealth["http_pool"] = true 

return allHealthy, componentHealth, nil
}

func (ci *ComponentIntegrator) UpdateCollectors(
redisCollector RedisMetricsCollector,
tmdbCollector TMDBMetricsCollector,
httpPoolCollector HTTPPoolMetricsCollector,
dbCollector DatabaseMetricsCollector,
) {
ci.mutex.Lock()
defer ci.mutex.Unlock()

if redisCollector != nil {
ci.redisCollector = redisCollector
}
if tmdbCollector != nil {
ci.tmdbCollector = tmdbCollector
}
if httpPoolCollector != nil {
ci.httpPoolCollector = httpPoolCollector
}
if dbCollector != nil {
ci.dbCollector = dbCollector
}

ci.lastUpdate = time.Now()
}

func (ci *ComponentIntegrator) GetLastUpdateTime() time.Time {
ci.mutex.RLock()
defer ci.mutex.RUnlock()
return ci.lastUpdate
}
