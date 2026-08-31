// Package main provides database abstraction for WaveFlix Hub.
//
// The database adapter layer enables transparent switching between SQLite and PostgreSQL
// without code duplication. It provides a unified interface for database operations
// and handles connection pooling, health checks, and connection statistics.
//
// The DatabaseAdapter interface abstracts all database operations, allowing the application
// to work with either SQLite or PostgreSQL seamlessly. This design enables gradual migration
// and rollback capabilities while maintaining a single codebase.
package main

import "database/sql"

// DatabaseAdapter provides a unified interface for database operations across different database backends.
//
// This interface abstracts the differences between SQLite and PostgreSQL, allowing the application
// to execute queries without knowledge of the underlying database type. The adapter handles
// SQL dialect translation, connection management, and database-specific features transparently.
//
// Implementations:
//   - sqliteAdapter: Wraps SQLite connections using modernc.org/sqlite driver
//   - postgresAdapter: Wraps PostgreSQL connections using pgx/v5 driver
//
// Example usage:
//
//	adapter, err := selector.Select()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	rows, err := adapter.Query("SELECT * FROM users WHERE id = ?", userID)
type DatabaseAdapter interface {
	// Exec executes a query without returning rows.
	// Typical use cases include INSERT, UPDATE, DELETE, and DDL statements.
	//
	// The query parameter may use either ? placeholders (SQLite style) or $N placeholders
	// (PostgreSQL style). The adapter handles translation as needed.
	//
	// Returns sql.Result containing information about the execution, such as
	// the number of rows affected and the last inserted ID (if applicable).
	//
	// Example:
	//   result, err := adapter.Exec("INSERT INTO users (email, username) VALUES (?, ?)", email, username)
	Exec(query string, args ...interface{}) (sql.Result, error)

	// Query executes a query that returns rows.
	// Typical use cases include SELECT statements.
	//
	// The returned *sql.Rows must be closed by the caller to release the database connection.
	// Always defer rows.Close() immediately after checking the error.
	//
	// Example:
	//   rows, err := adapter.Query("SELECT id, email FROM users WHERE created_at > ?", since)
	//   if err != nil {
	//       return err
	//   }
	//   defer rows.Close()
	//   for rows.Next() {
	//       // scan rows
	//   }
	Query(query string, args ...interface{}) (*sql.Rows, error)

	// QueryRow executes a query that is expected to return at most one row.
	// QueryRow always returns a non-nil value. Errors are deferred until Row's Scan method is called.
	//
	// If the query selects no rows, the Scan will return ErrNoRows.
	// Otherwise, Scan scans the first selected row and discards the rest.
	//
	// Example:
	//   var email string
	//   err := adapter.QueryRow("SELECT email FROM users WHERE id = ?", id).Scan(&email)
	//   if err == sql.ErrNoRows {
	//       // handle not found
	//   }
	QueryRow(query string, args ...interface{}) *sql.Row

	// Begin starts a transaction.
	//
	// The returned transaction must be committed or rolled back.
	// Use transactions to ensure atomicity of multiple operations.
	//
	// Example:
	//   tx, err := adapter.Begin()
	//   if err != nil {
	//       return err
	//   }
	//   defer tx.Rollback() // rollback if not committed
	//
	//   if _, err := tx.Exec("INSERT INTO users ..."); err != nil {
	//       return err
	//   }
	//   if _, err := tx.Exec("INSERT INTO profiles ..."); err != nil {
	//       return err
	//   }
	//   return tx.Commit()
	Begin() (*sql.Tx, error)

	// DbType returns the database type identifier.
	//
	// Returns:
	//   - "sqlite" for SQLite database connections
	//   - "postgres" for PostgreSQL database connections
	//
	// This method is used by the SQL compatibility layer to determine
	// whether query translation is needed, and by monitoring systems
	// to report the current database backend.
	DbType() string

	// Ping verifies a connection to the database is still alive.
	//
	// This method is typically called during health checks to ensure
	// the database connection is functional. It establishes a connection
	// if one is not available.
	//
	// Returns an error if the connection cannot be established or
	// if the database is not responding.
	Ping() error

	// Stats returns connection pool statistics for the database.
	//
	// Returns nil for database backends that do not support connection pooling (SQLite).
	// For PostgreSQL connections, returns statistics including:
	//   - OpenConnections: number of established connections
	//   - IdleConnections: number of idle connections waiting for reuse
	//   - InUse: number of connections currently in use
	//   - MaxOpen: maximum number of open connections allowed
	//   - MaxIdle: maximum number of idle connections allowed
	//
	// These statistics are useful for monitoring connection pool health
	// and tuning pool configuration parameters.
	Stats() *ConnectionStats
}

// ConnectionStats holds connection pool statistics for monitoring and diagnostics.
//
// These statistics are primarily relevant for PostgreSQL connections with connection pooling.
// SQLite adapters return nil from the Stats() method since SQLite does not use connection pooling.
//
// Use these statistics to:
//   - Monitor connection pool utilization
//   - Detect connection leaks (InUse staying high)
//   - Tune MaxOpen and MaxIdle settings for optimal performance
//   - Alert on connection pool exhaustion
type ConnectionStats struct {
	// OpenConnections is the total number of established connections to the database.
	// This includes both idle and in-use connections.
	OpenConnections int

	// IdleConnections is the number of idle connections waiting for reuse.
	// A healthy pool typically maintains some idle connections to handle traffic spikes.
	IdleConnections int

	// InUse is the number of connections currently being used for queries.
	// If this number approaches MaxOpen, the pool may be under-provisioned.
	InUse int

	// MaxOpen is the maximum number of open connections allowed.
	// Configured via PG_MAX_CONNS environment variable (default: 25).
	MaxOpen int

	// MaxIdle is the maximum number of idle connections allowed.
	// Configured via PG_MAX_IDLE_CONNS environment variable (default: 25).
	MaxIdle int
}
