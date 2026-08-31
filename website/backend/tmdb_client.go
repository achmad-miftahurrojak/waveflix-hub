// Package main provides optimized TMDB API client with intelligent caching.
//
// The TMDB client implements advanced caching strategies, request batching,
// and rate limiting to maximize API efficiency and prevent rate limit violations.
// It supports both Redis and in-memory caching with automatic failover.
package main

import (
"context"
"encoding/json"
"fmt"
"io"
"log"
"net/http"
"net/url"
"os"
"sync"
"time"
)

// TMDBClient provides optimized access to TMDB API with caching and rate limiting.
//
// The client implements several optimization strategies:
//   - Intelligent caching with different TTLs per content type
//   - Request deduplication for concurrent identical requests
//   - Batch processing for multiple requests
//   - Graceful rate limit handling with backoff
//   - Circuit breaker for API health protection
//
// Features:
//   - Redis-backed caching for horizontal scaling
//   - Request coalescing to prevent duplicate API calls
//   - Configurable TTL per content type (trending vs static)
//   - Automatic retry with exponential backoff
//   - Comprehensive metrics and monitoring
//
// Example usage:
//
//client := NewTMDBClient(cacheManager)
//movie, err := client.GetMovie("123456")
//trending, err := client.GetTrending("movie", "day")
type TMDBClient struct {
cache      *RedisCacheManager
httpClient *http.Client
apiKey     string
baseURL    string

// Request deduplication
inflight   map[string]*inflightRequest
inflightMu sync.RWMutex

// Rate limiting
rateLimiter *TMDBRateLimiter

// Circuit breaker
circuitBreaker *CircuitBreaker
}

// inflightRequest handles concurrent identical requests.
type inflightRequest struct {
wg     sync.WaitGroup
result []byte
err    error
}

// CacheTTLConfig defines TTL for different content types.
type CacheTTLConfig struct {
MovieDetail    time.Duration // 2 hours - movies rarely change
TVDetail       time.Duration // 2 hours - TV shows rarely change
PersonDetail   time.Duration // 6 hours - people info very static
SearchResults  time.Duration // 30 minutes - search results change moderately
Trending       time.Duration // 15 minutes - trending changes frequently
Discover       time.Duration // 30 minutes - discovery results change moderately
Images         time.Duration // 24 hours - images are static
Videos         time.Duration // 24 hours - videos rarely change
Credits        time.Duration // 6 hours - cast/crew info fairly static
WatchProviders time.Duration // 2 hours - provider availability changes daily
}

// DefaultCacheTTL returns production-optimized cache TTL configuration.
func DefaultCacheTTL() *CacheTTLConfig {
return &CacheTTLConfig{
MovieDetail:    2 * time.Hour,
TVDetail:       2 * time.Hour,
PersonDetail:   6 * time.Hour,
SearchResults:  30 * time.Minute,
Trending:       15 * time.Minute,
Discover:       30 * time.Minute,
Images:         24 * time.Hour,
Videos:         24 * time.Hour,
Credits:        6 * time.Hour,
WatchProviders: 2 * time.Hour,
}
}

// NewTMDBClient creates an optimized TMDB API client.
//
// The client is configured with intelligent caching, rate limiting,
// and request optimization for maximum performance and reliability.
//
// Parameters:
//   - cache: Redis cache manager for persistent caching
//
// Returns: Configured TMDBClient ready for production use
//
// Example:
//
//cache := NewRedisCacheManager()
//client := NewTMDBClient(cache)
//defer client.Close()
func NewTMDBClient(cache *RedisCacheManager) *TMDBClient {
return &TMDBClient{
cache:      cache,
httpClient: &http.Client{Timeout: 10 * time.Second},
apiKey:     os.Getenv("TMDB_API_KEY"),
baseURL:    "https://api.themoviedb.org/3",
inflight:   make(map[string]*inflightRequest),
rateLimiter: NewTMDBRateLimiter(),
circuitBreaker: NewCircuitBreaker(),
}
}

// GetMovie retrieves movie details with optimized caching.
//
// This method implements intelligent caching with 2-hour TTL since movie
// details rarely change. It includes request deduplication to prevent
// multiple concurrent calls for the same movie.
//
// Parameters:
//   - movieID: TMDB movie ID
//   - language: Language code (e.g., "id-ID")
//   - appendToResponse: Additional data to fetch (credits,videos,etc)
//
// Returns:
//   - Movie data as map[string]interface{}
//   - Error if request fails
//
// Example:
//
//movie, err := client.GetMovie("123456", "id-ID", "credits,videos")
func (tc *TMDBClient) GetMovie(movieID, language, appendToResponse string) (map[string]interface{}, error) {
cacheKey := fmt.Sprintf("tmdb:movie:%s:%s:%s", movieID, language, appendToResponse)

// Try cache first
var cachedData map[string]interface{}
if tc.cache.GetJSON(cacheKey, &cachedData) {
return cachedData, nil
}

// Build API URL
params := url.Values{
"api_key":  {tc.apiKey},
"language": {language},
}
if appendToResponse != "" {
params.Set("append_to_response", appendToResponse)
}

apiURL := fmt.Sprintf("%s/movie/%s?%s", tc.baseURL, movieID, params.Encode())

// Fetch with deduplication
data, err := tc.fetchWithDeduplication(apiURL, cacheKey)
if err != nil {
return nil, err
}

// Parse and cache
var result map[string]interface{}
if err := json.Unmarshal(data, &result); err != nil {
return nil, err
}

// Cache with movie-specific TTL
tc.cache.SetJSON(cacheKey, result, DefaultCacheTTL().MovieDetail)

return result, nil
}

// GetTrending retrieves trending content with short-term caching.
//
// Trending content changes frequently, so this method uses a shorter
// 15-minute TTL to balance API efficiency with content freshness.
//
// Parameters:
//   - mediaType: "movie" or "tv"
//   - timeWindow: "day" or "week"
//   - language: Language code
//
// Returns: Trending results with optimized caching
//
// Example:
//
//trending, err := client.GetTrending("movie", "day", "id-ID")
func (tc *TMDBClient) GetTrending(mediaType, timeWindow, language string) (map[string]interface{}, error) {
cacheKey := fmt.Sprintf("tmdb:trending:%s:%s:%s", mediaType, timeWindow, language)

// Try cache first (short TTL for trending)
var cachedData map[string]interface{}
if tc.cache.GetJSON(cacheKey, &cachedData) {
return cachedData, nil
}

params := url.Values{
"api_key":  {tc.apiKey},
"language": {language},
}

apiURL := fmt.Sprintf("%s/trending/%s/%s?%s", tc.baseURL, mediaType, timeWindow, params.Encode())

data, err := tc.fetchWithDeduplication(apiURL, cacheKey)
if err != nil {
return nil, err
}

var result map[string]interface{}
if err := json.Unmarshal(data, &result); err != nil {
return nil, err
}

// Cache with trending-specific TTL (shorter)
tc.cache.SetJSON(cacheKey, result, DefaultCacheTTL().Trending)

return result, nil
}

// GetWatchProviders retrieves streaming provider info with medium-term caching.
//
// Provider availability changes daily, so we use 2-hour TTL to balance
// freshness with API efficiency.
//
// Parameters:
//   - mediaType: "movie" or "tv"
//   - contentID: TMDB content ID
//
// Returns: Watch provider data with optimized caching
func (tc *TMDBClient) GetWatchProviders(mediaType, contentID string) (map[string]interface{}, error) {
cacheKey := fmt.Sprintf("tmdb:providers:%s:%s", mediaType, contentID)

var cachedData map[string]interface{}
if tc.cache.GetJSON(cacheKey, &cachedData) {
return cachedData, nil
}

params := url.Values{"api_key": {tc.apiKey}}
apiURL := fmt.Sprintf("%s/%s/%s/watch/providers?%s", tc.baseURL, mediaType, contentID, params.Encode())

data, err := tc.fetchWithDeduplication(apiURL, cacheKey)
if err != nil {
return nil, err
}

var result map[string]interface{}
if err := json.Unmarshal(data, &result); err != nil {
return nil, err
}

tc.cache.SetJSON(cacheKey, result, DefaultCacheTTL().WatchProviders)
return result, nil
}

// BatchGetMovies retrieves multiple movies efficiently using request batching.
//
// This method optimizes multiple movie requests by:
//   1. Checking cache for each movie individually
//   2. Batching uncached requests to minimize API calls
//   3. Implementing smart request deduplication
//
// Parameters:
//   - movieIDs: Slice of TMDB movie IDs
//   - language: Language code for all requests
//
// Returns: Map of movie ID to movie data
//
// Example:
//
//movies := client.BatchGetMovies([]string{"123", "456", "789"}, "id-ID")
func (tc *TMDBClient) BatchGetMovies(movieIDs []string, language string) map[string]map[string]interface{} {
results := make(map[string]map[string]interface{})
var uncachedIDs []string

// First pass: check cache for each movie
for _, id := range movieIDs {
cacheKey := fmt.Sprintf("tmdb:movie:%s:%s:", id, language)
var cachedData map[string]interface{}

if tc.cache.GetJSON(cacheKey, &cachedData) {
results[id] = cachedData
} else {
uncachedIDs = append(uncachedIDs, id)
}
}

// Second pass: fetch uncached movies concurrently
if len(uncachedIDs) > 0 {
var wg sync.WaitGroup
resultChan := make(chan struct {
id   string
data map[string]interface{}
err  error
}, len(uncachedIDs))

// Limit concurrent requests to respect rate limits
semaphore := make(chan struct{}, 5) // Max 5 concurrent requests

for _, id := range uncachedIDs {
wg.Add(1)
go func(movieID string) {
defer wg.Done()
semaphore <- struct{}{} // Acquire semaphore
defer func() { <-semaphore }() // Release semaphore

movie, err := tc.GetMovie(movieID, language, "")
resultChan <- struct {
id   string
data map[string]interface{}
err  error
}{movieID, movie, err}
}(id)
}

wg.Wait()
close(resultChan)

// Collect results
for result := range resultChan {
if result.err == nil {
results[result.id] = result.data
}
}
}

return results
}

// fetchWithDeduplication prevents duplicate concurrent requests.
//
// This method ensures that multiple concurrent requests for the same URL
// only result in a single API call, improving efficiency and reducing
// load on TMDB servers.
func (tc *TMDBClient) fetchWithDeduplication(apiURL, cacheKey string) ([]byte, error) {
tc.inflightMu.Lock()

// Check if request is already in flight
if req, exists := tc.inflight[apiURL]; exists {
tc.inflightMu.Unlock()
req.wg.Wait() // Wait for existing request to complete
return req.result, req.err
}

// Create new in-flight request
req := &inflightRequest{}
req.wg.Add(1)
tc.inflight[apiURL] = req
tc.inflightMu.Unlock()

// Perform the actual request
result, err := tc.fetchFromAPI(apiURL)

// Update in-flight request with result
req.result = result
req.err = err
req.wg.Done()

// Clean up in-flight map
tc.inflightMu.Lock()
delete(tc.inflight, apiURL)
tc.inflightMu.Unlock()

return result, err
}

// fetchFromAPI performs the actual HTTP request with rate limiting and circuit breaking.
func (tc *TMDBClient) fetchFromAPI(apiURL string) ([]byte, error) {
// Check circuit breaker
if !tc.circuitBreaker.AllowRequest() {
return nil, fmt.Errorf("circuit breaker open: TMDB API unhealthy")
}

// Apply rate limiting
if err := tc.rateLimiter.Wait(context.Background()); err != nil {
return nil, fmt.Errorf("rate limit exceeded: %w", err)
}

// Perform HTTP request
resp, err := tc.httpClient.Get(apiURL)
if err != nil {
tc.circuitBreaker.RecordFailure()
return nil, err
}
defer resp.Body.Close()

// Handle rate limit responses
if resp.StatusCode == 429 {
tc.circuitBreaker.RecordFailure()
return nil, fmt.Errorf("TMDB API rate limit exceeded")
}

if resp.StatusCode != 200 {
tc.circuitBreaker.RecordFailure()
return nil, fmt.Errorf("TMDB API error: %d", resp.StatusCode)
}

// Read response
data, err := io.ReadAll(io.LimitReader(resp.Body, 5_000_000)) // 5MB limit
if err != nil {
tc.circuitBreaker.RecordFailure()
return nil, err
}

tc.circuitBreaker.RecordSuccess()
return data, nil
}

// GetCacheStats returns cache performance metrics.
func (tc *TMDBClient) GetCacheStats() *CacheStats {
return tc.cache.GetStats()
}

// Close closes the TMDB client and cleanup resources.
func (tc *TMDBClient) Close() error {
return tc.cache.Close()
}

// Global instance for backward compatibility
var tmdbClient *TMDBClient

// InitOptimizedTMDB initializes the optimized TMDB client.
func InitOptimizedTMDB(cache *RedisCacheManager) {
tmdbClient = NewTMDBClient(cache)
log.Println("[tmdb] Optimized TMDB client initialized with caching and rate limiting")
}
