

package main

import (
"database/sql"
"fmt"
"log"
"strings"
"time"
)

type TableMigration struct {
TableName    string
RowCount     int64
MigratedRows int64
FailedRows   int64
StartTime    time.Time
EndTime      time.Time
Error        error
}

type MigrationManager struct {
sourceDB      *sql.DB
destDB        *sql.DB
config        *MigrationConfig
tables        []string
migrations    map[string]*TableMigration
totalRows     int64
migratedRows  int64
}

func NewMigrationManager(config *MigrationConfig) (*MigrationManager, error) {
manager := &MigrationManager{
config:     config,
migrations: make(map[string]*TableMigration),
}

sourceDB, err := sql.Open("sqlite", config.SourceFile)
if err != nil {
return nil, fmt.Errorf("failed to connect to source database: %w", err)
}
manager.sourceDB = sourceDB

destDB, err := sql.Open("pgx", config.DestinationURL)
if err != nil {
return nil, fmt.Errorf("failed to connect to destination database: %w", err)
}
manager.destDB = destDB

if err := manager.sourceDB.Ping(); err != nil {
return nil, fmt.Errorf("source database not accessible: %w", err)
}
if err := manager.destDB.Ping(); err != nil {
return nil, fmt.Errorf("destination database not accessible: %w", err)
}

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

func (m *MigrationManager) Execute() error {
if m.config.Verbose {
log.Println("[migrate] Starting data migration...")
}

if err := m.calculateTotalRows(); err != nil {
return fmt.Errorf("failed to calculate migration size: %w", err)
}

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

func (m *MigrationManager) calculateTotalRows() error {
for _, tableName := range m.tables {
query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
var count int64

err := m.sourceDB.QueryRow(query).Scan(&count)
if err != nil {

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

func (m *MigrationManager) migrateTable(tableName string) error {
migration := m.migrations[tableName]

if migration.RowCount == 0 {
if m.config.Verbose {
log.Printf("[migrate] Table %s is empty (skipping)", tableName)
}
return nil
}

columns, err := m.getTableColumns(tableName)
if err != nil {
return fmt.Errorf("failed to get table schema: %w", err)
}

selectQuery := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ", "), tableName)

placeholders := make([]string, len(columns))
for i := range placeholders {
placeholders[i] = fmt.Sprintf("$%d", i+1)
}
insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

rows, err := m.sourceDB.Query(selectQuery)
if err != nil {
return fmt.Errorf("failed to query source table: %w", err)
}
defer rows.Close()

stmt, err := m.destDB.Prepare(insertQuery)
if err != nil {
return fmt.Errorf("failed to prepare insert statement: %w", err)
}
defer stmt.Close()

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

func (m *MigrationManager) getTableColumns(tableName string) ([]string, error) {

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

func (m *MigrationManager) convertRowValues(values []interface{}, tableName string) []interface{} {
converted := make([]interface{}, len(values))

for i, value := range values {
switch v := value.(type) {
case []byte:

converted[i] = string(v)
case int64:

converted[i] = v
case float64:

converted[i] = v
case string:

converted[i] = v
case nil:

converted[i] = nil
default:

converted[i] = fmt.Sprintf("%v", v)
}
}

return converted
}

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
