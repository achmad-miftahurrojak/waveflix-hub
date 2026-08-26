package main

import (
	"database/sql"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

var db *sql.DB

const dbFile = "waveflix.db"
const legacyDBFile = "summertide.db"

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
	migrateDBFile()
	var err error
	db, err = sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatalf("Gagal buka DB: %v", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA synchronous=NORMAL;"); err != nil {
		log.Printf("Gagal set PRAGMA: %v", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)

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
		log.Fatalf("Gagal buat skema: %v", err)
	}

	db.Exec("ALTER TABLE users ADD COLUMN avatar TEXT DEFAULT ''")
	db.Exec("ALTER TABLE users ADD COLUMN banner TEXT DEFAULT ''")
	db.Exec("ALTER TABLE users ADD COLUMN bio TEXT DEFAULT ''")
	db.Exec("ALTER TABLE users ADD COLUMN name_font TEXT DEFAULT ''")
	db.Exec("ALTER TABLE users ADD COLUMN language TEXT DEFAULT 'id'")
	db.Exec("ALTER TABLE history ADD COLUMN runtime INTEGER DEFAULT 0")
	db.Exec("ALTER TABLE history ADD COLUMN progress INTEGER DEFAULT 0")
	// Add banner & bio columns to profiles (safe to run multiple times)
	db.Exec("ALTER TABLE profiles ADD COLUMN banner TEXT DEFAULT ''")
	db.Exec("ALTER TABLE profiles ADD COLUMN bio TEXT DEFAULT ''")

	db.Exec("ALTER TABLE watchlist ADD COLUMN profile_id INTEGER DEFAULT 0")
	db.Exec("ALTER TABLE favorites ADD COLUMN profile_id INTEGER DEFAULT 0")
	db.Exec("ALTER TABLE history ADD COLUMN profile_id INTEGER DEFAULT 0")

	// Create unique indexes that include profile_id
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_watchlist_profile_tmdb ON watchlist(profile_id, tmdb_id, media_type)")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_favorites_profile_tmdb ON favorites(profile_id, tmdb_id, media_type)")

	// Migration: Create default profiles for users who don't have any
	db.Exec(`
		INSERT INTO profiles (user_id, name, avatar)
		SELECT id, username, avatar FROM users
		WHERE id NOT IN (SELECT DISTINCT user_id FROM profiles)
	`)

	// Migration: Update existing records to point to the user's first profile
	db.Exec(`
		UPDATE watchlist 
		SET profile_id = (SELECT id FROM profiles WHERE profiles.user_id = watchlist.user_id LIMIT 1)
		WHERE profile_id = 0 OR profile_id IS NULL
	`)
	db.Exec(`
		UPDATE favorites 
		SET profile_id = (SELECT id FROM profiles WHERE profiles.user_id = favorites.user_id LIMIT 1)
		WHERE profile_id = 0 OR profile_id IS NULL
	`)
	db.Exec(`
		UPDATE history 
		SET profile_id = (SELECT id FROM profiles WHERE profiles.user_id = history.user_id LIMIT 1)
		WHERE profile_id = 0 OR profile_id IS NULL
	`)

	db.Exec("CREATE INDEX IF NOT EXISTS idx_watchlist_user_added ON watchlist(user_id, added_at DESC)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_favorites_user_added ON favorites(user_id, added_at DESC)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_history_user_watched ON history(user_id, watched_at DESC)")

	log.Println("Database SQLite siap (waveflix.db)")
}
