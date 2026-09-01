

package main

import (
"fmt"
"log"
)

type PostgreSQLSchemaBuilder struct {
db DatabaseAdapter
}

func NewPostgreSQLSchemaBuilder(db DatabaseAdapter) *PostgreSQLSchemaBuilder {
return &PostgreSQLSchemaBuilder{
db: db,
}
}

func (psb *PostgreSQLSchemaBuilder) CreateSchema() error {
log.Println("[schema] Creating PostgreSQL schema...")

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

func (psb *PostgreSQLSchemaBuilder) createEmailVerificationsTable() error {
query := `
CREATE TABLE IF NOT EXISTS email_verifications (
id SERIAL PRIMARY KEY,
token VARCHAR(255) UNIQUE NOT NULL,
expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
ALTER TABLE email_verifications ENABLE ROW LEVEL SECURITY;
`

if _, err := psb.db.Exec(query); err != nil {
return fmt.Errorf("failed to create email_verifications table: %w", err)
}

indexQuery := `
CREATE INDEX IF NOT EXISTS idx_email_verifications_expires 
ON email_verifications(expires_at)`

if _, err := psb.db.Exec(indexQuery); err != nil {
return fmt.Errorf("failed to create email_verifications expires index: %w", err)
}

log.Println("[schema] Created email_verifications table")
return nil
}

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

indexQuery := `
CREATE INDEX IF NOT EXISTS idx_profiles_user_id 
ON profiles(user_id)`

if _, err := psb.db.Exec(indexQuery); err != nil {
return fmt.Errorf("failed to create profiles user_id index: %w", err)
}

log.Println("[schema] Created profiles table")
return nil
}

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

userIndexQuery := `
CREATE INDEX IF NOT EXISTS idx_watchlist_user_added 
ON watchlist(user_id, added_at DESC)`

if _, err := psb.db.Exec(userIndexQuery); err != nil {
return fmt.Errorf("failed to create watchlist user_added index: %w", err)
}

log.Println("[schema] Created watchlist table")
return nil
}

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

userIndexQuery := `
CREATE INDEX IF NOT EXISTS idx_favorites_user_added 
ON favorites(user_id, added_at DESC)`

if _, err := psb.db.Exec(userIndexQuery); err != nil {
return fmt.Errorf("failed to create favorites user_added index: %w", err)
}

log.Println("[schema] Created favorites table")
return nil
}

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

userIndexQuery := `
CREATE INDEX IF NOT EXISTS idx_history_user_watched 
ON history(user_id, watched_at DESC)`

if _, err := psb.db.Exec(userIndexQuery); err != nil {
return fmt.Errorf("failed to create history user_watched index: %w", err)
}

continueWatchingQuery := `
CREATE INDEX IF NOT EXISTS idx_history_profile_incomplete 
ON history(profile_id, completed, watched_at DESC)`

if _, err := psb.db.Exec(continueWatchingQuery); err != nil {
return fmt.Errorf("failed to create history continue watching index: %w", err)
}

log.Println("[schema] Created history table")
return nil
}

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

