

package main

import (
"flag"
"fmt"
"log"
"os"
"path/filepath"
"strings"
)

type MigrationConfig struct {

SourceFile string

DestinationURL string

DryRun bool

Verbose bool

ContinueOnError bool

BatchSize int
}

func parseFlags() *MigrationConfig {
config := &MigrationConfig{
BatchSize: 1000, 
}

flag.StringVar(&config.SourceFile, "source", "", "Source SQLite database file path")
flag.StringVar(&config.DestinationURL, "dest", "", "Destination PostgreSQL connection URL")
flag.BoolVar(&config.DryRun, "dry-run", false, "Validate migration without transferring data")
flag.BoolVar(&config.Verbose, "verbose", false, "Enable detailed migration logging")
flag.BoolVar(&config.ContinueOnError, "continue-on-error", false, "Continue migration despite errors")
flag.IntVar(&config.BatchSize, "batch-size", 1000, "Records per batch (default: 1000)")

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

if config.SourceFile == "" {
config.SourceFile = getEnvDefault("SQLITE_FILE", "waveflix.db")
}

if config.DestinationURL == "" {
config.DestinationURL = os.Getenv("DATABASE_URL")

if config.DestinationURL == "" {
config.DestinationURL = buildPostgresURL()
}
}

return config
}

func validateConfig(config *MigrationConfig) error {

if config.SourceFile == "" {
return fmt.Errorf("source database file is required")
}

if _, err := os.Stat(config.SourceFile); os.IsNotExist(err) {
return fmt.Errorf("source database file not found: %s", config.SourceFile)
}

absPath, err := filepath.Abs(config.SourceFile)
if err == nil {
config.SourceFile = absPath
}

if config.DestinationURL == "" {
return fmt.Errorf("destination PostgreSQL URL is required (use -dest flag or DATABASE_URL environment variable)")
}

if !strings.HasPrefix(config.DestinationURL, "postgresql://") && !strings.HasPrefix(config.DestinationURL, "postgres://") {
return fmt.Errorf("destination URL must start with postgresql:// or postgres://")
}

if config.BatchSize <= 0 {
return fmt.Errorf("batch size must be positive, got: %d", config.BatchSize)
}

return nil
}

func buildPostgresURL() string {
host := os.Getenv("PG_HOST")
user := os.Getenv("PG_USER")
password := os.Getenv("PG_PASSWORD")
database := os.Getenv("PG_DATABASE")

if host == "" || user == "" || password == "" || database == "" {
return ""
}

port := getEnvDefault("PG_PORT", "5432")
sslmode := getEnvDefault("PG_SSLMODE", "require")

return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s",
user, password, host, port, database, sslmode)
}

func getEnvDefault(key, defaultValue string) string {
if value := os.Getenv(key); value != "" {
return value
}
return defaultValue
}

func printBanner(config *MigrationConfig) {
fmt.Println("=====================================")
fmt.Println("WaveFlix Hub Database Migration")
fmt.Println("=====================================")
fmt.Printf("Source:      %s\n", config.SourceFile)

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

config := parseFlags()

if err := validateConfig(config); err != nil {
log.Fatalf("Configuration error: %v", err)
}

printBanner(config)

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

fmt.Println("Migration command structure created successfully!")
fmt.Println("Actual migration logic will be implemented in subsequent tasks.")

if config.DryRun {
fmt.Println("\n[DRY RUN] No actual data transfer would occur.")
}
}
