package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
)

type GracefulShutdown struct {
	server         *http.Server
	db             *sql.DB
	redisClient    *redis.Client
	workerPool     *WorkerPool
	messageQueue   *MessageQueue
	shutdownFuncs  []func() error
	shutdownTimeout time.Duration
	mu             sync.RWMutex
	isShuttingDown bool
}

type ShutdownHook func() error

func NewGracefulShutdown(server *http.Server, db *sql.DB, redisClient *redis.Client, 
	workerPool *WorkerPool, messageQueue *MessageQueue) *GracefulShutdown {

	return &GracefulShutdown{
		server:          server,
		db:              db,
		redisClient:     redisClient,
		workerPool:      workerPool,
		messageQueue:    messageQueue,
		shutdownFuncs:   make([]func() error, 0),
		shutdownTimeout: 30 * time.Second, 
	}
}

func (gs *GracefulShutdown) AddShutdownHook(hook ShutdownHook) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.shutdownFuncs = append(gs.shutdownFuncs, hook)
}

func (gs *GracefulShutdown) SetTimeout(timeout time.Duration) {
	gs.shutdownTimeout = timeout
}

func (gs *GracefulShutdown) IsShuttingDown() bool {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.isShuttingDown
}

func (gs *GracefulShutdown) Start() {

	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, 
		os.Interrupt,    
		syscall.SIGTERM, 
		syscall.SIGQUIT, 
	)

	go gs.handleShutdownSignals(signalChan)

	log.Println("✅ Graceful shutdown handler initialized")
}

func (gs *GracefulShutdown) handleShutdownSignals(signalChan chan os.Signal) {

	sig := <-signalChan

	log.Printf("🛑 Received shutdown signal: %v", sig)

	gs.mu.Lock()
	gs.isShuttingDown = true
	gs.mu.Unlock()

	if err := gs.shutdown(); err != nil {
		log.Printf("❌ Error during graceful shutdown: %v", err)
		os.Exit(1)
	}

	log.Println("✅ Graceful shutdown completed successfully")
	os.Exit(0)
}

func (gs *GracefulShutdown) shutdown() error {
	log.Println("🔄 Starting graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), gs.shutdownTimeout)
	defer cancel()

	errChan := make(chan error, 10)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("🔄 Shutting down HTTP server...")

		serverCtx, serverCancel := context.WithTimeout(ctx, 15*time.Second)
		defer serverCancel()

		if err := gs.server.Shutdown(serverCtx); err != nil {
			errChan <- fmt.Errorf("HTTP server shutdown error: %w", err)
			return
		}

		log.Println("✅ HTTP server shut down successfully")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("🔄 Stopping worker pool...")

		if gs.workerPool != nil {
			if err := gs.workerPool.Shutdown(ctx); err != nil {
				errChan <- fmt.Errorf("worker pool shutdown error: %w", err)
				return
			}
		}

		log.Println("✅ Worker pool stopped successfully")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("🔄 Stopping message queue...")

		if gs.messageQueue != nil {
			if err := gs.messageQueue.Shutdown(ctx); err != nil {
				errChan <- fmt.Errorf("message queue shutdown error: %w", err)
				return
			}
		}

		log.Println("✅ Message queue stopped successfully")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		gs.executeShutdownHooks(errChan)
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("✅ All shutdown operations completed")
	case <-ctx.Done():
		return fmt.Errorf("shutdown timed out after %v", gs.shutdownTimeout)
	}

	if err := gs.closeDatabaseConnections(); err != nil {
		log.Printf("⚠️ Warning: Error closing database connections: %v", err)
	}

	close(errChan)
	var shutdownErrors []error
	for err := range errChan {
		shutdownErrors = append(shutdownErrors, err)
	}

	if len(shutdownErrors) > 0 {
		log.Printf("⚠️ Shutdown completed with %d errors:", len(shutdownErrors))
		for i, err := range shutdownErrors {
			log.Printf("  %d. %v", i+1, err)
		}

		return shutdownErrors[0]
	}

	return nil
}

func (gs *GracefulShutdown) executeShutdownHooks(errChan chan error) {
	log.Printf("🔄 Executing %d shutdown hooks...", len(gs.shutdownFuncs))

	gs.mu.RLock()
	hooks := make([]func() error, len(gs.shutdownFuncs))
	copy(hooks, gs.shutdownFuncs)
	gs.mu.RUnlock()

	for i, hook := range hooks {
		if err := hook(); err != nil {
			errChan <- fmt.Errorf("shutdown hook %d error: %w", i, err)
		}
	}

	log.Println("✅ All shutdown hooks executed")
}

func (gs *GracefulShutdown) closeDatabaseConnections() error {
	var errors []error

	if gs.redisClient != nil {
		log.Println("🔄 Closing Redis connection...")
		if err := gs.redisClient.Close(); err != nil {
			errors = append(errors, fmt.Errorf("Redis close error: %w", err))
		} else {
			log.Println("✅ Redis connection closed")
		}
	}

	if gs.db != nil {
		log.Println("🔄 Closing database connection...")
		if err := gs.db.Close(); err != nil {
			errors = append(errors, fmt.Errorf("database close error: %w", err))
		} else {
			log.Println("✅ Database connection closed")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("multiple connection close errors: %v", errors)
	}

	return nil
}

func (gs *GracefulShutdown) ShutdownMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if gs.IsShuttingDown() {
			w.Header().Set("Connection", "close")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error": "Service is shutting down", "status": "unavailable"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (gs *GracefulShutdown) PreflightShutdown() error {
	log.Println("🔄 Performing pre-shutdown checks...")

	if gs.workerPool != nil {
		activeJobs := gs.workerPool.GetActiveJobCount()
		if activeJobs > 0 {
			log.Printf("⚠️ Warning: %d active jobs will be interrupted", activeJobs)
		}
	}

	if gs.db != nil {
		stats := gs.db.Stats()
		if stats.InUse > 0 {
			log.Printf("⚠️ Warning: %d database connections still in use", stats.InUse)
		}
	}

	log.Println("✅ Pre-shutdown checks completed")
	return nil
}

func (gs *GracefulShutdown) EmergencyShutdown() {
	log.Println("🚨 EMERGENCY SHUTDOWN INITIATED")

	gs.mu.Lock()
	gs.isShuttingDown = true
	gs.mu.Unlock()

	if gs.server != nil {
		gs.server.Close()
	}

	if gs.db != nil {
		gs.db.Close()
	}

	if gs.redisClient != nil {
		gs.redisClient.Close()
	}

	log.Println("🚨 Emergency shutdown completed - process will exit")
	os.Exit(1)
}

func (gs *GracefulShutdown) WaitForShutdown() {
	for !gs.IsShuttingDown() {
		time.Sleep(100 * time.Millisecond)
	}
}

func (gs *GracefulShutdown) GetShutdownStatus() map[string]interface{} {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	status := map[string]interface{}{
		"is_shutting_down": gs.isShuttingDown,
		"timeout_seconds":  gs.shutdownTimeout.Seconds(),
		"hooks_registered": len(gs.shutdownFuncs),
	}

	if gs.workerPool != nil {
		status["active_workers"] = gs.workerPool.GetActiveJobCount()
	}

	if gs.db != nil {
		stats := gs.db.Stats()
		status["db_connections_in_use"] = stats.InUse
		status["db_connections_idle"] = stats.Idle
	}

	return status
}

func (gs *GracefulShutdown) HealthWithShutdownCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if gs.IsShuttingDown() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			response := map[string]interface{}{
				"status": "shutting_down",
				"message": "Service is gracefully shutting down",
				"timestamp": time.Now(),
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"status": "healthy",
			"timestamp": time.Now(),
			"uptime": time.Since(startTime).String(),
		}
		json.NewEncoder(w).Encode(response)
	}
}