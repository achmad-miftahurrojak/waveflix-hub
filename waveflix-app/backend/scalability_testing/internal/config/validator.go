
package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hamin-baek/waveflix-hub/scalability-testing/internal/interfaces"
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) ValidateRedis(config *interfaces.RedisConfig) []string {
	var errors []string

	if config.URL == "" {
		errors = append(errors, "Redis URL cannot be empty")
	} else {

		if !strings.Contains(config.URL, "://") {

			if !strings.Contains(config.URL, ":") {
				errors = append(errors, "Redis URL must include port (host:port)")
			}
		} else {

			if _, err := url.Parse(config.URL); err != nil {
				errors = append(errors, fmt.Sprintf("Invalid Redis URL format: %v", err))
			}
		}
	}

	if config.DB < 0 || config.DB > 15 {
		errors = append(errors, "Redis database number must be between 0 and 15")
	}

	if config.MaxRetries < 0 {
		errors = append(errors, "Redis max retries cannot be negative")
	}
	if config.MaxRetries > 10 {
		errors = append(errors, "Redis max retries should not exceed 10 (excessive retry attempts)")
	}

	if config.PoolSize <= 0 {
		errors = append(errors, "Redis pool size must be positive")
	}
	if config.PoolSize > 1000 {
		errors = append(errors, "Redis pool size is too large (>1000 connections)")
	}

	if config.PoolTimeout <= 0 {
		errors = append(errors, "Redis pool timeout must be positive")
	}
	if config.PoolTimeout > time.Minute*5 {
		errors = append(errors, "Redis pool timeout is too long (>5 minutes)")
	}

	if config.IdleTimeout <= 0 {
		errors = append(errors, "Redis idle timeout must be positive")
	}
	if config.IdleTimeout < time.Second*30 {
		errors = append(errors, "Redis idle timeout is too short (<30 seconds)")
	}

	if config.DefaultTTL <= 0 {
		errors = append(errors, "Redis default TTL must be positive")
	}
	if config.DefaultTTL > time.Hour*24 {
		errors = append(errors, "Redis default TTL is very long (>24 hours)")
	}

	return errors
}

func (v *Validator) ValidatePostgreSQL(config *interfaces.PostgreSQLConfig) []string {
	var errors []string

	if config.URL == "" {
		errors = append(errors, "PostgreSQL URL cannot be empty")
	} else {
		parsedURL, err := url.Parse(config.URL)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Invalid PostgreSQL URL format: %v", err))
		} else {

			if parsedURL.Scheme != "postgres" && parsedURL.Scheme != "postgresql" {
				errors = append(errors, "PostgreSQL URL must use postgres:// or postgresql:// scheme")
			}
			if parsedURL.Host == "" {
				errors = append(errors, "PostgreSQL URL must include host")
			}
		}
	}

	if config.MaxConnections <= 0 {
		errors = append(errors, "PostgreSQL max connections must be positive")
	}
	if config.MaxConnections > 200 {
		errors = append(errors, "PostgreSQL max connections is very high (>200)")
	}

	if config.MaxIdleConns < 0 {
		errors = append(errors, "PostgreSQL max idle connections cannot be negative")
	}
	if config.MaxIdleConns > config.MaxConnections {
		errors = append(errors, "PostgreSQL max idle connections cannot exceed max connections")
	}

	if config.ConnMaxLifetime <= 0 {
		errors = append(errors, "PostgreSQL connection max lifetime must be positive")
	}
	if config.ConnMaxLifetime < time.Minute*5 {
		errors = append(errors, "PostgreSQL connection max lifetime is too short (<5 minutes)")
	}

	if config.ConnMaxIdleTime <= 0 {
		errors = append(errors, "PostgreSQL connection max idle time must be positive")
	}
	if config.ConnMaxIdleTime > config.ConnMaxLifetime {
		errors = append(errors, "PostgreSQL connection max idle time should not exceed max lifetime")
	}

	validSSLModes := []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}
	isValidSSLMode := false
	for _, mode := range validSSLModes {
		if config.SSLMode == mode {
			isValidSSLMode = true
			break
		}
	}
	if !isValidSSLMode {
		errors = append(errors, fmt.Sprintf("Invalid PostgreSQL SSL mode: %s (valid: %v)", config.SSLMode, validSSLModes))
	}

	return errors
}

func (v *Validator) ValidateTMDB(config *interfaces.TMDBConfig) []string {
	var errors []string

	if config.APIKey == "" {
		errors = append(errors, "TMDB API key cannot be empty")
	} else {

		if len(config.APIKey) != 32 {
			errors = append(errors, "TMDB API key should be 32 characters long")
		}

		for _, char := range config.APIKey {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
				errors = append(errors, "TMDB API key should contain only hexadecimal characters")
				break
			}
		}
	}

	if config.BaseURL == "" {
		errors = append(errors, "TMDB base URL cannot be empty")
	} else {
		if _, err := url.Parse(config.BaseURL); err != nil {
			errors = append(errors, fmt.Sprintf("Invalid TMDB base URL: %v", err))
		} else if !strings.HasPrefix(config.BaseURL, "https://") {
			errors = append(errors, "TMDB base URL should use HTTPS")
		}
	}

	if config.RateLimit <= 0 {
		errors = append(errors, "TMDB rate limit must be positive")
	}
	if config.RateLimit > 40 {
		errors = append(errors, "TMDB rate limit exceeds API limits (40 requests per 10 seconds)")
	}

	if config.RatePeriod <= 0 {
		errors = append(errors, "TMDB rate period must be positive")
	}
	if config.RatePeriod < time.Second {
		errors = append(errors, "TMDB rate period is too short (<1 second)")
	}

	if config.Timeout <= 0 {
		errors = append(errors, "TMDB timeout must be positive")
	}
	if config.Timeout > time.Minute {
		errors = append(errors, "TMDB timeout is too long (>1 minute)")
	}

	if config.MaxRetries < 0 {
		errors = append(errors, "TMDB max retries cannot be negative")
	}
	if config.MaxRetries > 5 {
		errors = append(errors, "TMDB max retries is too high (>5)")
	}

	if config.BackoffBase <= 0 {
		errors = append(errors, "TMDB backoff base must be positive")
	}
	if config.BackoffBase > time.Second*10 {
		errors = append(errors, "TMDB backoff base is too long (>10 seconds)")
	}

	return errors
}

func (v *Validator) ValidateHTTP(config *interfaces.HTTPConfig) []string {
	var errors []string

	if config.MaxIdleConns <= 0 {
		errors = append(errors, "HTTP max idle connections must be positive")
	}
	if config.MaxIdleConns > 1000 {
		errors = append(errors, "HTTP max idle connections is very high (>1000)")
	}

	if config.MaxIdleConnsPerHost <= 0 {
		errors = append(errors, "HTTP max idle connections per host must be positive")
	}
	if config.MaxIdleConnsPerHost > config.MaxIdleConns {
		errors = append(errors, "HTTP max idle connections per host cannot exceed total max idle connections")
	}

	if config.MaxConnsPerHost <= 0 {
		errors = append(errors, "HTTP max connections per host must be positive")
	}
	if config.MaxConnsPerHost > 500 {
		errors = append(errors, "HTTP max connections per host is very high (>500)")
	}

	if config.IdleConnTimeout <= 0 {
		errors = append(errors, "HTTP idle connection timeout must be positive")
	}
	if config.IdleConnTimeout < time.Second*30 {
		errors = append(errors, "HTTP idle connection timeout is too short (<30 seconds)")
	}

	if config.TLSHandshakeTimeout <= 0 {
		errors = append(errors, "HTTP TLS handshake timeout must be positive")
	}
	if config.TLSHandshakeTimeout > time.Second*30 {
		errors = append(errors, "HTTP TLS handshake timeout is too long (>30 seconds)")
	}

	if config.ResponseHeaderTimeout <= 0 {
		errors = append(errors, "HTTP response header timeout must be positive")
	}
	if config.ResponseHeaderTimeout > time.Minute {
		errors = append(errors, "HTTP response header timeout is too long (>1 minute)")
	}

	if config.RequestTimeout <= 0 {
		errors = append(errors, "HTTP request timeout must be positive")
	}
	if config.RequestTimeout > time.Minute*5 {
		errors = append(errors, "HTTP request timeout is too long (>5 minutes)")
	}

	if config.TLSHandshakeTimeout > config.RequestTimeout {
		errors = append(errors, "HTTP TLS handshake timeout should not exceed request timeout")
	}

	if config.ResponseHeaderTimeout > config.RequestTimeout {
		errors = append(errors, "HTTP response header timeout should not exceed request timeout")
	}

	return errors
}

func (v *Validator) ValidateMonitoring(config *interfaces.MonitoringConfig) []string {
	var errors []string

	if config.MetricsInterval <= 0 {
		errors = append(errors, "Monitoring metrics interval must be positive")
	}
	if config.MetricsInterval < time.Second {
		errors = append(errors, "Monitoring metrics interval is too short (<1 second)")
	}
	if config.MetricsInterval > time.Minute*10 {
		errors = append(errors, "Monitoring metrics interval is too long (>10 minutes)")
	}

	if config.HistoryRetention <= 0 {
		errors = append(errors, "Monitoring history retention must be positive")
	}
	if config.HistoryRetention < time.Hour {
		errors = append(errors, "Monitoring history retention is too short (<1 hour)")
	}

	if config.DashboardPort <= 0 || config.DashboardPort > 65535 {
		errors = append(errors, "Monitoring dashboard port must be between 1 and 65535")
	}
	if config.MetricsPort <= 0 || config.MetricsPort > 65535 {
		errors = append(errors, "Monitoring metrics port must be between 1 and 65535")
	}
	if config.DashboardPort == config.MetricsPort {
		errors = append(errors, "Monitoring dashboard and metrics ports must be different")
	}

	reservedPorts := []int{22, 25, 53, 80, 110, 143, 443, 993, 995}
	for _, port := range reservedPorts {
		if config.DashboardPort == port {
			errors = append(errors, fmt.Sprintf("Monitoring dashboard port %d is reserved", port))
		}
		if config.MetricsPort == port {
			errors = append(errors, fmt.Sprintf("Monitoring metrics port %d is reserved", port))
		}
	}

	return errors
}

func (v *Validator) ValidateLoadTesting(config *interfaces.LoadTestingConfig) []string {
	var errors []string

	if config.MaxConcurrentUsers <= 0 {
		errors = append(errors, "Load testing max concurrent users must be positive")
	}
	if config.MaxConcurrentUsers > 10000 {
		errors = append(errors, "Load testing max concurrent users is very high (>10,000)")
	}

	if config.DefaultDuration <= 0 {
		errors = append(errors, "Load testing default duration must be positive")
	}
	if config.DefaultDuration < time.Second*30 {
		errors = append(errors, "Load testing default duration is too short (<30 seconds)")
	}
	if config.DefaultDuration > time.Hour*24 {
		errors = append(errors, "Load testing default duration is too long (>24 hours)")
	}

	if config.RampUpDuration < 0 {
		errors = append(errors, "Load testing ramp up duration cannot be negative")
	}
	if config.RampUpDuration > config.DefaultDuration {
		errors = append(errors, "Load testing ramp up duration should not exceed test duration")
	}

	if config.MetricsInterval <= 0 {
		errors = append(errors, "Load testing metrics interval must be positive")
	}
	if config.MetricsInterval < time.Second {
		errors = append(errors, "Load testing metrics interval is too short (<1 second)")
	}
	if config.MetricsInterval > time.Minute {
		errors = append(errors, "Load testing metrics interval is too long (>1 minute)")
	}

	if config.ResultsRetention <= 0 {
		errors = append(errors, "Load testing results retention must be positive")
	}
	if config.ResultsRetention < time.Hour*24 {
		errors = append(errors, "Load testing results retention is too short (<24 hours)")
	}

	return errors
}