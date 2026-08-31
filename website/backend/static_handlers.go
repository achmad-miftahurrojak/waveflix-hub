// Package main provides optimized static file handlers for WaveFlix Hub.
//
// These handlers implement CDN-like functionality for static assets including
// image optimization, compression, caching, and content delivery optimization.
package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// handleUploadsWithOptimization serves uploads directory with optimization.
func handleUploadsWithOptimization(w http.ResponseWriter, r *http.Request) {
	// Remove /uploads/ prefix
	path := strings.TrimPrefix(r.URL.Path, "/uploads/")
	if path == "" || path == "/" {
		http.NotFound(w, r)
		return
	}
	
	// Security: prevent directory traversal
	if strings.Contains(path, "..") || strings.Contains(path, "/./") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	
	filePath := filepath.Join("./uploads", path)
	
	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
		} else {
			http.Error(w, "File access error", http.StatusInternalServerError)
		}
		return
	}
	
	// Don't serve directories
	if info.IsDir() {
		http.Error(w, "Directory listing not allowed", http.StatusForbidden)
		return
	}
	
	// Set optimized headers
	setOptimizedFileHeaders(w, r, filePath, info)
	
	// Handle conditional requests (ETag/If-Modified-Since)
	if handleConditionalRequest(w, r, info) {
		return
	}
	
	// Serve compressed if possible
	if shouldCompress(r, filePath) {
		serveCompressed(w, r, filePath)
	} else {
		http.ServeFile(w, r, filePath)
	}
}

// handleCachedImages serves optimized cached images.
func handleCachedImages(w http.ResponseWriter, r *http.Request) {
	if cdnOptimizer == nil {
		http.NotFound(w, r)
		return
	}
	
	// Remove /cache/images/ prefix
	filename := strings.TrimPrefix(r.URL.Path, "/cache/images/")
	if filename == "" {
		http.NotFound(w, r)
		return
	}
	
	// Security check
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}
	
	cdnOptimizer.ServeOptimizedImage(w, r, filename)
}

// handleImageProxy handles TMDB image proxy requests.
func handleImageProxy(w http.ResponseWriter, r *http.Request) {
	if cdnOptimizer == nil {
		http.Error(w, "CDN optimizer not available", http.StatusServiceUnavailable)
		return
	}
	
	cdnOptimizer.HandleImageProxy(w, r)
}

// setOptimizedFileHeaders sets optimized caching and content headers.
func setOptimizedFileHeaders(w http.ResponseWriter, r *http.Request, filePath string, info os.FileInfo) {
	// Set Last-Modified
	w.Header().Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))
	
	// Set ETag based on file size and modification time
	etag := generateETag(info)
	w.Header().Set("ETag", etag)
	
	// Set content type based on file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	contentType := getContentType(ext)
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	
	// Set caching headers based on file type
	setCacheHeadersByType(w, ext)
	
	// Security headers for certain file types
	switch ext {
	case ".svg":
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	}
	
	// Enable range requests for media files
	if isMediaFile(ext) {
		w.Header().Set("Accept-Ranges", "bytes")
	}
}

// handleConditionalRequest handles ETag and If-Modified-Since headers.
func handleConditionalRequest(w http.ResponseWriter, r *http.Request, info os.FileInfo) bool {
	// Handle If-Modified-Since
	if ifModSince := r.Header.Get("If-Modified-Since"); ifModSince != "" {
		if modTime, err := time.Parse(http.TimeFormat, ifModSince); err == nil {
			if !info.ModTime().After(modTime) {
				w.WriteHeader(http.StatusNotModified)
				return true
			}
		}
	}
	
	// Handle If-None-Match (ETag)
	if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "" {
		etag := generateETag(info)
		if ifNoneMatch == etag {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}
	
	return false
}

// shouldCompress determines if file should be compressed.
func shouldCompress(r *http.Request, filePath string) bool {
	// Check if client accepts gzip
	acceptEncoding := r.Header.Get("Accept-Encoding")
	if !strings.Contains(acceptEncoding, "gzip") {
		return false
	}
	
	// Check file type
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

// serveCompressed serves file with gzip compression.
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

// generateETag generates an ETag for a file.
func generateETag(info os.FileInfo) string {
	return `"` + strconv.FormatInt(info.ModTime().Unix(), 16) + 
		   "-" + strconv.FormatInt(info.Size(), 16) + `"`
}

// getContentType returns content type for file extension.
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

// setCacheHeadersByType sets appropriate cache headers based on file type.
func setCacheHeadersByType(w http.ResponseWriter, ext string) {
	var maxAge time.Duration
	
	switch ext {
	case ".css", ".js":
		// Versioned assets - long cache
		maxAge = time.Hour * 24 * 365 // 1 year
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".avif", ".ico":
		// Images - medium cache
		maxAge = time.Hour * 24 * 30 // 30 days
		w.Header().Set("Cache-Control", "public, max-age=2592000")
	case ".woff", ".woff2", ".ttf", ".eot":
		// Fonts - long cache
		maxAge = time.Hour * 24 * 365 // 1 year
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	case ".pdf":
		// Documents - medium cache
		maxAge = time.Hour * 24 * 7 // 1 week
		w.Header().Set("Cache-Control", "public, max-age=604800")
	case ".mp4", ".webm", ".mp3":
		// Media files - long cache
		maxAge = time.Hour * 24 * 30 // 30 days
		w.Header().Set("Cache-Control", "public, max-age=2592000")
	default:
		// Other files - short cache
		maxAge = time.Hour * 24 // 1 day
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
	
	// Set Expires header
	w.Header().Set("Expires", time.Now().Add(maxAge).UTC().Format(http.TimeFormat))
}

// isMediaFile checks if file is a media file that supports range requests.
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

// CDN Status endpoint for monitoring
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

// Preload critical resources endpoint
func handlePreload(w http.ResponseWriter, r *http.Request) {
	// Set preload headers for critical resources
	w.Header().Set("Link", `</api/trending>; rel=preload; as=fetch`)
	w.Header().Add("Link", `</assets/logo.png>; rel=preload; as=image`)
	w.Header().Add("Link", `</assets/fonts/main.woff2>; rel=preload; as=font; type=font/woff2; crossorigin`)
	
	writeJSON(w, `{"preload":"enabled"}`)
}

// Resource hints for performance optimization
func setResourceHints(w http.ResponseWriter) {
	// DNS prefetch for external domains
	w.Header().Set("Link", `<//image.tmdb.org>; rel=dns-prefetch`)
	w.Header().Add("Link", `<//fonts.googleapis.com>; rel=dns-prefetch`)
	w.Header().Add("Link", `<//fonts.gstatic.com>; rel=dns-prefetch`)
	
	// Preconnect to critical external resources
	w.Header().Add("Link", `<//api.themoviedb.org>; rel=preconnect`)
}