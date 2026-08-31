// Package main provides PostgreSQL schema creation and management.
//
// The PostgreSQL schema builder creates tables, indexes, and foreign key constraints
// optimized for PostgreSQL with proper data types, indexing strategies, and referential
// integrity. It handles schema creation idempotently and supports migration from SQLite.
package main

import (
"fmt"
"log"
)

// PostgreSQLSchemaBuilder creates and manages PostgreSQL database schema.
//
// The schema builder translates the WaveFlix Hub data model into PostgreSQL-optimized
// table structures with appropriate data types, constraints, and indexes. It ensures
// referential integrity and provides optimal performance for the application's
// query patterns.
//
// Features:
//   - Idempotent schema creation (safe to run multiple times)
//   - PostgreSQL-specific data types (SERIAL, VARCHAR, TIMESTAMP WITH TIME ZONE)
//   - Optimized indexes for common query patterns
//   - Foreign key constraints with cascade deletion
//   - Proper NULL constraints and defaults
//
// Example usage:
//
//builder := NewPostgreSQLSchemaBuilder(db)
//if err := builder.CreateSchema(); err != nil {
//    log.Fatalf("Failed to create schema: %v", err)
//}
type PostgreSQLSchemaBuilder struct {
db DatabaseAdapter
}

// NewPostgreSQLSchemaBuilder creates a new PostgreSQL schema builder.
//
// The builder uses the provided database adapter to execute DDL statements.
// The adapter should already be connected to a PostgreSQL database.
//
// Parameters:
//   - db: Database adapter connected to PostgreSQL
//
// Returns: Configured PostgreSQLSchemaBuilder
//
// Example:
//
//builder := NewPostgreSQLSchemaBuilder(adapter)
func NewPostgreSQLSchemaBuilder(db DatabaseAdapter) *PostgreSQLSchemaBuilder {
return &PostgreSQLSchemaBuilder{
db: db,
}
}

// CreateSchema creates the complete WaveFlix Hub schema in PostgreSQL.
//
// This method creates all tables, indexes, and constraints required by the
// application. The schema creation is idempotent - it can be safely run
// multiple times without causing errors or data loss.
//
// Schema Creation Order:
//  1. Core tables (users, profiles)
//  2. Feature tables (watchlist, favorites, history)
//  3. Support tables (email_verifications)
//  4. Indexes for performance optimization
//  5. Foreign key constraints for referential integrity
//
// Returns: error if schema creation fails
//
// Example:
//
//if err := builder.CreateSchema(); err != nil {
//    log.Fatalf("Schema creation failed: %v", err)
//}
func (psb *PostgreSQLSchemaBuilder) CreateSchema() error {
log.Println("[schema] Creating PostgreSQL schema...")

// Create tables in dependency order
if err := psb.createUsersTable(); err != nil {
return fmt.Errorf("failed to create users table: %w", err)
}

if err := psb.createEmailVerificationsTable(); err != nil {
return fmt.Errorf("failed to create email_verifications table: %w", err)
}

if err := psb.createProfilesTable(); err != nil {
return fmt.Errorf("failed to create profiles table: %w", err)
}

if err := psb.createWatchlistTable(); err != nil {
return fmt.Errorf("failed to create watchlist table: %w", err)
}

if err := psb.createFavoritesTable(); err != nil {
return fmt.Errorf("failed to create favorites table: %w", err)
}

if err := psb.createHistoryTable(); err != nil {
return fmt.Errorf("failed to create history table: %w", err)
}

log.Println("[schema] PostgreSQL schema created successfully")
return nil
}

// createUsersTable creates the users table with PostgreSQL-optimized data types.
//
// The users table is the primary entity table containing user authentication
// and basic profile information. It uses PostgreSQL SERIAL for auto-incrementing
// primary keys and TIMESTAMP WITH TIME ZONE for proper timezone handling.
//
// Table Structure:
//   - id: SERIAL PRIMARY KEY (auto-incrementing integer)
//   - email: VARCHAR(255) UNIQUE NOT NULL (user's email address)
//   - username: VARCHAR(100) UNIQUE NOT NULL (display name)
//   - password_hash: VARCHAR(255) NOT NULL (bcrypt hash)
//   - is_verified: BOOLEAN DEFAULT FALSE (email verification status)
//   - created_at: TIMESTAMP WITH TIME ZONE DEFAULT NOW()
//   - updated_at: TIMESTAMP WITH TIME ZONE DEFAULT NOW()
//
// Constraints:
//   - Primary key on id
//   - Unique constraints on email and username
//   - NOT NULL constraints on required fields
//   - Default values for timestamps and boolean fields
func (psb *PostgreSQLSchemaBuilder) createUsersTable() error {
query := `
CREATE TABLE IF NOT EXISTS users (
id SERIAL PRIMARY KEY,
email VARCHAR(255) UNIQUE NOT NULL,
username VARCHAR(100) UNIQUE NOT NULL,
password_hash VARCHAR(255) NOT NULL,
role VARCHAR(20) DEFAULT 'user',
is_verified BOOLEAN DEFAULT FALSE,
created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS users_isolation_policy ON users;
CREATE POLICY users_isolation_policy ON users 
    FOR ALL 
    USING (id = NULLIF(current_setting('app.current_user_id', true), '')::integer);
`

if _, err := psb.db.Exec(query); err != nil {
return fmt.Errorf("failed to create users table: %w", err)
}

log.Println("[schema] Created users table")
return nil
}

// createEmailVerificationsTable creates the email verification tokens table.
//
// This table stores temporary tokens for email verification during user registration.
// Tokens have expiration times and are automatically cleaned up by the application.
//
// Table Structure:
//   - id: SERIAL PRIMARY KEY
//   - user_id: INTEGER NOT NULL (references users.id)
//   - token: VARCHAR(255) UNIQUE NOT NULL (verification token)
//   - expires_at: TIMESTAMP WITH TIME ZONE NOT NULL (token expiration)
//   - created_at: TIMESTAMP WITH TIME ZONE DEFAULT NOW()
//
// Foreign Keys:
//   - user_id references users(id) ON DELETE CASCADE
//
// Indexes:
//   - Index on expires_at for cleanup queries
//   - Unique index on token for fast lookups
func (psb *PostgreSQLSchemaBuilder) createEmailVerificationsTable() error {
query := `
CREATE TABLE IF NOT EXISTS email_verifications (
id SERIAL PRIMARY KEY,
user_id INTEGER NOT NULL,
token VARCHAR(255) UNIQUE NOT NULL,
expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
ALTER TABLE email_verifications ENABLE ROW LEVEL SECURITY;
`

if _, err := psb.db.Exec(query); err != nil {
return fmt.Errorf("failed to create email_verifications table: %w", err)
}

// Create index for efficient cleanup of expired tokens
indexQuery := `
CREATE INDEX IF NOT EXISTS idx_email_verifications_expires 
ON email_verifications(expires_at)`

if _, err := psb.db.Exec(indexQuery); err != nil {
return fmt.Errorf("failed to create email_verifications expires index: %w", err)
}

log.Println("[schema] Created email_verifications table")
return nil
}

// createProfilesTable creates the user profiles table for multi-profile support.
//
// Profiles allow users to create multiple viewing contexts (e.g., "Kids", "Parents")
// with separate watchlists, favorites, and viewing history. Each user can have
// multiple profiles.
//
// Table Structure:
//   - id: SERIAL PRIMARY KEY
//   - user_id: INTEGER NOT NULL (references users.id)
//   - name: VARCHAR(100) NOT NULL (profile display name)
//   - avatar_url: VARCHAR(255) (optional avatar image)
//   - is_default: BOOLEAN DEFAULT FALSE (primary profile flag)
//   - created_at: TIMESTAMP WITH TIME ZONE DEFAULT NOW()
//   - updated_at: TIMESTAMP WITH TIME ZONE DEFAULT NOW()
//
// Foreign Keys:
//   - user_id references users(id) ON DELETE CASCADE
//
// Constraints:
//   - Composite unique constraint on (user_id, name) to prevent duplicate profile names per user
func (psb *PostgreSQLSchemaBuilder) createProfilesTable() error {
query := `
CREATE TABLE IF NOT EXISTS profiles (
id SERIAL PRIMARY KEY,
user_id INTEGER NOT NULL,
name VARCHAR(100) NOT NULL,
avatar_url VARCHAR(255),
is_default BOOLEAN DEFAULT FALSE,
created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
UNIQUE(user_id, name)
);
ALTER TABLE profiles ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS profiles_isolation_policy ON profiles;
CREATE POLICY profiles_isolation_policy ON profiles 
    FOR ALL 
    USING (user_id = NULLIF(current_setting('app.current_user_id', true), '')::integer);
`

if _, err := psb.db.Exec(query); err != nil {
return fmt.Errorf("failed to create profiles table: %w", err)
}

// Create index for efficient user profile queries
indexQuery := `
CREATE INDEX IF NOT EXISTS idx_profiles_user_id 
ON profiles(user_id)`

if _, err := psb.db.Exec(indexQuery); err != nil {
return fmt.Errorf("failed to create profiles user_id index: %w", err)
}

log.Println("[schema] Created profiles table")
return nil
}

// createWatchlistTable creates the user watchlist table for "watch later" functionality.
//
// The watchlist allows users to save movies and TV shows they want to watch later.
// Each entry is associated with both a user and a specific profile for multi-profile support.
//
// Table Structure:
//   - id: SERIAL PRIMARY KEY
//   - user_id: INTEGER NOT NULL (references users.id)
//   - profile_id: INTEGER NOT NULL (references profiles.id)
//   - tmdb_id: INTEGER NOT NULL (The Movie Database ID)
//   - media_type: VARCHAR(20) NOT NULL ('movie' or 'tv')
//   - title: VARCHAR(255) NOT NULL (cached title for performance)
//   - poster_path: VARCHAR(255) (cached poster URL)
//   - added_at: TIMESTAMP WITH TIME ZONE DEFAULT NOW()
//
// Foreign Keys:
//   - user_id references users(id) ON DELETE CASCADE
//   - profile_id references profiles(id) ON DELETE CASCADE
//
// Constraints:
//   - Unique constraint on (profile_id, tmdb_id, media_type) to prevent duplicates
//
// Indexes:
//   - Composite index on (user_id, added_at DESC) for user watchlist queries
//   - Unique index on (profile_id, tmdb_id, media_type) for duplicate prevention
func (psb *PostgreSQLSchemaBuilder) createWatchlistTable() error {
query := `
CREATE TABLE IF NOT EXISTS watchlist (
id SERIAL PRIMARY KEY,
user_id INTEGER NOT NULL,
profile_id INTEGER NOT NULL,
tmdb_id INTEGER NOT NULL,
media_type VARCHAR(20) NOT NULL CHECK (media_type IN ('movie', 'tv')),
title VARCHAR(255) NOT NULL,
poster_path VARCHAR(255),
added_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE,
UNIQUE(profile_id, tmdb_id, media_type)
);
ALTER TABLE watchlist ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS watchlist_isolation_policy ON watchlist;
CREATE POLICY watchlist_isolation_policy ON watchlist 
    FOR ALL 
    USING (user_id = NULLIF(current_setting('app.current_user_id', true), '')::integer);
`

if _, err := psb.db.Exec(query); err != nil {
return fmt.Errorf("failed to create watchlist table: %w", err)
}

// Create index for efficient user watchlist queries (most recent first)
userIndexQuery := `
CREATE INDEX IF NOT EXISTS idx_watchlist_user_added 
ON watchlist(user_id, added_at DESC)`

if _, err := psb.db.Exec(userIndexQuery); err != nil {
return fmt.Errorf("failed to create watchlist user_added index: %w", err)
}

log.Println("[schema] Created watchlist table")
return nil
}

// createFavoritesTable creates the user favorites table for liked content.
//
// The favorites table stores movies and TV shows that users have marked as favorites.
// Like the watchlist, favorites are associated with specific user profiles.
//
// Table Structure:
//   - id: SERIAL PRIMARY KEY
//   - user_id: INTEGER NOT NULL (references users.id)
//   - profile_id: INTEGER NOT NULL (references profiles.id)
//   - tmdb_id: INTEGER NOT NULL (The Movie Database ID)
//   - media_type: VARCHAR(20) NOT NULL ('movie' or 'tv')
//   - title: VARCHAR(255) NOT NULL (cached title)
//   - poster_path: VARCHAR(255) (cached poster URL)
//   - added_at: TIMESTAMP WITH TIME ZONE DEFAULT NOW()
//
// Foreign Keys:
//   - user_id references users(id) ON DELETE CASCADE
//   - profile_id references profiles(id) ON DELETE CASCADE
//
// Constraints:
//   - Unique constraint on (profile_id, tmdb_id, media_type)
//   - Check constraint on media_type values
//
// Indexes:
//   - Composite index on (user_id, added_at DESC)
//   - Unique index on (profile_id, tmdb_id, media_type)
func (psb *PostgreSQLSchemaBuilder) createFavoritesTable() error {
query := `
CREATE TABLE IF NOT EXISTS favorites (
id SERIAL PRIMARY KEY,
user_id INTEGER NOT NULL,
profile_id INTEGER NOT NULL,
tmdb_id INTEGER NOT NULL,
media_type VARCHAR(20) NOT NULL CHECK (media_type IN ('movie', 'tv')),
title VARCHAR(255) NOT NULL,
poster_path VARCHAR(255),
added_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE,
UNIQUE(profile_id, tmdb_id, media_type)
);
ALTER TABLE favorites ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS favorites_isolation_policy ON favorites;
CREATE POLICY favorites_isolation_policy ON favorites 
    FOR ALL 
    USING (user_id = NULLIF(current_setting('app.current_user_id', true), '')::integer);
`

if _, err := psb.db.Exec(query); err != nil {
return fmt.Errorf("failed to create favorites table: %w", err)
}

// Create index for efficient user favorites queries
userIndexQuery := `
CREATE INDEX IF NOT EXISTS idx_favorites_user_added 
ON favorites(user_id, added_at DESC)`

if _, err := psb.db.Exec(userIndexQuery); err != nil {
return fmt.Errorf("failed to create favorites user_added index: %w", err)
}

log.Println("[schema] Created favorites table")
return nil
}

// createHistoryTable creates the viewing history table.
//
// The history table tracks what content users have watched, when they watched it,
// and their progress through the content. This enables "continue watching" features
// and viewing analytics.
//
// Table Structure:
//   - id: SERIAL PRIMARY KEY
//   - user_id: INTEGER NOT NULL (references users.id)
//   - profile_id: INTEGER NOT NULL (references profiles.id)
//   - tmdb_id: INTEGER NOT NULL (The Movie Database ID)
//   - media_type: VARCHAR(20) NOT NULL ('movie' or 'tv')
//   - season_number: INTEGER (for TV shows, NULL for movies)
//   - episode_number: INTEGER (for TV shows, NULL for movies)
//   - title: VARCHAR(255) NOT NULL (cached title)
//   - poster_path: VARCHAR(255) (cached poster URL)
//   - progress_seconds: INTEGER DEFAULT 0 (playback position)
//   - duration_seconds: INTEGER (total content duration)
//   - completed: BOOLEAN DEFAULT FALSE (finished watching)
//   - watched_at: TIMESTAMP WITH TIME ZONE DEFAULT NOW()
//   - updated_at: TIMESTAMP WITH TIME ZONE DEFAULT NOW()
//
// Foreign Keys:
//   - user_id references users(id) ON DELETE CASCADE
//   - profile_id references profiles(id) ON DELETE CASCADE
//
// Constraints:
//   - Check constraint on media_type values
//   - Check constraint on progress_seconds >= 0
//
// Indexes:
//   - Composite index on (user_id, watched_at DESC) for recent history
//   - Index on (profile_id, completed) for continue watching queries
func (psb *PostgreSQLSchemaBuilder) createHistoryTable() error {
query := `
CREATE TABLE IF NOT EXISTS history (
id SERIAL PRIMARY KEY,
user_id INTEGER NOT NULL,
profile_id INTEGER NOT NULL,
tmdb_id INTEGER NOT NULL,
media_type VARCHAR(20) NOT NULL CHECK (media_type IN ('movie', 'tv')),
season_number INTEGER,
episode_number INTEGER,
title VARCHAR(255) NOT NULL,
poster_path VARCHAR(255),
progress_seconds INTEGER DEFAULT 0 CHECK (progress_seconds >= 0),
duration_seconds INTEGER,
completed BOOLEAN DEFAULT FALSE,
watched_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE
);
ALTER TABLE history ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS history_isolation_policy ON history;
CREATE POLICY history_isolation_policy ON history 
    FOR ALL 
    USING (user_id = NULLIF(current_setting('app.current_user_id', true), '')::integer);
`

if _, err := psb.db.Exec(query); err != nil {
return fmt.Errorf("failed to create history table: %w", err)
}

// Create index for recent viewing history queries
userIndexQuery := `
CREATE INDEX IF NOT EXISTS idx_history_user_watched 
ON history(user_id, watched_at DESC)`

if _, err := psb.db.Exec(userIndexQuery); err != nil {
return fmt.Errorf("failed to create history user_watched index: %w", err)
}

// Create index for "continue watching" queries (incomplete items)
continueWatchingQuery := `
CREATE INDEX IF NOT EXISTS idx_history_profile_incomplete 
ON history(profile_id, completed, watched_at DESC)`

if _, err := psb.db.Exec(continueWatchingQuery); err != nil {
return fmt.Errorf("failed to create history continue watching index: %w", err)
}

log.Println("[schema] Created history table")
return nil
}

// DropSchema drops all WaveFlix Hub tables from PostgreSQL.
//
// This method is useful for testing and development scenarios where you need
// to completely reset the database schema. It drops tables in reverse dependency
// order to avoid foreign key constraint violations.
//
// ⚠️  WARNING: This method permanently deletes all data in the database.
// Use with extreme caution, especially in production environments.
//
// Drop Order:
//  1. Tables with foreign keys (history, favorites, watchlist, profiles, email_verifications)
//  2. Core tables (users)
//
// Returns: error if schema deletion fails
//
// Example:
//
//if err := builder.DropSchema(); err != nil {
//    log.Printf("Failed to drop schema: %v", err)
//}
func (psb *PostgreSQLSchemaBuilder) DropSchema() error {
log.Println("[schema] Dropping PostgreSQL schema...")

tables := []string{
"history",
"favorites", 
"watchlist",
"profiles",
"email_verifications",
"users",
}

for _, table := range tables {
query := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)
if _, err := psb.db.Exec(query); err != nil {
return fmt.Errorf("failed to drop table %s: %w", table, err)
}
log.Printf("[schema] Dropped table: %s", table)
}

log.Println("[schema] PostgreSQL schema dropped successfully")
return nil
}

