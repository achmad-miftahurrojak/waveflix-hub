// Package main provides table data migration functionality.
//
// This module handles the actual data transfer from SQLite to PostgreSQL
// with proper type conversions, batch processing, and error handling.
// It migrates tables in dependency order to respect foreign key constraints.
package main

import (
"database/sql"
"fmt"
"log"
"strings"
"time"
)

// TableMigration represents a single table migration operation.
//
// Each table migration includes metadata about the table structure,
// data transformation rules, and progress tracking.
type TableMigration struct {
TableName    string
RowCount     int64
MigratedRows int64
FailedRows   int64
StartTime    time.Time
EndTime      time.Time
Error        error
}

// MigrationManager coordinates the overall data migration process.
//
// The manager handles database connections, migration sequencing,
// progress tracking, and error recovery.
type MigrationManager struct {
sourceDB      *sql.DB
destDB        *sql.DB
config        *MigrationConfig
tables        []string
migrations    map[string]*TableMigration
totalRows     int64
migratedRows  int64
}

// NewMigrationManager creates a new migration manager.
//
// The manager establishes connections to both source and destination
// databases and initializes migration tracking structures.
func NewMigrationManager(config *MigrationConfig) (*MigrationManager, error) {
manager := &MigrationManager{
config:     config,
migrations: make(map[string]*TableMigration),
}

// Connect to source SQLite database
sourceDB, err := sql.Open("sqlite", config.SourceFile)
if err != nil {
return nil, fmt.Errorf("failed to connect to source database: %w", err)
}
manager.sourceDB = sourceDB

// Connect to destination PostgreSQL database  
destDB, err := sql.Open("pgx", config.DestinationURL)
if err != nil {
return nil, fmt.Errorf("failed to connect to destination database: %w", err)
}
manager.destDB = destDB

// Verify connections
if err := manager.sourceDB.Ping(); err != nil {
return nil, fmt.Errorf("source database not accessible: %w", err)
}
if err := manager.destDB.Ping(); err != nil {
return nil, fmt.Errorf("destination database not accessible: %w", err)
}

// Define migration order (respects foreign key dependencies)
manager.tables = []string{
"users",
"email_verifications", 
"profiles",
"watchlist",
"favorites",
"history",
}

return manager, nil
}

// Execute performs the complete data migration.
//
// The migration proceeds table-by-table in dependency order,
// with comprehensive progress tracking and error handling.
func (m *MigrationManager) Execute() error {
if m.config.Verbose {
log.Println("[migrate] Starting data migration...")
}

// Calculate total row counts for progress tracking
if err := m.calculateTotalRows(); err != nil {
return fmt.Errorf("failed to calculate migration size: %w", err)
}

// Migrate each table
for _, tableName := range m.tables {
migration := &TableMigration{
TableName: tableName,
StartTime: time.Now(),
}
m.migrations[tableName] = migration

if m.config.Verbose {
log.Printf("[migrate] Starting table: %s", tableName)
}

if err := m.migrateTable(tableName); err != nil {
migration.Error = err
migration.EndTime = time.Now()

if m.config.ContinueOnError {
log.Printf("[migrate] Table %s failed (continuing): %v", tableName, err)
continue
} else {
return fmt.Errorf("migration failed at table %s: %w", tableName, err)
}
}

migration.EndTime = time.Now()
if m.config.Verbose {
duration := migration.EndTime.Sub(migration.StartTime)
log.Printf("[migrate] Completed table %s: %d rows in %v", 
tableName, migration.MigratedRows, duration)
}
}

if m.config.Verbose {
log.Println("[migrate] Data migration completed")
}

return nil
}

// calculateTotalRows counts total records across all tables.
//
// This enables accurate progress reporting during migration.
func (m *MigrationManager) calculateTotalRows() error {
for _, tableName := range m.tables {
query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
var count int64

err := m.sourceDB.QueryRow(query).Scan(&count)
if err != nil {
// Table might not exist in source database
if m.config.Verbose {
log.Printf("[migrate] Table %s not found in source (skipping)", tableName)
}
continue
}

m.migrations[tableName] = &TableMigration{
TableName: tableName,
RowCount:  count,
}
m.totalRows += count
}

if m.config.Verbose {
log.Printf("[migrate] Total records to migrate: %d", m.totalRows)
}

return nil
}

// migrateTable migrates all data from one table.
//
// The migration uses batch processing for memory efficiency
// and includes proper type conversions for PostgreSQL compatibility.
func (m *MigrationManager) migrateTable(tableName string) error {
migration := m.migrations[tableName]

// Skip empty tables
if migration.RowCount == 0 {
if m.config.Verbose {
log.Printf("[migrate] Table %s is empty (skipping)", tableName)
}
return nil
}

// Get table schema
columns, err := m.getTableColumns(tableName)
if err != nil {
return fmt.Errorf("failed to get table schema: %w", err)
}

// Build SELECT query
selectQuery := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ", "), tableName)

// Build INSERT query with PostgreSQL placeholders
placeholders := make([]string, len(columns))
for i := range placeholders {
placeholders[i] = fmt.Sprintf("$%d", i+1)
}
insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

// Execute migration in batches
rows, err := m.sourceDB.Query(selectQuery)
if err != nil {
return fmt.Errorf("failed to query source table: %w", err)
}
defer rows.Close()

// Prepare destination statement
stmt, err := m.destDB.Prepare(insertQuery)
if err != nil {
return fmt.Errorf("failed to prepare insert statement: %w", err)
}
defer stmt.Close()

// Migrate rows in batches
var batchCount int
values := make([]interface{}, len(columns))
valuePtrs := make([]interface{}, len(columns))
for i := range values {
valuePtrs[i] = &values[i]
}

for rows.Next() {
if err := rows.Scan(valuePtrs...); err != nil {
migration.FailedRows++
if m.config.ContinueOnError {
continue
}
return fmt.Errorf("failed to scan row: %w", err)
}

// Convert SQLite types to PostgreSQL types
convertedValues := m.convertRowValues(values, tableName)

if _, err := stmt.Exec(convertedValues...); err != nil {
migration.FailedRows++
if m.config.ContinueOnError {
if m.config.Verbose {
log.Printf("[migrate] Failed to insert row in %s: %v", tableName, err)
}
continue
}
return fmt.Errorf("failed to insert row: %w", err)
}

migration.MigratedRows++
m.migratedRows++
batchCount++

// Progress reporting
if m.config.Verbose && batchCount%m.config.BatchSize == 0 {
progress := float64(m.migratedRows) / float64(m.totalRows) * 100
log.Printf("[migrate] Progress: %d/%d rows (%.1f%%)", 
m.migratedRows, m.totalRows, progress)
}
}

if err := rows.Err(); err != nil {
return fmt.Errorf("error iterating rows: %w", err)
}

return nil
}

// getTableColumns retrieves column names for a table.
//
// This is used to build dynamic INSERT queries that work
// with the actual table structure.
func (m *MigrationManager) getTableColumns(tableName string) ([]string, error) {
// For SQLite, get columns from PRAGMA table_info
query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
rows, err := m.sourceDB.Query(query)
if err != nil {
return nil, err
}
defer rows.Close()

var columns []string
for rows.Next() {
var cid int
var name, dataType string
var notNull, primaryKey int
var defaultValue sql.NullString

err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey)
if err != nil {
return nil, err
}

columns = append(columns, name)
}

return columns, nil
}

// convertRowValues converts SQLite values to PostgreSQL-compatible values.
//
// This handles type conversions and formatting differences between
// SQLite and PostgreSQL data types.
func (m *MigrationManager) convertRowValues(values []interface{}, tableName string) []interface{} {
converted := make([]interface{}, len(values))

for i, value := range values {
switch v := value.(type) {
case []byte:
// Convert byte arrays to strings (SQLite TEXT stored as bytes)
converted[i] = string(v)
case int64:
// Keep integers as-is (PostgreSQL handles conversion)
converted[i] = v
case float64:
// Keep floats as-is
converted[i] = v
case string:
// Keep strings as-is
converted[i] = v
case nil:
// Keep NULL values as-is
converted[i] = nil
default:
// Fallback: convert to string
converted[i] = fmt.Sprintf("%v", v)
}
}

return converted
}

// GetMigrationSummary returns a summary of the migration results.
//
// The summary includes per-table statistics and overall migration metrics.
func (m *MigrationManager) GetMigrationSummary() string {
var summary strings.Builder

summary.WriteString("Migration Summary\n")
summary.WriteString("=================\n")

totalMigrated := int64(0)
totalFailed := int64(0)

for _, tableName := range m.tables {
migration := m.migrations[tableName]
if migration == nil {
continue
}

totalMigrated += migration.MigratedRows
totalFailed += migration.FailedRows

duration := migration.EndTime.Sub(migration.StartTime)
status := "SUCCESS"
if migration.Error != nil {
status = "FAILED"
}

summary.WriteString(fmt.Sprintf("%-20s: %8d rows, %8s, %v\n",
tableName, migration.MigratedRows, status, duration))
}

summary.WriteString("-----------------\n")
summary.WriteString(fmt.Sprintf("%-20s: %8d rows migrated\n", "TOTAL", totalMigrated))

if totalFailed > 0 {
summary.WriteString(fmt.Sprintf("%-20s: %8d rows failed\n", "FAILED", totalFailed))
}

return summary.String()
}

// Close closes database connections.
//
// This should be called after migration completion to clean up resources.
func (m *MigrationManager) Close() error {
var errors []string

if err := m.sourceDB.Close(); err != nil {
errors = append(errors, fmt.Sprintf("source DB: %v", err))
}

if err := m.destDB.Close(); err != nil {
errors = append(errors, fmt.Sprintf("destination DB: %v", err))
}

if len(errors) > 0 {
return fmt.Errorf("cleanup errors: %s", strings.Join(errors, ", "))
}

return nil
}
