package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" 
	_ "modernc.org/sqlite"             
)


type MigrationConfig struct {
	SQLiteDB     string 
	PostgresURL  string 
	BatchSize    int    
	SkipVerify   bool   
	BackupFirst  bool   
}


type DataMigrator struct {
	config     MigrationConfig
	sqliteConn *sql.DB
	pgConn     *sql.DB
}


func NewDataMigrator(config MigrationConfig) *DataMigrator {
	return &DataMigrator{
		config: config,
	}
}


func (dm *DataMigrator) Migrate() error {
	log.Println("[migrate] Starting SQLite to PostgreSQL migration...")

	
	if err := dm.validateConfig(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	
	if dm.config.BackupFirst {
		if err := dm.createBackup(); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	
	if err := dm.connect(); err != nil {
		return fmt.Errorf("failed to connect to databases: %w", err)
	}
	defer dm.disconnect()

	
	if err := dm.createPostgreSQLSchema(); err != nil {
		return fmt.Errorf("failed to create PostgreSQL schema: %w", err)
	}

	
	tables := []string{"users", "email_verifications", "profiles", "watchlist", "favorites", "history"}
	for _, table := range tables {
		log.Printf("[migrate] Migrating table: %s", table)
		if err := dm.migrateTable(table); err != nil {
			return fmt.Errorf("failed to migrate table %s: %w", table, err)
		}
	}

	
	if !dm.config.SkipVerify {
		if err := dm.verifyMigration(); err != nil {
			return fmt.Errorf("migration verification failed: %w", err)
		}
	}

	log.Println("[migrate] Migration completed successfully!")
	return nil
}


func (dm *DataMigrator) validateConfig() error {
	if dm.config.SQLiteDB == "" {
		return fmt.Errorf("SQLite database path is required")
	}
	if dm.config.PostgresURL == "" {
		return fmt.Errorf("PostgreSQL connection URL is required")
	}
	if dm.config.BatchSize <= 0 {
		dm.config.BatchSize = 1000 
	}

	
	if _, err := os.Stat(dm.config.SQLiteDB); os.IsNotExist(err) {
		return fmt.Errorf("SQLite database file does not exist: %s", dm.config.SQLiteDB)
	}

	return nil
}


func (dm *DataMigrator) createBackup() error {
	timestamp := time.Now().Format("20060102_150405")
	backupPath := strings.Replace(dm.config.SQLiteDB, ".db", fmt.Sprintf("_backup_%s.db", timestamp), 1)

	log.Printf("[migrate] Creating backup: %s", backupPath)

	
	sourceFile, err := os.Open(dm.config.SQLiteDB)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(backupPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	
	buffer := make([]byte, 1024*1024) 
	for {
		n, err := sourceFile.Read(buffer)
		if n == 0 {
			break
		}
		if err != nil && err.Error() != "EOF" {
			return err
		}
		if _, err := destFile.Write(buffer[:n]); err != nil {
			return err
		}
	}

	log.Printf("[migrate] Backup created successfully: %s", backupPath)
	return nil
}


func (dm *DataMigrator) connect() error {
	var err error

	
	dm.sqliteConn, err = sql.Open("sqlite", dm.config.SQLiteDB)
	if err != nil {
		return fmt.Errorf("failed to connect to SQLite: %w", err)
	}

	
	if err := dm.sqliteConn.Ping(); err != nil {
		return fmt.Errorf("SQLite connection test failed: %w", err)
	}

	
	dm.pgConn, err = sql.Open("pgx", dm.config.PostgresURL)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	
	if err := dm.pgConn.Ping(); err != nil {
		return fmt.Errorf("PostgreSQL connection test failed: %w", err)
	}

	log.Println("[migrate] Database connections established")
	return nil
}


func (dm *DataMigrator) disconnect() {
	if dm.sqliteConn != nil {
		dm.sqliteConn.Close()
	}
	if dm.pgConn != nil {
		dm.pgConn.Close()
	}
}


func (dm *DataMigrator) createPostgreSQLSchema() error {
	log.Println("[migrate] Creating PostgreSQL schema...")

	
	adapter := &PostgreSQLMigrationAdapter{conn: dm.pgConn}
	schemaBuilder := NewPostgreSQLSchemaBuilder(adapter)

	return schemaBuilder.CreateSchema()
}


type PostgreSQLMigrationAdapter struct {
	conn *sql.DB
}

func (adapter *PostgreSQLMigrationAdapter) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return adapter.conn.Query(query, args...)
}

func (adapter *PostgreSQLMigrationAdapter) QueryRow(query string, args ...interface{}) *sql.Row {
	return adapter.conn.QueryRow(query, args...)
}

func (adapter *PostgreSQLMigrationAdapter) Exec(query string, args ...interface{}) (sql.Result, error) {
	return adapter.conn.Exec(query, args...)
}

func (adapter *PostgreSQLMigrationAdapter) DbType() string {
	return "postgres"
}


func (dm *DataMigrator) migrateTable(tableName string) error {
	switch tableName {
	case "users":
		return dm.migrateUsers()
	case "email_verifications":
		return dm.migrateEmailVerifications()
	case "profiles":
		return dm.migrateProfiles()
	case "watchlist":
		return dm.migrateWatchlist()
	case "favorites":
		return dm.migrateFavorites()
	case "history":
		return dm.migrateHistory()
	default:
		return fmt.Errorf("unsupported table: %s", tableName)
	}
}


func (dm *DataMigrator) migrateUsers() error {
	
	query := `SELECT id, email, username, password_hash, 
	          COALESCE(avatar, '') as avatar, 
	          COALESCE(banner, '') as banner,
	          COALESCE(bio, '') as bio,
	          COALESCE(name_font, '') as name_font,
	          COALESCE(language, 'id') as language,
	          created_at 
	          FROM users ORDER BY id`

	rows, err := dm.sqliteConn.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query users from SQLite: %w", err)
	}
	defer rows.Close()

	
	insertQuery := `INSERT INTO users (id, email, username, password_hash, is_verified, created_at, updated_at) 
	                VALUES ($1, $2, $3, $4, $5, $6, $7)`

	count := 0
	for rows.Next() {
		var id int64
		var email, username, passwordHash, avatar, banner, bio, nameFont, language, createdAt string

		err := rows.Scan(&id, &email, &username, &passwordHash, &avatar, &banner, &bio, &nameFont, &language, &createdAt)
		if err != nil {
			return fmt.Errorf("failed to scan user row: %w", err)
		}

		
		createdTime, err := time.Parse("2006-01-02 15:04:05", createdAt)
		if err != nil {
			createdTime = time.Now()
		}

		
		_, err = dm.pgConn.Exec(insertQuery, id, email, username, passwordHash, false, createdTime, createdTime)
		if err != nil {
			return fmt.Errorf("failed to insert user %d: %w", id, err)
		}

		count++
		if count%100 == 0 {
			log.Printf("[migrate] Migrated %d users...", count)
		}
	}

	log.Printf("[migrate] Successfully migrated %d users", count)
	return nil
}


func (dm *DataMigrator) migrateEmailVerifications() error {
	
	var count int
	err := dm.sqliteConn.QueryRow("SELECT COUNT(*) FROM email_verifications").Scan(&count)
	if err != nil {
		log.Printf("[migrate] email_verifications table not found or empty, skipping")
		return nil
	}

	if count == 0 {
		log.Printf("[migrate] No email verifications to migrate")
		return nil
	}

	
	log.Printf("[migrate] Skipping email_verifications migration (schema mismatch)")
	return nil
}


func (dm *DataMigrator) migrateProfiles() error {
	query := `SELECT id, user_id, name, COALESCE(avatar, '') as avatar, created_at 
	          FROM profiles ORDER BY id`

	rows, err := dm.sqliteConn.Query(query)
	if err != nil {
		log.Printf("[migrate] profiles table not found, skipping: %v", err)
		return nil
	}
	defer rows.Close()

	insertQuery := `INSERT INTO profiles (id, user_id, name, avatar_url, is_default, created_at, updated_at) 
	                VALUES ($1, $2, $3, $4, $5, $6, $7)`

	count := 0
	for rows.Next() {
		var id, userId int64
		var name, avatar, createdAt string

		err := rows.Scan(&id, &userId, &name, &avatar, &createdAt)
		if err != nil {
			return fmt.Errorf("failed to scan profile row: %w", err)
		}

		createdTime, err := time.Parse("2006-01-02 15:04:05", createdAt)
		if err != nil {
			createdTime = time.Now()
		}

		_, err = dm.pgConn.Exec(insertQuery, id, userId, name, avatar, count == 0, createdTime, createdTime)
		if err != nil {
			return fmt.Errorf("failed to insert profile %d: %w", id, err)
		}

		count++
	}

	log.Printf("[migrate] Successfully migrated %d profiles", count)
	return nil
}


func (dm *DataMigrator) migrateWatchlist() error {
	query := `SELECT id, user_id, COALESCE(profile_id, 0) as profile_id, tmdb_id, media_type, 
	          COALESCE(title, '') as title, COALESCE(poster_path, '') as poster_path, added_at 
	          FROM watchlist ORDER BY id`

	rows, err := dm.sqliteConn.Query(query)
	if err != nil {
		log.Printf("[migrate] watchlist table not found, skipping: %v", err)
		return nil
	}
	defer rows.Close()

	insertQuery := `INSERT INTO watchlist (id, user_id, profile_id, tmdb_id, media_type, title, poster_path, added_at) 
	                VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	count := 0
	for rows.Next() {
		var id, userId, profileId, tmdbId int64
		var mediaType, title, posterPath, addedAt string

		err := rows.Scan(&id, &userId, &profileId, &tmdbId, &mediaType, &title, &posterPath, &addedAt)
		if err != nil {
			return fmt.Errorf("failed to scan watchlist row: %w", err)
		}

		
		if profileId == 0 {
			profileId = 1
		}

		addedTime, err := time.Parse("2006-01-02 15:04:05", addedAt)
		if err != nil {
			addedTime = time.Now()
		}

		_, err = dm.pgConn.Exec(insertQuery, id, userId, profileId, tmdbId, mediaType, title, posterPath, addedTime)
		if err != nil {
			return fmt.Errorf("failed to insert watchlist item %d: %w", id, err)
		}

		count++
	}

	log.Printf("[migrate] Successfully migrated %d watchlist items", count)
	return nil
}


func (dm *DataMigrator) migrateFavorites() error {
	query := `SELECT id, user_id, COALESCE(profile_id, 0) as profile_id, tmdb_id, media_type, 
	          COALESCE(title, '') as title, COALESCE(poster_path, '') as poster_path, added_at 
	          FROM favorites ORDER BY id`

	rows, err := dm.sqliteConn.Query(query)
	if err != nil {
		log.Printf("[migrate] favorites table not found, skipping: %v", err)
		return nil
	}
	defer rows.Close()

	insertQuery := `INSERT INTO favorites (id, user_id, profile_id, tmdb_id, media_type, title, poster_path, added_at) 
	                VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	count := 0
	for rows.Next() {
		var id, userId, profileId, tmdbId int64
		var mediaType, title, posterPath, addedAt string

		err := rows.Scan(&id, &userId, &profileId, &tmdbId, &mediaType, &title, &posterPath, &addedAt)
		if err != nil {
			return fmt.Errorf("failed to scan favorites row: %w", err)
		}

		if profileId == 0 {
			profileId = 1
		}

		addedTime, err := time.Parse("2006-01-02 15:04:05", addedAt)
		if err != nil {
			addedTime = time.Now()
		}

		_, err = dm.pgConn.Exec(insertQuery, id, userId, profileId, tmdbId, mediaType, title, posterPath, addedTime)
		if err != nil {
			return fmt.Errorf("failed to insert favorite item %d: %w", id, err)
		}

		count++
	}

	log.Printf("[migrate] Successfully migrated %d favorite items", count)
	return nil
}


func (dm *DataMigrator) migrateHistory() error {
	query := `SELECT id, user_id, COALESCE(profile_id, 0) as profile_id, tmdb_id, media_type,
	          COALESCE(title, '') as title, COALESCE(poster_path, '') as poster_path,
	          COALESCE(season, 0) as season, COALESCE(episode, 0) as episode,
	          COALESCE(runtime, 0) as runtime, COALESCE(progress, 0) as progress,
	          watched_at 
	          FROM history ORDER BY id`

	rows, err := dm.sqliteConn.Query(query)
	if err != nil {
		log.Printf("[migrate] history table not found, skipping: %v", err)
		return nil
	}
	defer rows.Close()

	insertQuery := `INSERT INTO history (id, user_id, profile_id, tmdb_id, media_type, season_number, episode_number, 
	                title, poster_path, progress_seconds, duration_seconds, completed, watched_at, updated_at) 
	                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	count := 0
	for rows.Next() {
		var id, userId, profileId, tmdbId, season, episode, runtime, progress int64
		var mediaType, title, posterPath, watchedAt string

		err := rows.Scan(&id, &userId, &profileId, &tmdbId, &mediaType, &title, &posterPath, 
			            &season, &episode, &runtime, &progress, &watchedAt)
		if err != nil {
			return fmt.Errorf("failed to scan history row: %w", err)
		}

		if profileId == 0 {
			profileId = 1
		}

		watchedTime, err := time.Parse("2006-01-02 15:04:05", watchedAt)
		if err != nil {
			watchedTime = time.Now()
		}

		
		completed := false
		if runtime > 0 && progress > 0 {
			completionRate := float64(progress) / float64(runtime)
			completed = completionRate > 0.8
		}

		
		var seasonNum, episodeNum interface{}
		if season > 0 {
			seasonNum = season
		}
		if episode > 0 {
			episodeNum = episode
		}

		_, err = dm.pgConn.Exec(insertQuery, id, userId, profileId, tmdbId, mediaType, seasonNum, episodeNum,
			                   title, posterPath, progress, runtime, completed, watchedTime, watchedTime)
		if err != nil {
			return fmt.Errorf("failed to insert history item %d: %w", id, err)
		}

		count++
	}

	log.Printf("[migrate] Successfully migrated %d history items", count)
	return nil
}


func (dm *DataMigrator) verifyMigration() error {
	log.Println("[migrate] Verifying migration...")

	tables := []string{"users", "profiles", "watchlist", "favorites", "history"}

	for _, table := range tables {
		var sqliteCount, pgCount int

		
		err := dm.sqliteConn.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&sqliteCount)
		if err != nil {
			log.Printf("[migrate] Warning: could not count SQLite %s table: %v", table, err)
			continue
		}

		
		err = dm.pgConn.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&pgCount)
		if err != nil {
			return fmt.Errorf("could not count PostgreSQL %s table: %w", table, err)
		}

		if sqliteCount != pgCount {
			return fmt.Errorf("count mismatch for table %s: SQLite=%d, PostgreSQL=%d", table, sqliteCount, pgCount)
		}

		log.Printf("[migrate] ✓ Table %s: %d records migrated successfully", table, pgCount)
	}

	log.Println("[migrate] ✓ Migration verification completed successfully")
	return nil
}


func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run migrate_sqlite_to_postgres.go <sqlite_db_path> <postgres_url>")
		fmt.Println("Example: go run migrate_sqlite_to_postgres.go ./waveflix.db 'postgres://waveflix:password@localhost/waveflix?sslmode=disable'")
		os.Exit(1)
	}

	config := MigrationConfig{
		SQLiteDB:    os.Args[1],
		PostgresURL: os.Args[2],
		BatchSize:   1000,
		BackupFirst: true,
		SkipVerify:  false,
	}

	migrator := NewDataMigrator(config)
	if err := migrator.Migrate(); err != nil {
		log.Fatalf("[migrate] Migration failed: %v", err)
	}

	log.Println("[migrate] 🎉 Migration completed successfully!")
	log.Println("[migrate] Next steps:")
	log.Println("[migrate] 1. Update backend configuration to use PostgreSQL")
	log.Println("[migrate] 2. Test the application with PostgreSQL")
	log.Println("[migrate] 3. Update production deployment configurations")
}