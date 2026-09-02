

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
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

	return result, nil
}

func sanitizeTMDBData(data map[string]interface{}, mediaType string) {

	delete(data, "adult")
	delete(data, "belongs_to_collection") 
}

func checkVidlinkAvailability(media, id string) bool {
	url := fmt.Sprintf("https://vidlink.pro/api/b/%s/%s", media, id)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Head(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func checkMajorProvider(media, id string) bool {
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

	idRegion, ok := results["ID"].(map[string]interface{})
	if !ok {
		return false
	}

	majorProviders := map[int]bool{
		8:    true, // Netflix
		119:  true, // Amazon Prime
		350:  true, // Apple TV
		122:  true, // Disney+
		158:  true, // Viu
		483:  true, // MAX Stream
		489:  true, // Vidio
		1899: true, // HBO Max
	}

	checkList := []string{"flatrate", "rent", "buy"}
	for _, t := range checkList {
		if list, ok := idRegion[t].([]interface{}); ok {
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
	return false
}

func filterByMajorProviderParallel(results []interface{}, defaultMedia string) []interface{} {
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

			if checkMajorProvider(media, idStr) {
				mu.Lock()
				filtered = append(filtered, item)
				mu.Unlock()
			}
		}(item)
	}

	wg.Wait()
	return filtered
}
