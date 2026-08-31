// Package main provides concurrent TMDB API client for improved performance.
//
// This module optimizes TMDB API calls through connection pooling, request batching,
// and intelligent caching to minimize latency and maximize throughput while
// respecting rate limits.
package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// TMDBClient provides concurrent access to TMDB API with optimizations.
type TMDBClient struct {
	httpClient    *http.Client
	rateLimiter   *RateLimiter
	requestPool   *sync.Pool
	responseCache *ResponseCache
	metrics       *TMDBMetrics
}

// RateLimiter implements token bucket rate limiting for TMDB API.
type RateLimiter struct {
	tokens    chan struct{}
	refill    *time.Ticker
	mu        sync.Mutex
	lastRefill time.Time
}

// ResponseCache provides intelligent caching for TMDB responses.
type ResponseCache struct {
	cache   map[string]*CacheEntry
	mu      sync.RWMutex
	janitor *time.Ticker
}

// CacheEntry represents a cached TMDB response.
type CacheEntry struct {
	Data      interface{}
	ExpiresAt time.Time
	Hits      int64
	Size      int
}

// TMDBMetrics tracks TMDB client performance.
type TMDBMetrics struct {
	RequestsTotal     int64
	RequestsSucceeded int64
	RequestsFailed    int64
	CacheHits         int64
	CacheMisses       int64
	AvgResponseTime   time.Duration
	mu                sync.RWMutex
}

// NewTMDBClient creates a new optimized TMDB client.
func NewTMDBClient() *TMDBClient {
	client := &TMDBClient{
		httpClient:  createOptimizedHTTPClient(),
		rateLimiter: NewRateLimiter(40, time.Second*10), // TMDB allows 40 requests per 10 seconds
		requestPool: &sync.Pool{
			New: func() interface{} {
				return &http.Request{}
			},
		},
		responseCache: NewResponseCache(),
		metrics:       &TMDBMetrics{},
	}
	
	// Start metrics collection
	go client.collectMetrics()
	
	return client
}

// createOptimizedHTTPClient creates an HTTP client optimized for TMDB API calls.
func createOptimizedHTTPClient() *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}
	
	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

// NewRateLimiter creates a new token bucket rate limiter.
func NewRateLimiter(capacity int, refillInterval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		tokens:     make(chan struct{}, capacity),
		refill:     time.NewTicker(refillInterval / time.Duration(capacity)),
		lastRefill: time.Now(),
	}
	
	// Fill initial tokens
	for i := 0; i < capacity; i++ {
		rl.tokens <- struct{}{}
	}
	
	// Start refill process
	go rl.refillTokens()
	
	return rl
}

// Wait waits for a token to become available.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// refillTokens periodically refills the token bucket.
func (rl *RateLimiter) refillTokens() {
	for range rl.refill.C {
		select {
		case rl.tokens <- struct{}{}:
		default:
			// Bucket is full
		}
	}
}

// NewResponseCache creates a new response cache.
func NewResponseCache() *ResponseCache {
	rc := &ResponseCache{
		cache:   make(map[string]*CacheEntry),
		janitor: time.NewTicker(5 * time.Minute),
	}
	
	// Start cleanup process
	go rc.cleanup()
	
	return rc
}

// Get retrieves a cached response.
func (rc *ResponseCache) Get(key string) (interface{}, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	
	entry, exists := rc.cache[key]
	if !exists || time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	
	entry.Hits++
	return entry.Data, true
}

// Set stores a response in cache.
func (rc *ResponseCache) Set(key string, data interface{}, ttl time.Duration) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	rc.cache[key] = &CacheEntry{
		Data:      data,
		ExpiresAt: time.Now().Add(ttl),
		Hits:      0,
		Size:      estimateSize(data),
	}
}

// cleanup removes expired entries from cache.
func (rc *ResponseCache) cleanup() {
	for range rc.janitor.C {
		now := time.Now()
		rc.mu.Lock()
		
		for key, entry := range rc.cache {
			if now.After(entry.ExpiresAt) {
				delete(rc.cache, key)
			}
		}
		
		rc.mu.Unlock()
	}
}

// FetchConcurrent fetches multiple TMDB URLs concurrently.
func (tc *TMDBClient) FetchConcurrent(urls []string, maxConcurrency int) ([]map[string]interface{}, error) {
	if maxConcurrency <= 0 {
		maxConcurrency = 10
	}
	
	results := make([]map[string]interface{}, len(urls))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrency)
	
	for i, url := range urls {
		wg.Add(1)
		
		go func(index int, targetUrl string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			// Check cache first
			cacheKey := generateCacheKey(targetUrl)
			if cached, found := tc.responseCache.Get(cacheKey); found {
				if data, ok := cached.(map[string]interface{}); ok {
					results[index] = data
					tc.recordCacheHit()
					return
				}
			}
			
			tc.recordCacheMiss()
			
			// Wait for rate limit token
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			if err := tc.rateLimiter.Wait(ctx); err != nil {
				log.Printf("[tmdb] Rate limit wait timeout: %v", err)
				return
			}
			
			// Make HTTP request
			start := time.Now()
			data, err := tc.fetchSingle(targetUrl)
			duration := time.Since(start)
			
			if err != nil {
				tc.recordFailure(duration)
				log.Printf("[tmdb] Fetch failed: %v", err)
				return
			}
			
			tc.recordSuccess(duration)
			results[index] = data
			
			// Cache successful response
			tc.responseCache.Set(cacheKey, data, getCacheTTL(targetUrl))
			
		}(i, url)
	}
	
	wg.Wait()
	return results, nil
}

// fetchSingle fetches a single URL.
func (tc *TMDBClient) fetchSingle(url string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	// Set optimized headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("User-Agent", "WaveFlix-Hub/1.0")
	
	resp, err := tc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	
	return data, nil
}

// BatchFetch optimizes batch fetching with intelligent grouping.
func (tc *TMDBClient) BatchFetch(requests []TMDBRequest) ([]TMDBResponse, error) {
	// Group requests by endpoint type for optimal caching
	groups := tc.groupRequests(requests)
	
	var allResults []TMDBResponse
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	for _, group := range groups {
		wg.Add(1)
		
		go func(requestGroup []TMDBRequest) {
			defer wg.Done()
			
			groupResults := tc.processBatchGroup(requestGroup)
			
			mu.Lock()
			allResults = append(allResults, groupResults...)
			mu.Unlock()
		}(group)
	}
	
	wg.Wait()
	
	// Sort results back to original order
	sortedResults := make([]TMDBResponse, len(requests))
	for _, result := range allResults {
		sortedResults[result.Index] = result
	}
	
	return sortedResults, nil
}

// TMDBRequest represents a single TMDB API request.
type TMDBRequest struct {
	Index    int
	URL      string
	CacheKey string
	Priority int
}

// TMDBResponse represents a single TMDB API response.
type TMDBResponse struct {
	Index   int
	Data    map[string]interface{}
	Error   error
	Cached  bool
	Duration time.Duration
}

// groupRequests groups requests by endpoint type for optimal processing.
func (tc *TMDBClient) groupRequests(requests []TMDBRequest) [][]TMDBRequest {
	groups := make(map[string][]TMDBRequest)
	
	for _, req := range requests {
		endpoint := extractEndpoint(req.URL)
		groups[endpoint] = append(groups[endpoint], req)
	}
	
	var result [][]TMDBRequest
	for _, group := range groups {
		result = append(result, group)
	}
	
	return result
}

// processBatchGroup processes a group of similar requests.
func (tc *TMDBClient) processBatchGroup(requests []TMDBRequest) []TMDBResponse {
	results := make([]TMDBResponse, len(requests))
	var wg sync.WaitGroup
	
	// Process with limited concurrency
	semaphore := make(chan struct{}, 5)
	
	for i, req := range requests {
		wg.Add(1)
		
		go func(index int, request TMDBRequest) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			start := time.Now()
			response := TMDBResponse{Index: request.Index}
			
			// Check cache
			if cached, found := tc.responseCache.Get(request.CacheKey); found {
				if data, ok := cached.(map[string]interface{}); ok {
					response.Data = data
					response.Cached = true
					response.Duration = time.Since(start)
					results[index] = response
					tc.recordCacheHit()
					return
				}
			}
			
			tc.recordCacheMiss()
			
			// Rate limit
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			
			if err := tc.rateLimiter.Wait(ctx); err != nil {
				response.Error = err
				response.Duration = time.Since(start)
				results[index] = response
				return
			}
			
			// Fetch data
			data, err := tc.fetchSingle(request.URL)
			response.Duration = time.Since(start)
			
			if err != nil {
				response.Error = err
				tc.recordFailure(response.Duration)
			} else {
				response.Data = data
				tc.recordSuccess(response.Duration)
				
				// Cache result
				tc.responseCache.Set(request.CacheKey, data, getCacheTTL(request.URL))
			}
			
			results[index] = response
			
		}(i, req)
	}
	
	wg.Wait()
	return results
}

// Helper functions
func generateCacheKey(url string) string {
	return fmt.Sprintf("tmdb:%x", md5.Sum([]byte(url)))
}

func getCacheTTL(url string) time.Duration {
	if strings.Contains(url, "trending") {
		return 15 * time.Minute
	} else if strings.Contains(url, "detail") || strings.Contains(url, "person") {
		return 1 * time.Hour
	} else if strings.Contains(url, "search") {
		return 30 * time.Minute
	}
	return 30 * time.Minute // default
}

func extractEndpoint(url string) string {
	// Extract endpoint type from URL for grouping
	if strings.Contains(url, "/movie/") {
		return "movie"
	} else if strings.Contains(url, "/tv/") {
		return "tv"
	} else if strings.Contains(url, "/person/") {
		return "person"
	} else if strings.Contains(url, "/trending/") {
		return "trending"
	} else if strings.Contains(url, "/discover/") {
		return "discover"
	} else if strings.Contains(url, "/search/") {
		return "search"
	}
	return "other"
}

func estimateSize(data interface{}) int {
	// Rough estimation of data size
	if str, ok := data.(string); ok {
		return len(str)
	}
	// For complex objects, use a simple heuristic
	return 1024 // default 1KB
}

// Metrics methods
func (tc *TMDBClient) recordSuccess(duration time.Duration) {
	tc.metrics.mu.Lock()
	defer tc.metrics.mu.Unlock()
	
	tc.metrics.RequestsTotal++
	tc.metrics.RequestsSucceeded++
	tc.updateAvgResponseTime(duration)
}

func (tc *TMDBClient) recordFailure(duration time.Duration) {
	tc.metrics.mu.Lock()
	defer tc.metrics.mu.Unlock()
	
	tc.metrics.RequestsTotal++
	tc.metrics.RequestsFailed++
	tc.updateAvgResponseTime(duration)
}

func (tc *TMDBClient) recordCacheHit() {
	tc.metrics.mu.Lock()
	defer tc.metrics.mu.Unlock()
	
	tc.metrics.CacheHits++
}

func (tc *TMDBClient) recordCacheMiss() {
	tc.metrics.mu.Lock()
	defer tc.metrics.mu.Unlock()
	
	tc.metrics.CacheMisses++
}

func (tc *TMDBClient) updateAvgResponseTime(duration time.Duration) {
	// Simple moving average
	tc.metrics.AvgResponseTime = (tc.metrics.AvgResponseTime + duration) / 2
}

func (tc *TMDBClient) collectMetrics() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		tc.metrics.mu.RLock()
		
		hitRate := float64(0)
		totalRequests := tc.metrics.CacheHits + tc.metrics.CacheMisses
		if totalRequests > 0 {
			hitRate = float64(tc.metrics.CacheHits) / float64(totalRequests) * 100
		}
		
		log.Printf("[tmdb] Metrics: requests=%d, success_rate=%.1f%%, cache_hit_rate=%.1f%%, avg_response_time=%v",
			tc.metrics.RequestsTotal,
			float64(tc.metrics.RequestsSucceeded)/float64(tc.metrics.RequestsTotal)*100,
			hitRate,
			tc.metrics.AvgResponseTime)
		
		tc.metrics.mu.RUnlock()
	}
}

// GetMetrics returns current TMDB client metrics.
func (tc *TMDBClient) GetMetrics() *TMDBMetrics {
	tc.metrics.mu.RLock()
	defer tc.metrics.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	return &TMDBMetrics{
		RequestsTotal:     tc.metrics.RequestsTotal,
		RequestsSucceeded: tc.metrics.RequestsSucceeded,
		RequestsFailed:    tc.metrics.RequestsFailed,
		CacheHits:         tc.metrics.CacheHits,
		CacheMisses:       tc.metrics.CacheMisses,
		AvgResponseTime:   tc.metrics.AvgResponseTime,
	}
}

// Global concurrent TMDB client
var concurrentTMDBClient *TMDBClient

// initConcurrentTMDB initializes the global concurrent TMDB client
func initConcurrentTMDB() {
	concurrentTMDBClient = NewTMDBClient()
	log.Println("[tmdb] Concurrent TMDB client initialized")
}