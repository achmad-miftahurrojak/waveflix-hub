package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

const tmdbApiKey = "cff0f315183dd0830f0ef2ef924ae25c"
const tmdbBaseUrl = "https://api.themoviedb.org/3"

func handleHomepage(w http.ResponseWriter, r *http.Request) {
	page := r.URL.Query().Get("page")
	if page == "" {
		page = "1"
	}

	targetUrl := tmdbBaseUrl + "/trending/all/day?language=id-ID&page=" + page + "&api_key=" + tmdbApiKey
	proxyRequest(w, targetUrl)
}

func handleDiscover(w http.ResponseWriter, r *http.Request) {
	page := r.URL.Query().Get("page")
	if page == "" {
		page = "1"
	}
	genre := r.URL.Query().Get("genre")
	
	targetUrl := tmdbBaseUrl + "/discover/movie?language=id-ID&page=" + page + "&with_genres=" + genre + "&api_key=" + tmdbApiKey
	proxyRequest(w, targetUrl)
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	// [HARDCORE ROUTE] Ini adalah kerangka Scraper M3U8
	// Idealnya ini akan mem-bypass Cloudflare dan mengambil URL .m3u8 dari provider.
	// Karena keterbatasan memori (tidak bisa run headless browser Playwright), 
	// kita mock / gunakan stream publik untuk membuktikan arsitektur VideoJS + VTT berhasil.
	
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{
		"stream_url": "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4",
		"subtitle_url": "/dummy_sub.vtt"
	}`))
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":[]}`))
		return
	}

	searchQuery := url.QueryEscape(query)
	targetUrl := tmdbBaseUrl + "/search/multi?query=" + searchQuery + "&language=id-ID&page=1&api_key=" + tmdbApiKey
	proxyRequest(w, targetUrl)
}

func proxyRequest(w http.ResponseWriter, targetUrl string) {
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(targetUrl)
	if err != nil {
		log.Printf("Gagal fetch TMDB: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	for k, v := range resp.Header {
		if k != "Access-Control-Allow-Origin" && k != "Content-Security-Policy" {
			w.Header()[k] = v
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/homepage", handleHomepage)
	mux.HandleFunc("/api/discover", handleDiscover)
	mux.HandleFunc("/api/search", handleSearch)
	mux.HandleFunc("/api/stream", handleStream)

	// Melayani file statis dari folder frontend
	mux.Handle("/", http.FileServer(http.Dir("../frontend")))

	handler := enableCORS(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Summer Tide Backend berjalan di http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server gagal: %v", err)
	}
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// [SECURITY FIX] Hanya izinkan origin tertentu
		origin := r.Header.Get("Origin")
		allowedOrigins := []string{
			"http://localhost:8080",
			"http://127.0.0.1:8080",
		}
		
		isAllowed := false
		for _, o := range allowedOrigins {
			if o == origin {
				isAllowed = true
				break
			}
		}

		if isAllowed || origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		w.Header().Set("Content-Security-Policy", "default-src 'self' 'unsafe-inline' https://api.themoviedb.org https://image.tmdb.org https://ui-avatars.com https://images.unsplash.com https://fonts.googleapis.com https://fonts.gstatic.com https://cdnjs.cloudflare.com https://vjs.zencdn.net https://test-streams.mux.dev https://brenopolanski.github.io https://commondatastorage.googleapis.com; frame-src *; media-src 'self' https://test-streams.mux.dev https://commondatastorage.googleapis.com blob:;")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
