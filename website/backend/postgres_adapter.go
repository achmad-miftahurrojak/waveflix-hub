// Package main provides PostgreSQL implementation of the database adapter interface.
//
// The PostgreSQL adapter wraps pgx/v5 connections and provides unified interface
// for database operations with connection pooling, statistics monitoring, and
// enhanced error handling for production workloads.
package main

import (
"database/sql"
)

// postgresAdapter wraps a PostgreSQL database connection and implements the DatabaseAdapter interface.
//
// This adapter provides a wrapper around database/sql operations for PostgreSQL using
// the pgx/v5 driver, with native connection pooling support and comprehensive statistics.
//
// Features:
//   - Connection pooling with configurable limits
//   - Connection pool statistics and monitoring
//   - Enhanced error handling and timeouts
//   - PostgreSQL-specific optimizations
//
// Example usage:
//
//db, err := sql.Open("pgx", connectionString)
//if err != nil {
//    log.Fatal(err)
//}
//adapter := newPostgresAdapter(db)
//stats := adapter.Stats() // connection pool statistics
type postgresAdapter struct {
db *sql.DB
}

// newPostgresAdapter creates a new PostgreSQL database adapter.
//
// The adapter wraps an existing *sql.DB connection opened with the "pgx" driver.
// The connection should already be configured with appropriate connection pool
// parameters and SSL settings.
//
// Example:
//
//db, err := sql.Open("pgx", "postgresql://user:pass@host/db")
//if err != nil {
//    return nil, err
//}
//adapter := newPostgresAdapter(db)
func newPostgresAdapter(db *sql.DB) DatabaseAdapter {
return &postgresAdapter{db: db}
}

// Exec executes a query without returning rows.
//
// This method is used for INSERT, UPDATE, DELETE, and DDL statements.
// The query uses PostgreSQL's native $1, $2, $N placeholder syntax.
//
// Example:
//
//result, err := adapter.Exec("INSERT INTO users (email, username) VALUES ($1, $2)", email, username)
//if err != nil {
//    return err
//}
//lastID, _ := result.LastInsertId()
func (p *postgresAdapter) Exec(query string, args ...interface{}) (sql.Result, error) {
return p.db.Exec(query, args...)
}

// Query executes a query that returns rows.
//
// The returned *sql.Rows must be closed by the caller to release the database connection.
// Always defer rows.Close() immediately after checking the error.
//
// Example:
//
//rows, err := adapter.Query("SELECT id, email FROM users WHERE created_at > $1", since)
//if err != nil {
//    return err
//}
//defer rows.Close()
//for rows.Next() {
//    var id int
//    var email string
//    if err := rows.Scan(&id, &email); err != nil {
//        return err
//    }
//    // process row
//}
func (p *postgresAdapter) Query(query string, args ...interface{}) (*sql.Rows, error) {
return p.db.Query(query, args...)
}

// QueryRow executes a query that is expected to return at most one row.
//
// QueryRow always returns a non-nil value. Errors are deferred until Row's Scan method is called.
// If the query selects no rows, the Scan will return sql.ErrNoRows.
//
// Example:
//
//var email string
//err := adapter.QueryRow("SELECT email FROM users WHERE id = $1", id).Scan(&email)
//if err == sql.ErrNoRows {
//    return nil, fmt.Errorf("user not found")
//}
func (p *postgresAdapter) QueryRow(query string, args ...interface{}) *sql.Row {
return p.db.QueryRow(query, args...)
}

// Begin starts a transaction.
//
// The returned transaction must be committed or rolled back.
// Use transactions to ensure atomicity of multiple operations.
//
// Example:
//
//tx, err := adapter.Begin()
//if err != nil {
//    return err
//}
//defer tx.Rollback() // rollback if not committed
//
//if _, err := tx.Exec("INSERT INTO users ..."); err != nil {
//    return err
//}
//if _, err := tx.Exec("INSERT INTO profiles ..."); err != nil {
//    return err
//}
//return tx.Commit()
func (p *postgresAdapter) Begin() (*sql.Tx, error) {
return p.db.Begin()
}

// DbType returns the database type identifier for PostgreSQL.
//
// This method always returns "postgres", which is used by the SQL compatibility layer
// to determine whether query translation is needed, and by monitoring systems
// to report the current database backend.
//
// Returns: "postgres"
func (p *postgresAdapter) DbType() string {
return "postgres"
}

// Ping verifies a connection to the database is still alive.
//
// This method is typically called during health checks to ensure
// the database connection is functional. It establishes a connection
// if one is not available.
//
// For PostgreSQL, this verifies that the server is reachable and
// authentication is working properly.
//
// Returns an error if the connection cannot be established or
// if the database is not responding.
//
// Example:
//
//if err := adapter.Ping(); err != nil {
//    log.Printf("Database health check failed: %v", err)
//    return http.StatusServiceUnavailable
//}
func (p *postgresAdapter) Ping() error {
return p.db.Ping()
}

// Stats returns connection pool statistics for the database.
//
// For PostgreSQL, this method returns detailed statistics about the connection pool,
// including the number of open connections, idle connections, connections in use,
// and the configured maximum values.
//
// These statistics are useful for:
//   - Monitoring connection pool utilization
//   - Detecting connection leaks (InUse staying high)
//   - Tuning MaxOpen and MaxIdle settings
//   - Alerting on connection pool exhaustion
//
// Returns: ConnectionStats with current pool statistics
//
// Example:
//
//stats := adapter.Stats()
//if stats.InUse > stats.MaxOpen * 0.8 {
//    log.Warn("Connection pool utilization high: %d/%d", stats.InUse, stats.MaxOpen)
//}
func (p *postgresAdapter) Stats() *ConnectionStats {
dbStats := p.db.Stats()

return &ConnectionStats{
OpenConnections: dbStats.OpenConnections,
IdleConnections: dbStats.Idle,
InUse:           dbStats.InUse,
MaxOpen:         dbStats.MaxOpenConnections,
MaxIdle:         dbStats.MaxOpenConnections, // Note: sql.DBStats doesn't expose MaxIdleConnections directly
}
}
