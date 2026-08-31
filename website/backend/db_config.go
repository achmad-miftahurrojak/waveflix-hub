package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// DatabaseConfig holds configuration for database connection selection.
//
// This configuration determines which database backend to use (SQLite or PostgreSQL)
// based on environment variables. The configuration supports multiple input formats:
//   - Explicit DATABASE_TYPE selection
//   - Full connection URL via DATABASE_URL
//   - Individual connection parameters via PG_* environment variables
//   - Automatic fallback to SQLite when no PostgreSQL configuration is present
type DatabaseConfig struct {
	// Type specifies the database backend: "postgres" or "sqlite"
	// When empty, the type is determined automatically based on available configuration
	Type string

	// Postgres holds PostgreSQL-specific connection configuration
	// This is nil when using SQLite
	Postgres *PostgresConfig

	// SQLite holds SQLite-specific configuration
	// This is nil when using PostgreSQL
	SQLite *SQLiteConfig
}

// PostgresConfig holds PostgreSQL connection parameters.
//
// Connection parameters can be specified in two ways:
//  1. Via DATABASE_URL: A complete connection string in the format:
//     postgresql://user:password@host:port/database?sslmode=require
//  2. Via individual PG_* environment variables:
//     PG_HOST, PG_PORT, PG_USER, PG_PASSWORD, PG_DATABASE, PG_SSLMODE
//
// If both are provided, DATABASE_URL takes precedence.
type PostgresConfig struct {
	// URL is the full connection string (preferred method)
	// Format: postgresql://user:password@host:port/database?sslmode=require
	// When URL is provided, individual parameters (Host, Port, etc.) are ignored
	URL string

	// Host is the PostgreSQL server hostname or IP address
	// Default: localhost
	Host string

	// Port is the PostgreSQL server port
	// Default: 5432
	Port string

	// User is the database user for authentication
	// Required when using individual parameters
	User string

	// Password is the database password for authentication
	// Required when using individual parameters
	Password string

	// Database is the name of the database to connect to
	// Required when using individual parameters
	Database string

	// SSLMode specifies the SSL/TLS connection mode
	// Valid values: disable, require, verify-ca, verify-full
	// Default: require
	//
	// Security recommendations:
	//   - "disable": Only for local development, not production
	//   - "require": Ensures encrypted connection (recommended minimum)
	//   - "verify-ca": Verifies server certificate against CA
	//   - "verify-full": Verifies server certificate and hostname (most secure)
	SSLMode string
}

// SQLiteConfig holds SQLite-specific configuration.
type SQLiteConfig struct {
	// File is the path to the SQLite database file
	// Default: waveflix.db
	File string
}

// LoadDatabaseConfig loads database configuration from environment variables.
//
// Configuration priority order:
//  1. DATABASE_TYPE environment variable (explicit selection)
//  2. DATABASE_URL presence (implies PostgreSQL)
//  3. PG_HOST presence (implies PostgreSQL)
//  4. Fallback to SQLite
//
// Environment variables:
//   - DATABASE_TYPE: Explicit database selection ("sqlite" or "postgres")
//   - DATABASE_URL: Full PostgreSQL connection URL
//   - PG_HOST: PostgreSQL host
//   - PG_PORT: PostgreSQL port (default: 5432)
//   - PG_USER: PostgreSQL username
//   - PG_PASSWORD: PostgreSQL password
//   - PG_DATABASE: Database name
//   - PG_SSLMODE: SSL mode (default: require)
//   - PG_MAX_CONNS: Maximum open connections (default: 25)
//   - PG_MAX_IDLE_CONNS: Maximum idle connections (default: 25)
//
// Example usage:
//
//	config := LoadDatabaseConfig()
//	if config.Type == "postgres" {
//	    fmt.Printf("Using PostgreSQL at %s:%s\n", config.Postgres.Host, config.Postgres.Port)
//	}
func LoadDatabaseConfig() *DatabaseConfig {
	config := &DatabaseConfig{}

	// Check for explicit database type selection
	dbType := os.Getenv("DATABASE_TYPE")

	// Determine database type based on priority order
	switch {
	case dbType == "sqlite":
		config.Type = "sqlite"
		config.SQLite = loadSQLiteConfig()
		return config

	case dbType == "postgres":
		config.Type = "postgres"
		config.Postgres = loadPostgresConfig()
		return config

	case os.Getenv("DATABASE_URL") != "":
		// DATABASE_URL present implies PostgreSQL
		config.Type = "postgres"
		config.Postgres = loadPostgresConfig()
		return config

	case os.Getenv("PG_HOST") != "":
		// PG_HOST present implies PostgreSQL
		config.Type = "postgres"
		config.Postgres = loadPostgresConfig()
		return config

	default:
		// Fallback to SQLite
		config.Type = "sqlite"
		config.SQLite = loadSQLiteConfig()
		return config
	}
}

// loadPostgresConfig loads PostgreSQL configuration from environment variables.
func loadPostgresConfig() *PostgresConfig {
	config := &PostgresConfig{}

	// Check for DATABASE_URL first (takes precedence)
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		config.URL = databaseURL
		// Parse URL to extract individual components for reference
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

// loadSQLiteConfig loads SQLite configuration from environment variables.
func loadSQLiteConfig() *SQLiteConfig {
	return &SQLiteConfig{
		File: getEnvOrDefault("SQLITE_FILE", "waveflix.db"),
	}
}

// parsePostgresURL parses a PostgreSQL connection URL and extracts individual parameters.
//
// Format: postgresql://user:password@host:port/database?sslmode=require
//
// This function populates the individual fields (Host, Port, User, etc.) from the URL
// for reference and logging purposes, even though the URL is used directly for connection.
func parsePostgresURL(rawURL string, config *PostgresConfig) error {
	// Handle postgres:// scheme (common alternative to postgresql://)
	rawURL = strings.Replace(rawURL, "postgres://", "postgresql://", 1)

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("failed to parse DATABASE_URL: %w", err)
	}

	// Extract user and password
	if u.User != nil {
		config.User = u.User.Username()
		if password, ok := u.User.Password(); ok {
			config.Password = password
		}
	}

	// Extract host and port
	config.Host = u.Hostname()
	if port := u.Port(); port != "" {
		config.Port = port
	} else {
		config.Port = "5432" // default PostgreSQL port
	}

	// Extract database name (path without leading slash)
	config.Database = strings.TrimPrefix(u.Path, "/")

	// Extract SSL mode from query parameters
	if sslMode := u.Query().Get("sslmode"); sslMode != "" {
		config.SSLMode = sslMode
	} else {
		config.SSLMode = "require" // default
	}

	return nil
}

// buildPostgresURL constructs a PostgreSQL connection URL from individual parameters.
//
// Format: postgresql://user:password@host:port/database?sslmode=require
//
// This function is used when individual PG_* environment variables are provided
// instead of a full DATABASE_URL.
func buildPostgresURL(config *PostgresConfig) string {
	// URL encode password to handle special characters
	password := url.QueryEscape(config.Password)

	// Build connection URL
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

// getEnvOrDefault returns the value of an environment variable or a default value if not set.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns the integer value of an environment variable or a default value if not set.
//
// If the environment variable is set but cannot be parsed as an integer,
// the default value is returned and an error is logged.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		// Log parse error but don't fail, use default instead
		fmt.Printf("[config] Warning: Failed to parse %s as integer, using default %d\n", key, defaultValue)
	}
	return defaultValue
}

// ValidatePostgresConfig validates that required PostgreSQL configuration is present.
//
// Returns an error if critical configuration is missing.
func (c *PostgresConfig) ValidatePostgresConfig() error {
	if c.URL != "" {
		// If URL is provided, it should be complete
		return nil
	}

	// Validate individual parameters
	if c.Host == "" {
		return fmt.Errorf("PostgreSQL host is required (PG_HOST)")
	}
	if c.User == "" {
		return fmt.Errorf("PostgreSQL user is required (PG_USER)")
	}
	if c.Database == "" {
		return fmt.Errorf("PostgreSQL database is required (PG_DATABASE)")
	}
	// Password is optional for some auth methods (e.g., peer authentication)

	return nil
}

// ConnectionString returns the connection string for the PostgreSQL configuration.
//
// This is the string passed to sql.Open() to establish a connection.
func (c *PostgresConfig) ConnectionString() string {
	return c.URL
}

// String returns a sanitized string representation of the configuration for logging.
//
// Passwords and sensitive information are redacted.
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

// String returns a string representation of SQLite configuration for logging.
func (c *SQLiteConfig) String() string {
	return fmt.Sprintf("SQLite{File: %s}", c.File)
}
