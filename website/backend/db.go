package main

import (
"fmt"
"log"
"os"

_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
_ "modernc.org/sqlite"             // SQLite driver
)

var db DatabaseAdapter

const dbFile = "waveflix.db"
const legacyDBFile = "summertide.db"

// migrateDBFile handles legacy database file migration for SQLite
func migrateDBFile() {
if _, err := os.Stat(dbFile); err == nil {
return
}
if _, err := os.Stat(legacyDBFile); err == nil {
if err := os.Rename(legacyDBFile, dbFile); err != nil {
log.Printf("[db] gagal migrasi %s -> %s: %v", legacyDBFile, dbFile, err)
} else {
log.Printf("[db] migrasi %s -> %s berhasil", legacyDBFile, dbFile)
}
}
}

func initDB() {
log.Println("[db] Initializing database...")

// Keep legacy file migration for SQLite backward compatibility
migrateDBFile()

// Use DatabaseSelector to choose appropriate backend
selector := NewDatabaseSelector()
rawAdapter, err := selector.Select()
if err != nil {
log.Fatalf("[db] Failed to initialize database: %v", err)
}

// Wrap with SQL translation middleware
db = NewTranslatingDatabaseAdapter(rawAdapter)

// Get database type for conditional logic
dbType := db.DbType()
log.Printf("[db] Using database: %s", dbType)

// Create schema based on database type
if err := createSchema(dbType); err != nil {
log.Fatalf("[db] Failed to create schema: %v", err)
}

// Apply database-specific migrations and optimizations
if err := applyMigrations(dbType); err != nil {
log.Printf("[db] Warning: Some migrations failed: %v", err)
}

log.Println("[db] Database initialization completed successfully")
}

// createSchema creates the appropriate schema based on database type
func createSchema(dbType string) error {
switch dbType {
case "postgres":
// Use PostgreSQL schema builder
schemaBuilder := NewPostgreSQLSchemaBuilder(db)
return schemaBuilder.CreateSchema()

case "sqlite":
// Use existing SQLite schema
return createSQLiteSchema()

default:
return fmt.Errorf("unsupported database type: %s", dbType)
}
}

// createSQLiteSchema creates the original SQLite schema for backward compatibility
func createSQLiteSchema() error {
schema := `
CREATE TABLE IF NOT EXISTS users (
id            INTEGER PRIMARY KEY AUTOINCREMENT,
email         TEXT UNIQUE NOT NULL,
username      TEXT NOT NULL,
password_hash TEXT NOT NULL,
created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS email_verifications (
email      TEXT PRIMARY KEY,
code_hash  TEXT NOT NULL,
expires_at INTEGER NOT NULL,
attempts   INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS profiles (
id INTEGER PRIMARY KEY AUTOINCREMENT,
user_id INTEGER NOT NULL,
name TEXT NOT NULL,
avatar TEXT DEFAULT '',
created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS watchlist (
id          INTEGER PRIMARY KEY AUTOINCREMENT,
user_id     INTEGER NOT NULL,
profile_id  INTEGER DEFAULT 0,
tmdb_id     INTEGER NOT NULL,
media_type  TEXT NOT NULL,
title       TEXT,
poster_path TEXT,
vote_average REAL,
added_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS favorites (
id          INTEGER PRIMARY KEY AUTOINCREMENT,
user_id     INTEGER NOT NULL,
profile_id  INTEGER DEFAULT 0,
tmdb_id     INTEGER NOT NULL,
media_type  TEXT NOT NULL,
title       TEXT,
poster_path TEXT,
vote_average REAL,
added_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS history (
id          INTEGER PRIMARY KEY AUTOINCREMENT,
user_id     INTEGER NOT NULL,
profile_id  INTEGER DEFAULT 0,
tmdb_id     INTEGER NOT NULL,
media_type  TEXT NOT NULL,
title       TEXT,
poster_path TEXT,
vote_average REAL,
season      INTEGER,
episode     INTEGER,
runtime     INTEGER DEFAULT 0,
progress    INTEGER DEFAULT 0,
watched_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

if _, err := db.Exec(schema); err != nil {
return fmt.Errorf("failed to create SQLite schema: %w", err)
}

log.Println("[db] SQLite schema created successfully")
return nil
}

// applyMigrations applies database-specific migrations and optimizations
func applyMigrations(dbType string) error {
log.Printf("[db] Applying migrations for %s...", dbType)

if dbType == "sqlite" {
// Apply SQLite-specific migrations (existing logic)
return applySQLiteMigrations()
} else if dbType == "postgres" {
// PostgreSQL migrations are handled by the schema builder
log.Println("[db] PostgreSQL migrations completed by schema builder")
return nil
}

return nil
}

// applySQLiteMigrations applies the existing SQLite migration logic
func applySQLiteMigrations() error {
// Existing ALTER TABLE statements for SQLite backward compatibility
migrations := []string{
"ALTER TABLE users ADD COLUMN avatar TEXT DEFAULT ''",
"ALTER TABLE users ADD COLUMN banner TEXT DEFAULT ''",
"ALTER TABLE users ADD COLUMN bio TEXT DEFAULT ''",
"ALTER TABLE users ADD COLUMN name_font TEXT DEFAULT ''",
"ALTER TABLE users ADD COLUMN language TEXT DEFAULT 'id'",
"ALTER TABLE history ADD COLUMN runtime INTEGER DEFAULT 0",
"ALTER TABLE history ADD COLUMN progress INTEGER DEFAULT 0",
"ALTER TABLE profiles ADD COLUMN banner TEXT DEFAULT ''",
"ALTER TABLE profiles ADD COLUMN bio TEXT DEFAULT ''",
"ALTER TABLE watchlist ADD COLUMN profile_id INTEGER DEFAULT 0",
"ALTER TABLE favorites ADD COLUMN profile_id INTEGER DEFAULT 0",
"ALTER TABLE history ADD COLUMN profile_id INTEGER DEFAULT 0",
}

for _, migration := range migrations {
if _, err := db.Exec(migration); err != nil {
log.Printf("[db] Migration failed (continuing): %s - %v", migration, err)
}
}

// Create indexes
indexes := []string{
"CREATE UNIQUE INDEX IF NOT EXISTS idx_watchlist_profile_tmdb ON watchlist(profile_id, tmdb_id, media_type)",
"CREATE UNIQUE INDEX IF NOT EXISTS idx_favorites_profile_tmdb ON favorites(profile_id, tmdb_id, media_type)",
}

for _, index := range indexes {
if _, err := db.Exec(index); err != nil {
log.Printf("[db] Index creation failed (continuing): %s - %v", index, err)
}
}

// Data migrations
dataMigrations := []string{
`INSERT INTO profiles (user_id, name, avatar)
SELECT id, username, avatar FROM users
WHERE id NOT IN (SELECT DISTINCT user_id FROM profiles)`,

`UPDATE watchlist 
SET profile_id = (SELECT id FROM profiles WHERE profiles.user_id = watchlist.user_id LIMIT 1)
WHERE profile_id = 0 OR profile_id IS NULL`,

`UPDATE favorites 
SET profile_id = (SELECT id FROM profiles WHERE profiles.user_id = favorites.user_id LIMIT 1)
WHERE profile_id = 0 OR profile_id IS NULL`,

`UPDATE history 
SET profile_id = (SELECT id FROM profiles WHERE profiles.user_id = history.user_id LIMIT 1)
WHERE profile_id = 0 OR profile_id IS NULL`,
}

for _, migration := range dataMigrations {
if _, err := db.Exec(migration); err != nil {
log.Printf("[db] Data migration failed (continuing): %v", err)
}
}

log.Println("[db] SQLite migrations completed")
return nil
}


