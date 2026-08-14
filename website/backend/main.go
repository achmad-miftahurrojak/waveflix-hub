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
	IframeUrl string `json:"iframeUrl"`
	Error     string `json:"error,omitempty"`
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
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

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

	w.Header().Set("Content-Type", "application/json")

	if title == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(PlayResponse{Error: "Title is required"})
		return
	}

	cacheKey := fmt.Sprintf("%s-%s", strings.ToLower(title), year)

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

	log.Printf("[SCRAPING START] Mencari video: %s (%s)", title, year)
	
	iframeUrl, err := scrapeIdlix(title, year)
	
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
		IframeUrl: iframeUrl,
	})
}

func scrapeIdlix(title, year string) (string, error) {
	if globalBrowser == nil {
		return "", fmt.Errorf("browser global belum siap")
	}

	// [BUG FIX] Buat tab (Page) baru dari instance browser global yang sudah ada
	// Jauh lebih ringan di RAM daripada membuka browser baru
	page := stealth.MustPage(globalBrowser)
	defer page.MustClose()

	// [BUG FIX] Pindahkan domain hardcode ke Environment Variable
	domain := os.Getenv("IDLIX_DOMAIN")
	if domain == "" {
		domain = "tv.idlixofficial.co"
	}
	searchQuery := url.QueryEscape(title)
	searchUrl := fmt.Sprintf("https://%s/?s=%s", domain, searchQuery)

	log.Printf("[SCRAPING] Membuka URL: %s", searchUrl)

	// Timeout 15 detik agar tidak menggantung selamanya jika terkena Cloudflare Captcha
	err := page.Timeout(15 * time.Second).Navigate(searchUrl)
	if err != nil {
		return "", fmt.Errorf("gagal membuka situs pencarian: %v", err)
	}

	page.MustWaitLoad()
	
	// Tunggu Cloudflare challenge (bisa memakan waktu)
	time.Sleep(5 * time.Second)

	// 1. Cari elemen hasil film pertama (tunggu lebih lama karena Cloudflare)
	resultItem, err := page.Timeout(15 * time.Second).Element("article a, .result-item a, .item a")
	if err != nil {
		html, _ := page.HTML()
		log.Printf("=== DEBUG HTML Awal ===\n%.500s\n=======================", html)
		return "", fmt.Errorf("gagal menemukan hasil pencarian untuk %s: %v", title, err)
	}

	// 2. Klik dan tunggu halaman load
	err = resultItem.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return "", fmt.Errorf("gagal mengeklik film: %v", err)
	}
	page.MustWaitLoad()

	// 3. Cari elemen iframe
	iframeElement, err := page.Timeout(5 * time.Second).Element("iframe")
	if err != nil {
		return "", fmt.Errorf("gagal menemukan iframe video: %v", err)
	}

	// 4. Ekstrak src
	iframeSrc, err := iframeElement.Attribute("src")
	if err != nil || iframeSrc == nil {
		return "", fmt.Errorf("gagal mengekstrak src dari iframe")
	}

	return *iframeSrc, nil
}
