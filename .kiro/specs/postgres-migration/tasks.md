# Implementation Plan: PostgreSQL Migration

## Overview

This implementation plan converts the WaveFlix Hub backend from SQLite to PostgreSQL using a phased approach. The migration introduces a database abstraction layer, SQL compatibility translation, connection pooling, and a data migration tool while maintaining backward compatibility with SQLite. The implementation follows the design document's architecture with pgx/v5 driver integration and comprehensive testing at each phase.

## Tasks

- [x] 1. Setup infrastructure and database abstraction layer

  - [x] 1.1 Add pgx/v5 dependencies to go.mod

    - Add `github.com/jackc/pgx/v5` and `github.com/jackc/pgx/v5/stdlib` to go.mod
    - Run `go mod tidy` to fetch dependencies
    - Verify dependencies resolve without conflicts
    - _Requirements: 11.1, 11.2_

  - [x] 1.2 Create database adapter interface in db_adapter.go

    - Define `DatabaseAdapter` interface with Exec, Query, QueryRow, Begin, DbType, Ping, Stats methods
    - Define `ConnectionStats` struct for pool monitoring
    - Create package documentation for the adapter layer
    - _Requirements: 6.1_

  - [x] 1.3 Implement SQLite adapter in sqlite_adapter.go

    - Implement `sqliteAdapter` struct that wraps existing `*sql.DB`
    - Implement all `DatabaseAdapter` interface methods for SQLite
    - Return "sqlite" from DbType method
    - Return nil ConnectionStats (not applicable for SQLite)
    - _Requirements: 5.5, 6.1_

  - [x] 1.4 Create database configuration structures in db_config.go

    - Define `DatabaseConfig`, `PostgresConfig`, `SQLiteConfig` structs
    - Implement environment variable parsing for all PG\_\* variables
    - Support both `DATABASE_URL` and individual connection parameters
    - Parse connection string format: `postgresql://user:password@host:port/database?sslmode=require`
    - _Requirements: 1.1, 1.2, 1.3, 7.1, 7.2_

  - [x] 1.5 Implement database selector in db_selector.go
    - Create `DatabaseSelector` struct with `Select()` method
    - Implement priority logic: DATABASE_TYPE → DATABASE_URL → PG_HOST → SQLite fallback
    - Log the selected database type on initialization
    - Return appropriate adapter based on configuration
    - _Requirements: 5.1, 5.2, 5.3, 5.4_

- [x] 2. Checkpoint - Verify infrastructure setup

  - Ensure all dependencies are installed
  - Verify database adapter interface compiles
  - Ensure SQLite adapter passes basic compilation
  - Ensure all tests pass, ask the user if questions arise.

- [x] 3. Implement PostgreSQL adapter and connection pooling

  - [x] 3.1 Create PostgreSQL adapter in postgres_adapter.go

    - Implement `postgresAdapter` struct wrapping `*sql.DB` with pgx driver
    - Implement all `DatabaseAdapter` interface methods
    - Return "postgres" from DbType method
    - Implement Stats() method returning connection pool statistics
    - _Requirements: 11.3, 11.4, 11.5_

  - [x] 3.2 Implement connection pool manager in connection_pool.go

    - Define `ConnectionPoolConfig` and `PoolManager` structs
    - Implement `DefaultPoolConfig()` with environment variable overrides
    - Configure MaxOpenConns (default 25, override with PG_MAX_CONNS)
    - Configure MaxIdleConns (default 25, override with PG_MAX_IDLE_CONNS)
    - Set ConnMaxLifetime to 30 minutes
    - Set ConnMaxIdleTime to 10 minutes
    - _Requirements: 3.1, 3.2, 3.3, 3.4_

  - [x] 3.3 Implement connection health monitoring in connection_pool.go

    - Create background goroutine for periodic health checks
    - Ping database every minute (configurable)
    - Log health check failures
    - Provide graceful shutdown mechanism via done channel
    - _Requirements: 3.5, 3.6_

  - [x] 3.4 Implement connection error handler in connection_error.go

    - Define `ConnectionErrorHandler` struct with retry configuration
    - Implement `ConnectWithRetry()` method with exponential backoff
    - Start with 1 second backoff, multiply by 1.5 each retry
    - Cap backoff at 30 seconds (maxBackoff)
    - Retry up to 5 times by default
    - Log each retry attempt with attempt number
    - _Requirements: 1.5_

  - [x]\* 3.5 Write unit tests for connection pool configuration

    - **Property 5: Connection Pool Configuration Override**
    - **Validates: Requirements 3.3, 3.4**
    - Test PG_MAX_CONNS environment variable overrides MaxOpenConns
    - Test PG_MAX_IDLE_CONNS overrides MaxIdleConns
    - Test default values when env vars not set
    - Verify connection lifecycle settings (MaxLifetime, MaxIdleTime)

  - [x]\* 3.6 Write unit tests for connection retry logic
    - **Property 2: Connection Backoff Bounds**
    - **Validates: Requirements 1.5**
    - Test backoff duration increases exponentially (1.5^N factor)
    - Test backoff caps at maxBackoff (30 seconds)
    - Test retry stops after maxRetries attempts
    - Test successful connection on retry N

- [x] 4. Implement SQL compatibility layer

  - [x] 4.1 Create SQL translator in sql_translator.go

    - Define `SQLTranslator` struct with dbType field
    - Implement `Translate(query string) string` method
    - Call sub-translators for each dialect difference
    - Return original query unchanged for SQLite
    - _Requirements: 6.2_

  - [x] 4.2 Implement placeholder translation

    - Create `translatePlaceholders()` method in sql_translator.go
    - Convert `?` placeholders to `$1, $2, $3...` for PostgreSQL
    - Track parameter number as state variable
    - Handle literal `?` in string literals (don't translate)
    - _Requirements: 6.3_

  - [x] 4.3 Implement datetime function translation

    - Create `translateDateTime()` method in sql_translator.go
    - Convert `CURRENT_TIMESTAMP` to `NOW()` for PostgreSQL
    - Convert `datetime('now')` to `NOW()`
    - Handle variations like `strftime()`
    - _Requirements: 6.5_

  - [x] 4.4 Implement boolean literal translation

    - Create `translateBooleanLiterals()` method in sql_translator.go
    - Convert integer `1` to `TRUE` for PostgreSQL in boolean contexts
    - Convert integer `0` to `FALSE` for PostgreSQL in boolean contexts
    - Detect boolean context (e.g., after WHERE, in CASE expressions)
    - _Requirements: 6.4_

  - [x] 4.5 Implement UPSERT translation

    - Create `translateUpsert()` method in sql_translator.go
    - Convert `INSERT OR IGNORE` to `INSERT ... ON CONFLICT DO NOTHING`
    - Convert `INSERT OR REPLACE` to `INSERT ... ON CONFLICT DO UPDATE`
    - Parse table schema to identify conflict columns for ON CONFLICT clause
    - _Requirements: 6.2_

  - [x]\* 4.6 Write unit tests for SQL translation

    - **Property 1: Placeholder Translation Count Preservation**
    - **Validates: Requirements 6.3**
    - Test query with N `?` produces exactly N `$` placeholders
    - Test placeholder order is preserved
    - Test nested queries with multiple placeholders
    - Test edge cases: zero placeholders, single placeholder

  - [x]\* 4.7 Write unit tests for SQL dialect translations
    - Test datetime function conversions (CURRENT_TIMESTAMP, datetime('now'))
    - Test boolean literal conversions (1/0 to TRUE/FALSE)
    - Test UPSERT conversions (INSERT OR IGNORE/REPLACE)
    - Test no translation occurs for SQLite adapter
    - _Requirements: 6.4, 6.5_

- [x] 5. Checkpoint - Verify SQL compatibility layer

  - Run all SQL translator unit tests
  - Verify placeholder translation works correctly
  - Ensure datetime and boolean conversions are accurate
  - Ensure all tests pass, ask the user if questions arise.

- [x] 6. Create PostgreSQL schema with proper types

  - [x] 6.1 Implement PostgreSQL schema builder in postgres_schema.go

    - Create `createPostgresSchema()` function
    - Use PostgreSQL-specific types: SERIAL, VARCHAR, TIMESTAMP WITH TIME ZONE
    - Create all tables: users, email_verifications, profiles, watchlist, favorites, history
    - Add all foreign key constraints with ON DELETE CASCADE
    - Make all CREATE statements idempotent (IF NOT EXISTS)
    - _Requirements: 2.1, 2.5_

  - [x] 6.2 Create indexes in postgres_schema.go

    - Create unique index `idx_watchlist_profile_tmdb` on (profile_id, tmdb_id, media_type)
    - Create unique index `idx_favorites_profile_tmdb` on (profile_id, tmdb_id, media_type)
    - Create index `idx_watchlist_user_added` on (user_id, added_at DESC)
    - Create index `idx_favorites_user_added` on (user_id, added_at DESC)
    - Create index `idx_history_user_watched` on (user_id, watched_at DESC)
    - Create index `idx_profiles_user_id` on (user_id)
    - Create index `idx_email_verifications_expires` on (expires_at)
    - _Requirements: 2.2_

  - [x] 6.3 Apply foreign key constraints

    - Add FK constraint on profiles.user_id → users.id with ON DELETE CASCADE
    - Add FK constraint on watchlist.user_id → users.id with ON DELETE CASCADE
    - Add FK constraint on watchlist.profile_id → profiles.id with ON DELETE CASCADE
    - Add FK constraint on favorites.user_id → users.id with ON DELETE CASCADE
    - Add FK constraint on favorites.profile_id → profiles.id with ON DELETE CASCADE
    - Add FK constraint on history.user_id → users.id with ON DELETE CASCADE
    - Add FK constraint on history.profile_id → profiles.id with ON DELETE CASCADE
    - _Requirements: 2.3_

  - [x]\* 6.4 Write integration test for schema creation

    - **Property 4: Schema Idempotency**
    - **Validates: Requirements 2.4**
    - Use testcontainers-go to spin up PostgreSQL container
    - Run schema creation once, verify all tables exist
    - Run schema creation again, verify no errors
    - Verify schema identical after multiple runs
    - Clean up container after test

  - [x]\* 6.5 Write integration test for foreign key constraints
    - Use testcontainers-go for isolated PostgreSQL instance
    - Insert test user and profile
    - Delete user, verify profile is cascade-deleted
    - Insert test data in watchlist/favorites/history
    - Delete profile, verify cascade deletion works
    - _Requirements: 2.3_

- [x] 7. Update db.go to use database abstraction layer

  - [x] 7.1 Refactor initDB() to use DatabaseSelector

    - Replace direct `sql.Open("sqlite", ...)` with selector.Select()
    - Store returned DatabaseAdapter in global `db` variable
    - Keep legacy file migration logic for SQLite
    - Remove SQLite-specific PRAGMA statements for PostgreSQL
    - Call appropriate schema creation based on DbType()
    - _Requirements: 1.4, 5.1_

  - [x] 7.2 Integrate SQL translator into db adapter

    - Wrap DatabaseAdapter with translation middleware
    - Translate queries before execution based on DbType()
    - Apply translation to Exec, Query, QueryRow methods
    - Pass through Begin() without translation (operates on connection)
    - _Requirements: 6.1, 6.2_

  - [x] 7.3 Conditional schema initialization

    - If DbType() == "sqlite", run existing SQLite schema
    - If DbType() == "postgres", run PostgreSQL schema from postgres_schema.go
    - Apply idempotent ALTER TABLE statements for both
    - Log schema initialization completion
    - _Requirements: 2.4, 2.6_

  - [x]\* 7.4 Write integration test for database initialization
    - Test initialization with SQLite configuration
    - Test initialization with PostgreSQL configuration
    - Verify correct schema is created for each database type
    - Verify application can connect to both databases
    - _Requirements: 1.4, 5.5_

- [x] 8. Checkpoint - Verify database abstraction integration

  - Start application with DATABASE_TYPE=sqlite, verify it works
  - Start application with DATABASE_TYPE=postgres, verify connection
  - Verify schema is created correctly in PostgreSQL
  - Ensure all tests pass, ask the user if questions arise.

- [x] 9. Implement data migration tool

  - [x] 9.1 Create migration command structure in cmd/migrate/main.go

    - Create command-line tool with flags for source and destination
    - Parse connection strings for both SQLite and PostgreSQL
    - Validate source database exists and is readable
    - Validate destination database is accessible
    - _Requirements: 4.1_

  - [x] 9.2 Implement table data migration in migrate_data.go

    - Define list of tables to migrate in order (respecting FK dependencies)
    - For each table: SELECT all rows from SQLite, INSERT into PostgreSQL
    - Use transactions per table for atomicity
    - Handle type conversions: INTEGER→SERIAL, DATETIME→TIMESTAMP WITH TIME ZONE
    - Preserve auto-increment IDs using explicit ID insertion
    - _Requirements: 4.2, 4.3, 4.7_

  - [x] 9.3 Implement ID sequence reset in migrate_data.go

    - After inserting data with explicit IDs, reset PostgreSQL sequences
    - For each table with SERIAL column: `SELECT setval('table_id_seq', MAX(id)) FROM table`
    - Ensure next auto-generated ID continues from max existing ID
    - Log sequence reset for each table
    - _Requirements: 4.7_

  - [x] 9.4 Implement migration error handling and rollback

    - Wrap each table migration in a transaction
    - On error, rollback transaction and log detailed error message
    - Continue with remaining tables or stop based on flag
    - Provide detailed error context (table name, row data, SQL error)
    - _Requirements: 4.4_

  - [x] 9.5 Add migration summary reporting

    - Track migrated record count per table
    - Track failed record count per table
    - Print summary table at completion
    - Include elapsed time and throughput metrics
    - _Requirements: 4.5_

  - [x]\* 9.6 Write integration test for data migration
    - **Property 6: ID Preservation During Migration**
    - **Property 7: SQLite Type Transformation Validity**
    - **Validates: Requirements 4.2, 4.7**
    - Create SQLite database with test data including specific IDs
    - Run migration to PostgreSQL test container
    - Verify record count matches for all tables
    - Verify IDs are preserved exactly
    - Verify foreign key relationships are intact
    - Verify data types are correctly converted

- [x] 10. Enhance health check endpoint

  - [x] 10.1 Update /health endpoint in main.go

    - Add database type to health check response
    - Include connection pool statistics for PostgreSQL
    - Show open_connections, idle_connections, in_use, max_open
    - Return HTTP 503 if Ping() fails
    - _Requirements: 10.1, 10.2, 10.3, 10.4_

  - [x] 10.2 Add slow query logging

    - Wrap query execution with timing measurement
    - Log queries exceeding 1 second with full query text
    - Include query parameters in log (sanitized)
    - Add request ID for correlation
    - _Requirements: 10.5_

  - [x]\* 10.3 Write unit tests for health check
    - Test health check returns correct database type
    - Test health check includes pool stats for PostgreSQL
    - Test health check returns 503 on connection failure
    - Mock database adapter for isolated testing
    - _Requirements: 10.1, 10.2, 10.3_

- [x] 11. Update environment configuration

  - [x] 11.1 Update .env.example with PostgreSQL variables

    - Add DATABASE_TYPE with explanation
    - Add DATABASE_URL with example connection string
    - Add individual PG\_\* variables (PG_HOST, PG_PORT, PG_USER, PG_PASSWORD, PG_DATABASE)
    - Add PG_SSLMODE with options (disable, require, verify-full)
    - Add PG_MAX_CONNS and PG_MAX_IDLE_CONNS with defaults
    - Include comments explaining configuration priority
    - _Requirements: 7.1, 7.2, 7.5_

  - [x] 11.2 Create database configuration documentation in docs/database.md

    - Document configuration priority order
    - Provide example configurations for different scenarios
    - Document connection string format
    - Explain SSL/TLS configuration options
    - _Requirements: 7.3_

  - [x] 11.3 Document connection pool tuning in docs/database.md
    - Explain MaxOpenConns formula: CPU cores \* 2 + disk spindles
    - Document when to adjust pool size
    - Provide troubleshooting guide for connection exhaustion
    - Document monitoring queries for PostgreSQL
    - _Requirements: 7.4_

- [x] 12. Checkpoint - Verify configuration and documentation

  - Review .env.example completeness
  - Verify documentation accuracy
  - Test configuration with different scenarios
  - Ensure all tests pass, ask the user if questions arise.

- [x] 13. Implement comprehensive testing

  - [x]\* 13.1 Write integration tests for PostgreSQL adapter

    - Use testcontainers-go to spin up PostgreSQL for tests
    - Test Exec, Query, QueryRow methods
    - Test transaction Begin/Commit/Rollback
    - Test connection health (Ping)
    - Test connection pool statistics (Stats)
    - _Requirements: 11.3, 11.4_

  - [x]\* 13.2 Write integration tests for database selector

    - **Property 3: Database Selector Determinism**
    - **Validates: Requirements 5.1, 5.2, 5.3, 5.4**
    - Test DATABASE_TYPE=sqlite selects SQLite
    - Test DATABASE_TYPE=postgres selects PostgreSQL
    - Test DATABASE_URL present selects PostgreSQL
    - Test PG_HOST present selects PostgreSQL
    - Test fallback to SQLite when no config
    - Test same env vars always return same adapter type

  - [x]\* 13.3 Write migration integration tests

    - Create sample SQLite database with realistic data
    - Run full migration to PostgreSQL test container
    - Verify all data integrity checks pass
    - Test migration rollback on error
    - Test migration idempotency (can run twice safely)
    - _Requirements: 8.3, 8.4_

  - [x]\* 13.4 Write performance benchmark tests
    - Benchmark simple INSERT query on SQLite vs PostgreSQL
    - Benchmark SELECT with JOIN on both databases
    - Benchmark concurrent writes (10 goroutines)
    - Compare P50, P95, P99 latencies
    - Measure connection pool scaling behavior
    - _Requirements: 12.1, 12.2, 12.4, 12.5_

- [x] 14. Create rollback documentation

  - [x] 14.1 Document rollback procedure in docs/rollback.md

    - Step-by-step instructions to switch back to SQLite
    - Explain when rollback is appropriate
    - Document verification steps after rollback
    - Include troubleshooting common rollback issues
    - _Requirements: 9.3, 9.4_

  - [x] 14.2 Document data synchronization procedure (optional)

    - Explain when to sync data back from PostgreSQL to SQLite
    - Provide reverse migration script structure
    - Document risks and limitations
    - Include verification checklist
    - _Requirements: 9.1, 9.2_

  - [x] 14.3 Create rollback verification checklist in docs/rollback.md
    - Application starts successfully
    - Health check returns "ok"
    - Health check shows correct database type
    - User authentication works
    - All CRUD operations work (watchlist, favorites, history, profiles)
    - _Requirements: 9.5_

- [x] 15. Final integration and deployment preparation

  - [x] 15.1 Update README.md with database setup instructions

    - Add section on database configuration
    - Document how to start with SQLite vs PostgreSQL
    - Link to detailed configuration documentation
    - Add migration guide reference
    - _Requirements: 7.1_

  - [x] 15.2 Create deployment guide in docs/deployment.md

    - Document PostgreSQL deployment on common platforms
    - Provide Docker Compose example with PostgreSQL
    - Document environment variable configuration for production
    - Include security best practices (SSL, password management)
    - _Requirements: 7.3_

  - [x] 15.3 Add PostgreSQL to CI/CD pipeline
    - Update GitHub Actions workflow to test with PostgreSQL
    - Add PostgreSQL service container to CI
    - Run integration tests against PostgreSQL in CI
    - Ensure backward compatibility tests with SQLite still pass
    - _Requirements: 8.1, 8.2_

- [x] 16. Final checkpoint - End-to-end verification
  - Deploy application with PostgreSQL in test environment
  - Verify all endpoints work correctly
  - Run performance tests and validate metrics
  - Execute rollback procedure to verify it works
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional test tasks that can be skipped for faster MVP delivery
- Each task references specific requirements for traceability
- Property-based tests validate universal correctness properties from the design
- Checkpoints ensure incremental validation and provide natural pause points
- Integration tests use testcontainers-go for isolated PostgreSQL instances
- The implementation maintains backward compatibility with SQLite throughout
- Connection pool configuration can be tuned via environment variables without code changes
- The migration tool is a separate command-line utility for operational flexibility

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["1.2", "1.4"] },
    { "id": 2, "tasks": ["1.3", "1.5", "3.1", "4.1"] },
    { "id": 3, "tasks": ["3.2", "3.4", "4.2", "4.3", "4.4", "4.5"] },
    { "id": 4, "tasks": ["3.3", "3.5", "3.6", "4.6", "4.7", "6.1"] },
    { "id": 5, "tasks": ["6.2", "6.3", "6.4", "6.5"] },
    { "id": 6, "tasks": ["7.1", "9.1", "11.1"] },
    { "id": 7, "tasks": ["7.2", "7.3", "9.2", "11.2"] },
    { "id": 8, "tasks": ["7.4", "9.3", "9.4", "11.3"] },
    { "id": 9, "tasks": ["9.5", "9.6", "10.1"] },
    { "id": 10, "tasks": ["10.2", "10.3", "13.1", "13.2"] },
    { "id": 11, "tasks": ["13.3", "13.4", "14.1"] },
    { "id": 12, "tasks": ["14.2", "14.3", "15.1"] },
    { "id": 13, "tasks": ["15.2", "15.3"] }
  ]
}
```
