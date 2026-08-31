# Requirements Document

## Introduction

This document specifies the requirements for migrating WaveFlix Hub from SQLite to PostgreSQL. The migration enables horizontal scaling, improved write concurrency, connection pooling, and database replication capabilities essential for production-grade deployments.

## Glossary

- **PostgreSQL**: An open-source relational database management system with enterprise-grade features
- **SQLite**: The current embedded database using modernc.org/sqlite (pure Go implementation)
- **Connection Pool**: A cache of maintained database connections for reuse
- **PgBouncer**: A lightweight connection pooler for PostgreSQL
- **pgx**: A PostgreSQL driver for Go with built-in connection pooling
- **Migration_Window**: A planned period where the system undergoes database transition
- **Zero_Downtime**: Deployment strategy where service remains available during migration
- **Write_Concurrency**: The ability to handle multiple simultaneous write operations
- **Database_Replication**: Creating and maintaining multiple copies of a database for availability

## Requirements

### Requirement 1: PostgreSQL Connection Configuration

**User Story:** As a backend developer, I want to configure PostgreSQL connections through environment variables, so that the application can connect to different PostgreSQL instances across environments.

#### Acceptance Criteria

1. WHEN the application starts, THE Database_Connector SHALL read PostgreSQL connection parameters from environment variables
2. THE Database_Connector SHALL support the following environment variables: `DATABASE_URL`, `PG_HOST`, `PG_PORT`, `PG_USER`, `PG_PASSWORD`, `PG_DATABASE`, `PG_SSLMODE`
3. IF `DATABASE_URL` is provided, THE Database_Connector SHALL parse it and use it as the primary connection string
4. IF no PostgreSQL configuration is provided, THE Database_Connector SHALL fall back to SQLite for backward compatibility
5. WHEN a connection cannot be established, THE Database_Connector SHALL log the error and retry with exponential backoff up to 30 seconds

### Requirement 2: Schema Migration to PostgreSQL

**User Story:** As a backend developer, I want the application to automatically create PostgreSQL schema on startup, so that the database is ready for use without manual setup.

#### Acceptance Criteria

1. WHEN the application connects to PostgreSQL for the first time, THE Schema_Manager SHALL create all required tables: `users`, `email_verifications`, `profiles`, `watchlist`, `favorites`, `history`
2. THE Schema_Manager SHALL create all required indexes including `idx_watchlist_profile_tmdb`, `idx_favorites_profile_tmdb`, `idx_watchlist_user_added`, `idx_favorites_user_added`, `idx_history_user_watched`
3. THE Schema_Manager SHALL apply all foreign key constraints with `ON DELETE CASCADE` behavior
4. WHEN the schema already exists, THE Schema_Manager SHALL skip table creation without errors
5. THE Schema_Manager SHALL use PostgreSQL-specific types: `SERIAL` for auto-increment, `TIMESTAMP WITH TIME ZONE` for datetime fields
6. WHERE a schema change is needed, THE Schema_Manager SHALL apply idempotent ALTER TABLE statements safely

### Requirement 3: Connection Pooling

**User Story:** As a backend developer, I want efficient connection pooling for PostgreSQL, so that the application can handle high concurrent requests without connection exhaustion.

#### Acceptance Criteria

1. THE Connection_Pool_Manager SHALL configure a maximum of 25 open connections by default
2. THE Connection_Pool_Manager SHALL configure a maximum of 25 idle connections by default
3. WHEN the `PG_MAX_CONNS` environment variable is set, THE Connection_Pool_Manager SHALL use that value for maximum connections
4. WHEN the `PG_MAX_IDLE_CONNS` environment variable is set, THE Connection_Pool_Manager SHALL use that value for maximum idle connections
5. WHILE the application is running, THE Connection_Pool_Manager SHALL maintain connection health checks
6. WHEN a connection becomes stale, THE Connection_Pool_Manager SHALL close and replace it

### Requirement 4: Data Migration from SQLite to PostgreSQL

**User Story:** As a system administrator, I want to migrate existing data from SQLite to PostgreSQL, so that users retain their accounts and data after the transition.

#### Acceptance Criteria

1. WHEN the migration command is executed, THE Data_Migrator SHALL read all records from the SQLite database
2. THE Data_Migrator SHALL transform SQLite data types to PostgreSQL-compatible types
3. THE Data_Migrator SHALL insert all records into PostgreSQL within a single transaction per table
4. IF a migration error occurs, THE Data_Migrator SHALL roll back the current transaction and report the error
5. WHEN migration completes successfully, THE Data_Migrator SHALL output a summary of migrated records per table
6. THE Data_Migrator SHALL preserve all foreign key relationships during migration
7. THE Data_Migrator SHALL preserve all auto-increment IDs from SQLite to PostgreSQL

### Requirement 5: Backward Compatibility Mode

**User Story:** As a backend developer, I want the application to support both SQLite and PostgreSQL, so that the transition can be gradual and rollback is possible.

#### Acceptance Criteria

1. WHERE `DATABASE_TYPE` environment variable is set to `sqlite`, THE Database_Selector SHALL use SQLite as the database
2. WHERE `DATABASE_TYPE` environment variable is set to `postgres`, THE Database_Selector SHALL use PostgreSQL as the database
3. IF `DATABASE_TYPE` is not set and PostgreSQL configuration is present, THE Database_Selector SHALL default to PostgreSQL
4. IF `DATABASE_TYPE` is not set and no PostgreSQL configuration is present, THE Database_Selector SHALL default to SQLite
5. WHILE running in SQLite mode, THE Application SHALL function identically to the pre-migration version

### Requirement 6: SQL Compatibility Layer

**User Story:** As a backend developer, I want SQL queries to work with both SQLite and PostgreSQL, so that code duplication is minimized during the transition period.

#### Acceptance Criteria

1. THE SQL_Adapter SHALL provide a unified interface for executing queries across both database types
2. WHEN a query uses SQLite-specific syntax, THE SQL_Adapter SHALL translate it to PostgreSQL-equivalent syntax
3. THE SQL_Adapter SHALL handle placeholder parameter differences (`?` for SQLite, `$1, $2` for PostgreSQL)
4. THE SQL_Adapter SHALL handle boolean literal differences (`1/0` for SQLite, `TRUE/FALSE` for PostgreSQL)
5. THE SQL_Adapter SHALL handle datetime function differences (`datetime('now')` vs `NOW()`)

### Requirement 7: Environment Configuration Updates

**User Story:** As a DevOps engineer, I want updated environment configuration documentation, so that deployment processes include PostgreSQL settings.

#### Acceptance Criteria

1. THE Configuration_Documentation SHALL include all new PostgreSQL environment variables
2. THE Configuration_Documentation SHALL include example connection strings for various deployment scenarios
3. THE Configuration_Documentation SHALL include SSL/TLS configuration options
4. THE Configuration_Documentation SHALL include connection pool tuning recommendations
5. WHEN the `.env.example` file is updated, THE Configuration_Documentation SHALL reflect all available options

### Requirement 8: Migration Testing Strategy

**User Story:** As a QA engineer, I want comprehensive tests for the database migration, so that the migration is reliable and data integrity is verified.

#### Acceptance Criteria

1. WHEN running the test suite, THE Test_Framework SHALL include unit tests for PostgreSQL connection configuration
2. THE Test_Framework SHALL include integration tests for schema creation
3. THE Test_Framework SHALL include migration tests that verify data integrity after SQLite-to-PostgreSQL migration
4. THE Test_Framework SHALL include performance tests comparing query execution between SQLite and PostgreSQL
5. THE Test_Framework SHALL include tests for the SQL compatibility layer
6. WHERE a test fails, THE Test_Framework SHALL provide clear error messages indicating the database type and query involved

### Requirement 9: Rollback Plan

**User Story:** As a system administrator, I want a clear rollback procedure, so that the system can be restored to SQLite if migration issues occur.

#### Acceptance Criteria

1. WHEN a rollback is initiated, THE Rollback_Handler SHALL restore the application configuration to use SQLite
2. THE Rollback_Handler SHALL retain the SQLite database file during PostgreSQL operation
3. THE Rollback_Procedure_Documentation SHALL describe step-by-step rollback instructions
4. THE Rollback_Procedure_Documentation SHALL include verification steps to confirm successful rollback
5. WHEN rollback completes, THE Rollback_Handler SHALL verify database connectivity and functionality

### Requirement 10: Health Check and Monitoring

**User Story:** As a DevOps engineer, I want database health checks integrated into the application, so that connectivity issues are detected early.

#### Acceptance Criteria

1. WHEN the `/health` endpoint is called, THE Health_Check_Handler SHALL test database connectivity
2. THE Health_Check_Response SHALL include the database type (SQLite or PostgreSQL)
3. THE Health_Check_Response SHALL include connection pool statistics when using PostgreSQL
4. IF database connectivity fails, THE Health_Check_Handler SHALL return HTTP 503 Service Unavailable
5. WHILE the application is running, THE Health_Monitor SHALL log slow queries exceeding 1 second

### Requirement 11: PostgreSQL Driver Integration

**User Story:** As a backend developer, I want to integrate the pgx driver for PostgreSQL, so that the application has native PostgreSQL support with connection pooling.

#### Acceptance Criteria

1. THE Dependency_Manager SHALL add `github.com/jackc/pgx/v5` as a dependency
2. THE Dependency_Manager SHALL add `github.com/jackc/pgx/v5/stdlib` for database/sql compatibility
3. WHEN the application initializes, THE Driver_Manager SHALL register the pgx driver
4. THE Connection_Factory SHALL create connections using the pgx driver when PostgreSQL is configured
5. THE Connection_Factory SHALL use pgx connection pool configuration for optimal performance

### Requirement 12: Query Performance Optimization

**User Story:** As a backend developer, I want to optimize queries for PostgreSQL, so that the application performs well with the new database.

#### Acceptance Criteria

1. THE Query_Optimizer SHALL use PostgreSQL-specific query plans for complex queries
2. THE Query_Optimizer SHALL add appropriate indexes for frequently queried columns
3. WHEN a query is identified as slow, THE Query_Optimizer SHALL log the query plan for analysis
4. THE Query_Optimizer SHALL use prepared statements for repeated queries
5. THE Query_Optimizer SHALL implement query timeouts to prevent long-running queries
