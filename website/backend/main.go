package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
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
	mux := http.NewServeMux()
	mux.HandleFunc("/api/play", handlePlay)

	handler := enableCORS(mux)

	log.Println("Summer Tide Backend berjalan di http://localhost:8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
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
	if item, found := videoCache[cacheKey]; found && time.Now().Before(item.ExpiresAt) {
		cacheMutex.RUnlock()
		log.Printf("[CACHE HIT] %s", title)
		json.NewEncoder(w).Encode(PlayResponse{
			Title:     title,
			Year:      year,
			IframeUrl: item.IframeURL,
		})
		return
	}
	cacheMutex.RUnlock()

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
	// Menjalankan Headless Chrome via go-rod
	// Menggunakan argumen tambahan untuk meminimalisir deteksi bot
	l := launcher.New().
		Leakless(false).
		Headless(true).
		Set("disable-blink-features", "AutomationControlled")
		
	urlLauncher, err := l.Launch()
	if err != nil {
		return "", fmt.Errorf("gagal meluncurkan browser: %v", err)
	}
	defer l.Cleanup()

	browser := rod.New().ControlURL(urlLauncher).MustConnect()
	defer browser.MustClose()

	// Gunakan stealth plugin untuk bypass Cloudflare
	page := stealth.MustPage(browser)

	// Asumsi domain idlix saat ini
	searchQuery := url.QueryEscape(title)
	searchUrl := fmt.Sprintf("https://tv.idlixofficial.co/?s=%s", searchQuery)

	log.Printf("[SCRAPING] Membuka URL: %s", searchUrl)

	// Timeout 15 detik agar tidak menggantung selamanya jika terkena Cloudflare Captcha
	err = page.Timeout(15 * time.Second).Navigate(searchUrl)
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
