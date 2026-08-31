// Package main provides SQL translation middleware for database adapters.
//
// The translation middleware wraps DatabaseAdapter implementations and automatically
// translates SQL queries from SQLite syntax to PostgreSQL syntax when needed.
// This enables transparent query compatibility without changing application code.
package main

import (
"database/sql"
)

// TranslatingDatabaseAdapter wraps a DatabaseAdapter with SQL translation capability.
//
// This middleware automatically translates SQL queries based on the underlying
// database type. For SQLite, queries pass through unchanged. For PostgreSQL,
// queries are translated from SQLite syntax to PostgreSQL syntax.
//
// Features:
//   - Transparent query translation
//   - Preserves all DatabaseAdapter interface methods
//   - No performance impact for SQLite
//   - Automatic detection of database type
//
// Example usage:
//
//rawAdapter := selector.Select()
//db := NewTranslatingDatabaseAdapter(rawAdapter)
//// All queries are now automatically translated
//db.Query("SELECT * FROM users WHERE id = ?") // Becomes $1 for PostgreSQL
type TranslatingDatabaseAdapter struct {
adapter    DatabaseAdapter
translator *SQLTranslator
}

// NewTranslatingDatabaseAdapter creates a new translating database adapter.
//
// The wrapper detects the underlying database type and creates an appropriate
// SQL translator. The translator is cached for performance.
//
// Parameters:
//   - adapter: The underlying database adapter to wrap
//
// Returns: TranslatingDatabaseAdapter with automatic query translation
//
// Example:
//
//rawAdapter := selector.Select()
//db := NewTranslatingDatabaseAdapter(rawAdapter)
func NewTranslatingDatabaseAdapter(adapter DatabaseAdapter) DatabaseAdapter {
translator := NewSQLTranslator(adapter.DbType())

return &TranslatingDatabaseAdapter{
adapter:    adapter,
translator: translator,
}
}

// Exec executes a query without returning rows, with automatic translation.
//
// The query is translated based on the underlying database type before execution.
// For SQLite, the query passes through unchanged. For PostgreSQL, the query
// is converted from SQLite syntax to PostgreSQL syntax.
//
// Parameters:
//   - query: SQL query in SQLite syntax
//   - args: Query parameters
//
// Returns:
//   - sql.Result: Query execution result
//   - error: Execution or translation error
//
// Example:
//
//result, err := db.Exec("INSERT INTO users (email) VALUES (?)", email)
//// For PostgreSQL, becomes: "INSERT INTO users (email) VALUES ($1)"
func (tda *TranslatingDatabaseAdapter) Exec(query string, args ...interface{}) (sql.Result, error) {
translatedQuery := tda.translator.Translate(query)
return tda.adapter.Exec(translatedQuery, args...)
}

// Query executes a query that returns rows, with automatic translation.
//
// The query is translated based on the underlying database type before execution.
// The returned *sql.Rows behaves exactly the same regardless of the underlying database.
//
// Parameters:
//   - query: SQL query in SQLite syntax
//   - args: Query parameters
//
// Returns:
//   - *sql.Rows: Query result rows
//   - error: Execution or translation error
//
// Example:
//
//rows, err := db.Query("SELECT * FROM users WHERE active = 1")
//// For PostgreSQL, becomes: "SELECT * FROM users WHERE active = TRUE"
func (tda *TranslatingDatabaseAdapter) Query(query string, args ...interface{}) (*sql.Rows, error) {
translatedQuery := tda.translator.Translate(query)
return tda.adapter.Query(translatedQuery, args...)
}

// QueryRow executes a query that returns at most one row, with automatic translation.
//
// The query is translated based on the underlying database type before execution.
// The returned *sql.Row behaves exactly the same regardless of the underlying database.
//
// Parameters:
//   - query: SQL query in SQLite syntax
//   - args: Query parameters
//
// Returns:
//   - *sql.Row: Single query result row
//
// Example:
//
//var email string
//err := db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&email)
//// For PostgreSQL, becomes: "SELECT email FROM users WHERE id = $1"
func (tda *TranslatingDatabaseAdapter) QueryRow(query string, args ...interface{}) *sql.Row {
translatedQuery := tda.translator.Translate(query)
return tda.adapter.QueryRow(translatedQuery, args...)
}

// Begin starts a transaction without translation.
//
// Transaction operations work at the connection level and do not require
// SQL query translation. The returned *sql.Tx operates on the underlying
// database connection directly.
//
// Returns:
//   - *sql.Tx: Database transaction
//   - error: Transaction start error
//
// Example:
//
//tx, err := db.Begin()
//if err != nil {
//    return err
//}
//defer tx.Rollback()
//
//// Transaction operations work normally
//tx.Exec("INSERT INTO users ...")
//tx.Commit()
func (tda *TranslatingDatabaseAdapter) Begin() (*sql.Tx, error) {
return tda.adapter.Begin()
}

// DbType returns the underlying database type.
//
// This method passes through to the underlying adapter without modification.
// The database type is used internally for translation decisions.
//
// Returns: Database type string ("sqlite" or "postgres")
//
// Example:
//
//dbType := db.DbType()
//log.Printf("Using database: %s", dbType)
func (tda *TranslatingDatabaseAdapter) DbType() string {
return tda.adapter.DbType()
}

// Ping verifies the database connection.
//
// This method passes through to the underlying adapter without modification.
// Connection health checks do not require query translation.
//
// Returns: Connection error, if any
//
// Example:
//
//if err := db.Ping(); err != nil {
//    log.Printf("Database connection failed: %v", err)
//}
func (tda *TranslatingDatabaseAdapter) Ping() error {
return tda.adapter.Ping()
}

// Stats returns connection pool statistics.
//
// This method passes through to the underlying adapter without modification.
// Connection pool statistics are provided by the underlying database driver.
//
// Returns: ConnectionStats or nil (for SQLite)
//
// Example:
//
//stats := db.Stats()
//if stats != nil {
//    log.Printf("Open connections: %d", stats.OpenConnections)
//}
func (tda *TranslatingDatabaseAdapter) Stats() *ConnectionStats {
return tda.adapter.Stats()
}
