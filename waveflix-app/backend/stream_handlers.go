package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// HandleStreamAPI acts as a bridge to the external Python Decryptor microservice.
func HandleStreamAPI(w http.ResponseWriter, r *http.Request) {
	// Enable CORS if needed (already handled by global middleware if any, but let's be safe)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	media := r.URL.Query().Get("media")
	id := r.URL.Query().Get("id")
	season := r.URL.Query().Get("s")
	episode := r.URL.Query().Get("e")

	if media == "" || id == "" {
		http.Error(w, `{"error": "Missing media or id"}`, http.StatusBadRequest)
		return
	}

	var pythonServiceURL string
	if media == "movie" {
		pythonServiceURL = fmt.Sprintf("http://localhost:8000/stream?media=movie&id=%s", id)
	} else if media == "tv" {
		pythonServiceURL = fmt.Sprintf("http://localhost:8000/stream?media=tv&id=%s&season=%s&episode=%s", id, season, episode)
	} else {
		http.Error(w, `{"error": "Invalid media type"}`, http.StatusBadRequest)
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(pythonServiceURL)
	if err != nil {
		// Python service is down or unreachable
		http.Error(w, `{"error": "Streaming service is currently unavailable. Please ensure the Python decryptor is running on port 8000."}`, http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.WriteHeader(resp.StatusCode)
	}

	// Proxy the response directly to the frontend
	io.Copy(w, resp.Body)
}

// CheckEpisodeAvailability is a helper function to verify if an episode exists
// in the streaming provider before sending it to the frontend.
func (tc *TMDBClient) CheckEpisodeAvailability(media, id, season, episode string) bool {
	cacheKey := fmt.Sprintf("stream_avail:%s:%s:%s:%s", media, id, season, episode)
	var available bool
	if tc.cache.GetJSON(cacheKey, &available) {
		return available
	}

	var pythonServiceURL string
	if media == "movie" {
		pythonServiceURL = fmt.Sprintf("http://localhost:8000/stream?media=movie&id=%s", id)
	} else if media == "tv" {
		pythonServiceURL = fmt.Sprintf("http://localhost:8000/stream?media=tv&id=%s&season=%s&episode=%s", id, season, episode)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		resp, err := client.Get(pythonServiceURL)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		
		status := resp.StatusCode
		resp.Body.Close()
		
		if status == http.StatusOK {
			tc.cache.SetJSON(cacheKey, true, 24*time.Hour)
			return true
		} else if status == 404 {
			tc.cache.SetJSON(cacheKey, false, 24*time.Hour)
			return false
		} else if status == 429 || status == 500 {
			// Rate limited or Python error, wait and retry with jitter
			jitter := time.Duration(100 + (time.Now().UnixNano() % 500)) * time.Millisecond
			time.Sleep(time.Duration(1+i) * time.Second + jitter)
			continue
		}
		
		tc.cache.SetJSON(cacheKey, false, 2*time.Hour)
		return false
	}

	// If all retries failed, default to false
	tc.cache.SetJSON(cacheKey, false, 2*time.Hour)
	return false
}
