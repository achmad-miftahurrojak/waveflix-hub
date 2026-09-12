

package errors

import (
	"fmt"
	"runtime"
	"time"
)

type ErrorType string

const (

	ErrorTypeConfiguration ErrorType = "configuration"

	ErrorTypeIntegration ErrorType = "integration"

	ErrorTypeLoadTest ErrorType = "loadtest"

	ErrorTypeMonitoring ErrorType = "monitoring"

	ErrorTypeNetwork ErrorType = "network"

	ErrorTypeDatabase ErrorType = "database"

	ErrorTypeCache ErrorType = "cache"

	ErrorTypeAPI ErrorType = "api"

	ErrorTypeValidation ErrorType = "validation"

	ErrorTypeSystem ErrorType = "system"
)

type Severity string

const (
	SeverityCritical Severity = "critical" 
	SeverityHigh     Severity = "high"     
	SeverityMedium   Severity = "medium"   
	SeverityLow      Severity = "low"      
	SeverityInfo     Severity = "info"     
)

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

func (e *ScalabilityError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s:%s] %s: %s (caused by: %v)", e.Type, e.Severity, e.Component, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s:%s] %s: %s", e.Type, e.Severity, e.Component, e.Message)
}

func (e *ScalabilityError) Unwrap() error {
	return e.Cause
}

type RecoveryStrategy struct {
	Action      RecoveryAction `json:"action"`
	RetryCount  int           `json:"retry_count"`
	RetryDelay  time.Duration `json:"retry_delay"`
	BackoffType BackoffType   `json:"backoff_type"`
	Fallback    string        `json:"fallback,omitempty"`
	Timeout     time.Duration `json:"timeout,omitempty"`
}

type RecoveryAction string

const (
	RecoveryRetry    RecoveryAction = "retry"
	RecoveryFallback RecoveryAction = "fallback"
	RecoveryIgnore   RecoveryAction = "ignore"
	RecoveryFail     RecoveryAction = "fail"
	RecoveryDegrade  RecoveryAction = "degrade"
)

type BackoffType string

const (
	BackoffLinear      BackoffType = "linear"
	BackoffExponential BackoffType = "exponential"
	BackoffFixed       BackoffType = "fixed"
	BackoffJittered    BackoffType = "jittered"
)

type ErrorHandler interface {
	HandleError(err *ScalabilityError) error
	ShouldRetry(err *ScalabilityError) bool
	GetRecoveryStrategy(err *ScalabilityError) *RecoveryStrategy
}

type DefaultErrorHandler struct{}

func NewDefaultErrorHandler() *DefaultErrorHandler {
	return &DefaultErrorHandler{}
}

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
		return nil 
	case RecoveryDegrade:
		return h.handleDegrade(err, strategy)
	case RecoveryFail:
		return err
	default:
		return err
	}
}

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

func (h *DefaultErrorHandler) GetRecoveryStrategy(err *ScalabilityError) *RecoveryStrategy {
	switch err.Type {
	case ErrorTypeConfiguration:
		return &RecoveryStrategy{
			Action:     RecoveryFail, 
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
			Action:     RecoveryFail, 
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

func (h *DefaultErrorHandler) handleRetry(err *ScalabilityError, strategy *RecoveryStrategy) error {

	return err
}

func (h *DefaultErrorHandler) handleFallback(err *ScalabilityError, strategy *RecoveryStrategy) error {

	return fmt.Errorf("falling back to %s due to: %w", strategy.Fallback, err)
}

func (h *DefaultErrorHandler) handleDegrade(err *ScalabilityError, strategy *RecoveryStrategy) error {

	return nil
}

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

func (e *ScalabilityError) WithDetails(details map[string]interface{}) *ScalabilityError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

func (e *ScalabilityError) WithRecovery(strategy *RecoveryStrategy) *ScalabilityError {
	e.Recovery = strategy
	return e
}

func (e *ScalabilityError) IsRetryable() bool {
	return e.Retryable
}

func (e *ScalabilityError) IsCritical() bool {
	return e.Severity == SeverityCritical
}