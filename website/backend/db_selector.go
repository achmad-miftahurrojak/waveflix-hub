// Package main provides database selection logic for WaveFlix Hub.
//
// The database selector determines which database backend to use (SQLite or PostgreSQL)
// based on environment variable configuration. It implements a priority-based selection
// algorithm that enables gradual migration and rollback capabilities.
package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
	_ "modernc.org/sqlite"             // SQLite driver
)

// DatabaseSelector chooses the appropriate database backend based on configuration.
//
// The selector implements a priority-based algorithm to determine which database
// to use, enabling both explicit configuration and automatic detection. This design
// supports gradual migration scenarios and rollback capabilities.
//
// Selection Priority Order:
//  1. DATABASE_TYPE environment variable (explicit selection: "sqlite" or "postgres")
//  2. DATABASE_URL presence (implies PostgreSQL)
//  3. PG_HOST presence (implies PostgreSQL with individual parameters)
//  4. Fallback to SQLite (backward compatibility)
//
// Example usage:
//
//	selector := NewDatabaseSelector()
//	adapter, err := selector.Select()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("Using database: %s", adapter.DbType())
type DatabaseSelector struct {
	config *DatabaseConfig
}

// NewDatabaseSelector creates a new database selector with loaded configuration.
//
// The selector reads environment variables on initialization and caches the
// configuration for the selection decision. This ensures consistent behavior
// even if environment variables change during application runtime.
//
// Returns a DatabaseSelector ready to perform database selection.
func NewDatabaseSelector() *DatabaseSelector {
	return &DatabaseSelector{
		config: LoadDatabaseConfig(),
	}
}

// Select chooses and returns the appropriate database adapter based on configuration.
//
// This method implements the core selection logic, evaluating environment variables
// in priority order and returning either a SQLite or PostgreSQL adapter. The selection
// is deterministic: identical environment variables always produce the same result.
//
// Selection Algorithm:
//  1. If DATABASE_TYPE="sqlite", use SQLite explicitly
//  2. If DATABASE_TYPE="postgres", use PostgreSQL explicitly
//  3. If DATABASE_URL is set, use PostgreSQL (URL-based config)
//  4. If PG_HOST is set, use PostgreSQL (parameter-based config)
//  5. Otherwise, fallback to SQLite for backward compatibility
//
// The method logs the selected database type for operational visibility.
//
// Returns:
//   - DatabaseAdapter: A configured adapter for the selected database
//   - error: Configuration or connection errors
//
// Example:
//
//	selector := NewDatabaseSelector()
//	adapter, err := selector.Select()
//	if err != nil {
//	    log.Fatalf("Failed to initialize database: %v", err)
//	}
//	// Use adapter for all database operations
func (s *DatabaseSelector) Select() (DatabaseAdapter, error) {
	switch s.config.Type {
	case "postgres":
		log.Printf("[db] Selected database: PostgreSQL (explicit configuration)")
		return s.newPostgresAdapter()

	case "sqlite":
		log.Printf("[db] Selected database: SQLite (explicit configuration)")
		return s.newSQLiteAdapter()

	default:
		// This should not happen if LoadDatabaseConfig works correctly,
		// but handle it gracefully
		log.Printf("[db] Warning: Unknown database type '%s', falling back to SQLite", s.config.Type)
		return s.newSQLiteAdapter()
	}
}

// newPostgresAdapter creates a PostgreSQL database adapter with connection pooling.
//
// This method establishes a connection to PostgreSQL using the pgx/v5 driver,
// configures connection pooling parameters, and wraps the connection in a
// DatabaseAdapter interface.
//
// Connection Configuration:
//   - Driver: pgx/v5 with database/sql compatibility
//   - Connection string: from DATABASE_URL or built from PG_* variables
//   - Connection pool: configured via PG_MAX_CONNS and PG_MAX_IDLE_CONNS
//   - Health checks: periodic ping to detect stale connections
//
// The method validates the PostgreSQL configuration before attempting connection
// and returns descriptive errors for common configuration issues.
//
// Returns:
//   - DatabaseAdapter: A PostgreSQL adapter with connection pooling
//   - error: Configuration validation or connection errors
//
// Error Cases:
//   - Missing required configuration (host, user, database)
//   - Connection refused (PostgreSQL not running or unreachable)
//   - Authentication failure (invalid credentials)
//   - SSL/TLS errors (certificate validation failures)
func (s *DatabaseSelector) newPostgresAdapter() (DatabaseAdapter, error) {
	if s.config.Postgres == nil {
		return nil, fmt.Errorf("PostgreSQL configuration is missing")
	}

	// Validate required PostgreSQL configuration
	if err := s.config.Postgres.ValidatePostgresConfig(); err != nil {
		return nil, fmt.Errorf("invalid PostgreSQL configuration: %w", err)
	}

	// Log connection info (sanitized, no password)
	log.Printf("[db] Connecting to PostgreSQL: %s", s.config.Postgres.String())

	// Open connection using pgx driver with database/sql compatibility
	// Driver name "pgx" is registered by importing github.com/jackc/pgx/v5/stdlib
	db, err := sql.Open("pgx", s.config.Postgres.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	// Configure connection pool
	// These settings can be overridden via PG_MAX_CONNS and PG_MAX_IDLE_CONNS
	maxConns := getEnvInt("PG_MAX_CONNS", 25)
	maxIdleConns := getEnvInt("PG_MAX_IDLE_CONNS", 25)

	db.SetMaxOpenConns(maxConns)
	db.SetMaxIdleConns(maxIdleConns)

	log.Printf("[db] PostgreSQL connection pool configured: MaxOpen=%d, MaxIdle=%d", maxConns, maxIdleConns)

	// Verify connectivity with a ping
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	log.Println("[db] PostgreSQL connection established successfully")

	// Create and return the PostgreSQL adapter
	// The adapter will be implemented in postgres_adapter.go (future task)
	return newPostgresAdapter(db), nil
}

// newSQLiteAdapter creates a SQLite database adapter.
//
// This method establishes a connection to the SQLite database file,
// applies SQLite-specific PRAGMA settings for optimal performance,
// and wraps the connection in a DatabaseAdapter interface.
//
// SQLite Configuration:
//   - Driver: modernc.org/sqlite (pure Go implementation)
//   - Database file: waveflix.db (default) or SQLITE_FILE environment variable
//   - WAL mode: Enabled for better concurrency
//   - Busy timeout: 5000ms to handle concurrent access
//   - Synchronous mode: NORMAL for balance between safety and performance
//
// The method handles legacy database file migration (summertide.db → waveflix.db)
// for backward compatibility with older deployments.
//
// Returns:
//   - DatabaseAdapter: A SQLite adapter configured for optimal performance
//   - error: File access or connection errors
//
// Error Cases:
//   - Database file not accessible (permission issues)
//   - Database file corrupted
//   - SQLite driver initialization failures
func (s *DatabaseSelector) newSQLiteAdapter() (DatabaseAdapter, error) {
	if s.config.SQLite == nil {
		return nil, fmt.Errorf("SQLite configuration is missing")
	}

	dbFile := s.config.SQLite.File
	log.Printf("[db] Connecting to SQLite: %s", dbFile)

	// Open SQLite connection using modernc.org/sqlite driver
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Apply SQLite PRAGMA settings for optimal performance
	// WAL mode improves concurrent read/write performance
	// Busy timeout prevents immediate failures under contention
	// NORMAL synchronous mode balances safety and performance
	pragmas := "PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA synchronous=NORMAL;"
	if _, err := db.Exec(pragmas); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set SQLite PRAGMA settings: %w", err)
	}

	// Configure connection pool for SQLite
	// While SQLite is not a client-server database, the database/sql package
	// still maintains a connection pool. Limit to 25 connections to prevent
	// excessive file handle usage.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)

	// Verify database is accessible
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to verify SQLite connection: %w", err)
	}

	log.Println("[db] SQLite connection established successfully")

	// Create and return the SQLite adapter
	return newSQLiteAdapter(db), nil
}

// GetDatabaseType returns the configured database type without creating an adapter.
//
// This utility method is useful for logging, monitoring, and conditional logic
// that needs to know the database type before adapter creation.
//
// Returns: "postgres" or "sqlite"
func (s *DatabaseSelector) GetDatabaseType() string {
	return s.config.Type
}
