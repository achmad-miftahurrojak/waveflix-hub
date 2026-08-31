package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

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
		TracesSampleRate: 0.1, 
		Debug:            environment == "development",
	})
	if err != nil {
		log.Printf("[sentry] Gagal inisialisasi Sentry: %v", err)
		return
	}

	log.Printf("[sentry] Sentry aktif (env: %s)", environment)
}

func flushSentry() {
	sentry.Flush(2 * time.Second)
}

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

func captureError(err error) {
	if err != nil {
		sentry.CaptureException(err)
	}
}
