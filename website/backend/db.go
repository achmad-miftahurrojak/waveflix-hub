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

// migrateDBFile: kalau DB baru belum ada tapi DB lama ada, ganti namanya supaya
// akun/watchlist/history user lama tetap terbawa (bukan bikin DB kosong baru).
func migrateDBFile() {
	if _, err := os.Stat(dbFile); err == nil {
		return // DB baru sudah ada
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
	db.SetMaxOpenConns(1) // SQLite: hindari "database is locked"

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		email         TEXT UNIQUE NOT NULL,
		username      TEXT NOT NULL,
		password_hash TEXT NOT NULL,
		created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS watchlist (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id     INTEGER NOT NULL,
		tmdb_id     INTEGER NOT NULL,
		media_type  TEXT NOT NULL,
		title       TEXT,
		poster_path TEXT,
		vote_average REAL,
		added_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, tmdb_id, media_type)
	);

	CREATE TABLE IF NOT EXISTS favorites (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id     INTEGER NOT NULL,
		tmdb_id     INTEGER NOT NULL,
		media_type  TEXT NOT NULL,
		title       TEXT,
		poster_path TEXT,
		vote_average REAL,
		added_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, tmdb_id, media_type)
	);

	CREATE TABLE IF NOT EXISTS history (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id     INTEGER NOT NULL,
		tmdb_id     INTEGER NOT NULL,
		media_type  TEXT NOT NULL,
		title       TEXT,
		poster_path TEXT,
		vote_average REAL,
		season      INTEGER,
		episode     INTEGER,
		watched_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(schema); err != nil {
		log.Fatalf("Gagal buat skema: %v", err)
	}

	// Kolom profil tambahan (avatar/banner). ALTER diabaikan errornya kalau kolom
	// sudah ada (SQLite tidak punya ADD COLUMN IF NOT EXISTS).
	db.Exec("ALTER TABLE users ADD COLUMN avatar TEXT DEFAULT ''")
	db.Exec("ALTER TABLE users ADD COLUMN banner TEXT DEFAULT ''")
	db.Exec("ALTER TABLE users ADD COLUMN bio TEXT DEFAULT ''")
	db.Exec("ALTER TABLE users ADD COLUMN name_font TEXT DEFAULT ''")

	log.Println("Database SQLite siap (waveflix.db)")
}
