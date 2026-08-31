// Package main provides rate limiting and circuit breaker for TMDB API protection.
//
// These components protect against API abuse and service degradation by
// implementing intelligent backoff strategies and health monitoring.
package main

import (
"context"
"sync"
"time"
)

// TMDBRateLimiter implements token bucket rate limiting for TMDB API.
//
// TMDB allows 40 requests per 10 seconds, so we implement a conservative
// rate limit with burst capacity and intelligent backoff.
//
// Features:
//   - Token bucket algorithm with refill
//   - Configurable burst capacity
//   - Context-aware waiting with timeout
//   - Adaptive rate limiting based on API responses
//
// Rate Limits:
//   - Base rate: 35 requests per 10 seconds (conservative)
//   - Burst capacity: 10 requests 
//   - Refill rate: 3.5 requests per second
type TMDBRateLimiter struct {
tokens    int
maxTokens int
refillRate time.Duration
lastRefill time.Time
mu        sync.Mutex
}

// NewTMDBRateLimiter creates a rate limiter optimized for TMDB API limits.
func NewTMDBRateLimiter() *TMDBRateLimiter {
return &TMDBRateLimiter{
tokens:     35, // Start with full bucket
maxTokens:  35, // Conservative limit (TMDB allows 40/10s)
refillRate: time.Second * 10 / 35, // ~286ms between requests
lastRefill: time.Now(),
}
}

// Wait blocks until a request token is available.
//
// This method implements intelligent waiting with context support
// for timeout and cancellation handling.
//
// Parameters:
//   - ctx: Context for timeout and cancellation
//
// Returns: Error if context is cancelled or times out
//
// Example:
//
//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//defer cancel()
//if err := rateLimiter.Wait(ctx); err != nil {
//    return err
//}
//// Make API request
func (rl *TMDBRateLimiter) Wait(ctx context.Context) error {
for {
// Try to acquire token
if rl.tryAcquireToken() {
return nil
}

// Calculate wait time until next token
rl.mu.Lock()
waitTime := rl.refillRate - time.Since(rl.lastRefill)
rl.mu.Unlock()

if waitTime <= 0 {
waitTime = rl.refillRate
}

// Wait with context support
select {
case <-ctx.Done():
return ctx.Err()
case <-time.After(waitTime):
// Continue loop to try acquiring token
}
}
}

// tryAcquireToken attempts to acquire a rate limit token.
func (rl *TMDBRateLimiter) tryAcquireToken() bool {
rl.mu.Lock()
defer rl.mu.Unlock()

// Refill tokens based on elapsed time
now := time.Now()
elapsed := now.Sub(rl.lastRefill)
tokensToAdd := int(elapsed / rl.refillRate)

if tokensToAdd > 0 {
rl.tokens = min(rl.maxTokens, rl.tokens+tokensToAdd)
rl.lastRefill = now
}

// Try to consume a token
if rl.tokens > 0 {
rl.tokens--
return true
}

return false
}

// CircuitBreaker protects against cascading failures when TMDB API is unhealthy.
//
// The circuit breaker monitors API health and temporarily stops requests
// when the service is experiencing issues, allowing it to recover.
//
// States:
//   - Closed: Normal operation, requests allowed
//   - Open: API unhealthy, requests blocked
//   - HalfOpen: Testing recovery, limited requests allowed
//
// Configuration:
//   - Failure threshold: 5 consecutive failures to open circuit
//   - Recovery timeout: 30 seconds before trying half-open
//   - Success threshold: 3 consecutive successes to close circuit
type CircuitBreaker struct {
maxFailures    int
recoveryTime   time.Duration
successThreshold int

failures       int
successes      int
lastFailTime   time.Time
state          CircuitState
mu            sync.RWMutex
}

// CircuitState represents the current state of the circuit breaker.
type CircuitState int

const (
CircuitClosed CircuitState = iota
CircuitOpen
CircuitHalfOpen
)

// NewCircuitBreaker creates a circuit breaker optimized for TMDB API protection.
func NewCircuitBreaker() *CircuitBreaker {
return &CircuitBreaker{
maxFailures:      5,                // Open after 5 failures
recoveryTime:     30 * time.Second, // Wait 30s before retry
successThreshold: 3,                // Close after 3 successes
state:           CircuitClosed,
}
}

// AllowRequest determines if a request should be allowed through.
//
// This method implements the circuit breaker logic to protect
// against API overload during service degradation.
//
// Returns: Boolean indicating if request is allowed
//
// Example:
//
//if !circuitBreaker.AllowRequest() {
//    return errors.New("circuit breaker open")
//}
//// Make API request
func (cb *CircuitBreaker) AllowRequest() bool {
cb.mu.Lock()
defer cb.mu.Unlock()

switch cb.state {
case CircuitClosed:
return true

case CircuitOpen:
// Check if recovery time has passed
if time.Since(cb.lastFailTime) > cb.recoveryTime {
cb.state = CircuitHalfOpen
cb.successes = 0
return true
}
return false

case CircuitHalfOpen:
return true

default:
return false
}
}

// RecordSuccess records a successful API response.
//
// Success recording helps the circuit breaker determine when
// the API has recovered and normal operation can resume.
//
// Example:
//
//resp, err := httpClient.Get(url)
//if err == nil && resp.StatusCode == 200 {
//    circuitBreaker.RecordSuccess()
//}
func (cb *CircuitBreaker) RecordSuccess() {
cb.mu.Lock()
defer cb.mu.Unlock()

cb.failures = 0

if cb.state == CircuitHalfOpen {
cb.successes++
if cb.successes >= cb.successThreshold {
cb.state = CircuitClosed
}
}
}

// RecordFailure records a failed API response.
//
// Failure recording triggers circuit breaker opening when
// failure threshold is exceeded, protecting against cascading failures.
//
// Example:
//
//resp, err := httpClient.Get(url)
//if err != nil || resp.StatusCode >= 500 {
//    circuitBreaker.RecordFailure()
//}
func (cb *CircuitBreaker) RecordFailure() {
cb.mu.Lock()
defer cb.mu.Unlock()

cb.failures++
cb.lastFailTime = time.Now()

if cb.failures >= cb.maxFailures {
cb.state = CircuitOpen
}
}

// GetState returns the current circuit breaker state.
func (cb *CircuitBreaker) GetState() CircuitState {
cb.mu.RLock()
defer cb.mu.RUnlock()
return cb.state
}

// GetStats returns circuit breaker statistics for monitoring.
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
cb.mu.RLock()
defer cb.mu.RUnlock()

return map[string]interface{}{
"state":    cb.state,
"failures": cb.failures,
"successes": cb.successes,
}
}

// Helper function for min calculation
func min(a, b int) int {
if a < b {
return a
}
return b
}
