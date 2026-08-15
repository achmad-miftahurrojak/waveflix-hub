package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

const tmdbBaseUrl = "https://api.themoviedb.org/3"
const traktBaseUrl = "https://api.trakt.tv"
const watchRegion = "ID" // Indonesia

// Kredensial diisi dari .env / environment saat startup (lihat loadConfig).
var (
	tmdbApiKey        string
	traktClientID     string
	traktClientSecret string
)

// loadConfig memuat .env lalu mengisi kredensial dari environment.
func loadConfig() {
	loadDotEnv(".env")
	// Fallback TMDB key lama supaya app tetap jalan kalau .env belum diisi.
	tmdbApiKey = getenv("TMDB_API_KEY", "cff0f315183dd0830f0ef2ef924ae25c")
	traktClientID = getenv("TRAKT_CLIENT_ID", "")
	traktClientSecret = getenv("TRAKT_CLIENT_SECRET", "")

	if traktClientID == "" {
		log.Println("[config] TRAKT_CLIENT_ID kosong — endpoint Trakt belum aktif. Isi di backend/.env")
	} else {
		log.Println("[config] Trakt client_id termuat ✓")
	}
}

// Bahasa asli dominan per negara → memperketat filter "produksi negara X".
// India (IN) sengaja tidak dimasukkan karena bahasanya beragam.
var countryLang = map[string]string{
	"ID": "id", // Indonesia
	"US": "en", // Amerika Serikat
	"KR": "ko", // Korea
	"JP": "ja", // Jepang
	"CN": "zh", // China
	"GB": "en", // Inggris
	"TH": "th", // Thailand
}

func normalizeMedia(m string) string {
	if m == "tv" {
		return "tv"
	}
	return "movie"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// ============================================================================
// TRENDING  ->  /api/trending?media=all|movie|tv&page=1  (global, dipakai fallback)
// ============================================================================
func handleTrending(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	media := q.Get("media")
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
//               &country=US&sort_by=popularity.desc&page=1&released_before=...
// ============================================================================
func handleDiscover(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	media := normalizeMedia(q.Get("media"))
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
		// Perketat: cocokkan bahasa asli ke negaranya supaya co-production /
		// judul yang salah-label origin tidak ikut muncul. India (banyak bahasa)
		// sengaja dilewati → cukup pakai origin_country saja.
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
	media := normalizeMedia(q.Get("media"))
	id := q.Get("id")
	if id == "" {
		writeJSON(w, `{"error":"id kosong"}`)
		return
	}

	base := tmdbBaseUrl + "/" + media + "/" + id
	data, err := fetchJSON(base + "?language=en-US&append_to_response=credits,videos,recommendations,similar&api_key=" + tmdbApiKey)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		writeJSON(w, `{"error":"gagal ambil detail"}`)
		return
	}

	// Fallback deskripsi ke bahasa Inggris kalau versi Indonesia kosong.
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

// ============================================================================
// SEASON  ->  /api/season?id=123&season=1   (daftar episode satu musim)
// ============================================================================
func handleSeason(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := q.Get("id")
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
	media := normalizeMedia(q.Get("media"))
	id := q.Get("id")
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
var httpClient = http.Client{Timeout: 10 * time.Second}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(body))
}

func fetchJSON(targetUrl string) (map[string]interface{}, error) {
	resp, err := httpClient.Get(targetUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func proxyRequest(w http.ResponseWriter, targetUrl string) {
	resp, err := httpClient.Get(targetUrl)
	if err != nil {
		log.Printf("Gagal fetch TMDB: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, `{"error":"gagal menghubungi TMDB"}`)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func main() {
	loadConfig()
	initDB()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/trending", handleTrending)
	mux.HandleFunc("/api/homepage", handleHomepage)
	mux.HandleFunc("/api/discover", handleDiscover)
	mux.HandleFunc("/api/detail", handleDetail)
	mux.HandleFunc("/api/season", handleSeason)
	mux.HandleFunc("/api/images", handleImages)
	mux.HandleFunc("/api/search", handleSearch)
	mux.HandleFunc("/api/stream", handleStream)

	// Auth
	mux.HandleFunc("/api/auth/register", handleRegister)
	mux.HandleFunc("/api/auth/login", handleLogin)
	mux.HandleFunc("/api/auth/me", requireAuth(handleMe))

	// Data user (butuh token)
	mux.HandleFunc("/api/watchlist", requireAuth(handleWatchlist))
	mux.HandleFunc("/api/favorites", requireAuth(handleFavorites))
	mux.HandleFunc("/api/history", requireAuth(handleHistory))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"service":"waveflix-api","status":"ok"}`)
	})

	handler := enableCORS(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Summer Tide API berjalan di http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server gagal: %v", err)
	}
}

func enableCORS(next http.Handler) http.Handler {
	allowedOrigins := map[string]bool{
		"http://localhost:3000": true,
		"http://127.0.0.1:3000": true,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
