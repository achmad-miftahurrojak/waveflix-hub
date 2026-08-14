package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

// In-memory cache untuk menyimpan link iframe yang sudah di-scrape
var (
	videoCache = make(map[string]CacheItem)
	cacheMutex sync.RWMutex
	
	// Global Browser Pool untuk mencegah Memory Leak / OOM
	globalBrowser *rod.Browser
)

type CacheItem struct {
	IframeURL string
	ExpiresAt time.Time
}

type PlayResponse struct {
	Title     string `json:"title"`
	Year      string `json:"year"`
	Type      string `json:"type,omitempty"`
	Season    string `json:"season,omitempty"`
	Episode   string `json:"episode,omitempty"`
	IframeUrl string `json:"iframeUrl"`
	Error     string `json:"error,omitempty"`
}

type ListResponse struct {
	Titles []string `json:"titles"`
}

var (
	homepageCache     []string
	homepageCacheTime time.Time
	homepageMutex     sync.RWMutex
)

func handleHomepage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	homepageMutex.RLock()
	if len(homepageCache) > 0 && time.Since(homepageCacheTime) < 1*time.Hour {
		titles := homepageCache
		homepageMutex.RUnlock()
		json.NewEncoder(w).Encode(ListResponse{Titles: titles})
		return
	}
	homepageMutex.RUnlock()

	titles, err := scrapeList("") // empty means homepage
	if err != nil {
		log.Printf("Gagal scrape homepage: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	homepageMutex.Lock()
	homepageCache = titles
	homepageCacheTime = time.Now()
	homepageMutex.Unlock()

	json.NewEncoder(w).Encode(ListResponse{Titles: titles})
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	w.Header().Set("Content-Type", "application/json")

	if query == "" {
		json.NewEncoder(w).Encode(ListResponse{Titles: []string{}})
		return
	}

	titles, err := scrapeList(query)
	if err != nil {
		log.Printf("Gagal scrape search %s: %v", query, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(ListResponse{Titles: titles})
}

func scrapeList(query string) ([]string, error) {
	if globalBrowser == nil {
		return nil, fmt.Errorf("browser global belum siap")
	}

	page := stealth.MustPage(globalBrowser)
	defer page.MustClose()

	domain := os.Getenv("IDLIX_DOMAIN")
	if domain == "" {
		domain = "z2.idlixku.com"
	}

	var targetUrl string
	if query == "" {
		targetUrl = fmt.Sprintf("https://%s/", domain)
	} else {
		searchQuery := url.QueryEscape(query)
		targetUrl = fmt.Sprintf("https://%s/?s=%s", domain, searchQuery)
	}

	log.Printf("[SCRAPING LIST] Membuka URL: %s", targetUrl)
	err := page.Timeout(15 * time.Second).Navigate(targetUrl)
	if err != nil {
		return nil, fmt.Errorf("gagal navigasi: %v", err)
	}

	page.MustWaitLoad()
	time.Sleep(3 * time.Second) // Tunggu Cloudflare

	var titles []string
	elements, err := page.Timeout(10 * time.Second).Elements("article.item, .result-item, .item")
	if err != nil {
		log.Printf("Tidak ada item ditemukan (mungkin kosong atau diblokir)")
		return titles, nil
	}

	for _, el := range elements {
		titleEl, err := el.Element("h3, .title, h2")
		if err == nil {
			txt, err := titleEl.Text()
			if err == nil && txt != "" {
				titles = append(titles, strings.TrimSpace(txt))
			}
		}
	}

	if len(titles) > 15 {
		titles = titles[:15]
	}
	return titles, nil
}

func main() {
	// [BUG FIX] Luncurkan 1 Browser Global saja agar tidak OOM
	l := launcher.New().
		Leakless(false).
		Headless(true).
		Set("disable-blink-features", "AutomationControlled")
	urlLauncher, err := l.Launch()
	if err != nil {
		log.Fatalf("Gagal meluncurkan browser: %v", err)
	}
	defer l.Cleanup()

	globalBrowser = rod.New().ControlURL(urlLauncher).MustConnect()
	defer globalBrowser.MustClose()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/play", handlePlay)
	mux.HandleFunc("/api/homepage", handleHomepage)
	mux.HandleFunc("/api/search", handleSearch)

	// [FEATURE] Serve file Frontend statis agar tidak 404 saat buka root URL
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
		// [SECURITY FIX] Hanya izinkan origin tertentu (atau origin request jika sesuai)
		origin := r.Header.Get("Origin")
		allowedOrigins := []string{
			"http://localhost:8080",
			"http://127.0.0.1:8080",
			// Tambahkan domain production Anda di sini nantinya
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
			// Fallback aman
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		// [SECURITY FIX] Menambahkan header CSP untuk mencegah XSS & iframe injection
		w.Header().Set("Content-Security-Policy", "default-src 'self' 'unsafe-inline' https://api.themoviedb.org https://image.tmdb.org https://ui-avatars.com https://images.unsplash.com https://fonts.googleapis.com https://fonts.gstatic.com https://cdnjs.cloudflare.com; frame-src *;")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handlePlay(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title")
	year := r.URL.Query().Get("year")
	mediaType := r.URL.Query().Get("type") // "movie" or "tv"
	season := r.URL.Query().Get("season")
	episode := r.URL.Query().Get("episode")

	if mediaType == "" {
		mediaType = "movie"
	}

	w.Header().Set("Content-Type", "application/json")

	if title == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(PlayResponse{Error: "Title is required"})
		return
	}

	cacheKey := fmt.Sprintf("%s-%s-%s-%s-%s", strings.ToLower(title), year, mediaType, season, episode)

	// Cek Cache
	cacheMutex.RLock()
	item, found := videoCache[cacheKey]
	cacheMutex.RUnlock()

	if found {
		if time.Now().Before(item.ExpiresAt) {
			log.Printf("[CACHE HIT] %s", title)
			json.NewEncoder(w).Encode(PlayResponse{
				Title:     title,
				Year:      year,
				Type:      mediaType,
				Season:    season,
				Episode:   episode,
				IframeUrl: item.IframeURL,
			})
			return
		} else {
			// [BUG FIX] Hapus item yang sudah expired dari map (Garbage Collection)
			cacheMutex.Lock()
			delete(videoCache, cacheKey)
			cacheMutex.Unlock()
			log.Printf("[CACHE EXPIRED] Menghapus %s dari memori", title)
		}
	}

	log.Printf("[SCRAPING START] Mencari video: %s (%s) Type: %s", title, year, mediaType)
	
	iframeUrl, err := scrapeIdlix(title, year, mediaType, season, episode)
	
	if err != nil || iframeUrl == "" {
		log.Printf("[SCRAPING FAILED] %s: %v", title, err)
		iframeUrl = "" // Kosongkan agar frontend menampilkan alert error
	} else {
		log.Printf("[SCRAPING SUCCESS] %s -> %s", title, iframeUrl)
		
		// Simpan ke Cache (Berlaku 2 Jam)
		cacheMutex.Lock()
		videoCache[cacheKey] = CacheItem{
			IframeURL: iframeUrl,
			ExpiresAt: time.Now().Add(2 * time.Hour),
		}
		cacheMutex.Unlock()
	}

	json.NewEncoder(w).Encode(PlayResponse{
		Title:     title,
		Year:      year,
		Type:      mediaType,
		Season:    season,
		Episode:   episode,
		IframeUrl: iframeUrl,
	})
}

// Fungsi helper membersihkan judul menjadi slug (contoh: "Deadpool & Wolverine" -> "deadpool-wolverine")
func createSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "&", "")
	slug = strings.ReplaceAll(slug, ":", "")
	slug = strings.ReplaceAll(slug, "'", "")
	slug = strings.ReplaceAll(slug, ".", "")
	slug = strings.ReplaceAll(slug, "--", "-")
	return strings.Trim(slug, "-")
}

func scrapeIdlix(title, year, mediaType, season, episode string) (string, error) {
	if globalBrowser == nil {
		return "", fmt.Errorf("browser global belum siap")
	}

	page := stealth.MustPage(globalBrowser)
	defer page.MustClose()

	domain := os.Getenv("IDLIX_DOMAIN")
	if domain == "" {
		domain = "z2.idlixku.com"
	}

	// [FEATURE] Smart Direct URL Navigation
	// Mencoba navigasi langsung tanpa lewat kolom pencarian (jauh lebih cepat dan akurat)
	slug := createSlug(title)
	var directUrl string

	if mediaType == "tv" {
		directUrl = fmt.Sprintf("https://%s/episode/%s-season-%s-episode-%s/", domain, slug, season, episode)
	} else {
		directUrl = fmt.Sprintf("https://%s/movie/%s-%s/", domain, slug, year)
	}

	log.Printf("[SCRAPING] Mencoba Direct URL: %s", directUrl)
	
	err := page.Timeout(15 * time.Second).Navigate(directUrl)
	if err != nil {
		return "", fmt.Errorf("gagal membuka direct URL: %v", err)
	}

	page.MustWaitLoad()
	time.Sleep(5 * time.Second) // Tunggu Cloudflare

	// Cek apakah halaman 404 (Not Found). Jika bukan 404, kita bisa langsung ambil iframe.
	pageTitle := ""
	if res, err := page.Eval("() => document.title"); err == nil {
		pageTitle = res.Value.Str()
	}
	if !strings.Contains(strings.ToLower(pageTitle), "page not found") && !strings.Contains(strings.ToLower(pageTitle), "error 404") {
		// Halaman ada! Langsung cari iframe.
		iframeElement, err := page.Timeout(5 * time.Second).Element("iframe")
		if err == nil {
			iframeSrc, err := iframeElement.Attribute("src")
			if err == nil && iframeSrc != nil {
				return *iframeSrc, nil
			}
		}
	}

	// [FALLBACK] Jika Direct URL gagal (404), gunakan Search Bar
	log.Printf("[SCRAPING FALLBACK] Direct URL gagal. Menggunakan pencarian web untuk: %s", title)
	
	searchQuery := url.QueryEscape(title)
	searchUrl := fmt.Sprintf("https://%s/?s=%s", domain, searchQuery)

	err = page.Timeout(15 * time.Second).Navigate(searchUrl)
	if err != nil {
		return "", fmt.Errorf("gagal membuka situs pencarian: %v", err)
	}

	page.MustWaitLoad()
	time.Sleep(5 * time.Second) // Tunggu Cloudflare lagi

	// 1. Cari elemen hasil pencarian pertama
	resultItem, err := page.Timeout(15 * time.Second).Element("article a, .result-item a, .item a")
	if err != nil {
		return "", fmt.Errorf("gagal menemukan hasil pencarian untuk %s: %v", title, err)
	}

	// 2. Klik hasil pencarian (Ini akan menuju halaman Movie, atau halaman utama Series)
	err = resultItem.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return "", fmt.Errorf("gagal mengeklik film: %v", err)
	}
	page.MustWaitLoad()

	// Jika ini adalah TV Series, kita saat ini berada di halaman Series Utama.
	// Kita harus mengklik Season dan Episode yang benar.
	if mediaType == "tv" {
		log.Printf("[SCRAPING TV] Mencari tombol episode %s season %s...", episode, season)
		// Struktur episode idlix biasanya ada di dalam <ul><li> dengan class 'episodiotitle' atau link berisi nomor episode.
		// Sangat sulit mencari tombol dinamis tanpa klik langsung, jadi kita akan mencoba extract semua link di halaman ini,
		// dan mencari URL yang cocok dengan pola /episode/judul-season-x-episode-y/
		
		links, err := page.Elements("a")
		if err == nil {
			targetPattern := fmt.Sprintf("season-%s-episode-%s", season, episode)
			found := false
			for _, link := range links {
				href, err := link.Attribute("href")
				if err == nil && href != nil && strings.Contains(*href, targetPattern) {
					log.Printf("[SCRAPING TV] Ditemukan link episode yang cocok: %s", *href)
					link.Click(proto.InputMouseButtonLeft, 1)
					page.MustWaitLoad()
					time.Sleep(3 * time.Second)
					found = true
					break
				}
			}
			if !found {
				return "", fmt.Errorf("gagal menemukan tombol season %s episode %s di halaman series", season, episode)
			}
		}
	}

	// 3. Halaman video seharusnya sudah terbuka (baik movie maupun episode series). Cari elemen iframe
	iframeElement, err := page.Timeout(5 * time.Second).Element("iframe")
	if err != nil {
		return "", fmt.Errorf("gagal menemukan iframe video: %v", err)
	}

	iframeSrc, err := iframeElement.Attribute("src")
	if err != nil || iframeSrc == nil {
		return "", fmt.Errorf("gagal mengekstrak src dari iframe")
	}

	return *iframeSrc, nil
}
