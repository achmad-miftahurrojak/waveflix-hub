// Package main provides robust connection error handling with retry logic.
//
// The connection error handler implements exponential backoff retry strategies
// for database connection failures, helping applications recover gracefully
// from temporary network issues, database restarts, and other transient errors.
package main

import (
"database/sql"
"fmt"
"log"
"time"
)

// ConnectionErrorHandler manages connection retry logic with exponential backoff.
//
// The error handler implements a robust retry strategy that helps applications
// recover from temporary database connectivity issues. It uses exponential backoff
// to avoid overwhelming a recovering database server while providing reasonable
// response times for transient failures.
//
// Features:
//   - Configurable maximum retry attempts
//   - Exponential backoff with jitter reduction
//   - Backoff ceiling to prevent excessive delays
//   - Detailed retry attempt logging
//   - Graceful failure after max attempts
//
// Retry Strategy:
//   - Initial backoff: 1 second
//   - Backoff multiplier: 1.5x per attempt
//   - Maximum backoff: 30 seconds (prevents excessive delays)
//   - Default max attempts: 5 (configurable)
//
// Example usage:
//
//handler := NewConnectionErrorHandler(5, 30*time.Second)
//db, err := handler.ConnectWithRetry("pgx", connectionString)
//if err != nil {
//    log.Fatalf("Failed to connect after retries: %v", err)
//}
type ConnectionErrorHandler struct {
// maxRetries is the maximum number of connection attempts (including initial attempt).
// Default: 5 attempts
maxRetries int

// maxBackoff is the maximum delay between retry attempts.
// Prevents exponential backoff from creating excessive delays.
// Default: 30 seconds
maxBackoff time.Duration

// backoffMultiplier determines how quickly backoff delays increase.
// Each retry attempt multiplies the delay by this factor.
// Default: 1.5 (50% increase per attempt)
backoffMultiplier float64

// initialBackoff is the delay before the first retry attempt.
// Default: 1 second
initialBackoff time.Duration
}

// NewConnectionErrorHandler creates a new connection error handler with specified parameters.
//
// The handler uses exponential backoff to space out retry attempts, helping avoid
// overwhelming a database that may be experiencing temporary issues.
//
// Parameters:
//   - maxRetries: Maximum number of connection attempts (including initial attempt)
//   - maxBackoff: Maximum delay between retry attempts (prevents excessive waits)
//
// Returns: Configured ConnectionErrorHandler
//
// Example:
//
//// 5 attempts with maximum 30-second backoff
//handler := NewConnectionErrorHandler(5, 30*time.Second)
//
//// More aggressive retry for critical services
//handler := NewConnectionErrorHandler(10, 60*time.Second)
func NewConnectionErrorHandler(maxRetries int, maxBackoff time.Duration) *ConnectionErrorHandler {
return &ConnectionErrorHandler{
maxRetries:        maxRetries,
maxBackoff:        maxBackoff,
backoffMultiplier: 1.5,
initialBackoff:    1 * time.Second,
}
}

// ConnectWithRetry attempts to establish a database connection with retry logic.
//
// This method implements exponential backoff retry strategy for database connections.
// It's particularly useful during application startup, database failover scenarios,
// and when connecting to databases that may be temporarily unavailable.
//
// Retry Behavior:
//  1. Attempt immediate connection (attempt 1)
//  2. If failed and retries remain, wait with exponential backoff
//  3. Retry connection (attempts 2-N)
//  4. Return connection on success, or error after maxRetries attempts
//
// Backoff Calculation:
//   - Attempt 1: immediate (no delay)
//   - Attempt 2: 1.0 seconds
//   - Attempt 3: 1.5 seconds
//   - Attempt 4: 2.25 seconds
//   - Attempt 5: 3.375 seconds
//   - Capped at maxBackoff (default: 30 seconds)
//
// Parameters:
//   - driverName: Database driver name (e.g., "pgx" for PostgreSQL)
//   - connectionString: Database connection string or DSN
//
// Returns:
//   - *sql.DB: Successfully opened database connection
//   - error: Connection error if all retry attempts failed
//
// Example:
//
//handler := NewConnectionErrorHandler(5, 30*time.Second)
//db, err := handler.ConnectWithRetry("pgx", "postgresql://user:pass@localhost/db")
//if err != nil {
//    log.Fatalf("Database connection failed: %v", err)
//}
//defer db.Close()
func (h *ConnectionErrorHandler) ConnectWithRetry(driverName, connectionString string) (*sql.DB, error) {
var lastErr error
backoffDuration := h.initialBackoff

for attempt := 1; attempt <= h.maxRetries; attempt++ {
// Attempt connection
db, err := sql.Open(driverName, connectionString)
if err == nil {
// Connection opened successfully, verify with ping
pingErr := db.Ping(); if pingErr == nil {
if attempt > 1 {
log.Printf("[db] Connection established successfully on attempt %d", attempt)
}
return db, nil
}
// Ping failed, close connection and retry
db.Close()
lastErr = pingErr
} else {
lastErr = err
}

// If this wasn't the last attempt, wait and retry
if attempt < h.maxRetries {
log.Printf("[db] Connection attempt %d failed: %v (retrying in %v)", 
attempt, lastErr, backoffDuration)

time.Sleep(backoffDuration)

// Calculate next backoff duration with exponential increase
backoffDuration = time.Duration(float64(backoffDuration) * h.backoffMultiplier)

// Cap backoff at maximum to prevent excessive delays
if backoffDuration > h.maxBackoff {
backoffDuration = h.maxBackoff
}
}
}

// All retry attempts failed
return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", h.maxRetries, lastErr)
}

// ConnectWithRetryAndValidation attempts connection with custom validation logic.
//
// This method extends ConnectWithRetry by allowing custom validation of the
// database connection beyond the standard Ping() check. This is useful for
// applications that need to verify specific database state, run initialization
// queries, or check permissions before considering a connection successful.
//
// The validation function receives the newly opened connection and should
// return nil if the connection is acceptable, or an error if additional
// setup or checks are needed.
//
// Parameters:
//   - driverName: Database driver name
//   - connectionString: Database connection string
//   - validateFn: Custom validation function for the connection
//
// Returns:
//   - *sql.DB: Successfully opened and validated database connection
//   - error: Connection or validation error if all attempts failed
//
// Example:
//
//validateFn := func(db *sql.DB) error {
//    var version string
//    err := db.QueryRow("SELECT version()").Scan(&version)
//    if err != nil {
//        return err
//    }
//    log.Printf("Connected to: %s", version)
//    return nil
//}
//
//db, err := handler.ConnectWithRetryAndValidation("pgx", connectionString, validateFn)
func (h *ConnectionErrorHandler) ConnectWithRetryAndValidation(
driverName, connectionString string,
validateFn func(*sql.DB) error) (*sql.DB, error) {

var lastErr error
backoffDuration := h.initialBackoff

for attempt := 1; attempt <= h.maxRetries; attempt++ {
// Attempt connection
db, err := sql.Open(driverName, connectionString)
if err == nil {
// Connection opened successfully, verify with ping
pingErr := db.Ping(); if pingErr == nil {
// Run custom validation
if validateErr := validateFn(db); validateErr == nil {
if attempt > 1 {
log.Printf("[db] Connection established and validated successfully on attempt %d", attempt)
}
return db, nil
} else {
lastErr = validateErr
}
} else {
lastErr = pingErr
}
// Validation or ping failed, close connection and retry
db.Close()
} else {
lastErr = err
}

// If this wasn't the last attempt, wait and retry
if attempt < h.maxRetries {
log.Printf("[db] Connection attempt %d failed: %v (retrying in %v)", 
attempt, lastErr, backoffDuration)

time.Sleep(backoffDuration)

// Calculate next backoff duration
backoffDuration = time.Duration(float64(backoffDuration) * h.backoffMultiplier)
if backoffDuration > h.maxBackoff {
backoffDuration = h.maxBackoff
}
}
}

// All retry attempts failed
return nil, fmt.Errorf("failed to connect and validate database after %d attempts: %w", h.maxRetries, lastErr)
}

// IsRetriableError determines whether a database error should trigger a retry.
//
// Some database errors indicate permanent issues (authentication failure, 
// invalid database name) while others indicate temporary problems (connection
// timeout, server restart). This method helps distinguish between the two.
//
// Retriable error patterns include:
//   - Connection timeouts
//   - Connection refused (server temporarily down)
//   - Network unreachable
//   - Connection reset by peer
//   - Server temporarily overloaded
//
// Non-retriable error patterns include:
//   - Authentication failures
//   - Database does not exist
//   - SSL/TLS certificate errors
//   - Permission denied
//
// Parameters:
//   - err: Database error to evaluate
//
// Returns: true if the error indicates a potentially temporary issue
//
// Example:
//
//if err != nil && !handler.IsRetriableError(err) {
//    log.Fatalf("Permanent database error: %v", err)
//}
func (h *ConnectionErrorHandler) IsRetriableError(err error) bool {
if err == nil {
return false
}

errorMsg := err.Error()

// Common retriable error patterns
retriablePatterns := []string{
"connection refused",
"connection timed out",
"connection timeout",
"network unreachable",
"connection reset by peer",
"server closed the connection",
"broken pipe",
"no such host", // DNS temporary failure
"temporary failure in name resolution",
}

for _, pattern := range retriablePatterns {
if containsIgnoreCase(errorMsg, pattern) {
return true
}
}

// Non-retriable error patterns (permanent issues)
permanentPatterns := []string{
"authentication failed",
"password authentication failed",
"database does not exist",
"role does not exist",
"permission denied",
"ssl certificate verify failed",
"invalid connection string",
}

for _, pattern := range permanentPatterns {
if containsIgnoreCase(errorMsg, pattern) {
return false
}
}

// Default to retriable for unknown errors
// This err-on-the-side-of-retrying approach helps with unexpected transient issues
return true
}

// containsIgnoreCase performs case-insensitive substring search.
//
// This helper function is used internally for error pattern matching.
//
// Parameters:
//   - s: String to search in
//   - substr: Substring to search for
//
// Returns: true if substr is found in s (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
// Simple case-insensitive contains check
// Convert both strings to lowercase for comparison
sLower := ""
substrLower := ""

for _, r := range s {
if r >= 'A' && r <= 'Z' {
sLower += string(r + 32) // Convert to lowercase
} else {
sLower += string(r)
}
}

for _, r := range substr {
if r >= 'A' && r <= 'Z' {
substrLower += string(r + 32) // Convert to lowercase
} else {
substrLower += string(r)
}
}

// Check if substrLower is contained in sLower
if len(substrLower) == 0 {
return true
}
if len(sLower) < len(substrLower) {
return false
}

for i := 0; i <= len(sLower)-len(substrLower); i++ {
match := true
for j := 0; j < len(substrLower); j++ {
if sLower[i+j] != substrLower[j] {
match = false
break
}
}
if match {
return true
}
}

return false
}


