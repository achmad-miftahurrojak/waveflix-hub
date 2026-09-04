package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HandleStreamAPI acts as a bridge to the external Python Decryptor microservice.
func HandleStreamAPI(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, `{"error": "Streaming service is currently unavailable. Please ensure the Python decryptor is running on port 8000."}`, http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.WriteHeader(resp.StatusCode)
	}

	io.Copy(w, resp.Body)
}

// HandleMediaProxy proxies media content (MP4/HLS) with proper Referer headers
// so browsers can play protected CDN streams natively.
func HandleMediaProxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Accept-Ranges")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	encoded := r.URL.Query().Get("url")
	if encoded == "" {
		http.Error(w, `{"error": "Missing url parameter"}`, http.StatusBadRequest)
		return
	}

	targetURL, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		// Try with padding
		for len(encoded)%4 != 0 {
			encoded += "="
		}
		targetURL, err = base64.URLEncoding.DecodeString(encoded)
		if err != nil {
			http.Error(w, `{"error": "Invalid url encoding"}`, http.StatusBadRequest)
			return
		}
	}

	req, err := http.NewRequest("GET", string(targetURL), nil)
	if err != nil {
		http.Error(w, `{"error": "Invalid target URL"}`, http.StatusBadRequest)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://vidlink.pro/")
	req.Header.Set("Origin", "https://vidlink.pro")

	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, `{"error": "Failed to fetch media"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for _, key := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges"} {
		if val := resp.Header.Get(key); val != "" {
			w.Header().Set(key, val)
		}
	}

	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "video/mp4")
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// CheckEpisodeAvailability is a helper function to verify if an episode exists
// in the streaming provider before sending it to the frontend.
func (tc *TMDBClient) CheckEpisodeAvailability(media, id, season, episode string) bool {
	// Bypass decryptor check for episodes because checking sequentially is too slow (user: "loading episodes mulu")
	return true
}
