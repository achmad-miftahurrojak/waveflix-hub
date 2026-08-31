// Package main provides CDN optimization and static asset management for WaveFlix Hub.
//
// This module handles image optimization, static asset caching, CDN integration,
// and intelligent content delivery to minimize bandwidth usage and improve
// user experience across different regions and devices.
package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/image/draw"
)

// CDNOptimizer manages content delivery network optimizations.
type CDNOptimizer struct {
	cacheDir        string
	tmdbImageCache  map[string]*CachedImage
	mu              sync.RWMutex
	maxCacheSize    int64
	currentCacheSize int64
	
	// Configuration
	enableWebP       bool
	enableAVIF       bool
	jpegQuality      int
	enableLazyLoad   bool
	cdnBaseURL       string
}

// CachedImage represents a cached and optimized image.
type CachedImage struct {
	OriginalURL    string
	LocalPath      string
	OptimizedPaths map[string]string // format -> path
	Size           int64
	CreatedAt      time.Time
	AccessedAt     time.Time
	AccessCount    int64
}

// ImageOptimizationRequest represents a request for image optimization.
type ImageOptimizationRequest struct {
	SourceURL    string
	Width        int
	Height       int
	Quality      int
	Format       string // webp, avif, jpeg, png
	Lazy         bool
}

// StaticAssetConfig holds static asset optimization configuration.
type StaticAssetConfig struct {
	EnableCompression bool
	EnableBrotli      bool
	CacheMaxAge       time.Duration
	CDNEnabled        bool
	CDNBaseURL        string
}

// NewCDNOptimizer creates a new CDN optimizer instance.
func NewCDNOptimizer() *CDNOptimizer {
	cacheDir := "./cache/images"
	os.MkdirAll(cacheDir, 0755)
	
	optimizer := &CDNOptimizer{
		cacheDir:       cacheDir,
		tmdbImageCache: make(map[string]*CachedImage),
		maxCacheSize:   2 * 1024 * 1024 * 1024, // 2GB
		enableWebP:     true,
		enableAVIF:     false, // Enable when supported
		jpegQuality:    85,
		enableLazyLoad: true,
		cdnBaseURL:     getEnv("CDN_BASE_URL", ""),
	}
	
	// Start cleanup routine
	go optimizer.cleanupRoutine()
	
	log.Println("[cdn] CDN optimizer initialized")
	return optimizer
}

// OptimizeTMDBImage optimizes TMDB images for different formats and sizes.
func (co *CDNOptimizer) OptimizeTMDBImage(imageURL string, width, height int, format string) (string, error) {
	if imageURL == "" {
		return "", fmt.Errorf("empty image URL")
	}
	
	cacheKey := co.generateCacheKey(imageURL, width, height, format)
	
	co.mu.RLock()
	cached, exists := co.tmdbImageCache[cacheKey]
	if exists {
		cached.AccessedAt = time.Now()
		cached.AccessCount++
		co.mu.RUnlock()
		
		if optimizedPath, ok := cached.OptimizedPaths[format]; ok {
			return co.getPublicURL(optimizedPath), nil
		}
	}
	co.mu.RUnlock()
	
	// Download and optimize image
	optimizedPath, err := co.downloadAndOptimize(imageURL, width, height, format)
	if err != nil {
		return "", err
	}
	
	// Cache the result
	co.mu.Lock()
	if !exists {
		cached = &CachedImage{
			OriginalURL:    imageURL,
			OptimizedPaths: make(map[string]string),
			CreatedAt:      time.Now(),
			AccessedAt:     time.Now(),
			AccessCount:    1,
		}
		co.tmdbImageCache[cacheKey] = cached
	}
	cached.OptimizedPaths[format] = optimizedPath
	co.mu.Unlock()
	
	return co.getPublicURL(optimizedPath), nil
}

// downloadAndOptimize downloads an image and optimizes it.
func (co *CDNOptimizer) downloadAndOptimize(imageURL string, width, height int, format string) (string, error) {
	// Download original image
	resp, err := http.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image: %d", resp.StatusCode)
	}
	
	// Decode image
	img, originalFormat, err := image.Decode(resp.Body)
	if err != nil {
		return "", err
	}
	
	log.Printf("[cdn] Downloaded image: %s (format: %s, size: %dx%d)", 
		imageURL, originalFormat, img.Bounds().Max.X, img.Bounds().Max.Y)
	
	// Resize if needed
	if width > 0 || height > 0 {
		img = co.resizeImage(img, width, height)
	}
	
	// Generate output filename
	hash := md5.Sum([]byte(fmt.Sprintf("%s_%d_%d_%s", imageURL, width, height, format)))
	filename := hex.EncodeToString(hash[:]) + "." + format
	outputPath := filepath.Join(co.cacheDir, filename)
	
	// Save optimized image
	err = co.saveOptimizedImage(img, outputPath, format)
	if err != nil {
		return "", err
	}
	
	// Record file size
	if stat, err := os.Stat(outputPath); err == nil {
		co.currentCacheSize += stat.Size()
	}
	
	return filename, nil
}

// resizeImage resizes an image maintaining aspect ratio.
func (co *CDNOptimizer) resizeImage(src image.Image, maxWidth, maxHeight int) image.Image {
	bounds := src.Bounds()
	srcWidth := bounds.Max.X
	srcHeight := bounds.Max.Y
	
	// Calculate new dimensions maintaining aspect ratio
	var newWidth, newHeight int
	
	if maxWidth == 0 {
		maxWidth = srcWidth
	}
	if maxHeight == 0 {
		maxHeight = srcHeight
	}
	
	ratioX := float64(maxWidth) / float64(srcWidth)
	ratioY := float64(maxHeight) / float64(srcHeight)
	ratio := ratioX
	if ratioY < ratioX {
		ratio = ratioY
	}
	
	newWidth = int(float64(srcWidth) * ratio)
	newHeight = int(float64(srcHeight) * ratio)
	
	// Don't upscale
	if ratio > 1.0 {
		return src
	}
	
	// Create new image
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	
	return dst
}

// saveOptimizedImage saves an image in the specified format with optimization.
func (co *CDNOptimizer) saveOptimizedImage(img image.Image, path, format string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		options := &jpeg.Options{Quality: co.jpegQuality}
		return jpeg.Encode(file, img, options)
	case "png":
		return png.Encode(file, img)
	case "webp":
		// For WebP, we'd need to add WebP encoding library
		// For now, fallback to JPEG
		log.Printf("[cdn] WebP not implemented, using JPEG")
		options := &jpeg.Options{Quality: co.jpegQuality}
		return jpeg.Encode(file, img, options)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// generateCacheKey generates a cache key for an image optimization request.
func (co *CDNOptimizer) generateCacheKey(imageURL string, width, height int, format string) string {
	key := fmt.Sprintf("%s_%d_%d_%s", imageURL, width, height, format)
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

// getPublicURL returns the public URL for a cached image.
func (co *CDNOptimizer) getPublicURL(filename string) string {
	if co.cdnBaseURL != "" {
		return co.cdnBaseURL + "/cache/images/" + filename
	}
	return "/cache/images/" + filename
}

// ServeOptimizedImage serves an optimized image with proper headers.
func (co *CDNOptimizer) ServeOptimizedImage(w http.ResponseWriter, r *http.Request, filename string) {
	filePath := filepath.Join(co.cacheDir, filename)
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	
	// Set caching headers
	co.setCacheHeaders(w, time.Hour*24*30) // 30 days
	
	// Set content type based on file extension
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	case ".avif":
		w.Header().Set("Content-Type", "image/avif")
	}
	
	// Enable compression for supported formats
	w.Header().Set("Content-Encoding", "gzip")
	
	// Serve file
	http.ServeFile(w, r, filePath)
	
	log.Printf("[cdn] Served optimized image: %s", filename)
}

// setCacheHeaders sets appropriate caching headers.
func (co *CDNOptimizer) setCacheHeaders(w http.ResponseWriter, maxAge time.Duration) {
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable", int(maxAge.Seconds())))
	w.Header().Set("Expires", time.Now().Add(maxAge).UTC().Format(http.TimeFormat))
	w.Header().Set("ETag", fmt.Sprintf(`"%d"`, time.Now().Unix()))
}

// HandleImageProxy handles TMDB image proxy requests with optimization.
func (co *CDNOptimizer) HandleImageProxy(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()
	imageURL := query.Get("url")
	widthStr := query.Get("w")
	heightStr := query.Get("h")
	format := query.Get("f")
	
	if imageURL == "" {
		http.Error(w, "Missing url parameter", http.StatusBadRequest)
		return
	}
	
	// Decode URL
	decodedURL, err := url.QueryUnescape(imageURL)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	
	// Parse dimensions
	width := 0
	height := 0
	if widthStr != "" {
		width, _ = strconv.Atoi(widthStr)
	}
	if heightStr != "" {
		height, _ = strconv.Atoi(heightStr)
	}
	
	// Default format
	if format == "" {
		format = "jpeg"
	}
	
	// Validate dimensions (prevent abuse)
	if width > 2000 || height > 2000 {
		http.Error(w, "Dimensions too large", http.StatusBadRequest)
		return
	}
	
	// Optimize and serve
	optimizedURL, err := co.OptimizeTMDBImage(decodedURL, width, height, format)
	if err != nil {
		log.Printf("[cdn] Image optimization failed: %v", err)
		http.Error(w, "Optimization failed", http.StatusInternalServerError)
		return
	}
	
	// Redirect to optimized image
	http.Redirect(w, r, optimizedURL, http.StatusFound)
}

// cleanupRoutine periodically cleans up old cached images.
func (co *CDNOptimizer) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		co.cleanup()
	}
}

// cleanup removes old and unused cached images.
func (co *CDNOptimizer) cleanup() {
	co.mu.Lock()
	defer co.mu.Unlock()
	
	now := time.Now()
	maxAge := time.Hour * 24 * 7 // 7 days
	
	var toDelete []string
	
	for key, cached := range co.tmdbImageCache {
		// Delete if not accessed recently and old
		if now.Sub(cached.AccessedAt) > maxAge && cached.AccessCount < 10 {
			toDelete = append(toDelete, key)
			
			// Delete files
			for _, path := range cached.OptimizedPaths {
				fullPath := filepath.Join(co.cacheDir, path)
				if err := os.Remove(fullPath); err == nil {
					if stat, err := os.Stat(fullPath); err == nil {
						co.currentCacheSize -= stat.Size()
					}
				}
			}
		}
	}
	
	// Remove from cache
	for _, key := range toDelete {
		delete(co.tmdbImageCache, key)
	}
	
	log.Printf("[cdn] Cleaned up %d cached images, cache size: %d bytes", 
		len(toDelete), co.currentCacheSize)
}

// GetCacheStats returns cache statistics.
func (co *CDNOptimizer) GetCacheStats() map[string]interface{} {
	co.mu.RLock()
	defer co.mu.RUnlock()
	
	return map[string]interface{}{
		"cached_images":    len(co.tmdbImageCache),
		"cache_size_bytes": co.currentCacheSize,
		"cache_size_mb":    co.currentCacheSize / 1024 / 1024,
		"max_cache_mb":     co.maxCacheSize / 1024 / 1024,
		"cache_utilization": float64(co.currentCacheSize) / float64(co.maxCacheSize) * 100,
	}
}

// Static Asset Optimization
type StaticAssetOptimizer struct {
	config StaticAssetConfig
}

// NewStaticAssetOptimizer creates a new static asset optimizer.
func NewStaticAssetOptimizer(config StaticAssetConfig) *StaticAssetOptimizer {
	return &StaticAssetOptimizer{
		config: config,
	}
}

// OptimizeStaticAssets handles static asset optimization middleware.
func (sao *StaticAssetOptimizer) OptimizeStaticAssets(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set caching headers for static assets
		if sao.isStaticAsset(r.URL.Path) {
			sao.setStaticAssetHeaders(w, r)
		}
		
		next(w, r)
	}
}

// isStaticAsset checks if the request is for a static asset.
func (sao *StaticAssetOptimizer) isStaticAsset(path string) bool {
	staticExtensions := []string{".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot"}
	
	for _, ext := range staticExtensions {
		if strings.HasSuffix(strings.ToLower(path), ext) {
			return true
		}
	}
	
	return false
}

// setStaticAssetHeaders sets optimized headers for static assets.
func (sao *StaticAssetOptimizer) setStaticAssetHeaders(w http.ResponseWriter, r *http.Request) {
	// Set long-term caching for static assets
	maxAge := sao.config.CacheMaxAge
	if maxAge == 0 {
		maxAge = time.Hour * 24 * 365 // 1 year default
	}
	
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable", int(maxAge.Seconds())))
	w.Header().Set("Expires", time.Now().Add(maxAge).UTC().Format(http.TimeFormat))
	
	// Enable compression
	if sao.config.EnableCompression {
		w.Header().Set("Vary", "Accept-Encoding")
	}
	
	// Set content type optimization headers
	ext := strings.ToLower(filepath.Ext(r.URL.Path))
	switch ext {
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	}
}

// Global CDN optimizer instance
var cdnOptimizer *CDNOptimizer
var staticAssetOptimizer *StaticAssetOptimizer

// initCDNOptimizer initializes CDN optimization
func initCDNOptimizer() {
	cdnOptimizer = NewCDNOptimizer()
	
	staticConfig := StaticAssetConfig{
		EnableCompression: true,
		EnableBrotli:      true,
		CacheMaxAge:       time.Hour * 24 * 365,
		CDNEnabled:        getEnv("CDN_ENABLED", "false") == "true",
		CDNBaseURL:        getEnv("CDN_BASE_URL", ""),
	}
	
	staticAssetOptimizer = NewStaticAssetOptimizer(staticConfig)
	
	log.Println("[cdn] CDN optimization initialized")
}