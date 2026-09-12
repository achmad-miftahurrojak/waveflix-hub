

package main

import (
"context"
"encoding/json"
"log"
"os"
"sync"
"time"

"github.com/redis/go-redis/v9"
)

type RedisCacheManager struct {
client    *redis.Client
enabled   bool
fallback  *MemoryCache 
defaultTTL time.Duration
}

type MemoryCache struct {
cache map[string]cacheItem
mu    sync.RWMutex
}

type CacheConfig struct {
RedisURL     string
Password     string
DB           int
MaxRetries   int
PoolSize     int
PoolTimeout  time.Duration
IdleTimeout  time.Duration
DefaultTTL   time.Duration
}

func NewRedisCacheManager() *RedisCacheManager {
config := loadCacheConfig()

rdb := redis.NewClient(&redis.Options{
Addr:         config.RedisURL,
Password:     config.Password,
DB:           config.DB,
MaxRetries:   config.MaxRetries,
PoolSize:     config.PoolSize,
PoolTimeout:  config.PoolTimeout,

})

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

enabled := true
if err := rdb.Ping(ctx).Err(); err != nil {
log.Printf("[cache] Redis connection failed: %v", err)
log.Printf("[cache] Falling back to in-memory cache (limited scalability)")
enabled = false
} else {
log.Printf("[cache] Redis connected successfully: %s", config.RedisURL)
}

memCache := &MemoryCache{
cache: make(map[string]cacheItem),
}

go memCache.cleanup()

return &RedisCacheManager{
client:     rdb,
enabled:    enabled,
fallback:   memCache,
defaultTTL: config.DefaultTTL,
}
}

func loadCacheConfig() *CacheConfig {
return &CacheConfig{
RedisURL:     getEnvDefault("REDIS_URL", "localhost:6379"),
Password:     os.Getenv("REDIS_PASSWORD"),
DB:           getEnvInt("REDIS_DB", 0),
MaxRetries:   getEnvInt("REDIS_MAX_RETRIES", 3),
PoolSize:     getEnvInt("REDIS_POOL_SIZE", 20),
PoolTimeout:  time.Duration(getEnvInt("REDIS_POOL_TIMEOUT", 30)) * time.Second,
IdleTimeout:  time.Duration(getEnvInt("REDIS_IDLE_TIMEOUT", 300)) * time.Second,
DefaultTTL:   time.Duration(getEnvInt("CACHE_TTL_MINUTES", 15)) * time.Minute,
}
}

func (rcm *RedisCacheManager) Get(key string) ([]byte, bool) {
if rcm.enabled {
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()

data, err := rcm.client.Get(ctx, key).Bytes()
if err == nil {
return data, true
}

if err != redis.Nil {
log.Printf("[cache] Redis GET error for key %s: %v", key, err)
}
}

return rcm.fallback.Get(key)
}

func (rcm *RedisCacheManager) Set(key string, data []byte, ttl time.Duration) {
if ttl == 0 {
ttl = rcm.defaultTTL
}

if rcm.enabled {
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()

err := rcm.client.Set(ctx, key, data, ttl).Err()
if err == nil {
return 
}

log.Printf("[cache] Redis SET error for key %s: %v", key, err)
}

rcm.fallback.Set(key, data, ttl)
}

func (rcm *RedisCacheManager) Delete(key string) {
if rcm.enabled {
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()

err := rcm.client.Del(ctx, key).Err()
if err != nil {
log.Printf("[cache] Redis DEL error for key %s: %v", key, err)
}
}

rcm.fallback.Delete(key)
}

func (rcm *RedisCacheManager) GetJSON(key string, dest interface{}) bool {
data, found := rcm.Get(key)
if !found {
return false
}

err := json.Unmarshal(data, dest)
if err != nil {
log.Printf("[cache] JSON unmarshal error for key %s: %v", key, err)
return false
}

return true
}

func (rcm *RedisCacheManager) SetJSON(key string, data interface{}, ttl time.Duration) error {
jsonData, err := json.Marshal(data)
if err != nil {
return err
}

rcm.Set(key, jsonData, ttl)
return nil
}

func (rcm *RedisCacheManager) GetStats() *CacheStats {
stats := &CacheStats{
Enabled: rcm.enabled,
}

if rcm.enabled {
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()

info := rcm.client.Info(ctx)
if info.Err() == nil {

stats.RedisConnected = true
}
}

stats.MemoryCacheSize = len(rcm.fallback.cache)
return stats
}

func (rcm *RedisCacheManager) Close() error {
if rcm.client != nil {
return rcm.client.Close()
}
return nil
}

type CacheStats struct {
Enabled         bool `json:"enabled"`
RedisConnected  bool `json:"redis_connected"`
MemoryCacheSize int  `json:"memory_cache_size"`
}

func (mc *MemoryCache) Get(key string) ([]byte, bool) {
mc.mu.RLock()
defer mc.mu.RUnlock()

item, exists := mc.cache[key]
if !exists || time.Now().After(item.expireAt) {
return nil, false
}

return item.data, true
}

func (mc *MemoryCache) Set(key string, data []byte, ttl time.Duration) {
mc.mu.Lock()
defer mc.mu.Unlock()

mc.cache[key] = cacheItem{
data:     data,
expireAt: time.Now().Add(ttl),
}
}

func (mc *MemoryCache) Delete(key string) {
mc.mu.Lock()
defer mc.mu.Unlock()

delete(mc.cache, key)
}

func (mc *MemoryCache) cleanup() {
ticker := time.NewTicker(5 * time.Minute)
defer ticker.Stop()

for {
select {
case <-ticker.C:
now := time.Now()
mc.mu.Lock()
for k, v := range mc.cache {
if now.After(v.expireAt) {
delete(mc.cache, k)
}
}
mc.mu.Unlock()
}
}
}

func getEnvDefault(key, defaultValue string) string {
if value := os.Getenv(key); value != "" {
return value
}
return defaultValue
}
