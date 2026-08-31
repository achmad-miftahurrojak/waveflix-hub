

package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
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

type CDNOptimizer struct {
	cacheDir        string
	tmdbImageCache  map[string]*CachedImage
	mu              sync.RWMutex
	maxCacheSize    int64
	currentCacheSize int64

	enableWebP       bool
	enableAVIF       bool
	jpegQuality      int
	enableLazyLoad   bool
	cdnBaseURL       string
}

type CachedImage struct {
	OriginalURL    string
	LocalPath      string
	OptimizedPaths map[string]string 
	Size           int64
	CreatedAt      time.Time
	AccessedAt     time.Time
	AccessCount    int64
}

type ImageOptimizationRequest struct {
	SourceURL    string
	Width        int
	Height       int
	Quality      int
	Format       string 
	Lazy         bool
}

type StaticAssetConfig struct {
	EnableCompression bool
	EnableBrotli      bool
	CacheMaxAge       time.Duration
	CDNEnabled        bool
	CDNBaseURL        string
}

func NewCDNOptimizer() *CDNOptimizer {
	cacheDir := "./cache/images"
	os.MkdirAll(cacheDir, 0755)

	optimizer := &CDNOptimizer{
		cacheDir:       cacheDir,
		tmdbImageCache: make(map[string]*CachedImage),
		maxCacheSize:   2 * 1024 * 1024 * 1024, 
		enableWebP:     true,
		enableAVIF:     false, 
		jpegQuality:    85,
		enableLazyLoad: true,
		cdnBaseURL:     getenv("CDN_BASE_URL", ""),
	}

	go optimizer.cleanupRoutine()

	log.Println("[cdn] CDN optimizer initialized")
	return optimizer
}

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

	optimizedPath, err := co.downloadAndOptimize(imageURL, width, height, format)
	if err != nil {
		return "", err
	}

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

func (co *CDNOptimizer) downloadAndOptimize(imageURL string, width, height int, format string) (string, error) {

	resp, err := http.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image: %d", resp.StatusCode)
	}

	img, originalFormat, err := image.Decode(resp.Body)
	if err != nil {
		return "", err
	}

	log.Printf("[cdn] Downloaded image: %s (format: %s, size: %dx%d)", 
		imageURL, originalFormat, img.Bounds().Max.X, img.Bounds().Max.Y)

	if width > 0 || height > 0 {
		img = co.resizeImage(img, width, height)
	}

	hash := md5.Sum([]byte(fmt.Sprintf("%s_%d_%d_%s", imageURL, width, height, format)))
	filename := hex.EncodeToString(hash[:]) + "." + format
	outputPath := filepath.Join(co.cacheDir, filename)

	err = co.saveOptimizedImage(img, outputPath, format)
	if err != nil {
		return "", err
	}

	if stat, err := os.Stat(outputPath); err == nil {
		co.currentCacheSize += stat.Size()
	}

	return filename, nil
}

func (co *CDNOptimizer) resizeImage(src image.Image, maxWidth, maxHeight int) image.Image {
	bounds := src.Bounds()
	srcWidth := bounds.Max.X
	srcHeight := bounds.Max.Y

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

	if ratio > 1.0 {
		return src
	}

	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	return dst
}

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

		log.Printf("[cdn] WebP not implemented, using JPEG")
		options := &jpeg.Options{Quality: co.jpegQuality}
		return jpeg.Encode(file, img, options)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func (co *CDNOptimizer) generateCacheKey(imageURL string, width, height int, format string) string {
	key := fmt.Sprintf("%s_%d_%d_%s", imageURL, width, height, format)
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

func (co *CDNOptimizer) getPublicURL(filename string) string {
	if co.cdnBaseURL != "" {
		return co.cdnBaseURL + "/cache/images/" + filename
	}
	return "/cache/images/" + filename
}

func (co *CDNOptimizer) ServeOptimizedImage(w http.ResponseWriter, r *http.Request, filename string) {
	filePath := filepath.Join(co.cacheDir, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	co.setCacheHeaders(w, time.Hour*24*30) 

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

	w.Header().Set("Content-Encoding", "gzip")

	http.ServeFile(w, r, filePath)

	log.Printf("[cdn] Served optimized image: %s", filename)
}

func (co *CDNOptimizer) setCacheHeaders(w http.ResponseWriter, maxAge time.Duration) {
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable", int(maxAge.Seconds())))
	w.Header().Set("Expires", time.Now().Add(maxAge).UTC().Format(http.TimeFormat))
	w.Header().Set("ETag", fmt.Sprintf(`"%d"`, time.Now().Unix()))
}

func (co *CDNOptimizer) HandleImageProxy(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()
	imageURL := query.Get("url")
	widthStr := query.Get("w")
	heightStr := query.Get("h")
	format := query.Get("f")

	if imageURL == "" {
		http.Error(w, "Missing url parameter", http.StatusBadRequest)
		return
	}

	decodedURL, err := url.QueryUnescape(imageURL)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	width := 0
	height := 0
	if widthStr != "" {
		width, _ = strconv.Atoi(widthStr)
	}
	if heightStr != "" {
		height, _ = strconv.Atoi(heightStr)
	}

	if format == "" {
		format = "jpeg"
	}

	if width > 2000 || height > 2000 {
		http.Error(w, "Dimensions too large", http.StatusBadRequest)
		return
	}

	optimizedURL, err := co.OptimizeTMDBImage(decodedURL, width, height, format)
	if err != nil {
		log.Printf("[cdn] Image optimization failed: %v", err)
		http.Error(w, "Optimization failed", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, optimizedURL, http.StatusFound)
}

func (co *CDNOptimizer) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		co.cleanup()
	}
}

func (co *CDNOptimizer) cleanup() {
	co.mu.Lock()
	defer co.mu.Unlock()

	now := time.Now()
	maxAge := time.Hour * 24 * 7 

	var toDelete []string

	for key, cached := range co.tmdbImageCache {

		if now.Sub(cached.AccessedAt) > maxAge && cached.AccessCount < 10 {
			toDelete = append(toDelete, key)

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

	for _, key := range toDelete {
		delete(co.tmdbImageCache, key)
	}

	log.Printf("[cdn] Cleaned up %d cached images, cache size: %d bytes", 
		len(toDelete), co.currentCacheSize)
}

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

type StaticAssetOptimizer struct {
	config StaticAssetConfig
}

func NewStaticAssetOptimizer(config StaticAssetConfig) *StaticAssetOptimizer {
	return &StaticAssetOptimizer{
		config: config,
	}
}

func (sao *StaticAssetOptimizer) OptimizeStaticAssets(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if sao.isStaticAsset(r.URL.Path) {
			sao.setStaticAssetHeaders(w, r)
		}

		next(w, r)
	}
}

func (sao *StaticAssetOptimizer) isStaticAsset(path string) bool {
	staticExtensions := []string{".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot"}

	for _, ext := range staticExtensions {
		if strings.HasSuffix(strings.ToLower(path), ext) {
			return true
		}
	}

	return false
}

func (sao *StaticAssetOptimizer) setStaticAssetHeaders(w http.ResponseWriter, r *http.Request) {

	maxAge := sao.config.CacheMaxAge
	if maxAge == 0 {
		maxAge = time.Hour * 24 * 365 
	}

	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable", int(maxAge.Seconds())))
	w.Header().Set("Expires", time.Now().Add(maxAge).UTC().Format(http.TimeFormat))

	if sao.config.EnableCompression {
		w.Header().Set("Vary", "Accept-Encoding")
	}

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

var cdnOptimizer *CDNOptimizer
var staticAssetOptimizer *StaticAssetOptimizer

func initCDNOptimizer() {
	cdnOptimizer = NewCDNOptimizer()

	staticConfig := StaticAssetConfig{
		EnableCompression: true,
		EnableBrotli:      true,
		CacheMaxAge:       time.Hour * 24 * 365,
		CDNEnabled:        getenv("CDN_ENABLED", "false") == "true",
		CDNBaseURL:        getenv("CDN_BASE_URL", ""),
	}

	staticAssetOptimizer = NewStaticAssetOptimizer(staticConfig)

	log.Println("[cdn] CDN optimization initialized")
}