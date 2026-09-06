package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type cachedStream struct {
	body      []byte
	expiresAt time.Time
}

var (
	streamCache   = make(map[string]*cachedStream)
	streamCacheMu sync.RWMutex
	requestGroup  singleflight.Group
)

func getCachedStream(key string) ([]byte, bool) {
	streamCacheMu.RLock()
	defer streamCacheMu.RUnlock()
	c, ok := streamCache[key]
	if !ok || time.Now().After(c.expiresAt) {
		return nil, false
	}
	return c.body, true
}

func setCachedStream(key string, body []byte, ttl time.Duration) {
	streamCacheMu.Lock()
	defer streamCacheMu.Unlock()
	streamCache[key] = &cachedStream{body: body, expiresAt: time.Now().Add(ttl)}
}

// HandleStreamAPI acts as a bridge to the external extractor microservice.
// Caches successful stream responses for 3 hours (token validity ~4h).
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

	cacheKey := fmt.Sprintf("%s:%s:%s:%s", media, id, season, episode)
	if cached, ok := getCachedStream(cacheKey); ok {
		w.Write(cached)
		return
	}

	// Use singleflight to prevent duplicate simultaneous extractions
	v, err, _ := requestGroup.Do(cacheKey, func() (interface{}, error) {
		// Check cache again just in case it was populated while waiting
		if cached, ok := getCachedStream(cacheKey); ok {
			return cached, nil
		}

		var extractorURL string
		if media == "movie" {
			extractorURL = fmt.Sprintf("http://localhost:8000/stream?media=movie&id=%s", id)
		} else if media == "tv" {
			extractorURL = fmt.Sprintf("http://localhost:8000/stream?media=tv&id=%s&season=%s&episode=%s", id, season, episode)
		}

		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Get(extractorURL)
		if err != nil {
			return nil, fmt.Errorf(`{"error": "Streaming service unavailable."}`)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf(`{"error": "Failed to read stream response"}`)
		}

		if resp.StatusCode == http.StatusOK {
			var parsed map[string]interface{}
			if json.Unmarshal(body, &parsed) == nil {
				if sources, ok := parsed["sources"]; ok {
					if arr, ok := sources.([]interface{}); ok && len(arr) > 0 {
						setCachedStream(cacheKey, body, 3*time.Hour)
					}
				}
			}
			return body, nil
		}
		
		return body, fmt.Errorf("%s", string(body))
	})

	if err != nil {
		errStr := err.Error()
		if len(errStr) > 0 && errStr[0] == '{' {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(errStr))
		} else {
			http.Error(w, errStr, http.StatusServiceUnavailable)
		}
		return
	}

	w.Write(v.([]byte))
}

// HandleMediaProxy proxies media content (MP4/HLS) with proper Referer headers.
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
	req.Header.Set("Referer", "https://vidsrc.in/")
	req.Header.Set("Origin", "https://vidsrc.in")

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

func (tc *TMDBClient) CheckEpisodeAvailability(media, id, season, episode string) bool {
	return true
}
