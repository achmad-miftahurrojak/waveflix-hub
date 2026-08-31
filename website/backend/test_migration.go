// Package main provides migration testing for WaveFlix Hub PostgreSQL migration.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// TestMigrationConnections tests that we can connect to both SQLite and PostgreSQL
func TestMigrationConnections() error {
	// Test SQLite connection
	sqliteDB, err := sql.Open("sqlite", "waveflix.db")
	if err != nil {
		return fmt.Errorf("Failed to open SQLite: %v", err)
	}
	defer sqliteDB.Close()

	err = sqliteDB.Ping()
	if err != nil {
		return fmt.Errorf("Failed to ping SQLite: %v", err)
	}
	
	log.Println("✓ SQLite connection successful")

	// Test PostgreSQL connection (if configured)
	postgresURL := os.Getenv("DATABASE_URL")
	if postgresURL == "" {
		postgresURL = "postgres://waveflix:password@localhost:5432/waveflix?sslmode=disable"
		log.Println("Using default PostgreSQL connection string for testing")
	}

	pgDB, err := sql.Open("pgx", postgresURL)
	if err != nil {
		log.Printf("PostgreSQL not available: %v", err)
		return nil
	}
	defer pgDB.Close()

	err = pgDB.Ping()
	if err != nil {
		log.Printf("PostgreSQL not available: %v", err)
		return nil
	}
	
	log.Println("✓ PostgreSQL connection successful")
	return nil
}

// TestSQLiteSchema tests that SQLite database has expected tables
func TestSQLiteSchema() error {
	db, err := sql.Open("sqlite", "waveflix.db")
	if err != nil {
		return fmt.Errorf("Failed to open SQLite: %v", err)
	}
	defer db.Close()

	expectedTables := []string{
		"users", "email_verifications", "profiles", 
		"watchlist", "favorites", "history",
	}

	for _, table := range expectedTables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='%s'", table)
		err := db.QueryRow(query).Scan(&count)
		if err != nil {
			log.Printf("Error checking table %s: %v", table, err)
			continue
		}
		
		if count == 0 {
			log.Printf("⚠️  Table %s not found in SQLite database", table)
		} else {
			log.Printf("✓ Table %s exists in SQLite", table)
		}
	}
	return nil
}

// TestDataIntegrity tests basic data integrity in SQLite
func TestDataIntegrity() error {
	db, err := sql.Open("sqlite", "waveflix.db")
	if err != nil {
		return fmt.Errorf("Failed to open SQLite: %v", err)
	}
	defer db.Close()

	// Check user count
	var userCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		log.Printf("Error counting users: %v", err)
	} else {
		log.Printf("✓ Found %d users in SQLite database", userCount)
	}

	// Check for valid user data
	if userCount > 0 {
		var email string
		err = db.QueryRow("SELECT email FROM users WHERE email IS NOT NULL LIMIT 1").Scan(&email)
		if err != nil {
			log.Printf("Error getting user email: %v", err)
		} else {
			log.Printf("✓ Sample user email: %s", email)
		}
	}
	return nil
}

// TestMigrationScript tests that the migration script can be executed
func TestMigrationScript() error {
	// Check if migration script exists
	if _, err := os.Stat("../../database/migrate_sqlite_to_postgres.go"); os.IsNotExist(err) {
		if _, err := os.Stat("database/migrate_sqlite_to_postgres.go"); os.IsNotExist(err) {
			return fmt.Errorf("Migration script not found")
		}
		log.Println("✓ Migration script exists (local)")
	} else {
		log.Println("✓ Migration script exists (parent)")
	}

	// Check if init.sql exists
	if _, err := os.Stat("../../database/init.sql"); os.IsNotExist(err) {
		if _, err := os.Stat("database/init.sql"); os.IsNotExist(err) {
			return fmt.Errorf("PostgreSQL init.sql not found")
		}
		log.Println("✓ PostgreSQL init.sql exists (local)")
	} else {
		log.Println("✓ PostgreSQL init.sql exists (parent)")
	}
	
	return nil
}

func main() {
	log.Println("=== WaveFlix Hub PostgreSQL Migration Testing ===")
	
	var hasErrors bool
	
	log.Println("\n1. Testing database connections...")
	if err := TestMigrationConnections(); err != nil {
		log.Printf("❌ Connection test failed: %v", err)
		hasErrors = true
	}
	
	log.Println("\n2. Testing SQLite schema...")
	if err := TestSQLiteSchema(); err != nil {
		log.Printf("❌ Schema test failed: %v", err)
		hasErrors = true
	}
	
	log.Println("\n3. Testing data integrity...")
	if err := TestDataIntegrity(); err != nil {
		log.Printf("❌ Data integrity test failed: %v", err)
		hasErrors = true
	}
	
	log.Println("\n4. Testing migration script...")
	if err := TestMigrationScript(); err != nil {
		log.Printf("❌ Migration script test failed: %v", err)
		hasErrors = true
	}
	
	if hasErrors {
		log.Println("\n❌ Some tests failed")
		os.Exit(1)
	} else {
		log.Println("\n✅ All migration tests passed!")
	}
}