// Package main provides parallelized handlers for WaveFlix Hub API endpoints.
//
// These handlers use worker pools to process CPU-intensive tasks in parallel,
// improving response times and throughput for operations like data processing,
// TMDB API calls, and content analysis.
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// handleDetailBatchParallel processes multiple detail requests in parallel.
func handleDetailBatchParallel(w http.ResponseWriter, r *http.Request) {
	ids := parseBatchIDs(r.URL.Query().Get("ids"))
	if len(ids) == 0 {
		httpError(w, http.StatusBadRequest, "ids parameter required")
		return
	}
	
	media := normalizeMedia(r.URL.Query().Get("media"))
	if media == "" {
		httpError(w, http.StatusBadRequest, "media parameter required")
		return
	}

	// Limit batch size to prevent abuse
	if len(ids) > 50 {
		httpError(w, http.StatusBadRequest, "too many IDs requested (max 50)")
		return
	}

	// Process requests in parallel using worker pools
	results := make(map[string]interface{})
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	// Use semaphore to limit concurrent TMDB API calls
	semaphore := make(chan struct{}, 10)

	for _, id := range ids {
		wg.Add(1)
		
		// Submit task to worker pool
		err := SubmitBackgroundTask(TaskTypeTMDBAPI, map[string]interface{}{
			"id":    id,
			"media": media,
			"query": r.URL.Query(),
			"callback": func(result interface{}, err error) {
				defer wg.Done()
				
				if err == nil {
					mu.Lock()
					results[id] = result
					mu.Unlock()
				}
			},
		})
		
		// Fallback to direct processing if worker pool unavailable
		if err != nil {
			go func(itemID string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				
				// Fetch data directly
				base := tmdbBaseUrl + "/" + media + "/" + itemID
				imageLangs := getImageLangs(r.URL.Query())
				targetUrl := base + "?language=" + getTmdbLang(r.URL.Query()) + 
					"&append_to_response=credits,videos,recommendations,similar,images" +
					"&include_image_language=" + imageLangs + "&api_key=" + tmdbApiKey

				data, fetchErr := fetchJSON(targetUrl, media)
				if fetchErr == nil {
					sanitizeTMDBData(data, media)
					
					mu.Lock()
					results[itemID] = data
					mu.Unlock()
					
					recordExternalAPICall("tmdb", "success")
				} else {
					recordExternalAPICall("tmdb", "error")
				}
			}(id)
		}
	}

	// Wait for all requests to complete with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All requests completed
	case <-time.After(30 * time.Second):
		// Timeout - return partial results
		log.Printf("[parallel] Batch request timeout, returning partial results (%d/%d)", len(results), len(ids))
	}

	response := map[string]interface{}{
		"results": results,
		"total":   len(results),
		"requested": len(ids),
	}

	responseJSON, _ := json.Marshal(response)
	writeJSON(w, string(responseJSON))
}

// handleDiscoverParallel processes discover requests with parallel data enrichment.
func handleDiscoverParallel(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	media := getMediaParam(q)
	
	// First, get basic discover results
	p := url.Values{}
	p.Set("api_key", tmdbApiKey)
	p.Set("language", getTmdbLang(q))
	p.Set("include_image_language", getImageLangs(q))
	p.Set("watch_region", "ID")
	
	// Copy discover parameters
	for key, values := range q {
		if key != "media" && len(values) > 0 {
			p.Set(key, values[0])
		}
	}
	
	targetUrl := tmdbBaseUrl + "/discover/" + media + "?" + p.Encode()
	data, err := fetchJSON(targetUrl, media)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		writeJSON(w, `{"error":"TMDB API error"}`)
		return
	}
	
	// Parallel enhancement of results
	if results, ok := data["results"].([]interface{}); ok {
		enhanceResultsParallel(results, media)
	}
	
	responseJSON, _ := json.Marshal(data)
	writeJSON(w, string(responseJSON))
}

// enhanceResultsParallel enhances discover results in parallel.
func enhanceResultsParallel(results []interface{}, media string) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Limit concurrent enhancements
	
	for i, result := range results {
		if item, ok := result.(map[string]interface{}); ok {
			wg.Add(1)
			
			go func(index int, data map[string]interface{}) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				
				// Submit enhancement task to worker pool
				err := SubmitBackgroundTask(TaskTypeDataProcess, map[string]interface{}{
					"type":   "enhance_result",
					"data":   data,
					"media":  media,
					"index":  index,
				})
				
				if err != nil {
					// Fallback enhancement
					enhanceResultItem(data, media)
				}
			}(i, item)
		}
	}
	
	// Wait for enhancements with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// All enhancements completed
	case <-time.After(5 * time.Second):
		// Timeout - continue with partial enhancements
		log.Printf("[parallel] Result enhancement timeout")
	}
}

// enhanceResultItem enhances a single result item.
func enhanceResultItem(item map[string]interface{}, media string) {
	// Add provider availability check
	if id, ok := item["id"].(float64); ok {
		idStr := strconv.Itoa(int(id))
		
		// Check vidlink availability in background
		go func() {
			available := checkVidlinkAvailability(media, idStr)
			// Store result in cache for future use
			if cache != nil {
				cacheKey := "vidlink:" + media + ":" + idStr
				cache.SetJSON(cacheKey, map[string]bool{"available": available}, 1*time.Hour)
			}
		}()
		
		// Check major provider availability
		go func() {
			hasMajorProvider := checkMajorProvider(media, idStr)
			if cache != nil {
				cacheKey := "provider:" + media + ":" + idStr
				cache.SetJSON(cacheKey, map[string]bool{"major_provider": hasMajorProvider}, 6*time.Hour)
			}
		}()
	}
	
	// Sanitize data
	sanitizeTMDBData(item, media)
}

// handleSearchParallel processes search with parallel result processing.
func handleSearchParallel(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		httpError(w, http.StatusBadRequest, "query parameter required")
		return
	}
	
	page := r.URL.Query().Get("page")
	if page == "" {
		page = "1"
	}
	
	imageLangs := getImageLangs(r.URL.Query())
	targetUrl := tmdbBaseUrl + "/search/multi?query=" + url.QueryEscape(query) +
		"&language=" + getTmdbLang(r.URL.Query()) + "&include_image_language=" + imageLangs + 
		"&page=" + page + "&include_adult=false&api_key=" + tmdbApiKey
	
	data, err := fetchJSON(targetUrl, "multi")
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		writeJSON(w, `{"error":"TMDB API error"}`)
		return
	}
	
	// Process search results in parallel
	if results, ok := data["results"].([]interface{}); ok {
		processSearchResultsParallel(results)
	}
	
	responseJSON, _ := json.Marshal(data)
	writeJSON(w, string(responseJSON))
}

// processSearchResultsParallel processes search results with parallel enhancement.
func processSearchResultsParallel(results []interface{}) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 8) // Limit concurrent processing
	
	for _, result := range results {
		if item, ok := result.(map[string]interface{}); ok {
			wg.Add(1)
			
			go func(data map[string]interface{}) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				
				// Submit to worker pool
				err := SubmitBackgroundTask(TaskTypeDataProcess, map[string]interface{}{
					"type": "process_search_result",
					"data": data,
				})
				
				if err != nil {
					// Fallback processing
					processSearchResult(data)
				}
			}(item)
		}
	}
	
	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		log.Printf("[parallel] Search result processing timeout")
	}
}

// processSearchResult processes a single search result.
func processSearchResult(item map[string]interface{}) {
	// Determine media type
	mediaType := "movie" // default
	if mt, ok := item["media_type"].(string); ok {
		mediaType = mt
	}
	
	// Sanitize and filter
	sanitizeTMDBData(item, mediaType)
	
	// Record content view for analytics
	recordContentView(mediaType)
}

// Concurrent TMDB API batch processor
func processTMDBBatch(ids []string, media string, query url.Values) map[string]interface{} {
	results := make(map[string]interface{})
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	// Create worker pool for this batch
	maxWorkers := 10
	if len(ids) < maxWorkers {
		maxWorkers = len(ids)
	}
	
	jobQueue := make(chan string, len(ids))
	
	// Start workers
	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobQueue {
				// Process single item
				base := tmdbBaseUrl + "/" + media + "/" + id
				imageLangs := getImageLangs(query)
				targetUrl := base + "?language=" + getTmdbLang(query) + 
					"&append_to_response=credits,videos,recommendations,similar,images" +
					"&include_image_language=" + imageLangs + "&api_key=" + tmdbApiKey

				data, err := fetchJSON(targetUrl, media)
				if err == nil {
					sanitizeTMDBData(data, media)
					
					mu.Lock()
					results[id] = data
					mu.Unlock()
					
					recordExternalAPICall("tmdb", "success")
				} else {
					recordExternalAPICall("tmdb", "error")
				}
			}
		}()
	}
	
	// Send jobs
	for _, id := range ids {
		jobQueue <- id
	}
	close(jobQueue)
	
	// Wait for completion
	wg.Wait()
	
	return results
}

// Helper function to process large datasets in chunks
func processInChunks[T any](items []T, chunkSize int, processor func([]T)) {
	for i := 0; i < len(items); i += chunkSize {
		end := i + chunkSize
		if end > len(items) {
			end = len(items)
		}
		
		chunk := items[i:end]
		processor(chunk)
	}
}

// Parallel content recommendation processing
func processRecommendationsParallel(data map[string]interface{}, media string) {
	if recommendations, ok := data["recommendations"].(map[string]interface{}); ok {
		if results, ok := recommendations["results"].([]interface{}); ok {
			// Process recommendations in background
			go func() {
				var wg sync.WaitGroup
				semaphore := make(chan struct{}, 3)
				
				for _, rec := range results {
					if item, ok := rec.(map[string]interface{}); ok {
						wg.Add(1)
						
						go func(recItem map[string]interface{}) {
							defer wg.Done()
							semaphore <- struct{}{}
							defer func() { <-semaphore }()
							
							sanitizeTMDBData(recItem, media)
						}(item)
					}
				}
				
				wg.Wait()
			}()
		}
	}
}