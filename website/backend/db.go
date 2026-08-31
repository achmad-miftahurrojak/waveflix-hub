package main

import (
	"database/sql"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
)

var db DatabaseAdapter

func initDB() {
	log.Println("[db] Initializing database...")

	config := LoadPostgresConfig()
	if err := config.ValidateConfig(); err != nil {
		log.Fatalf("[db] Invalid PostgreSQL configuration: %v", err)
	}

	log.Printf("[db] Connecting to PostgreSQL: %s", config.String())

	sqlDB, err := sql.Open("pgx", config.ConnectionString())
	if err != nil {
		log.Fatalf("[db] Failed to open PostgreSQL connection: %v", err)
	}

	maxConns := getEnvInt("PG_MAX_CONNS", 25)
	maxIdleConns := getEnvInt("PG_MAX_IDLE_CONNS", 25)

	sqlDB.SetMaxOpenConns(maxConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)

	log.Printf("[db] PostgreSQL connection pool configured: MaxOpen=%d, MaxIdle=%d", maxConns, maxIdleConns)

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		log.Fatalf("[db] Failed to connect to PostgreSQL: %v", err)
	}

	log.Println("[db] PostgreSQL connection established successfully")

	db = newPostgresAdapter(sqlDB)

	// Create schema using PostgreSQL schema builder
	schemaBuilder := NewPostgreSQLSchemaBuilder(db)
	if err := schemaBuilder.CreateSchema(); err != nil {
		log.Fatalf("[db] Failed to create schema: %v", err)
	}

	log.Println("[db] Database initialization completed successfully")
}
