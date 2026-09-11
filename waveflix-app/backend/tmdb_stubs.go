package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
	"strings"
	"strconv"
)

const (
	TaskTypeTMDBAPI    = "tmdb_api"
	TaskTypeDataProcess = "data_process"
	TaskTypeEmailSend   = "email_send"
)

var tmdbBaseUrl = "https://api.themoviedb.org/3"

var tmdbApiKey = ""

func getTmdbApiKey() string {
	if tmdbApiKey != "" {
		return tmdbApiKey
	}
	tmdbApiKey = os.Getenv("TMDB_API_KEY")
	return tmdbApiKey
}

func getMediaParam(q url.Values) string {
	return normalizeMedia(q.Get("media"))
}

func getTmdbLang(q url.Values) string {
	if lang := q.Get("language"); lang != "" {
		return lang
	}
	if lang := q.Get("lang"); lang != "" {
		if lang == "en" {
			return "en-US"
		}
		return "id-ID"
	}
	return "id-ID"
}

func getImageLangs(q url.Values) string {
	if lang := q.Get("include_image_language"); lang != "" {
		return lang
	}
	return "en,null"
}

func fetchJSON(targetURL string, mediaType string) (map[string]interface{}, error) {
	// Generate cache key
	cacheKey := "tmdb:fetch:" + targetURL
	
	// Check cache
	if cache != nil {
		if cachedData, found := cache.Get(cacheKey); found && cachedData != nil {
			var result map[string]interface{}
			if err := json.Unmarshal(cachedData, &result); err == nil {
				return result, nil
			}
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(targetURL)
	if err != nil {
		return nil, fmt.Errorf("[tmdb] fetch error (%s): %w", mediaType, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[tmdb] non-200 response (%s): %d", mediaType, resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("[tmdb] decode error (%s): %w", mediaType, err)
	}

	// Save to cache
	if cache != nil {
		if jsonData, err := json.Marshal(result); err == nil {
			cache.Set(cacheKey, jsonData, 30*time.Minute)
		}
	}

	return result, nil
}

func sanitizeTMDBData(data map[string]interface{}, mediaType string) {

	delete(data, "adult")
	delete(data, "belongs_to_collection") 
}

func checkVidlinkAvailability(media, id string) bool {
	var url string
	if media == "movie" {
		url = fmt.Sprintf("https://vidlink.pro/movie/%s", id)
	} else {
		url = fmt.Sprintf("https://vidlink.pro/tv/%s/1/1", id)
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Head(url)
	if err != nil {
		return true // Default to true if network error so we don't break the site
	}
	defer resp.Body.Close()
	// 200 OK means it exists. 429 means we are rate limited (assume it exists to avoid empty catalog).
	// If it's a 404 or 500, it actually doesn't exist.
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusTooManyRequests
}

func checkMajorProvider(media, id string, requestedProvider string) bool {
	if tmdbClient == nil {
		return false
	}
	data, err := tmdbClient.GetWatchProviders(media, id)
	if err != nil {
		return false
	}

	results, ok := data["results"].(map[string]interface{})
	if !ok {
		return false
	}

	majorProviders := make(map[int]bool)
	if requestedProvider != "" {
		for _, p := range strings.Split(requestedProvider, "|") {
			if pid, err := strconv.Atoi(p); err == nil {
				majorProviders[pid] = true
			}
		}
	} else {
		majorProviders = map[int]bool{
			8:    true, // Netflix
			119:  true, // Amazon Prime
			350:  true, // Apple TV
			122:  true, // Disney+
			158:  true, // Viu
			483:  true, // MAX Stream
			489:  true, // Vidio
			1899: true, // HBO Max
		}
	}

	checkList := []string{"flatrate", "rent", "buy", "free", "ads"}
	for _, regionData := range results {
		if regionMap, ok := regionData.(map[string]interface{}); ok {
			for _, t := range checkList {
				if list, ok := regionMap[t].([]interface{}); ok {
					for _, item := range list {
						if provider, ok := item.(map[string]interface{}); ok {
							if pid, ok := provider["provider_id"].(float64); ok {
								if majorProviders[int(pid)] {
									return true
								}
							}
						}
					}
				}
			}
		}
	}

	return false
}

func filterByMajorProviderParallel(results []interface{}, defaultMedia string, requestedProvider string) []interface{} {
	var mu sync.Mutex
	filtered := make([]interface{}, 0, len(results))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)

	for _, item := range results {
		wg.Add(1)
		go func(item interface{}) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			data, ok := item.(map[string]interface{})
			if !ok {
				return
			}
			
			media := defaultMedia
			if mt, ok := data["media_type"].(string); ok {
				media = mt
			}

			if media == "person" {
				mu.Lock()
				filtered = append(filtered, item)
				mu.Unlock()
				return
			}
			
			idStr := ""
			if id, ok := data["id"].(float64); ok {
				idStr = fmt.Sprintf("%.0f", id)
			}
			
			if idStr == "" {
				return
			}

			var isMajor bool
			var isVidlink bool
			var wgItem sync.WaitGroup
			
			wgItem.Add(2)
			go func() {
				defer wgItem.Done()
				isMajor = checkMajorProvider(media, idStr, requestedProvider)
			}()
			go func() {
				defer wgItem.Done()
				isVidlink = checkVidlinkAvailability(media, idStr)
			}()
			wgItem.Wait()

			if !isMajor {
				fmt.Printf("[DEBUG] ID %s (%s) dropped by checkMajorProvider\n", idStr, media)
			}
			if !isVidlink {
				fmt.Printf("[DEBUG] ID %s (%s) dropped by checkVidlinkAvailability\n", idStr, media)
			}

			if isMajor && isVidlink {
				mu.Lock()
				filtered = append(filtered, item)
				mu.Unlock()
			}
		}(item)
	}

	wg.Wait()
	return filtered
}
