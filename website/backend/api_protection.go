

package main

import (
"context"
"sync"
"time"
)

type TMDBRateLimiter struct {
tokens    int
maxTokens int
refillRate time.Duration
lastRefill time.Time
mu        sync.Mutex
}

func NewTMDBRateLimiter() *TMDBRateLimiter {
return &TMDBRateLimiter{
tokens:     35, 
maxTokens:  35, 
refillRate: time.Second * 10 / 35, 
lastRefill: time.Now(),
}
}

func (rl *TMDBRateLimiter) Wait(ctx context.Context) error {
for {

if rl.tryAcquireToken() {
return nil
}

rl.mu.Lock()
waitTime := rl.refillRate - time.Since(rl.lastRefill)
rl.mu.Unlock()

if waitTime <= 0 {
waitTime = rl.refillRate
}

select {
case <-ctx.Done():
return ctx.Err()
case <-time.After(waitTime):

}
}
}

func (rl *TMDBRateLimiter) tryAcquireToken() bool {
rl.mu.Lock()
defer rl.mu.Unlock()

now := time.Now()
elapsed := now.Sub(rl.lastRefill)
tokensToAdd := int(elapsed / rl.refillRate)

if tokensToAdd > 0 {
rl.tokens = min(rl.maxTokens, rl.tokens+tokensToAdd)
rl.lastRefill = now
}

if rl.tokens > 0 {
rl.tokens--
return true
}

return false
}

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

type CircuitState int

const (
CircuitClosed CircuitState = iota
CircuitOpen
CircuitHalfOpen
)

func NewCircuitBreaker() *CircuitBreaker {
return &CircuitBreaker{
maxFailures:      5,                
recoveryTime:     30 * time.Second, 
successThreshold: 3,                
state:           CircuitClosed,
}
}

func (cb *CircuitBreaker) AllowRequest() bool {
cb.mu.Lock()
defer cb.mu.Unlock()

switch cb.state {
case CircuitClosed:
return true

case CircuitOpen:

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

func (cb *CircuitBreaker) RecordFailure() {
cb.mu.Lock()
defer cb.mu.Unlock()

cb.failures++
cb.lastFailTime = time.Now()

if cb.failures >= cb.maxFailures {
cb.state = CircuitOpen
}
}

func (cb *CircuitBreaker) GetState() CircuitState {
cb.mu.RLock()
defer cb.mu.RUnlock()
return cb.state
}

func (cb *CircuitBreaker) GetStats() map[string]interface{} {
cb.mu.RLock()
defer cb.mu.RUnlock()

return map[string]interface{}{
"state":    cb.state,
"failures": cb.failures,
"successes": cb.successes,
}
}

func min(a, b int) int {
if a < b {
return a
}
return b
}
