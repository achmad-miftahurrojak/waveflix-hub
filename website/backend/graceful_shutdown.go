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

// GracefulShutdown manages graceful shutdown of the application
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

// ShutdownHook represents a function to call during shutdown
type ShutdownHook func() error

// NewGracefulShutdown creates a new graceful shutdown manager
func NewGracefulShutdown(server *http.Server, db *sql.DB, redisClient *redis.Client, 
	workerPool *WorkerPool, messageQueue *MessageQueue) *GracefulShutdown {
	
	return &GracefulShutdown{
		server:          server,
		db:              db,
		redisClient:     redisClient,
		workerPool:      workerPool,
		messageQueue:    messageQueue,
		shutdownFuncs:   make([]func() error, 0),
		shutdownTimeout: 30 * time.Second, // Default 30 seconds timeout
	}
}

// AddShutdownHook adds a function to be called during shutdown
func (gs *GracefulShutdown) AddShutdownHook(hook ShutdownHook) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.shutdownFuncs = append(gs.shutdownFuncs, hook)
}

// SetTimeout sets the shutdown timeout duration
func (gs *GracefulShutdown) SetTimeout(timeout time.Duration) {
	gs.shutdownTimeout = timeout
}

// IsShuttingDown returns whether the application is currently shutting down
func (gs *GracefulShutdown) IsShuttingDown() bool {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.isShuttingDown
}

// Start begins listening for shutdown signals and managing graceful shutdown
func (gs *GracefulShutdown) Start() {
	// Create channel to listen for interrupt signals
	signalChan := make(chan os.Signal, 1)
	
	// Register the channel to receive specific signals
	signal.Notify(signalChan, 
		os.Interrupt,    // SIGINT (Ctrl+C)
		syscall.SIGTERM, // SIGTERM (graceful shutdown)
		syscall.SIGQUIT, // SIGQUIT (graceful shutdown with core dump)
	)
	
	// Start goroutine to handle shutdown signals
	go gs.handleShutdownSignals(signalChan)
	
	log.Println("✅ Graceful shutdown handler initialized")
}

// handleShutdownSignals processes shutdown signals and initiates graceful shutdown
func (gs *GracefulShutdown) handleShutdownSignals(signalChan chan os.Signal) {
	// Block until we receive a signal
	sig := <-signalChan
	
	log.Printf("🛑 Received shutdown signal: %v", sig)
	
	// Mark that we're shutting down
	gs.mu.Lock()
	gs.isShuttingDown = true
	gs.mu.Unlock()
	
	// Start graceful shutdown process
	if err := gs.shutdown(); err != nil {
		log.Printf("❌ Error during graceful shutdown: %v", err)
		os.Exit(1)
	}
	
	log.Println("✅ Graceful shutdown completed successfully")
	os.Exit(0)
}

// Shutdown performs the actual graceful shutdown sequence
func (gs *GracefulShutdown) shutdown() error {
	log.Println("🔄 Starting graceful shutdown...")
	
	// Create context with timeout for the entire shutdown process
	ctx, cancel := context.WithTimeout(context.Background(), gs.shutdownTimeout)
	defer cancel()
	
	// Create error channel to collect errors from concurrent shutdowns
	errChan := make(chan error, 10)
	var wg sync.WaitGroup
	
	// Step 1: Stop accepting new connections (HTTP server)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("🔄 Shutting down HTTP server...")
		
		// Create a shorter timeout for server shutdown
		serverCtx, serverCancel := context.WithTimeout(ctx, 15*time.Second)
		defer serverCancel()
		
		if err := gs.server.Shutdown(serverCtx); err != nil {
			errChan <- fmt.Errorf("HTTP server shutdown error: %w", err)
			return
		}
		
		log.Println("✅ HTTP server shut down successfully")
	}()
	
	// Step 2: Stop background workers
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
	
	// Step 3: Stop message queue processing
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
	
	// Step 4: Execute custom shutdown hooks
	wg.Add(1)
	go func() {
		defer wg.Done()
		gs.executeShutdownHooks(errChan)
	}()
	
	// Wait for all shutdown operations to complete or timeout
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
	
	// Step 5: Close database connections (final step)
	if err := gs.closeDatabaseConnections(); err != nil {
		log.Printf("⚠️ Warning: Error closing database connections: %v", err)
	}
	
	// Collect any errors that occurred during shutdown
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
		// Return the first error
		return shutdownErrors[0]
	}
	
	return nil
}

// executeShutdownHooks runs all registered shutdown hooks
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

// closeDatabaseConnections closes database and Redis connections
func (gs *GracefulShutdown) closeDatabaseConnections() error {
	var errors []error
	
	// Close Redis connection
	if gs.redisClient != nil {
		log.Println("🔄 Closing Redis connection...")
		if err := gs.redisClient.Close(); err != nil {
			errors = append(errors, fmt.Errorf("Redis close error: %w", err))
		} else {
			log.Println("✅ Redis connection closed")
		}
	}
	
	// Close database connection
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

// Middleware that checks if the service is shutting down
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

// PreflightShutdown performs pre-shutdown health checks and notifications
func (gs *GracefulShutdown) PreflightShutdown() error {
	log.Println("🔄 Performing pre-shutdown checks...")
	
	// Check if there are critical operations in progress
	if gs.workerPool != nil {
		activeJobs := gs.workerPool.GetActiveJobCount()
		if activeJobs > 0 {
			log.Printf("⚠️ Warning: %d active jobs will be interrupted", activeJobs)
		}
	}
	
	// Check database connections
	if gs.db != nil {
		stats := gs.db.Stats()
		if stats.InUse > 0 {
			log.Printf("⚠️ Warning: %d database connections still in use", stats.InUse)
		}
	}
	
	// Notify external systems (if needed)
	// This could include service discovery, load balancers, etc.
	
	log.Println("✅ Pre-shutdown checks completed")
	return nil
}

// Emergency shutdown for critical situations (non-graceful)
func (gs *GracefulShutdown) EmergencyShutdown() {
	log.Println("🚨 EMERGENCY SHUTDOWN INITIATED")
	
	gs.mu.Lock()
	gs.isShuttingDown = true
	gs.mu.Unlock()
	
	// Force close all connections immediately
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

// WaitForShutdown blocks until shutdown is initiated
func (gs *GracefulShutdown) WaitForShutdown() {
	for !gs.IsShuttingDown() {
		time.Sleep(100 * time.Millisecond)
	}
}

// GetShutdownStatus returns current shutdown status information
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

// Health check endpoint that considers shutdown state  
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
		
		// Normal health check
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