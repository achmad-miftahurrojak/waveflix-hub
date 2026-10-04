

package main

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func handleUploadsWithOptimization(w http.ResponseWriter, r *http.Request) {

	path := strings.TrimPrefix(r.URL.Path, "/uploads/")
	if path == "" || path == "/" {
		http.NotFound(w, r)
		return
	}

	if strings.Contains(path, "..") || strings.Contains(path, "/./") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("./uploads", path)

	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
		} else {
			http.Error(w, "File access error", http.StatusInternalServerError)
		}
		return
	}

	if info.IsDir() {
		http.Error(w, "Directory listing not allowed", http.StatusForbidden)
		return
	}

	setOptimizedFileHeaders(w, r, filePath, info)

	if handleConditionalRequest(w, r, info) {
		return
	}

	if shouldCompress(r, filePath) {
		serveCompressed(w, r, filePath)
	} else {
		http.ServeFile(w, r, filePath)
	}
}

func handleCachedImages(w http.ResponseWriter, r *http.Request) {
	if cdnOptimizer == nil {
		http.NotFound(w, r)
		return
	}

	filename := strings.TrimPrefix(r.URL.Path, "/cache/images/")
	if filename == "" {
		http.NotFound(w, r)
		return
	}

	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	cdnOptimizer.ServeOptimizedImage(w, r, filename)
}

func handleImageProxy(w http.ResponseWriter, r *http.Request) {
	if cdnOptimizer == nil {
		http.Error(w, "CDN optimizer not available", http.StatusServiceUnavailable)
		return
	}

	cdnOptimizer.HandleImageProxy(w, r)
}

func setOptimizedFileHeaders(w http.ResponseWriter, r *http.Request, filePath string, info os.FileInfo) {

	w.Header().Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))

	etag := generateETag(info)
	w.Header().Set("ETag", etag)

	ext := strings.ToLower(filepath.Ext(filePath))
	contentType := getContentType(ext)
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	setCacheHeadersByType(w, ext)

	switch ext {
	case ".svg":
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	}

	if isMediaFile(ext) {
		w.Header().Set("Accept-Ranges", "bytes")
	}
}

func handleConditionalRequest(w http.ResponseWriter, r *http.Request, info os.FileInfo) bool {

	if ifModSince := r.Header.Get("If-Modified-Since"); ifModSince != "" {
		if modTime, err := time.Parse(http.TimeFormat, ifModSince); err == nil {
			if !info.ModTime().After(modTime) {
				w.WriteHeader(http.StatusNotModified)
				return true
			}
		}
	}

	if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "" {
		etag := generateETag(info)
		if ifNoneMatch == etag {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}

	return false
}

func shouldCompress(r *http.Request, filePath string) bool {

	acceptEncoding := r.Header.Get("Accept-Encoding")
	if !strings.Contains(acceptEncoding, "gzip") {
		return false
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	compressibleTypes := map[string]bool{
		".css":  true,
		".js":   true,
		".html": true,
		".htm":  true,
		".xml":  true,
		".json": true,
		".txt":  true,
		".svg":  true,
	}

	return compressibleTypes[ext]
}

func serveCompressed(w http.ResponseWriter, r *http.Request, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Vary", "Accept-Encoding")

	gz := gzip.NewWriter(w)
	defer gz.Close()

	_, err = io.Copy(gz, file)
	if err != nil {
		log.Printf("[static] Compression error: %v", err)
	}
}

func generateETag(info os.FileInfo) string {
	return `"` + strconv.FormatInt(info.ModTime().Unix(), 16) + 
		   "-" + strconv.FormatInt(info.Size(), 16) + `"`
}

func getContentType(ext string) string {
	types := map[string]string{
		".css":  "text/css; charset=utf-8",
		".js":   "application/javascript; charset=utf-8",
		".json": "application/json; charset=utf-8",
		".html": "text/html; charset=utf-8",
		".htm":  "text/html; charset=utf-8",
		".xml":  "application/xml; charset=utf-8",
		".txt":  "text/plain; charset=utf-8",
		".svg":  "image/svg+xml",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".webp": "image/webp",
		".avif": "image/avif",
		".ico":  "image/x-icon",
		".woff": "font/woff",
		".woff2": "font/woff2",
		".ttf":  "font/ttf",
		".eot":  "application/vnd.ms-fontobject",
		".pdf":  "application/pdf",
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".mp3":  "audio/mpeg",
	}

	return types[ext]
}

func setCacheHeadersByType(w http.ResponseWriter, ext string) {
	var maxAge time.Duration

	switch ext {
	case ".css", ".js":

		maxAge = time.Hour * 24 * 365 
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".avif", ".ico":

		maxAge = time.Hour * 24 * 30 
		w.Header().Set("Cache-Control", "public, max-age=2592000")
	case ".woff", ".woff2", ".ttf", ".eot":

		maxAge = time.Hour * 24 * 365 
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	case ".pdf":

		maxAge = time.Hour * 24 * 7 
		w.Header().Set("Cache-Control", "public, max-age=604800")
	case ".mp4", ".webm", ".mp3":

		maxAge = time.Hour * 24 * 30 
		w.Header().Set("Cache-Control", "public, max-age=2592000")
	default:

		maxAge = time.Hour * 24 
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}

	w.Header().Set("Expires", time.Now().Add(maxAge).UTC().Format(http.TimeFormat))
}

func isMediaFile(ext string) bool {
	mediaTypes := map[string]bool{
		".mp4":  true,
		".webm": true,
		".mp3":  true,
		".wav":  true,
		".ogg":  true,
		".m4v":  true,
		".mov":  true,
		".avi":  true,
	}

	return mediaTypes[ext]
}

func handleCDNStatus(w http.ResponseWriter, r *http.Request) {
	if cdnOptimizer == nil {
		writeJSON(w, `{"status":"disabled","error":"CDN optimizer not initialized"}`)
		return
	}

	stats := cdnOptimizer.GetCacheStats()
	stats["status"] = "enabled"

	responseJSON, _ := json.Marshal(stats)
	writeJSON(w, string(responseJSON))
}

func handlePreload(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Link", `</api/trending>; rel=preload; as=fetch`)
	w.Header().Add("Link", `</assets/logo.png>; rel=preload; as=image`)
	w.Header().Add("Link", `</assets/fonts/main.woff2>; rel=preload; as=font; type=font/woff2; crossorigin`)

	writeJSON(w, `{"preload":"enabled"}`)
}

func setResourceHints(w http.ResponseWriter) {
}
