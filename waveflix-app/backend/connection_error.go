

package main

import (
"database/sql"
"fmt"
"log"
"time"
)

type ConnectionErrorHandler struct {

maxRetries int

maxBackoff time.Duration

backoffMultiplier float64

initialBackoff time.Duration
}

func NewConnectionErrorHandler(maxRetries int, maxBackoff time.Duration) *ConnectionErrorHandler {
return &ConnectionErrorHandler{
maxRetries:        maxRetries,
maxBackoff:        maxBackoff,
backoffMultiplier: 1.5,
initialBackoff:    1 * time.Second,
}
}

func (h *ConnectionErrorHandler) ConnectWithRetry(driverName, connectionString string) (*sql.DB, error) {
var lastErr error
backoffDuration := h.initialBackoff

for attempt := 1; attempt <= h.maxRetries; attempt++ {

db, err := sql.Open(driverName, connectionString)
if err == nil {

pingErr := db.Ping(); if pingErr == nil {
if attempt > 1 {
log.Printf("[db] Connection established successfully on attempt %d", attempt)
}
return db, nil
}

db.Close()
lastErr = pingErr
} else {
lastErr = err
}

if attempt < h.maxRetries {
log.Printf("[db] Connection attempt %d failed: %v (retrying in %v)", 
attempt, lastErr, backoffDuration)

time.Sleep(backoffDuration)

backoffDuration = time.Duration(float64(backoffDuration) * h.backoffMultiplier)

if backoffDuration > h.maxBackoff {
backoffDuration = h.maxBackoff
}
}
}

return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", h.maxRetries, lastErr)
}

func (h *ConnectionErrorHandler) ConnectWithRetryAndValidation(
driverName, connectionString string,
validateFn func(*sql.DB) error) (*sql.DB, error) {

var lastErr error
backoffDuration := h.initialBackoff

for attempt := 1; attempt <= h.maxRetries; attempt++ {

db, err := sql.Open(driverName, connectionString)
if err == nil {

pingErr := db.Ping(); if pingErr == nil {

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

db.Close()
} else {
lastErr = err
}

if attempt < h.maxRetries {
log.Printf("[db] Connection attempt %d failed: %v (retrying in %v)", 
attempt, lastErr, backoffDuration)

time.Sleep(backoffDuration)

backoffDuration = time.Duration(float64(backoffDuration) * h.backoffMultiplier)
if backoffDuration > h.maxBackoff {
backoffDuration = h.maxBackoff
}
}
}

return nil, fmt.Errorf("failed to connect and validate database after %d attempts: %w", h.maxRetries, lastErr)
}

func (h *ConnectionErrorHandler) IsRetriableError(err error) bool {
if err == nil {
return false
}

errorMsg := err.Error()

retriablePatterns := []string{
"connection refused",
"connection timed out",
"connection timeout",
"network unreachable",
"connection reset by peer",
"server closed the connection",
"broken pipe",
"no such host", 
"temporary failure in name resolution",
}

for _, pattern := range retriablePatterns {
if containsIgnoreCase(errorMsg, pattern) {
return true
}
}

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

return true
}

func containsIgnoreCase(s, substr string) bool {

sLower := ""
substrLower := ""

for _, r := range s {
if r >= 'A' && r <= 'Z' {
sLower += string(r + 32) 
} else {
sLower += string(r)
}
}

for _, r := range substr {
if r >= 'A' && r <= 'Z' {
substrLower += string(r + 32) 
} else {
substrLower += string(r)
}
}

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

