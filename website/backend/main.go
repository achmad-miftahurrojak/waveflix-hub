package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const tmdbBaseUrl = "https://api.themoviedb.org/3"
const traktBaseUrl = "https://api.trakt.tv"
const watchRegion = "ID" // Indonesia

var (
	tmdbApiKey        string
	traktClientID     string
	traktClientSecret string
)

func loadConfig() {
	loadDotEnv(".env")
	tmdbApiKey = getenv("TMDB_API_KEY", "")
	if tmdbApiKey == "" {
		log.Println("[config] WARNING: TMDB_API_KEY kosong. Call ke TMDB akan gagal. Isi di backend/.env")
	}
	traktClientID = getenv("TRAKT_CLIENT_ID", "")
	traktClientSecret = getenv("TRAKT_CLIENT_SECRET", "")

	if traktClientID == "" {
		log.Println("[config] TRAKT_CLIENT_ID kosong — endpoint Trakt belum aktif. Isi di backend/.env")
	} else {
		log.Println("[config] Trakt client_id termuat ✓")
	}
}

var countryLang = map[string]string{
	"ID": "id",
	"US": "en",
	"KR": "ko",
	"JP": "ja",
	"CN": "zh",
	"GB": "en",
	"TH": "th",
}

func normalizeMedia(m string) string {
	if m == "tv" {
		return "tv"
	}
	return "movie"
}

func getMediaParam(q url.Values) string {
	m := q.Get("media")
	if m == "" {
		m = q.Get("media_type")
	}
	return normalizeMedia(m)
}

func getIDParam(q url.Values) string {
	id := q.Get("id")
	if id == "" {
		id = q.Get("tmdb_id")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return "" // invalid non-numeric ID
		}
	}
	return id
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

type cacheItem struct {
	data     []byte
	expireAt time.Time
}

var (
	memCache = make(map[string]cacheItem)
	cacheMu  sync.RWMutex
)

func init() {
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			now := time.Now()
			cacheMu.Lock()
			for k, v := range memCache {
				if now.After(v.expireAt) {
					delete(memCache, k)
				}
			}
			cacheMu.Unlock()
		}
	}()
}

// ============================================================================
// TRENDING  ->  /api/trending?media=all|movie|tv&page=1  (global, dipakai fallback)
// ============================================================================
func handleTrending(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	media := q.Get("media")
	if media == "" {
		media = q.Get("media_type")
	}
	if media != "movie" && media != "tv" {
		media = "all"
	}
	page := firstNonEmpty(q.Get("page"), "1")

	p := url.Values{}
	p.Set("api_key", tmdbApiKey)
	p.Set("language", "en-US")
	p.Set("region", watchRegion)
	p.Set("page", page)

	proxyRequest(w, tmdbBaseUrl+"/trending/"+media+"/day?"+p.Encode())
}

func handleHomepage(w http.ResponseWriter, r *http.Request) { handleTrending(w, r) }

// ============================================================================
// DISCOVER  ->  /api/discover?media=movie&provider=8&genre=28&year=2024
//
//	&country=US&sort_by=popularity.desc&page=1&released_before=...
//
// ============================================================================
func handleDiscover(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	media := getMediaParam(q)
	page := firstNonEmpty(q.Get("page"), "1")

	p := url.Values{}
	p.Set("api_key", tmdbApiKey)
	p.Set("language", "en-US")
	p.Set("watch_region", watchRegion)
	p.Set("page", page)
	p.Set("sort_by", firstNonEmpty(q.Get("sort_by"), "popularity.desc"))
	p.Set("include_adult", "false")
	p.Set("vote_count.gte", firstNonEmpty(q.Get("min_votes"), "0"))

	if v := q.Get("provider"); v != "" {
		p.Set("with_watch_providers", v)
	}
	if v := q.Get("genre"); v != "" {
		p.Set("with_genres", v)
	}
	if v := q.Get("country"); v != "" {
		p.Set("with_origin_country", v)
		if lang, ok := countryLang[v]; ok {
			p.Set("with_original_language", lang)
		}
	}
	if v := q.Get("year"); v != "" {
		if media == "tv" {
			p.Set("first_air_date_year", v)
		} else {
			p.Set("primary_release_year", v)
		}
	}
	if v := q.Get("released_before"); v != "" {
		if media == "tv" {
			p.Set("first_air_date.lte", v)
		} else {
			p.Set("primary_release_date.lte", v)
		}
		if q.Get("min_votes") == "" {
			p.Set("vote_count.gte", "20")
		}
	}
	if v := q.Get("released_after"); v != "" {
		if media == "tv" {
			p.Set("first_air_date.gte", v)
		} else {
			p.Set("primary_release_date.gte", v)
		}
	}

	proxyRequest(w, tmdbBaseUrl+"/discover/"+media+"?"+p.Encode())
}

// ============================================================================
// DETAIL  ->  /api/detail?media=movie&id=123
// Menyertakan credits (cast). Kalau overview (en-US) kosong -> fallback en-US.
// ============================================================================
func handleDetail(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	media := getMediaParam(q)
	id := getIDParam(q)
	if id == "" {
		writeJSON(w, `{"error":"id kosong"}`)
		return
	}

	base := tmdbBaseUrl + "/" + media + "/" + id
	data, err := fetchJSON(base + "?language=en-US&append_to_response=credits,videos,recommendations,similar,images&include_image_language=en,null&api_key=" + tmdbApiKey)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		writeJSON(w, `{"error":"gagal ambil detail"}`)
		return
	}

	if ov, _ := data["overview"].(string); ov == "" {
		if en, err := fetchJSON(base + "?language=en-US&api_key=" + tmdbApiKey); err == nil {
			if enOv, ok := en["overview"].(string); ok && enOv != "" {
				data["overview"] = enOv
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func handleDetailBatch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	media := getMediaParam(q)
	idsStr := q.Get("ids")
	if idsStr == "" {
		writeJSON(w, `{"error":"ids kosong"}`)
		return
	}
	ids := strings.Split(idsStr, ",")

	type result struct {
		id   string
		data map[string]interface{}
		err  error
	}
	ch := make(chan result, len(ids))

	for _, id := range ids {
		go func(id string) {
			base := tmdbBaseUrl + "/" + media + "/" + id
			targetUrl := base + "?language=en-US&append_to_response=credits,videos,recommendations,similar,images&include_image_language=en,null&api_key=" + tmdbApiKey
			data, err := fetchJSON(targetUrl)
			if err != nil {
				ch <- result{id: id, err: err}
				return
			}

			if ov, _ := data["overview"].(string); ov == "" {
				if en, err := fetchJSON(base + "?language=en-US&api_key=" + tmdbApiKey); err == nil {
					if enOv, ok := en["overview"].(string); ok && enOv != "" {
						data["overview"] = enOv
					}
				}
			}
			ch <- result{id: id, data: data}
		}(id)
	}

	out := make(map[string]interface{})
	for i := 0; i < len(ids); i++ {
		res := <-ch
		if res.err == nil {
			out[res.id] = res.data
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// ============================================================================
// SEASON  ->  /api/season?id=123&season=1   (daftar episode satu musim)
// ============================================================================
func handleSeason(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := getIDParam(q)
	season := firstNonEmpty(q.Get("season"), "1")
	if id == "" {
		writeJSON(w, `{"episodes":[]}`)
		return
	}
	proxyRequest(w, tmdbBaseUrl+"/tv/"+id+"/season/"+season+"?language=en-US&api_key="+tmdbApiKey)
}

// ============================================================================
// IMAGES  ->  /api/images?media=movie&id=123   (logo hero, en+null)
// ============================================================================
func handleImages(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	media := getMediaParam(q)
	id := getIDParam(q)
	if id == "" {
		writeJSON(w, `{"logos":[]}`)
		return
	}
	proxyRequest(w, tmdbBaseUrl+"/"+media+"/"+id+"/images?include_image_language=en,null&api_key="+tmdbApiKey)
}

// ============================================================================
// SEARCH  ->  /api/search?q=...
// ============================================================================
func handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, `{"results":[]}`)
		return
	}
	page := firstNonEmpty(r.URL.Query().Get("page"), "1")
	targetUrl := tmdbBaseUrl + "/search/multi?query=" + url.QueryEscape(query) +
		"&language=en-US&page=" + page + "&include_adult=false&api_key=" + tmdbApiKey
	proxyRequest(w, targetUrl)
}

// ============================================================================
// STREAM (mock — kerangka scraper M3U8)
// ============================================================================
func handleStream(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, `{
		"stream_url": "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4",
		"subtitle_url": "/dummy_sub.vtt"
	}`)
}

// ============================================================================
// HELPERS
// ============================================================================
var httpClient = http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	},
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(body))
}

func fetchJSON(targetUrl string) (map[string]interface{}, error) {
	cacheKey := "json:" + targetUrl
	cacheMu.RLock()
	if item, exists := memCache[cacheKey]; exists && time.Now().Before(item.expireAt) {
		cacheMu.RUnlock()
		var out map[string]interface{}
		json.Unmarshal(item.data, &out)
		return out, nil
	}
	cacheMu.RUnlock()

	resp, err := httpClient.Get(targetUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	cacheMu.Lock()
	memCache[cacheKey] = cacheItem{
		data:     bodyBytes,
		expireAt: time.Now().Add(15 * time.Minute),
	}
	cacheMu.Unlock()

	var out map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func proxyRequest(w http.ResponseWriter, targetUrl string) {
	cacheKey := "proxy:" + targetUrl
	cacheMu.RLock()
	if item, exists := memCache[cacheKey]; exists && time.Now().Before(item.expireAt) {
		cacheMu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(item.data)
		return
	}
	cacheMu.RUnlock()

	resp, err := httpClient.Get(targetUrl)
	if err != nil {
		log.Printf("Gagal fetch TMDB: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, `{"error":"gagal menghubungi TMDB"}`)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	cacheMu.Lock()
	memCache[cacheKey] = cacheItem{
		data:     bodyBytes,
		expireAt: time.Now().Add(15 * time.Minute),
	}
	cacheMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(bodyBytes)
}

func main() {
	loadConfig()
	initJWTSecret()
	initDB()

	os.MkdirAll("./uploads", 0755)

	mux := http.NewServeMux()
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))
	mux.HandleFunc("/api/trending", handleTrending)
	mux.HandleFunc("/api/homepage", handleHomepage)
	mux.HandleFunc("/api/discover", handleDiscover)
	mux.HandleFunc("/api/detail", handleDetail)
	mux.HandleFunc("/api/detail-batch", handleDetailBatch)
	mux.HandleFunc("/api/season", handleSeason)
	mux.HandleFunc("/api/images", handleImages)
	mux.HandleFunc("/api/search", handleSearch)
	mux.HandleFunc("/api/stream", handleStream)

	// Auth
	mux.HandleFunc("/api/auth/register", rateLimit(3, time.Minute, handleRegister))
	mux.HandleFunc("/api/auth/login", rateLimit(5, time.Minute, handleLogin))
	mux.HandleFunc("/api/auth/check-email", rateLimit(5, time.Minute, handleCheckEmail))
	mux.HandleFunc("/api/auth/me", requireAuth(handleMe))
	mux.HandleFunc("/api/auth/profile", requireAuth(handleUpdateProfile))
	mux.HandleFunc("/api/auth/password", requireAuth(handleChangePassword))
	mux.HandleFunc("/api/auth/avatar", requireAuth(uploadImage("avatar")))
	mux.HandleFunc("/api/auth/banner", requireAuth(uploadImage("banner")))

	// Data user
	mux.HandleFunc("/api/watchlist", requireAuth(handleWatchlist))
	mux.HandleFunc("/api/favorites", requireAuth(handleFavorites))
	mux.HandleFunc("/api/history", requireAuth(handleHistory))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			writeJSON(w, `{"status":"error","database":"disconnected"}`)
			return
		}
		writeJSON(w, `{"status":"ok","database":"connected"}`)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"service":"waveflix-api","status":"ok"}`)
	})

	handler := enableCORS(mux)
	handler = secureHeaders(handler)
	handler = recoveryMiddleware(handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Summer Tide API berjalan di http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server gagal: %v", err)
	}
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				writeJSON(w, `{"error":"internal server error"}`)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

func enableCORS(next http.Handler) http.Handler {
	allowedOrigins := map[string]bool{
		"http://localhost:3000": true,
		"http://127.0.0.1:3000": true,
	}
	if envOrigin := os.Getenv("ALLOWED_ORIGIN"); envOrigin != "" {
		allowedOrigins[envOrigin] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if origin != "" && len(allowedOrigins) == 2 { // fallback untuk dev lokal jika belum set origin produksi
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
