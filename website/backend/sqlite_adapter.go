// Package main provides SQLite implementation of the database adapter interface.
//
// The SQLite adapter wraps the existing modernc.org/sqlite driver and provides
// a unified interface for database operations. This enables transparent switching
// between SQLite and PostgreSQL without code changes in the application layer.
package main

import "database/sql"

// sqliteAdapter wraps a SQLite database connection and implements the DatabaseAdapter interface.
//
// This adapter provides a thin wrapper around database/sql operations for SQLite,
// maintaining compatibility with the existing SQLite-based implementation while
// enabling gradual migration to PostgreSQL.
//
// Example usage:
//
//	db, err := sql.Open("sqlite", "waveflix.db")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	adapter := &sqliteAdapter{db: db}
//	rows, err := adapter.Query("SELECT * FROM users WHERE id = ?", userID)
type sqliteAdapter struct {
	db *sql.DB
}

// newSQLiteAdapter creates a new SQLite database adapter.
//
// The adapter wraps an existing *sql.DB connection opened with the "sqlite" driver.
// The connection should already be configured with appropriate PRAGMA settings
// and connection pool parameters.
//
// Example:
//
//	db, err := sql.Open("sqlite", "waveflix.db")
//	if err != nil {
//	    return nil, err
//	}
//	adapter := newSQLiteAdapter(db)
func newSQLiteAdapter(db *sql.DB) DatabaseAdapter {
	return &sqliteAdapter{db: db}
}

// Exec executes a query without returning rows.
//
// This method is used for INSERT, UPDATE, DELETE, and DDL statements.
// The query uses SQLite's native ? placeholder syntax.
//
// Example:
//
//	result, err := adapter.Exec("INSERT INTO users (email, username) VALUES (?, ?)", email, username)
//	if err != nil {
//	    return err
//	}
//	lastID, _ := result.LastInsertId()
func (s *sqliteAdapter) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.db.Exec(query, args...)
}

// Query executes a query that returns rows.
//
// The returned *sql.Rows must be closed by the caller to release the database connection.
// Always defer rows.Close() immediately after checking the error.
//
// Example:
//
//	rows, err := adapter.Query("SELECT id, email FROM users WHERE created_at > ?", since)
//	if err != nil {
//	    return err
//	}
//	defer rows.Close()
//	for rows.Next() {
//	    var id int
//	    var email string
//	    if err := rows.Scan(&id, &email); err != nil {
//	        return err
//	    }
//	    // process row
//	}
func (s *sqliteAdapter) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.Query(query, args...)
}

// QueryRow executes a query that is expected to return at most one row.
//
// QueryRow always returns a non-nil value. Errors are deferred until Row's Scan method is called.
// If the query selects no rows, the Scan will return sql.ErrNoRows.
//
// Example:
//
//	var email string
//	err := adapter.QueryRow("SELECT email FROM users WHERE id = ?", id).Scan(&email)
//	if err == sql.ErrNoRows {
//	    return nil, fmt.Errorf("user not found")
//	}
func (s *sqliteAdapter) QueryRow(query string, args ...interface{}) *sql.Row {
	return s.db.QueryRow(query, args...)
}

// Begin starts a transaction.
//
// The returned transaction must be committed or rolled back.
// Use transactions to ensure atomicity of multiple operations.
//
// Example:
//
//	tx, err := adapter.Begin()
//	if err != nil {
//	    return err
//	}
//	defer tx.Rollback() // rollback if not committed
//
//	if _, err := tx.Exec("INSERT INTO users ..."); err != nil {
//	    return err
//	}
//	if _, err := tx.Exec("INSERT INTO profiles ..."); err != nil {
//	    return err
//	}
//	return tx.Commit()
func (s *sqliteAdapter) Begin() (*sql.Tx, error) {
	return s.db.Begin()
}

// DbType returns the database type identifier for SQLite.
//
// This method always returns "sqlite", which is used by the SQL compatibility layer
// to determine whether query translation is needed, and by monitoring systems
// to report the current database backend.
//
// Returns: "sqlite"
func (s *sqliteAdapter) DbType() string {
	return "sqlite"
}

// Ping verifies a connection to the database is still alive.
//
// This method is typically called during health checks to ensure
// the database connection is functional. It establishes a connection
// if one is not available.
//
// For SQLite, this verifies that the database file is accessible
// and the connection is valid.
//
// Returns an error if the connection cannot be established or
// if the database is not responding.
//
// Example:
//
//	if err := adapter.Ping(); err != nil {
//	    log.Printf("Database health check failed: %v", err)
//	    return http.StatusServiceUnavailable
//	}
func (s *sqliteAdapter) Ping() error {
	return s.db.Ping()
}

// Stats returns connection pool statistics for the database.
//
// For SQLite, this method always returns nil because SQLite is an embedded
// database that does not use connection pooling in the same way as
// client-server databases like PostgreSQL.
//
// SQLite connections are lightweight and typically not pooled, as each
// connection directly accesses the database file. The database/sql package
// may maintain a small pool internally, but these statistics are not
// meaningful for SQLite and are not exposed.
//
// Returns: nil (connection pool statistics not applicable for SQLite)
func (s *sqliteAdapter) Stats() *ConnectionStats {
	return nil
}
