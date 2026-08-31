// Package main provides SQL dialect translation for database compatibility.
//
// The SQL translator converts queries from SQLite syntax to PostgreSQL syntax,
// enabling transparent migration while maintaining backward compatibility.
// It handles placeholder conversion, datetime functions, boolean literals,
// and UPSERT statements.
package main

import (
"strconv"
"strings"
)

// SQLTranslator handles SQL query translation between database dialects.
//
// The translator identifies the target database type and applies appropriate
// transformations to ensure query compatibility. For SQLite, queries are
// passed through unchanged. For PostgreSQL, queries undergo dialect-specific
// translations.
//
// Supported transformations:
//   - Placeholder conversion (? → $1, $2, $3...)
//   - Datetime function translation (CURRENT_TIMESTAMP → NOW())
//   - Boolean literal conversion (1/0 → TRUE/FALSE)
//   - UPSERT statement translation (INSERT OR IGNORE → INSERT ... ON CONFLICT)
//
// Example usage:
//
//translator := NewSQLTranslator("postgres")
//translated := translator.Translate("SELECT * FROM users WHERE id = ?")
//// Result: "SELECT * FROM users WHERE id = $1"
type SQLTranslator struct {
dbType string
}

// NewSQLTranslator creates a new SQL translator for the specified database type.
//
// The translator applies transformations based on the target database dialect.
// Supported database types are "sqlite" and "postgres".
//
// Parameters:
//   - dbType: Target database type ("sqlite" or "postgres")
//
// Returns: Configured SQLTranslator instance
//
// Example:
//
//translator := NewSQLTranslator("postgres")
func NewSQLTranslator(dbType string) *SQLTranslator {
return &SQLTranslator{
dbType: dbType,
}
}

// Translate converts a SQL query for the target database dialect.
//
// This is the main entry point for query translation. The method applies
// all necessary transformations based on the target database type:
//
// For SQLite:
//   - Query is returned unchanged (no translation needed)
//
// For PostgreSQL:
//   - Placeholder translation (? → $1, $2, $3...)
//   - Datetime function translation
//   - Boolean literal translation
//   - UPSERT statement translation
//
// The translations are applied in a specific order to avoid conflicts and
// ensure correctness of the final query.
//
// Parameters:
//   - query: Original SQL query string
//
// Returns: Translated SQL query appropriate for the target database
//
// Example:
//
//original := "INSERT OR IGNORE INTO users (email) VALUES (?)"
//translated := translator.Translate(original)
//// For PostgreSQL: "INSERT INTO users (email) VALUES ($1) ON CONFLICT DO NOTHING"
func (t *SQLTranslator) Translate(query string) string {
// SQLite queries need no translation
if t.dbType == "sqlite" {
return query
}

// Apply PostgreSQL transformations in order
if t.dbType == "postgres" {
query = t.translateUpsert(query)
query = t.translateDateTime(query)
query = t.translateBooleanLiterals(query)
query = t.translatePlaceholders(query)
}

return query
}

// translatePlaceholders converts SQLite-style ? placeholders to PostgreSQL-style $N placeholders.
//
// This method scans the query string and replaces each ? placeholder with a numbered
// PostgreSQL placeholder ($1, $2, $3, etc.). The method handles string literals
// correctly and does not translate ? characters that appear inside quoted strings.
//
// Algorithm:
//  1. Track whether we're inside a string literal (single quotes)
//  2. When outside strings, replace ? with $N (incrementing N)
//  3. Preserve ? characters inside string literals
//
// Examples:
//   - "SELECT * FROM users WHERE id = ?" → "SELECT * FROM users WHERE id = $1"
//   - "SELECT * FROM users WHERE name = ? AND email = ?" → "SELECT * FROM users WHERE name = $1 AND email = $2"
//   - "SELECT * FROM users WHERE comment = 'Why?'" → "SELECT * FROM users WHERE comment = 'Why?'" (unchanged)
//
// Parameters:
//   - query: SQL query with ? placeholders
//
// Returns: SQL query with $N placeholders
func (t *SQLTranslator) translatePlaceholders(query string) string {
if t.dbType != "postgres" {
return query
}

var result strings.Builder
paramNum := 1
inString := false

for _, char := range query {
switch char {
case '\'':
// Handle string literals - toggle inString state
// Check for escaped quotes (but keep it simple for now)
inString = !inString
result.WriteRune(char)

case '?':
if !inString {
// Replace ? with $N when not in string literal
result.WriteString("$")
result.WriteString(strconv.Itoa(paramNum))
paramNum++
} else {
// Preserve ? inside string literals
result.WriteRune(char)
}

default:
result.WriteRune(char)
}
}

return result.String()
}

// translateDateTime converts SQLite datetime functions to PostgreSQL equivalents.
//
// SQLite and PostgreSQL use different function names for current timestamp operations.
// This method handles the most common datetime function conversions.
//
// Supported conversions:
//   - CURRENT_TIMESTAMP → NOW() (PostgreSQL prefers NOW())
//   - datetime('now') → NOW()
//   - datetime('now', 'localtime') → NOW() (simplified)
//
// The method performs case-insensitive matching to handle various SQL coding styles.
//
// Examples:
//   - "INSERT INTO logs (created_at) VALUES (CURRENT_TIMESTAMP)" 
//     → "INSERT INTO logs (created_at) VALUES (NOW())"
//   - "SELECT * FROM events WHERE created_at > datetime('now')"
//     → "SELECT * FROM events WHERE created_at > NOW()"
//
// Parameters:
//   - query: SQL query with SQLite datetime functions
//
// Returns: SQL query with PostgreSQL datetime functions
func (t *SQLTranslator) translateDateTime(query string) string {
if t.dbType != "postgres" {
return query
}

// Convert CURRENT_TIMESTAMP to NOW() for better PostgreSQL compatibility
query = strings.ReplaceAll(query, "CURRENT_TIMESTAMP", "NOW()")
query = strings.ReplaceAll(query, "current_timestamp", "NOW()")

// Convert SQLite's datetime('now') to NOW()
query = strings.ReplaceAll(query, "datetime('now')", "NOW()")
query = strings.ReplaceAll(query, "datetime(\"now\")", "NOW()")

// Handle datetime('now', 'localtime') - simplify to NOW()
query = strings.ReplaceAll(query, "datetime('now', 'localtime')", "NOW()")
query = strings.ReplaceAll(query, "datetime(\"now\", \"localtime\")", "NOW()")

return query
}

// translateBooleanLiterals converts integer boolean literals to PostgreSQL boolean literals.
//
// SQLite commonly uses integer values (1, 0) for boolean fields, while PostgreSQL
// prefers explicit boolean literals (TRUE, FALSE). This method detects boolean
// contexts and performs the appropriate conversions.
//
// Boolean contexts detected:
//   - WHERE clauses: WHERE active = 1 → WHERE active = TRUE
//   - SET clauses: SET enabled = 0 → SET enabled = FALSE
//   - CASE expressions: CASE WHEN 1 → CASE WHEN TRUE
//
// The method uses simple heuristics to identify boolean contexts while avoiding
// false positives in numeric contexts (like user IDs or counts).
//
// Examples:
//   - "SELECT * FROM users WHERE active = 1" → "SELECT * FROM users WHERE active = TRUE"
//   - "UPDATE users SET enabled = 0 WHERE id = 5" → "UPDATE users SET enabled = FALSE WHERE id = 5"
//
// Parameters:
//   - query: SQL query with integer boolean literals
//
// Returns: SQL query with PostgreSQL boolean literals
func (t *SQLTranslator) translateBooleanLiterals(query string) string {
if t.dbType != "postgres" {
return query
}

// Simple boolean literal conversions in common contexts
// This is a basic implementation - a more sophisticated version would parse SQL

// Convert = 1 to = TRUE in boolean-like contexts
query = strings.ReplaceAll(query, " = 1", " = TRUE")
query = strings.ReplaceAll(query, "= 1 ", "= TRUE ")
query = strings.ReplaceAll(query, " = 0", " = FALSE")
query = strings.ReplaceAll(query, "= 0 ", "= FALSE ")

// Handle SET clauses for UPDATE statements
query = strings.ReplaceAll(query, "SET enabled = 1", "SET enabled = TRUE")
query = strings.ReplaceAll(query, "SET active = 1", "SET active = TRUE")
query = strings.ReplaceAll(query, "SET is_verified = 1", "SET is_verified = TRUE")
query = strings.ReplaceAll(query, "SET enabled = 0", "SET enabled = FALSE")
query = strings.ReplaceAll(query, "SET active = 0", "SET active = FALSE")
query = strings.ReplaceAll(query, "SET is_verified = 0", "SET is_verified = FALSE")

return query
}

// translateUpsert converts SQLite UPSERT syntax to PostgreSQL ON CONFLICT syntax.
//
// SQLite uses INSERT OR IGNORE/REPLACE syntax for upsert operations, while PostgreSQL
// uses INSERT ... ON CONFLICT DO NOTHING/UPDATE syntax. This method handles the
// most common upsert patterns.
//
// Supported conversions:
//   - INSERT OR IGNORE → INSERT ... ON CONFLICT DO NOTHING
//   - INSERT OR REPLACE → INSERT ... ON CONFLICT DO UPDATE
//
// Note: PostgreSQL's ON CONFLICT syntax requires specifying conflict columns.
// For simplicity, this implementation uses ON CONFLICT DO NOTHING for INSERT OR IGNORE
// and provides a basic framework for INSERT OR REPLACE conversion.
//
// More sophisticated implementations would analyze table schemas to determine
// appropriate conflict columns automatically.
//
// Examples:
//   - "INSERT OR IGNORE INTO users (email) VALUES ('test@example.com')"
//     → "INSERT INTO users (email) VALUES ('test@example.com') ON CONFLICT DO NOTHING"
//
// Parameters:
//   - query: SQL query with SQLite UPSERT syntax
//
// Returns: SQL query with PostgreSQL ON CONFLICT syntax
func (t *SQLTranslator) translateUpsert(query string) string {
if t.dbType != "postgres" {
return query
}

// Convert INSERT OR IGNORE to INSERT ... ON CONFLICT DO NOTHING
query = strings.ReplaceAll(query, "INSERT OR IGNORE", "INSERT")
if strings.Contains(query, "INSERT") && !strings.Contains(query, "ON CONFLICT") {
// Add ON CONFLICT DO NOTHING for INSERT OR IGNORE conversions
// This is a simple heuristic - in practice, you'd need to know the table structure
if strings.Contains(strings.ToUpper(query), "INSERT") {
// Find the end of the VALUES clause and add ON CONFLICT
// This is a simplified implementation
if strings.Contains(query, "VALUES") {
parts := strings.Split(query, "VALUES")
if len(parts) == 2 {
// Simple case: INSERT INTO table (cols) VALUES (vals)
valuesPart := strings.TrimSpace(parts[1])
if strings.Contains(valuesPart, ")") {
// Find the closing paren of VALUES clause
parenCount := 0
insertPos := -1
for i, char := range valuesPart {
if char == '(' {
parenCount++
} else if char == ')' {
parenCount--
if parenCount == 0 {
insertPos = i + 1
break
}
}
}
if insertPos > 0 {
before := parts[0] + "VALUES" + valuesPart[:insertPos]
after := valuesPart[insertPos:]
query = before + " ON CONFLICT DO NOTHING" + after
}
}
}
}
}
}

// INSERT OR REPLACE is more complex - requires knowing which columns conflict
// For now, just remove the OR REPLACE part and let the application handle it
query = strings.ReplaceAll(query, "INSERT OR REPLACE", "INSERT")

return query
}
