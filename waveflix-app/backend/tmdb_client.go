

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

type TMDBClient struct {
cache      *RedisCacheManager
httpClient *http.Client
apiKey     string
baseURL    string

inflight   map[string]*inflightRequest
inflightMu sync.RWMutex

rateLimiter *TMDBRateLimiter

circuitBreaker *CircuitBreaker
}

type inflightRequest struct {
wg     sync.WaitGroup
result []byte
err    error
}

type CacheTTLConfig struct {
MovieDetail    time.Duration 
TVDetail       time.Duration 
PersonDetail   time.Duration 
SearchResults  time.Duration 
Trending       time.Duration 
Discover       time.Duration 
Images         time.Duration 
Videos         time.Duration 
Credits        time.Duration 
WatchProviders time.Duration 
}

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

func (tc *TMDBClient) GetMovie(movieID, language, appendToResponse string) (map[string]interface{}, error) {
cacheKey := fmt.Sprintf("tmdb:movie:%s:%s:%s", movieID, language, appendToResponse)

var cachedData map[string]interface{}
if tc.cache.GetJSON(cacheKey, &cachedData) {
return cachedData, nil
}

params := url.Values{
"api_key":  {tc.apiKey},
"language": {language},
}
if appendToResponse != "" {
params.Set("append_to_response", appendToResponse)
}

apiURL := fmt.Sprintf("%s/movie/%s?%s", tc.baseURL, movieID, params.Encode())

data, err := tc.fetchWithDeduplication(apiURL, cacheKey)
if err != nil {
return nil, err
}

var result map[string]interface{}
if err := json.Unmarshal(data, &result); err != nil {
return nil, err
}

tc.cache.SetJSON(cacheKey, result, DefaultCacheTTL().MovieDetail)

return result, nil
}

func (tc *TMDBClient) GetTV(tvID, language, appendToResponse string) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("tmdb:tv:%s:%s:%s", tvID, language, appendToResponse)

	var cachedData map[string]interface{}
	if tc.cache.GetJSON(cacheKey, &cachedData) {
		return cachedData, nil
	}

	params := url.Values{
		"api_key":  {tc.apiKey},
		"language": {language},
		"include_image_language": {"en,null"},
	}
	if appendToResponse != "" {
		params.Set("append_to_response", appendToResponse)
	}

	apiURL := fmt.Sprintf("%s/tv/%s?%s", tc.baseURL, tvID, params.Encode())

	data, err := tc.fetchWithDeduplication(apiURL, cacheKey)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	tc.cache.SetJSON(cacheKey, result, DefaultCacheTTL().TVDetail)

	return result, nil
}

func (tc *TMDBClient) GetTrending(mediaType, timeWindow, language string) (map[string]interface{}, error) {
cacheKey := fmt.Sprintf("tmdb:trending:%s:%s:%s", mediaType, timeWindow, language)

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

tc.cache.SetJSON(cacheKey, result, DefaultCacheTTL().Trending)

return result, nil
}

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

func (tc *TMDBClient) BatchGetMovies(movieIDs []string, language string) map[string]map[string]interface{} {
results := make(map[string]map[string]interface{})
var uncachedIDs []string

for _, id := range movieIDs {
cacheKey := fmt.Sprintf("tmdb:movie:%s:%s:", id, language)
var cachedData map[string]interface{}

if tc.cache.GetJSON(cacheKey, &cachedData) {
results[id] = cachedData
} else {
uncachedIDs = append(uncachedIDs, id)
}
}

if len(uncachedIDs) > 0 {
var wg sync.WaitGroup
resultChan := make(chan struct {
id   string
data map[string]interface{}
err  error
}, len(uncachedIDs))

semaphore := make(chan struct{}, 5) 

for _, id := range uncachedIDs {
wg.Add(1)
go func(movieID string) {
defer wg.Done()
semaphore <- struct{}{} 
defer func() { <-semaphore }() 

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

for result := range resultChan {
if result.err == nil {
results[result.id] = result.data
}
}
}

return results
}

func (tc *TMDBClient) fetchWithDeduplication(apiURL, cacheKey string) ([]byte, error) {
tc.inflightMu.Lock()

if req, exists := tc.inflight[apiURL]; exists {
tc.inflightMu.Unlock()
req.wg.Wait() 
return req.result, req.err
}

req := &inflightRequest{}
req.wg.Add(1)
tc.inflight[apiURL] = req
tc.inflightMu.Unlock()

result, err := tc.fetchFromAPI(apiURL)

req.result = result
req.err = err
req.wg.Done()

tc.inflightMu.Lock()
delete(tc.inflight, apiURL)
tc.inflightMu.Unlock()

return result, err
}

func (tc *TMDBClient) fetchFromAPI(apiURL string) ([]byte, error) {

if !tc.circuitBreaker.AllowRequest() {
return nil, fmt.Errorf("circuit breaker open: TMDB API unhealthy")
}

if err := tc.rateLimiter.Wait(context.Background()); err != nil {
return nil, fmt.Errorf("rate limit exceeded: %w", err)
}

resp, err := tc.httpClient.Get(apiURL)
if err != nil {
tc.circuitBreaker.RecordFailure()
return nil, err
}
defer resp.Body.Close()

if resp.StatusCode == 429 {
tc.circuitBreaker.RecordFailure()
return nil, fmt.Errorf("TMDB API rate limit exceeded")
}

if resp.StatusCode != 200 {
tc.circuitBreaker.RecordFailure()
return nil, fmt.Errorf("TMDB API error: %d", resp.StatusCode)
}

data, err := io.ReadAll(io.LimitReader(resp.Body, 5_000_000)) 
if err != nil {
tc.circuitBreaker.RecordFailure()
return nil, err
}

tc.circuitBreaker.RecordSuccess()
return data, nil
}

func (tc *TMDBClient) GetCacheStats() *CacheStats {
return tc.cache.GetStats()
}

func (tc *TMDBClient) Close() error {
return tc.cache.Close()
}

func (tc *TMDBClient) HandleTrending(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	media := getMediaParam(q)
	timeWindow := q.Get("time_window")
	if timeWindow == "" {
		timeWindow = "day"
	}
	language := getTmdbLang(q)

	data, err := tc.GetTrending(media, timeWindow, language)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		writeJSON(w, `{"error":"TMDB API error"}`)
		return
	}
	sanitizeTMDBData(data, media)
	if results, ok := data["results"].([]interface{}); ok {
		filtered := filterByMajorProviderParallel(results, media, q.Get("provider"))
		data["results"] = filtered
	}
	responseJSON, _ := json.Marshal(data)
	writeJSON(w, string(responseJSON))
}

func (tc *TMDBClient) HandleDetail(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, `{"error":"id required"}`)
		return
	}

	media := getMediaParam(q)
	language := getTmdbLang(q)
	var data map[string]interface{}
	var err error

	if media == "tv" {
		data, err = tc.GetTV(id, language, "credits,videos,recommendations,similar,images")
	} else {
		data, err = tc.GetMovie(id, language, "credits,videos,recommendations,similar,images")
	}

	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		writeJSON(w, `{"error":"TMDB API error"}`)
		return
	}
	sanitizeTMDBData(data, media)
	responseJSON, _ := json.Marshal(data)
	writeJSON(w, string(responseJSON))
}

func (tc *TMDBClient) HandlePerson(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, `{"error":"id required"}`)
		return
	}
	targetUrl := fmt.Sprintf("%s/person/%s?api_key=%s&language=%s&append_to_response=combined_credits,images", tc.baseURL, id, tc.apiKey, getTmdbLang(q))
	data, err := tc.fetchWithDeduplication(targetUrl, "tmdb:person:"+id)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		writeJSON(w, `{"error":"TMDB API error"}`)
		return
	}

	writeJSON(w, string(data))
}

func (tc *TMDBClient) HandleSeason(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := q.Get("id")
	season := q.Get("season")
	if id == "" || season == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, `{"error":"id and season required"}`)
		return
	}

	language := getTmdbLang(q)
	cacheKey := fmt.Sprintf("tmdb:season:%s:%s:%s", id, season, language)
	targetUrl := fmt.Sprintf("%s/tv/%s/season/%s?api_key=%s&language=%s", tc.baseURL, id, season, tc.apiKey, language)
	
	data, err := tc.fetchWithDeduplication(targetUrl, cacheKey)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		writeJSON(w, `{"error":"TMDB API error"}`)
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err == nil {
		if episodes, ok := result["episodes"].([]interface{}); ok {
			var availableEpisodes []interface{}
			var mu sync.Mutex
			var wg sync.WaitGroup

			semaphore := make(chan struct{}, 3) // max 3 concurrent checks to prevent rate limiting

			for _, epInter := range episodes {
				ep, ok := epInter.(map[string]interface{})
				if !ok {
					continue
				}
				epNum := fmt.Sprintf("%v", ep["episode_number"])
				
				wg.Add(1)
				go func(ep map[string]interface{}, episodeNumber string) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					if tc.CheckEpisodeAvailability("tv", id, season, episodeNumber) {
						mu.Lock()
						availableEpisodes = append(availableEpisodes, ep)
						mu.Unlock()
					}
				}(ep, epNum)
			}
			wg.Wait()
			
			result["episodes"] = availableEpisodes
			filteredData, _ := json.Marshal(result)
			writeJSON(w, string(filteredData))
			return
		}
	}

	writeJSON(w, string(data))
}

var tmdbClient *TMDBClient

func InitOptimizedTMDB(cache *RedisCacheManager) {
tmdbClient = NewTMDBClient(cache)
log.Println("[tmdb] Optimized TMDB client initialized with caching and rate limiting")
}
