package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	loadDotEnv(".env")
	log.Println("[main] Starting WaveFlix Hub backend...")

	initSentry()
	defer flushSentry()
	log.Println("[main] Sentry initialized")

	initDB()
	log.Println("[main] Database connected")

	cache = NewRedisCacheManager()
	log.Println("[main] Cache initialized")

	messageQueue = NewMessageQueue()
	log.Println("[main] Message queue initialized")

	workerPoolManager = &WorkerPoolManager{}
	log.Println("[main] Worker pool manager initialized")

	initMetrics()
	go collectSystemMetrics()
	log.Println("[main] Metrics initialized")

	mux := http.NewServeMux()
	setupRoutes(mux)

	handler := withRateLimit(mux)
	handler = withMetrics(handler)
	handler = withSecurity(handler)
	handler = withCORS(handler)
	handler = sentryMiddleware(handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("[main] Server listening on http://0.0.0.0:%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("[main] Server failed: %v", err)
	}
}

func setupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/register", handleRegister)
	mux.HandleFunc("/api/login", handleLogin)
	mux.HandleFunc("/api/logout", handleLogout)
	mux.HandleFunc("/api/me", requireAuth(handleMe))
	mux.HandleFunc("/api/check-email", handleCheckEmail)
	mux.HandleFunc("/api/update-profile", requireAuth(handleUpdateProfile))
	mux.HandleFunc("/api/change-password", requireAuth(handleChangePassword))

	mux.HandleFunc("/api/auth/register", handleRegister)
	mux.HandleFunc("/api/auth/login", handleLogin)
	mux.HandleFunc("/api/auth/logout", handleLogout)
	mux.HandleFunc("/api/auth/me", requireAuth(handleMe))
	mux.HandleFunc("/api/auth/check-email", handleCheckEmail)
	mux.HandleFunc("/api/auth/profile", requireAuth(handleUpdateProfile))
	mux.HandleFunc("/api/auth/password", requireAuth(handleChangePassword))
	mux.HandleFunc("/api/auth/avatar", requireAuth(uploadImage("avatar_url")))
	mux.HandleFunc("/api/auth/banner", requireAuth(uploadImage("banner")))

	mux.HandleFunc("/api/profiles", requireAuth(handleProfiles))
	mux.HandleFunc("/api/profiles/", requireAuth(handleProfileDetail))

	mux.HandleFunc("/api/send-code", handleSendCode)
	mux.HandleFunc("/api/send-code-async", handleSendCodeAsync)
	mux.HandleFunc("/api/auth/send-code", handleSendCode)

	mux.HandleFunc("/api/history", requireAuth(handleHistory))
	mux.HandleFunc("/api/watchlist", requireAuth(handleWatchlist))
	mux.HandleFunc("/api/favorites", requireAuth(handleFavorites))

	tmdbClient := NewTMDBClient(cache)
	mux.HandleFunc("/api/trending", tmdbClient.HandleTrending)
	mux.HandleFunc("/api/detail", tmdbClient.HandleDetail)
	mux.HandleFunc("/api/person", tmdbClient.HandlePerson)
	mux.HandleFunc("/api/season", tmdbClient.HandleSeason)
	mux.HandleFunc("/api/search", handleSearchParallel)
	mux.HandleFunc("/api/discover", handleDiscoverParallel)
	mux.HandleFunc("/api/batch", handleDetailBatchParallel)
	mux.HandleFunc("/api/stream", HandleStreamAPI)
	mux.HandleFunc("/api/media-proxy", HandleMediaProxy)

	mux.HandleFunc("/uploads/", handleUploadsWithOptimization)
	mux.HandleFunc("/api/images/", handleCachedImages)
	mux.HandleFunc("/api/image-proxy", handleImageProxy)
	mux.HandleFunc("/api/cdn-status", handleCDNStatus)
	mux.HandleFunc("/api/preload", handlePreload)

	hc := NewHealthChecker(db, cache, "1.0.0")
	mux.HandleFunc("/health", hc.HealthCheckHandler)
	mux.HandleFunc("/ready", hc.ReadinessHandler)
	mux.Handle("/metrics", metricsHandler())

	log.Printf("[main] %d routes registered", 23)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-Profile-ID, Cache-Control, Pragma, Expires")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func withSecurity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

func withRateLimit(next http.Handler) http.Handler {
	return rateLimit("global", 100, time.Minute, next.ServeHTTP)
}

func withMetrics(next http.Handler) http.Handler {
	return metricsMiddleware(next.ServeHTTP)
}
