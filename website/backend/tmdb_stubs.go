// Package main provides TMDB API constants, global vars, and helper stubs
// yang digunakan bersama oleh handlers, parallel_handlers, dan async_handlers.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

// ─── TMDB API constants ───────────────────────────────────────────────────────

// TaskType constants untuk message queue.
const (
	TaskTypeTMDBAPI    = "tmdb_api"
	TaskTypeDataProcess = "data_process"
	TaskTypeEmailSend   = "email_send"
)

// ─── TMDB API globals ─────────────────────────────────────────────────────────

// tmdbBaseUrl adalah base URL TMDB API v3.
var tmdbBaseUrl = "https://api.themoviedb.org/3"

// tmdbApiKey dibaca dari environment variable TMDB_API_KEY.
var tmdbApiKey = os.Getenv("TMDB_API_KEY")

// ─── Query helpers ────────────────────────────────────────────────────────────

// getMediaParam mengambil nilai query param "media" ("movie" atau "tv").
// Default ke "movie" jika tidak diset atau tidak valid.
func getMediaParam(q url.Values) string {
	return normalizeMedia(q.Get("media"))
}

// getTmdbLang mengambil language param dari query; default "id-ID".
func getTmdbLang(q url.Values) string {
	if lang := q.Get("language"); lang != "" {
		return lang
	}
	return "id-ID"
}

// getImageLangs mengambil image language param; default "id,null".
func getImageLangs(q url.Values) string {
	if lang := q.Get("include_image_language"); lang != "" {
		return lang
	}
	return "id,null"
}

// ─── TMDB fetch helpers ───────────────────────────────────────────────────────

// fetchJSON melakukan GET request ke TMDB API dan mem-parse JSON response.
// Parameter mediaType dipakai untuk logging saja.
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

// sanitizeTMDBData membersihkan field sensitif atau tidak perlu dari TMDB response.
// Dipanggil in-place agar hasilnya langsung tersimpan di map yang sama.
func sanitizeTMDBData(data map[string]interface{}, mediaType string) {
	// Hapus field yang tidak diperlukan frontend
	delete(data, "adult")
	delete(data, "belongs_to_collection") // detail khusus movie
}

// ─── Provider availability helpers ───────────────────────────────────────────

// checkVidlinkAvailability mengecek apakah konten tersedia di vidlink.
// Ini adalah stub — implementasi aslinya ada di handlers.
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

// checkMajorProvider mengecek apakah konten tersedia di provider streaming utama.
// Ini adalah stub — implementasi aslinya ada di handlers.
func checkMajorProvider(media, id string) bool {
	// Simplified check — always return false until proper implementation
	return false
}
