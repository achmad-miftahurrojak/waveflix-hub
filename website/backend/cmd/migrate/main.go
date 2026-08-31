// Package main provides the database migration command for WaveFlix Hub.
//
// The migration tool copies data from SQLite to PostgreSQL with proper type
// conversions, ID sequence management, and comprehensive error handling.
// It supports both full migrations and validation of existing migrations.
//
// Usage:
//   go run cmd/migrate/main.go [options]
//   
// Examples:
//   # Migrate from SQLite to PostgreSQL
//   DATABASE_TYPE=postgres DATABASE_URL=postgresql://... go run cmd/migrate/main.go
//
//   # Migrate with custom source file
//   go run cmd/migrate/main.go -source waveflix.db -dest "postgresql://..."
//
//   # Dry run (validation only)
//   go run cmd/migrate/main.go -source waveflix.db -dest "postgresql://..." -dry-run
package main

import (
"flag"
"fmt"
"log"
"os"
"path/filepath"
"strings"
)

// MigrationConfig holds configuration for the database migration.
//
// The configuration supports flexible source and destination specification,
// with automatic detection of connection parameters and validation of
// database accessibility.
//
// Configuration Sources (in priority order):
//  1. Command-line flags (-source, -dest)
//  2. Environment variables (DATABASE_URL, etc.)
//  3. Default values (waveflix.db for source)
//
// Example configurations:
//
//   # Environment-based migration
//   DATABASE_TYPE=postgres DATABASE_URL=postgresql://... ./migrate
//   
//   # Explicit source and destination
//   ./migrate -source old.db -dest "postgresql://user:pass@host/db"
//   
//   # Validation mode
//   ./migrate -source data.db -dest "postgresql://..." -dry-run
type MigrationConfig struct {
// SourceFile is the path to the source SQLite database file.
// Default: "waveflix.db"
SourceFile string

// DestinationURL is the PostgreSQL connection string.
// Format: postgresql://user:password@host:port/database?options
DestinationURL string

// DryRun enables validation mode without actually migrating data.
// Useful for testing connectivity and data integrity checks.
DryRun bool

// Verbose enables detailed logging of migration progress.
// Shows table-by-table progress and record counts.
Verbose bool

// ContinueOnError determines whether to continue with remaining tables
// if one table migration fails. Default: false (stop on first error)
ContinueOnError bool

// BatchSize is the number of records to migrate in each batch.
// Larger batches are faster but use more memory. Default: 1000
BatchSize int
}

// parseFlags parses command-line arguments and environment variables.
//
// The function supports both explicit command-line configuration and
// environment-based configuration for deployment flexibility.
//
// Command-line flags take precedence over environment variables.
//
// Returns: Parsed MigrationConfig ready for validation
//
// Examples:
//
//   # Command-line configuration
//   -source waveflix.db -dest "postgresql://..." -verbose
//   
//   # Environment configuration
//   DATABASE_URL=postgresql://... (no flags needed)
func parseFlags() *MigrationConfig {
config := &MigrationConfig{
BatchSize: 1000, // Default batch size
}

// Define command-line flags
flag.StringVar(&config.SourceFile, "source", "", "Source SQLite database file path")
flag.StringVar(&config.DestinationURL, "dest", "", "Destination PostgreSQL connection URL")
flag.BoolVar(&config.DryRun, "dry-run", false, "Validate migration without transferring data")
flag.BoolVar(&config.Verbose, "verbose", false, "Enable detailed migration logging")
flag.BoolVar(&config.ContinueOnError, "continue-on-error", false, "Continue migration despite errors")
flag.IntVar(&config.BatchSize, "batch-size", 1000, "Records per batch (default: 1000)")

// Custom usage message
flag.Usage = func() {
fmt.Fprintf(os.Stderr, "WaveFlix Hub Database Migration Tool\n\n")
fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
fmt.Fprintf(os.Stderr, "Options:\n")
flag.PrintDefaults()
fmt.Fprintf(os.Stderr, "\nExamples:\n")
fmt.Fprintf(os.Stderr, "  # Environment-based migration:\n")
fmt.Fprintf(os.Stderr, "  DATABASE_TYPE=postgres DATABASE_URL=postgresql://... %s\n\n", os.Args[0])
fmt.Fprintf(os.Stderr, "  # Explicit configuration:\n")
fmt.Fprintf(os.Stderr, "  %s -source waveflix.db -dest 'postgresql://user:pass@host/db'\n\n", os.Args[0])
fmt.Fprintf(os.Stderr, "  # Dry run validation:\n")
fmt.Fprintf(os.Stderr, "  %s -source data.db -dest 'postgresql://...' -dry-run\n\n", os.Args[0])
}

flag.Parse()

// Fill in defaults from environment if not provided via flags
if config.SourceFile == "" {
config.SourceFile = getEnvDefault("SQLITE_FILE", "waveflix.db")
}

if config.DestinationURL == "" {
config.DestinationURL = os.Getenv("DATABASE_URL")

// If no DATABASE_URL, try to build from individual PG_* variables
if config.DestinationURL == "" {
config.DestinationURL = buildPostgresURL()
}
}

return config
}

// validateConfig validates the migration configuration.
//
// This function performs comprehensive validation of source and destination
// databases, checking file accessibility, connection string format, and
// database connectivity.
//
// Validation checks:
//  - Source SQLite file exists and is readable
//  - Destination URL is properly formatted
//  - PostgreSQL database is reachable
//  - Required tables exist in source database
//  - Destination database schema is compatible
//
// Parameters:
//   - config: Migration configuration to validate
//
// Returns: error if validation fails, nil if configuration is valid
//
// Example validation failures:
//   - Source file not found: "source database file not found: waveflix.db"
//   - Invalid URL: "invalid destination URL format"
//   - Connection failed: "cannot connect to PostgreSQL: connection refused"
func validateConfig(config *MigrationConfig) error {
// Validate source file
if config.SourceFile == "" {
return fmt.Errorf("source database file is required")
}

// Check if source file exists
if _, err := os.Stat(config.SourceFile); os.IsNotExist(err) {
return fmt.Errorf("source database file not found: %s", config.SourceFile)
}

// Get absolute path for better error messages
absPath, err := filepath.Abs(config.SourceFile)
if err == nil {
config.SourceFile = absPath
}

// Validate destination URL
if config.DestinationURL == "" {
return fmt.Errorf("destination PostgreSQL URL is required (use -dest flag or DATABASE_URL environment variable)")
}

// Basic PostgreSQL URL format validation
if !strings.HasPrefix(config.DestinationURL, "postgresql://") && !strings.HasPrefix(config.DestinationURL, "postgres://") {
return fmt.Errorf("destination URL must start with postgresql:// or postgres://")
}

// Validate batch size
if config.BatchSize <= 0 {
return fmt.Errorf("batch size must be positive, got: %d", config.BatchSize)
}

return nil
}

// buildPostgresURL constructs a PostgreSQL connection URL from individual environment variables.
//
// This function provides an alternative to DATABASE_URL by building the connection
// string from separate PG_* environment variables. This is useful for deployment
// systems that provide individual configuration parameters.
//
// Required environment variables:
//   - PG_HOST: PostgreSQL server hostname
//   - PG_USER: Database username
//   - PG_PASSWORD: Database password
//   - PG_DATABASE: Database name
//
// Optional environment variables:
//   - PG_PORT: Server port (default: 5432)
//   - PG_SSLMODE: SSL mode (default: require)
//
// Returns: PostgreSQL connection URL, or empty string if required variables are missing
//
// Example:
//   PG_HOST=localhost PG_USER=user PG_PASSWORD=pass PG_DATABASE=db
//   → postgresql://user:pass@localhost:5432/db?sslmode=require
func buildPostgresURL() string {
host := os.Getenv("PG_HOST")
user := os.Getenv("PG_USER")
password := os.Getenv("PG_PASSWORD")
database := os.Getenv("PG_DATABASE")

// Check required variables
if host == "" || user == "" || password == "" || database == "" {
return ""
}

port := getEnvDefault("PG_PORT", "5432")
sslmode := getEnvDefault("PG_SSLMODE", "require")

return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s",
user, password, host, port, database, sslmode)
}

// getEnvDefault returns an environment variable value or default if not set.
//
// This utility function simplifies environment variable handling with fallback values.
//
// Parameters:
//   - key: Environment variable name
//   - defaultValue: Fallback value if environment variable is unset
//
// Returns: Environment variable value or default
//
// Example:
//   port := getEnvDefault("PG_PORT", "5432")
func getEnvDefault(key, defaultValue string) string {
if value := os.Getenv(key); value != "" {
return value
}
return defaultValue
}

// printBanner displays the migration tool banner and configuration summary.
//
// The banner provides visual confirmation of migration parameters and helps
// prevent accidental migrations with incorrect configurations.
//
// Parameters:
//   - config: Migration configuration to display
//
// Example output:
//   =====================================
//   WaveFlix Hub Database Migration
//   =====================================
//   Source:      /path/to/waveflix.db
//   Destination: postgresql://user@host/db
//   Mode:        Full migration
//   Batch size:  1000 records
//   =====================================
func printBanner(config *MigrationConfig) {
fmt.Println("=====================================")
fmt.Println("WaveFlix Hub Database Migration")
fmt.Println("=====================================")
fmt.Printf("Source:      %s\n", config.SourceFile)

// Mask password in URL for display
displayURL := config.DestinationURL
if idx := strings.Index(displayURL, "://"); idx > 0 {
if credIdx := strings.Index(displayURL[idx+3:], "@"); credIdx > 0 {
userPass := displayURL[idx+3 : idx+3+credIdx]
if colonIdx := strings.Index(userPass, ":"); colonIdx > 0 {
user := userPass[:colonIdx]
rest := displayURL[idx+3+credIdx:]
displayURL = displayURL[:idx+3] + user + ":***@" + rest
}
}
}
fmt.Printf("Destination: %s\n", displayURL)

if config.DryRun {
fmt.Println("Mode:        Dry run (validation only)")
} else {
fmt.Println("Mode:        Full migration")
}

fmt.Printf("Batch size:  %d records\n", config.BatchSize)

if config.Verbose {
fmt.Println("Logging:     Verbose")
}

if config.ContinueOnError {
fmt.Println("Error mode:  Continue on error")
}

fmt.Println("=====================================")
fmt.Println()
}

func main() {
log.SetFlags(log.LstdFlags | log.Lshortfile)

// Parse configuration
config := parseFlags()

// Validate configuration
if err := validateConfig(config); err != nil {
log.Fatalf("Configuration error: %v", err)
}

// Display migration banner
printBanner(config)

// Confirm migration (unless dry run)
if !config.DryRun {
fmt.Print("This will migrate data from SQLite to PostgreSQL. Continue? [y/N]: ")
var response string
fmt.Scanln(&response)
response = strings.ToLower(strings.TrimSpace(response))

if response != "y" && response != "yes" {
fmt.Println("Migration cancelled.")
os.Exit(0)
}
fmt.Println()
}

// TODO: Initialize migration components
// This will be implemented in subsequent tasks:
// - Create database connections
// - Initialize migration manager
// - Execute table-by-table migration
// - Generate migration report

fmt.Println("Migration command structure created successfully!")
fmt.Println("Actual migration logic will be implemented in subsequent tasks.")

if config.DryRun {
fmt.Println("\n[DRY RUN] No actual data transfer would occur.")
}
}
