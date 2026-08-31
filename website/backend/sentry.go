package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

// initSentry menginisialisasi Sentry untuk error tracking.
// Panggil ini di awal main(), sebelum server start.
// DSN diambil dari env SENTRY_DSN. Kalau kosong, Sentry dinonaktifkan.
func initSentry() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		log.Println("[sentry] SENTRY_DSN tidak dikonfigurasi, error tracking dinonaktifkan")
		return
	}

	environment := os.Getenv("APP_ENV")
	if environment == "" {
		environment = "development"
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      environment,
		TracesSampleRate: 0.1, // Track 10% request sebagai trace
		Debug:            environment == "development",
	})
	if err != nil {
		log.Printf("[sentry] Gagal inisialisasi Sentry: %v", err)
		return
	}

	log.Printf("[sentry] Sentry aktif (env: %s)", environment)
}

// flushSentry memastikan semua event Sentry terkirim sebelum aplikasi shutdown.
func flushSentry() {
	sentry.Flush(2 * time.Second)
}

// sentryMiddleware adalah HTTP middleware yang menangkap panic dan error
// dari setiap HTTP request dan melaporkannya ke Sentry.
func sentryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetRequest(r)

		defer func() {
			if err := recover(); err != nil {
				hub.RecoverWithContext(r.Context(), err)
				sentry.Flush(2 * time.Second)
				httpError(w, http.StatusInternalServerError, "internal server error")
			}
		}()

		ctx := sentry.SetHubOnContext(r.Context(), hub)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// captureError melaporkan error ke Sentry jika Sentry aktif.
// Bisa dipakai di handler manapun untuk melaporkan error non-fatal.
func captureError(err error) {
	if err != nil {
		sentry.CaptureException(err)
	}
}
