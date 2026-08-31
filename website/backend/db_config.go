package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// PostgresConfig holds PostgreSQL connection parameters.
type PostgresConfig struct {
	URL      string
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

// LoadPostgresConfig loads PostgreSQL configuration from environment variables.
func LoadPostgresConfig() *PostgresConfig {
	config := &PostgresConfig{}

	// Check for DATABASE_URL first (takes precedence)
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		config.URL = databaseURL
		parsePostgresURL(databaseURL, config)
		return config
	}

	// Load individual parameters
	config.Host = getEnvOrDefault("PG_HOST", "localhost")
	config.Port = getEnvOrDefault("PG_PORT", "5432")
	config.User = os.Getenv("PG_USER")
	config.Password = os.Getenv("PG_PASSWORD")
	config.Database = os.Getenv("PG_DATABASE")
	config.SSLMode = getEnvOrDefault("PG_SSLMODE", "require")

	// Build connection URL from individual parameters
	config.URL = buildPostgresURL(config)

	return config
}

func parsePostgresURL(rawURL string, config *PostgresConfig) error {
	rawURL = strings.Replace(rawURL, "postgres://", "postgresql://", 1)

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("failed to parse DATABASE_URL: %w", err)
	}

	if u.User != nil {
		config.User = u.User.Username()
		if password, ok := u.User.Password(); ok {
			config.Password = password
		}
	}

	config.Host = u.Hostname()
	if port := u.Port(); port != "" {
		config.Port = port
	} else {
		config.Port = "5432"
	}

	config.Database = strings.TrimPrefix(u.Path, "/")

	if sslMode := u.Query().Get("sslmode"); sslMode != "" {
		config.SSLMode = sslMode
	} else {
		config.SSLMode = "require"
	}

	return nil
}

func buildPostgresURL(config *PostgresConfig) string {
	password := url.QueryEscape(config.Password)
	connURL := fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		config.User,
		password,
		config.Host,
		config.Port,
		config.Database,
		config.SSLMode,
	)
	return connURL
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		fmt.Printf("[config] Warning: Failed to parse %s as integer, using default %d\n", key, defaultValue)
	}
	return defaultValue
}

func (c *PostgresConfig) ValidateConfig() error {
	if c.URL != "" {
		return nil
	}

	if c.Host == "" {
		return fmt.Errorf("PostgreSQL host is required (PG_HOST)")
	}
	if c.User == "" {
		return fmt.Errorf("PostgreSQL user is required (PG_USER)")
	}
	if c.Database == "" {
		return fmt.Errorf("PostgreSQL database is required (PG_DATABASE)")
	}
	return nil
}

func (c *PostgresConfig) ConnectionString() string {
	return c.URL
}

func (c *PostgresConfig) String() string {
	return fmt.Sprintf(
		"PostgreSQL{Host: %s, Port: %s, User: %s, Database: %s, SSLMode: %s}",
		c.Host,
		c.Port,
		c.User,
		c.Database,
		c.SSLMode,
	)
}
