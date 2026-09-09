

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

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

	if len(ids) > 50 {
		httpError(w, http.StatusBadRequest, "too many IDs requested (max 50)")
		return
	}

	results := make(map[string]interface{})
	var mu sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, 10)

	for _, id := range ids {
		wg.Add(1)

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

		if err != nil {
			go func(itemID string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				base := tmdbBaseUrl + "/" + media + "/" + itemID
				imageLangs := getImageLangs(r.URL.Query())
				targetUrl := base + "?language=" + getTmdbLang(r.URL.Query()) + 
					"&append_to_response=credits,videos,recommendations,similar,images" +
					"&include_image_language=" + imageLangs + "&api_key=" + getTmdbApiKey()

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

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:

	case <-time.After(30 * time.Second):

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

func handleDiscoverParallel(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	media := getMediaParam(q)

	p := url.Values{}
	p.Set("api_key", getTmdbApiKey())
	p.Set("language", getTmdbLang(q))
	p.Set("include_image_language", getImageLangs(q))

	if provider := q.Get("provider"); provider != "" {
		p.Set("with_watch_providers", provider)
		p.Set("watch_region", "ID")
		p.Set("watch_monetization_types", "flatrate|free|ads")
	} else {
		p.Set("with_watch_providers", "8|119|350|122|158|483|489|1899")
		p.Set("watch_region", "ID")
		p.Set("watch_monetization_types", "flatrate|free|ads")
	}

	for key, values := range q {
		if key != "media" && key != "provider" && len(values) > 0 {
			val := values[0]
			switch key {
			case "genre":
				p.Set("with_genres", val)
			case "year":
				if media == "movie" {
					p.Set("primary_release_year", val)
				} else {
					p.Set("first_air_date_year", val)
				}
			case "country":
				p.Set("with_origin_country", val)
			case "network":
				p.Set("with_networks", val)
			case "released_after":
				if media == "movie" {
					p.Set("primary_release_date.gte", val)
				} else {
					p.Set("first_air_date.gte", val)
				}
			case "released_before":
				if media == "movie" {
					p.Set("primary_release_date.lte", val)
				} else {
					p.Set("first_air_date.lte", val)
				}
			case "sort_by":
				p.Set(key, val)
			default:
				p.Set(key, val)
			}
		}
	}

	collectionID := q.Get("collection")

	var data map[string]interface{}
	var err error

	if collectionID != "" && media == "movie" {
		ids := strings.Split(collectionID, ",")
		var allParts []interface{}
		
		for _, cid := range ids {
			targetUrl := tmdbBaseUrl + "/collection/" + cid + "?api_key=" + getTmdbApiKey() + "&language=" + getTmdbLang(q)
			collData, fetchErr := fetchJSON(targetUrl, media)
			if fetchErr == nil {
				if parts, ok := collData["parts"].([]interface{}); ok {
					allParts = append(allParts, parts...)
				}
			}
		}

		if len(allParts) > 0 {
			filteredParts := filterByMajorProviderParallel(allParts, media)
			data = map[string]interface{}{
				"page":          1,
				"results":       filteredParts,
				"total_pages":   1,
				"total_results": len(filteredParts),
			}
		} else {
			err = fmt.Errorf("no collection parts found")
		}
	} else {
		targetUrl := tmdbBaseUrl + "/discover/" + media + "?" + p.Encode()
		data, err = fetchJSON(targetUrl, media)
	}

	if err != nil {
		log.Printf("[tmdb] Error in handleDiscoverParallel: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		writeJSON(w, `{"error":"TMDB API error"}`)
		return
	}


	if results, ok := data["results"].([]interface{}); ok {
		filtered := enhanceResultsParallel(results, media)
		data["results"] = filtered
		// DO NOT overwrite total_results or total_pages so infinite scroll works
	}

	responseJSON, _ := json.Marshal(data)
	writeJSON(w, string(responseJSON))
}

func enhanceResultsParallel(results []interface{}, media string) []interface{} {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) 
	
	keep := make([]bool, len(results))

	for i, result := range results {
		if item, ok := result.(map[string]interface{}); ok {
			wg.Add(1)

			go func(index int, data map[string]interface{}) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				if id, ok := data["id"].(float64); ok {
					idStr := strconv.Itoa(int(id))
					
					var isMajor bool
					var isVidlink bool
					var wgItem sync.WaitGroup
					
					wgItem.Add(2)
					go func() {
						defer wgItem.Done()
						isMajor = checkMajorProvider(media, idStr)
					}()
					go func() {
						defer wgItem.Done()
						isVidlink = checkVidlinkAvailability(media, idStr)
					}()
					wgItem.Wait()

					if !isMajor || !isVidlink {
						keep[index] = false
						return
					}
				}
				keep[index] = true

				err := SubmitBackgroundTask(TaskTypeDataProcess, map[string]interface{}{
					"type":   "enhance_result",
					"data":   data,
					"media":  media,
					"index":  index,
				})
				if err != nil {
					enhanceResultItem(data, media)
				}
			}(i, item)
		} else {
			keep[i] = true
		}
	}

	wg.Wait()

	var filtered []interface{}
	for i, k := range keep {
		if k {
			filtered = append(filtered, results[i])
		}
	}
	return filtered
}

func enhanceResultItem(item map[string]interface{}, media string) {

	if id, ok := item["id"].(float64); ok {
		idStr := strconv.Itoa(int(id))

		go func() {
			available := checkVidlinkAvailability(media, idStr)

			if cache != nil {
				cacheKey := "vidlink:" + media + ":" + idStr
				cache.SetJSON(cacheKey, map[string]bool{"available": available}, 1*time.Hour)
			}
		}()

		go func() {
			hasMajorProvider := checkMajorProvider(media, idStr)
			if cache != nil {
				cacheKey := "provider:" + media + ":" + idStr
				cache.SetJSON(cacheKey, map[string]bool{"major_provider": hasMajorProvider}, 6*time.Hour)
			}
		}()
	}

	sanitizeTMDBData(item, media)
}

func handleSearchParallel(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		query = r.URL.Query().Get("query")
	}
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
		"&page=" + page + "&include_adult=false&api_key=" + getTmdbApiKey()

	data, err := fetchJSON(targetUrl, "multi")
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		writeJSON(w, `{"error":"TMDB API error"}`)
		return
	}

	if results, ok := data["results"].([]interface{}); ok {
		processSearchResultsParallel(results)
	}

	responseJSON, _ := json.Marshal(data)
	writeJSON(w, string(responseJSON))
}

func processSearchResultsParallel(results []interface{}) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 8) 

	for _, result := range results {
		if item, ok := result.(map[string]interface{}); ok {
			wg.Add(1)

			go func(data map[string]interface{}) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				err := SubmitBackgroundTask(TaskTypeDataProcess, map[string]interface{}{
					"type": "process_search_result",
					"data": data,
				})

				if err != nil {

					processSearchResult(data)
				}
			}(item)
		}
	}

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

func processSearchResult(item map[string]interface{}) {

	mediaType := "movie" 
	if mt, ok := item["media_type"].(string); ok {
		mediaType = mt
	}

	sanitizeTMDBData(item, mediaType)

	recordContentView(mediaType)
}

func processTMDBBatch(ids []string, media string, query url.Values) map[string]interface{} {
	results := make(map[string]interface{})
	var mu sync.Mutex
	var wg sync.WaitGroup

	maxWorkers := 10
	if len(ids) < maxWorkers {
		maxWorkers = len(ids)
	}

	jobQueue := make(chan string, len(ids))

	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobQueue {

				base := tmdbBaseUrl + "/" + media + "/" + id
				imageLangs := getImageLangs(query)
				targetUrl := base + "?language=" + getTmdbLang(query) + 
					"&append_to_response=credits,videos,recommendations,similar,images" +
					"&include_image_language=" + imageLangs + "&api_key=" + getTmdbApiKey()

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

	for _, id := range ids {
		jobQueue <- id
	}
	close(jobQueue)

	wg.Wait()

	return results
}

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

func processRecommendationsParallel(data map[string]interface{}, media string) {
	if recommendations, ok := data["recommendations"].(map[string]interface{}); ok {
		if results, ok := recommendations["results"].([]interface{}); ok {

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