// Package errors provides standardized error handling for the scalability testing system.
//
// This package defines error types, error handling strategies, and recovery mechanisms
// for different categories of failures in the testing and monitoring system.
package errors

import (
	"fmt"
	"runtime"
	"time"
)

// ErrorType represents the category of error for appropriate handling strategy.
type ErrorType string

const (
	// Configuration errors - startup validation and settings issues
	ErrorTypeConfiguration ErrorType = "configuration"
	
	// Integration errors - component integration failures
	ErrorTypeIntegration ErrorType = "integration"
	
	// LoadTest errors - load testing execution failures  
	ErrorTypeLoadTest ErrorType = "loadtest"
	
	// Monitoring errors - metrics collection and alerting failures
	ErrorTypeMonitoring ErrorType = "monitoring"
	
	// Network errors - connectivity and external service issues
	ErrorTypeNetwork ErrorType = "network"
	
	// Database errors - PostgreSQL connection and query issues
	ErrorTypeDatabase ErrorType = "database"
	
	// Cache errors - Redis and memory cache issues
	ErrorTypeCache ErrorType = "cache"
	
	// API errors - TMDB and external API failures
	ErrorTypeAPI ErrorType = "api"
	
	// Validation errors - test validation and assertion failures
	ErrorTypeValidation ErrorType = "validation"
	
	// System errors - resource exhaustion and system-level issues
	ErrorTypeSystem ErrorType = "system"
)

// Severity indicates the impact level of an error.
type Severity string

const (
	SeverityCritical Severity = "critical" // System cannot function
	SeverityHigh     Severity = "high"     // Major functionality impacted
	SeverityMedium   Severity = "medium"   // Minor functionality impacted
	SeverityLow      Severity = "low"      // Minimal impact
	SeverityInfo     Severity = "info"     // Informational only
)

// ScalabilityError represents a structured error with context and recovery information.
type ScalabilityError struct {
	Type        ErrorType              `json:"type"`
	Severity    Severity               `json:"severity"`
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	Component   string                 `json:"component"`
	Operation   string                 `json:"operation"`
	Cause       error                  `json:"cause,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Stack       string                 `json:"stack,omitempty"`
	Recovery    *RecoveryStrategy      `json:"recovery,omitempty"`
	Retryable   bool                   `json:"retryable"`
}

// Error implements the error interface.
func (e *ScalabilityError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s:%s] %s: %s (caused by: %v)", e.Type, e.Severity, e.Component, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s:%s] %s: %s", e.Type, e.Severity, e.Component, e.Message)
}

// Unwrap returns the underlying cause error for error unwrapping.
func (e *ScalabilityError) Unwrap() error {
	return e.Cause
}

// RecoveryStrategy defines how to handle and recover from specific errors.
type RecoveryStrategy struct {
	Action      RecoveryAction `json:"action"`
	RetryCount  int           `json:"retry_count"`
	RetryDelay  time.Duration `json:"retry_delay"`
	BackoffType BackoffType   `json:"backoff_type"`
	Fallback    string        `json:"fallback,omitempty"`
	Timeout     time.Duration `json:"timeout,omitempty"`
}

// RecoveryAction defines the type of recovery action to take.
type RecoveryAction string

const (
	RecoveryRetry    RecoveryAction = "retry"
	RecoveryFallback RecoveryAction = "fallback"
	RecoveryIgnore   RecoveryAction = "ignore"
	RecoveryFail     RecoveryAction = "fail"
	RecoveryDegrade  RecoveryAction = "degrade"
)

// BackoffType defines the backoff strategy for retries.
type BackoffType string

const (
	BackoffLinear      BackoffType = "linear"
	BackoffExponential BackoffType = "exponential"
	BackoffFixed       BackoffType = "fixed"
	BackoffJittered    BackoffType = "jittered"
)

// ErrorHandler defines the interface for handling different error types.
type ErrorHandler interface {
	HandleError(err *ScalabilityError) error
	ShouldRetry(err *ScalabilityError) bool
	GetRecoveryStrategy(err *ScalabilityError) *RecoveryStrategy
}

// DefaultErrorHandler provides standard error handling strategies.
type DefaultErrorHandler struct{}

// NewDefaultErrorHandler creates a new default error handler.
func NewDefaultErrorHandler() *DefaultErrorHandler {
	return &DefaultErrorHandler{}
}

// HandleError processes errors according to their type and severity.
func (h *DefaultErrorHandler) HandleError(err *ScalabilityError) error {
	strategy := h.GetRecoveryStrategy(err)
	if strategy == nil {
		return err
	}

	switch strategy.Action {
	case RecoveryRetry:
		return h.handleRetry(err, strategy)
	case RecoveryFallback:
		return h.handleFallback(err, strategy)
	case RecoveryIgnore:
		return nil // Ignore the error
	case RecoveryDegrade:
		return h.handleDegrade(err, strategy)
	case RecoveryFail:
		return err
	default:
		return err
	}
}

// ShouldRetry determines if an error should be retried.
func (h *DefaultErrorHandler) ShouldRetry(err *ScalabilityError) bool {
	if !err.Retryable {
		return false
	}

	strategy := h.GetRecoveryStrategy(err)
	if strategy == nil {
		return false
	}

	return strategy.Action == RecoveryRetry && strategy.RetryCount > 0
}

// GetRecoveryStrategy returns the appropriate recovery strategy for an error.
func (h *DefaultErrorHandler) GetRecoveryStrategy(err *ScalabilityError) *RecoveryStrategy {
	switch err.Type {
	case ErrorTypeConfiguration:
		return &RecoveryStrategy{
			Action:     RecoveryFail, // Configuration errors are not recoverable
			RetryCount: 0,
		}

	case ErrorTypeNetwork, ErrorTypeAPI:
		return &RecoveryStrategy{
			Action:      RecoveryRetry,
			RetryCount:  3,
			RetryDelay:  time.Second * 2,
			BackoffType: BackoffExponential,
			Timeout:     time.Second * 30,
		}

	case ErrorTypeCache:
		return &RecoveryStrategy{
			Action:      RecoveryFallback,
			RetryCount:  2,
			RetryDelay:  time.Millisecond * 500,
			BackoffType: BackoffLinear,
			Fallback:    "memory_cache",
		}

	case ErrorTypeDatabase:
		return &RecoveryStrategy{
			Action:      RecoveryRetry,
			RetryCount:  5,
			RetryDelay:  time.Second * 1,
			BackoffType: BackoffJittered,
			Timeout:     time.Second * 15,
		}

	case ErrorTypeMonitoring:
		return &RecoveryStrategy{
			Action:     RecoveryDegrade,
			RetryCount: 2,
			RetryDelay: time.Second * 5,
			Fallback:   "local_logging",
		}

	case ErrorTypeValidation:
		return &RecoveryStrategy{
			Action:     RecoveryFail, // Validation errors indicate test issues
			RetryCount: 0,
		}

	default:
		return &RecoveryStrategy{
			Action:      RecoveryRetry,
			RetryCount:  1,
			RetryDelay:  time.Second * 1,
			BackoffType: BackoffFixed,
		}
	}
}

// handleRetry implements retry logic with backoff.
func (h *DefaultErrorHandler) handleRetry(err *ScalabilityError, strategy *RecoveryStrategy) error {
	// This would be implemented by the calling code using the strategy
	// Return the error to indicate retry should be handled upstream
	return err
}

// handleFallback implements fallback logic.
func (h *DefaultErrorHandler) handleFallback(err *ScalabilityError, strategy *RecoveryStrategy) error {
	// Log the fallback action
	return fmt.Errorf("falling back to %s due to: %w", strategy.Fallback, err)
}

// handleDegrade implements service degradation.
func (h *DefaultErrorHandler) handleDegrade(err *ScalabilityError, strategy *RecoveryStrategy) error {
	// Log the degradation and continue with reduced functionality
	return nil
}

// Error constructors for different error types

// NewConfigurationError creates a configuration-related error.
func NewConfigurationError(component, operation, message string, cause error) *ScalabilityError {
	return &ScalabilityError{
		Type:      ErrorTypeConfiguration,
		Severity:  SeverityCritical,
		Code:      "CONFIG_ERROR",
		Message:   message,
		Component: component,
		Operation: operation,
		Cause:     cause,
		Timestamp: time.Now(),
		Stack:     getStack(),
		Retryable: false,
	}
}

// NewIntegrationError creates an integration test failure error.
func NewIntegrationError(component, operation, message string, cause error) *ScalabilityError {
	return &ScalabilityError{
		Type:      ErrorTypeIntegration,
		Severity:  SeverityHigh,
		Code:      "INTEGRATION_ERROR",
		Message:   message,
		Component: component,
		Operation: operation,
		Cause:     cause,
		Timestamp: time.Now(),
		Stack:     getStack(),
		Retryable: true,
	}
}

// NewLoadTestError creates a load test execution error.
func NewLoadTestError(component, operation, message string, cause error) *ScalabilityError {
	return &ScalabilityError{
		Type:      ErrorTypeLoadTest,
		Severity:  SeverityMedium,
		Code:      "LOADTEST_ERROR",
		Message:   message,
		Component: component,
		Operation: operation,
		Cause:     cause,
		Timestamp: time.Now(),
		Stack:     getStack(),
		Retryable: true,
	}
}

// NewMonitoringError creates a monitoring system error.
func NewMonitoringError(component, operation, message string, cause error) *ScalabilityError {
	return &ScalabilityError{
		Type:      ErrorTypeMonitoring,
		Severity:  SeverityMedium,
		Code:      "MONITORING_ERROR",
		Message:   message,
		Component: component,
		Operation: operation,
		Cause:     cause,
		Timestamp: time.Now(),
		Stack:     getStack(),
		Retryable: true,
	}
}

// NewNetworkError creates a network connectivity error.
func NewNetworkError(component, operation, message string, cause error) *ScalabilityError {
	return &ScalabilityError{
		Type:      ErrorTypeNetwork,
		Severity:  SeverityHigh,
		Code:      "NETWORK_ERROR",
		Message:   message,
		Component: component,
		Operation: operation,
		Cause:     cause,
		Timestamp: time.Now(),
		Stack:     getStack(),
		Retryable: true,
	}
}

// NewDatabaseError creates a database-related error.
func NewDatabaseError(component, operation, message string, cause error) *ScalabilityError {
	return &ScalabilityError{
		Type:      ErrorTypeDatabase,
		Severity:  SeverityHigh,
		Code:      "DATABASE_ERROR",
		Message:   message,
		Component: component,
		Operation: operation,
		Cause:     cause,
		Timestamp: time.Now(),
		Stack:     getStack(),
		Retryable: true,
	}
}

// NewCacheError creates a cache-related error.
func NewCacheError(component, operation, message string, cause error) *ScalabilityError {
	return &ScalabilityError{
		Type:      ErrorTypeCache,
		Severity:  SeverityMedium,
		Code:      "CACHE_ERROR",
		Message:   message,
		Component: component,
		Operation: operation,
		Cause:     cause,
		Timestamp: time.Now(),
		Stack:     getStack(),
		Retryable: true,
	}
}

// NewAPIError creates an external API error.
func NewAPIError(component, operation, message string, cause error) *ScalabilityError {
	return &ScalabilityError{
		Type:      ErrorTypeAPI,
		Severity:  SeverityMedium,
		Code:      "API_ERROR", 
		Message:   message,
		Component: component,
		Operation: operation,
		Cause:     cause,
		Timestamp: time.Now(),
		Stack:     getStack(),
		Retryable: true,
	}
}

// NewValidationError creates a validation failure error.
func NewValidationError(component, operation, message string, cause error) *ScalabilityError {
	return &ScalabilityError{
		Type:      ErrorTypeValidation,
		Severity:  SeverityHigh,
		Code:      "VALIDATION_ERROR",
		Message:   message,
		Component: component,
		Operation: operation,
		Cause:     cause,
		Timestamp: time.Now(),
		Stack:     getStack(),
		Retryable: false,
	}
}

// NewSystemError creates a system-level error.
func NewSystemError(component, operation, message string, cause error) *ScalabilityError {
	return &ScalabilityError{
		Type:      ErrorTypeSystem,
		Severity:  SeverityCritical,
		Code:      "SYSTEM_ERROR",
		Message:   message,
		Component: component,
		Operation: operation,
		Cause:     cause,
		Timestamp: time.Now(),
		Stack:     getStack(),
		Retryable: false,
	}
}

// getStack returns the current stack trace for error context.
func getStack() string {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			return string(buf[:n])
		}
		buf = make([]byte, 2*len(buf))
	}
}

// WithDetails adds additional context details to an error.
func (e *ScalabilityError) WithDetails(details map[string]interface{}) *ScalabilityError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// WithRecovery sets a custom recovery strategy for an error.
func (e *ScalabilityError) WithRecovery(strategy *RecoveryStrategy) *ScalabilityError {
	e.Recovery = strategy
	return e
}

// IsRetryable returns whether the error can be retried.
func (e *ScalabilityError) IsRetryable() bool {
	return e.Retryable
}

// IsCritical returns whether the error is critical severity.
func (e *ScalabilityError) IsCritical() bool {
	return e.Severity == SeverityCritical
}