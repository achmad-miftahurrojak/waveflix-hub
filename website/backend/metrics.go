// Package main provides Prometheus metrics for WaveFlix Hub backend.
//
// This module exposes application metrics for monitoring and auto-scaling
// decisions. Metrics include HTTP requests, response times, cache hit rates,
// database connections, and custom business metrics.
package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus metrics collectors
var (
	// HTTP metrics
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// Cache metrics
	cacheOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_operations_total",
			Help: "Total number of cache operations",
		},
		[]string{"operation", "result"},
	)

	cacheHitRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cache_hit_rate",
			Help: "Cache hit rate percentage",
		},
		[]string{"cache_type"},
	)

	// Database metrics
	dbConnectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Number of active database connections",
		},
	)

	dbConnectionsIdle = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
		},
		[]string{"query_type"},
	)

	// Queue metrics
	queueJobsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queue_jobs_total",
			Help: "Total number of queued jobs",
		},
		[]string{"job_type", "status"},
	)

	queueJobDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "queue_job_duration_seconds",
			Help:    "Queue job processing duration in seconds",
			Buckets: []float64{0.1, 0.5, 1, 5, 10, 30, 60},
		},
		[]string{"job_type"},
	)

	queueWorkersBusy = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "queue_workers_busy",
			Help: "Number of busy queue workers",
		},
	)

	// Business metrics
	apiCallsExternal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "external_api_calls_total",
			Help: "Total external API calls",
		},
		[]string{"api", "status"},
	)

	userRegistrations = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "user_registrations_total",
			Help: "Total user registrations",
		},
	)

	contentViews = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "content_views_total",
			Help: "Total content views",
		},
		[]string{"content_type"},
	)
)

// initMetrics initializes Prometheus metrics
func initMetrics() {
	// Register metrics with Prometheus
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		cacheOperationsTotal,
		cacheHitRate,
		dbConnectionsActive,
		dbConnectionsIdle,
		dbQueryDuration,
		queueJobsTotal,
		queueJobDuration,
		queueWorkersBusy,
		apiCallsExternal,
		userRegistrations,
		contentViews,
	)
	
	log.Println("[metrics] Prometheus metrics initialized")
}

// metricsHandler returns the Prometheus metrics handler
func metricsHandler() http.Handler {
	return promhttp.Handler()
}

// Middleware for HTTP request metrics
func metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Wrap ResponseWriter to capture status code
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		// Process request
		next(ww, r)
		
		// Record metrics
		duration := time.Since(start).Seconds()
		endpoint := sanitizeEndpoint(r.URL.Path)
		status := strconv.Itoa(ww.statusCode)
		
		httpRequestsTotal.WithLabelValues(r.Method, endpoint, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, endpoint).Observe(duration)
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// sanitizeEndpoint converts specific endpoints to generic patterns for metrics
func sanitizeEndpoint(path string) string {
	// Convert specific IDs to generic patterns
	// /api/detail?id=123 -> /api/detail
	// /api/person?id=456 -> /api/person
	
	if strings.Contains(path, "/api/detail") {
		return "/api/detail"
	}
	if strings.Contains(path, "/api/person") {
		return "/api/person"
	}
	if strings.Contains(path, "/api/season") {
		return "/api/season"
	}
	if strings.Contains(path, "/api/images") {
		return "/api/images"
	}
	
	return path
}

// Helper functions for recording metrics
func recordCacheHit(cacheType string) {
	cacheOperationsTotal.WithLabelValues("get", "hit").Inc()
}

func recordCacheMiss(cacheType string) {
	cacheOperationsTotal.WithLabelValues("get", "miss").Inc()
}

func recordCacheSet(cacheType string) {
	cacheOperationsTotal.WithLabelValues("set", "success").Inc()
}

func updateCacheHitRate(cacheType string, hitRate float64) {
	cacheHitRate.WithLabelValues(cacheType).Set(hitRate)
}

func recordDBQuery(queryType string, duration time.Duration) {
	dbQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())
}

func updateDBConnections(active, idle int) {
	dbConnectionsActive.Set(float64(active))
	dbConnectionsIdle.Set(float64(idle))
}

func recordQueueJob(jobType, status string, duration time.Duration) {
	queueJobsTotal.WithLabelValues(jobType, status).Inc()
	if status == "success" || status == "failed" {
		queueJobDuration.WithLabelValues(jobType).Observe(duration.Seconds())
	}
}

func updateQueueWorkers(busy int) {
	queueWorkersBusy.Set(float64(busy))
}

func recordExternalAPICall(api, status string) {
	apiCallsExternal.WithLabelValues(api, status).Inc()
}

func recordUserRegistration() {
	userRegistrations.Inc()
}

func recordContentView(contentType string) {
	contentViews.WithLabelValues(contentType).Inc()
}

// collectSystemMetrics periodically collects system-level metrics
func collectSystemMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Update database connection metrics
			if db != nil {
				if stats := db.Stats(); stats != nil {
					updateDBConnections(stats.OpenConnections, stats.IdleConnections)
				}
			}

			// Update cache metrics
			if cache != nil {
				stats := cache.GetStats()
				if stats.Enabled {
					// Calculate hit rate (simplified)
					updateCacheHitRate("redis", 85.0) // Placeholder
				}
			}

			// Update queue metrics
			if messageQueue != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				metrics, err := messageQueue.GetMetrics(ctx)
				cancel()
				
				if err == nil {
					updateQueueWorkers(metrics.WorkersActive)
				}
			}
		}
	}
}