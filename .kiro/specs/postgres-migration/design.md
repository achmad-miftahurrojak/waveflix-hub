# Design Document: PostgreSQL Migration

## Overview

This document describes the technical design for migrating WaveFlix Hub from SQLite to PostgreSQL. The migration enables horizontal scaling, improved write concurrency, connection pooling, and database replication capabilities essential for production-grade deployments.

### Goals

- Enable PostgreSQL as primary database with pgx driver
- Maintain backward compatibility with SQLite during transition
- Support zero-downtime migration of existing data
- Preserve all existing functionality without code duplication

### Non-Goals

- Multi-region database replication (future consideration)
- Read replica configuration (out of scope)
- Database sharding (out of scope)

## Architecture

### Component Diagram

```mermaid
graph TB
    subgraph "Application Layer"
        HTTP[HTTP Handlers]
        AUTH[Auth Module]
        PROFILES[Profiles Module]
        USERDATA[User Data Module]
    end

    subgraph "Database Abstraction Layer"
        DBA[Database Adapter Interface]
        SQLA[SQL Compatibility Layer]
        POOL[Connection Pool Manager]
    end

    subgraph "Database Drivers"
        PGX[pgx/v5 Driver]
        SQLITE[modernc.org/sqlite]
    end

    subgraph "Configuration"
        ENV[Environment Variables]
        CFG[Config Loader]
    end

    subgraph "Databases"
        PG[(PostgreSQL)]
        SQLITEDB[(SQLite)]
    end

    HTTP --> DBA
    AUTH --> DBA
    PROFILES --> DBA
    USERDATA --> DBA

    DBA --> SQLA
    SQLA --> POOL
    POOL --> PGX
    POOL --> SQLITE

    CFG --> ENV
    CFG --> DBA

    PGX --> PG
    SQLITE --> SQLITEDB
```

### Design Decisions

| Decision                              | Rationale                                                                                                  |
| ------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| Use pgx/v5 driver                     | Industry-standard PostgreSQL driver for Go with built-in connection pooling and database/sql compatibility |
| Abstract database layer via interface | Enables transparent switching between SQLite and PostgreSQL without code duplication                       |
| Translate SQL at runtime              | Avoids maintaining two codebases; translates `?` to `$1, $2` and handles dialect differences               |
| Keep SQLite as fallback               | Ensures rollback capability and gradual migration path                                                     |
| Connection pooling via pgx            | Native pooling with health checks, no external dependency like PgBouncer required                          |

## Components and Interfaces

### Database Adapter Interface

The core abstraction that allows the application to work with both SQLite and PostgreSQL transparently.

```go
// DatabaseAdapter provides a unified interface for database operations
type DatabaseAdapter interface {
    // Core operations
    Exec(query string, args ...interface{}) (sql.Result, error)
    Query(query string, args ...interface{}) (*sql.Rows, error)
    QueryRow(query string, args ...interface{}) *sql.Row

    // Transaction support
    Begin() (*sql.Tx, error)

    // Adapter info
    DbType() string  // "postgres" or "sqlite"

    // Health check
    Ping() error

    // Connection stats (PostgreSQL only)
    Stats() *ConnectionStats
}

type ConnectionStats struct {
    OpenConnections int
    IdleConnections int
    InUse           int
    MaxOpen         int
    MaxIdle         int
}
```

### SQL Compatibility Layer

Translates SQL queries between SQLite and PostgreSQL dialects at runtime.

```go
// SQLTranslator handles dialect conversion
type SQLTranslator struct {
    dbType string
}

func (t *SQLTranslator) Translate(query string) string {
    if t.dbType == "sqlite" {
        return query // No translation needed
    }

    // PostgreSQL translations
    query = t.translatePlaceholders(query)
    query = t.translateDateTime(query)
    query = t.translateBooleanLiterals(query)
    query = t.translateUpsert(query)

    return query
}

// translatePlaceholders converts ? to $1, $2, ...
func (t *SQLTranslator) translatePlaceholders(query string) string {
    var result strings.Builder
    paramNum := 1
    for _, ch := range query {
        if ch == '?' {
            result.WriteString(fmt.Sprintf("$%d", paramNum))
            paramNum++
        } else {
            result.WriteRune(ch)
        }
    }
    return result.String()
}
```

### Translation Rules

| SQLite Syntax       | PostgreSQL Equivalent               |
| ------------------- | ----------------------------------- |
| `?` (placeholder)   | `$1, $2, $3...`                     |
| `CURRENT_TIMESTAMP` | `NOW()` or `CURRENT_TIMESTAMP`      |
| `datetime('now')`   | `NOW()`                             |
| `1` / `0` (boolean) | `TRUE` / `FALSE`                    |
| `INSERT OR IGNORE`  | `INSERT ... ON CONFLICT DO NOTHING` |
| `INSERT OR REPLACE` | `INSERT ... ON CONFLICT DO UPDATE`  |
| `AUTOINCREMENT`     | `SERIAL` or `IDENTITY`              |
| `PRAGMA` statements | PostgreSQL config (remove)          |

### Connection Pool Manager

Manages database connections with pooling for PostgreSQL.

```go
// ConnectionPoolConfig holds pool configuration
type ConnectionPoolConfig struct {
    MaxOpen     int
    MaxIdle     int
    MaxLifetime time.Duration
    MaxIdleTime time.Duration
    HealthCheck time.Duration
}

// DefaultPoolConfig returns sensible defaults
func DefaultPoolConfig() *ConnectionPoolConfig {
    return &ConnectionPoolConfig{
        MaxOpen:     getEnvInt("PG_MAX_CONNS", 25),
        MaxIdle:     getEnvInt("PG_MAX_IDLE_CONNS", 25),
        MaxLifetime: 30 * time.Minute,
        MaxIdleTime: 10 * time.Minute,
        HealthCheck: 1 * time.Minute,
    }
}

// PoolManager manages connection health
type PoolManager struct {
    db   *sql.DB
    conf *ConnectionPoolConfig
    done chan struct{}
}

func (p *PoolManager) Start() {
    ticker := time.NewTicker(p.conf.HealthCheck)
    go func() {
        for {
            select {
            case <-ticker.C:
                if err := p.db.Ping(); err != nil {
                    log.Printf("[pool] health check failed: %v", err)
                }
            case <-p.done:
                ticker.Stop()
                return
            }
        }
    }()
}
```

### Database Selector

Determines which database to use based on configuration.

```go
// DatabaseSelector chooses the appropriate database backend
type DatabaseSelector struct {
    config *DatabaseConfig
}

type DatabaseConfig struct {
    Type     string // "postgres" or "sqlite"
    Postgres *PostgresConfig
    SQLite   *SQLiteConfig
}

type PostgresConfig struct {
    URL      string // Full connection URL (preferred)
    Host     string
    Port     string
    User     string
    Password string
    Database string
    SSLMode  string
}

type SQLiteConfig struct {
    File string
}

func (s *DatabaseSelector) Select() (DatabaseAdapter, error) {
    // Priority order:
    // 1. DATABASE_TYPE env var (explicit)
    // 2. DATABASE_URL present -> PostgreSQL
    // 3. PG_HOST present -> PostgreSQL
    // 4. Fallback to SQLite

    dbType := os.Getenv("DATABASE_TYPE")

    switch {
    case dbType == "sqlite":
        return s.newSQLiteAdapter()
    case dbType == "postgres":
        return s.newPostgresAdapter()
    case os.Getenv("DATABASE_URL") != "":
        return s.newPostgresAdapter()
    case os.Getenv("PG_HOST") != "":
        return s.newPostgresAdapter()
    default:
        log.Println("[db] No PostgreSQL config found, falling back to SQLite")
        return s.newSQLiteAdapter()
    }
}
```

## Data Models

### PostgreSQL Schema

The PostgreSQL schema uses PostgreSQL-specific types and constraints:

```sql
-- Users table
CREATE TABLE IF NOT EXISTS users (
    id            SERIAL PRIMARY KEY,
    email         VARCHAR(254) UNIQUE NOT NULL,
    username      VARCHAR(80) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    avatar        TEXT DEFAULT '',
    banner        TEXT DEFAULT '',
    bio           TEXT DEFAULT '',
    name_font     VARCHAR(80) DEFAULT '',
    language      VARCHAR(10) DEFAULT 'id',
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Email verifications table
CREATE TABLE IF NOT EXISTS email_verifications (
    email      VARCHAR(254) PRIMARY KEY,
    code_hash  VARCHAR(255) NOT NULL,
    expires_at BIGINT NOT NULL,
    attempts   INTEGER DEFAULT 0
);

-- Profiles table
CREATE TABLE IF NOT EXISTS profiles (
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       VARCHAR(80) NOT NULL,
    avatar     TEXT DEFAULT '',
    banner     TEXT DEFAULT '',
    bio        TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Watchlist table
CREATE TABLE IF NOT EXISTS watchlist (
    id           SERIAL PRIMARY KEY,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    profile_id   INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    tmdb_id      INTEGER NOT NULL,
    media_type   VARCHAR(10) NOT NULL,
    title        TEXT,
    poster_path  TEXT,
    vote_average REAL,
    added_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Favorites table
CREATE TABLE IF NOT EXISTS favorites (
    id           SERIAL PRIMARY KEY,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    profile_id   INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    tmdb_id      INTEGER NOT NULL,
    media_type   VARCHAR(10) NOT NULL,
    title        TEXT,
    poster_path  TEXT,
    vote_average REAL,
    added_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- History table
CREATE TABLE IF NOT EXISTS history (
    id           SERIAL PRIMARY KEY,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    profile_id   INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    tmdb_id      INTEGER NOT NULL,
    media_type   VARCHAR(10) NOT NULL,
    title        TEXT,
    poster_path  TEXT,
    vote_average REAL,
    season       INTEGER,
    episode      INTEGER,
    runtime      INTEGER DEFAULT 0,
    progress     INTEGER DEFAULT 0,
    watched_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_watchlist_profile_tmdb
    ON watchlist(profile_id, tmdb_id, media_type);
CREATE UNIQUE INDEX IF NOT EXISTS idx_favorites_profile_tmdb
    ON favorites(profile_id, tmdb_id, media_type);
CREATE INDEX IF NOT EXISTS idx_watchlist_user_added
    ON watchlist(user_id, added_at DESC);
CREATE INDEX IF NOT EXISTS idx_favorites_user_added
    ON favorites(user_id, added_at DESC);
CREATE INDEX IF NOT EXISTS idx_history_user_watched
    ON history(user_id, watched_at DESC);
CREATE INDEX IF NOT EXISTS idx_profiles_user_id
    ON profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_email_verifications_expires
    ON email_verifications(expires_at);
```

### Type Mapping

| SQLite Type | PostgreSQL Type             | Notes                              |
| ----------- | --------------------------- | ---------------------------------- |
| `INTEGER`   | `INTEGER` / `SERIAL`        | Use SERIAL for auto-increment PKs  |
| `TEXT`      | `TEXT` / `VARCHAR(n)`       | Use VARCHAR for constrained fields |
| `REAL`      | `REAL` / `DOUBLE PRECISION` | Equivalent                         |
| `DATETIME`  | `TIMESTAMP WITH TIME ZONE`  | PostgreSQL stores timezone         |
| `BOOLEAN`   | `BOOLEAN`                   | SQLite uses 0/1 integers           |

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Placeholder Translation Count Preservation

*For any* SQL query string containing N question mark (?) placeholders, the translation to PostgreSQL format SHALL produce a query with exactly N numbered placeholders ($1 through $N), maintaining the same parameter order.

**Validates: Requirements 6.3**

### Property 2: Connection Backoff Bounds

*For any* retry attempt number N during connection failure, the exponential backoff duration SHALL be at least `initialBackoff * 1.5^N` and at most maxBackoff (30 seconds), ensuring retries do not overwhelm the system.

**Validates: Requirements 1.5**

### Property 3: Database Selector Determinism

*For any* identical set of environment variables (DATABASE_TYPE, DATABASE_URL, PG_HOST), the database selector SHALL always return the same database adapter type.

**Validates: Requirements 5.1, 5.2, 5.3, 5.4**

### Property 4: Schema Idempotency

*For any* number of schema creation executions greater than or equal to 1, the resulting database schema SHALL be identical with no errors raised on subsequent executions.

**Validates: Requirements 2.4**

### Property 5: Connection Pool Configuration Override

*For any* valid integer value V in the PG_MAX_CONNS environment variable, the resulting connection pool's MaxOpenConns SHALL equal V. Similarly, PG_MAX_IDLE_CONNS SHALL override MaxIdleConns.

**Validates: Requirements 3.3, 3.4**

### Property 6: ID Preservation During Migration

*For any* record in SQLite with auto-increment ID X, the corresponding record in PostgreSQL after migration SHALL have the identical ID X.

**Validates: Requirements 4.7**

### Property 7: SQLite Type Transformation Validity

*For any* SQLite column value of type T (INTEGER, TEXT, REAL, DATETIME), the transformed value SHALL be compatible with its PostgreSQL type mapping (INTEGER/SERIAL, TEXT/VARCHAR, REAL/DOUBLE PRECISION, TIMESTAMP WITH TIME ZONE).

**Validates: Requirements 4.2**

### Property 8: Health Response Database Type Inclusion

*For any* database backend in use (SQLite or PostgreSQL), the /health endpoint response SHALL include the "database" field with the correct database type string.

**Validates: Requirements 10.2**

### Property 9: Health Response Pool Statistics for PostgreSQL

*For any* PostgreSQL connection in use, the /health endpoint response SHALL include connection pool statistics (open_connections, idle_connections, in_use, max_open).

**Validates: Requirements 10.3**
## Error Handling

### Connection Error Handling

```go
// ConnectionErrorHandler manages database connection errors
type ConnectionErrorHandler struct {
    maxRetries     int
    initialBackoff time.Duration
    maxBackoff     time.Duration
}

func DefaultConnectionErrorHandler() *ConnectionErrorHandler {
    return &ConnectionErrorHandler{
        maxRetries:     5,
        initialBackoff: 1 * time.Second,
        maxBackoff:     30 * time.Second,
    }
}

func (h *ConnectionErrorHandler) ConnectWithRetry(connectFn func() error) error {
    var lastErr error
    backoff := h.initialBackoff

    for attempt := 0; attempt < h.maxRetries; attempt++ {
        err := connectFn()
        if err == nil {
            return nil
        }

        lastErr = err
        log.Printf("[db] connection attempt %d failed: %v", attempt+1, err)

        if attempt < h.maxRetries-1 {
            time.Sleep(backoff)
            backoff = time.Duration(float64(backoff) * 1.5)
            if backoff > h.maxBackoff {
                backoff = h.maxBackoff
            }
        }
    }

    return fmt.Errorf("connection failed after %d retries: %w", h.maxRetries, lastErr)
}
```

### Error Propagation

| Error Type                | Handling Strategy              | User Response                  |
| ------------------------- | ------------------------------ | ------------------------------ |
| Connection refused        | Retry with exponential backoff | HTTP 503 Service Unavailable   |
| Authentication failed     | Log and exit (config error)    | Application fails to start     |
| Query timeout             | Log slow query, return error   | HTTP 500 Internal Server Error |
| Constraint violation      | Log with context               | HTTP 409 Conflict              |
| Deadlock detected         | Retry once                     | HTTP 500 (logged)              |
| Connection pool exhausted | Return error immediately       | HTTP 503 Service Unavailable   |

### Query Timeout Configuration

```go
// QueryTimeoutConfig defines timeout settings
type QueryTimeoutConfig struct {
    Default   time.Duration
    LongQuery time.Duration
}

func DefaultQueryTimeoutConfig() *QueryTimeoutConfig {
    return &QueryTimeoutConfig{
        Default:   5 * time.Second,
        LongQuery: 30 * time.Second,
    }
}

// TimeoutMiddleware wraps queries with timeout
func TimeoutMiddleware(db DatabaseAdapter, timeout time.Duration) DatabaseAdapter {
    return &timeoutAdapter{
        db:      db,
        timeout: timeout,
    }
}
```

## Testing Strategy

### Unit Tests

Unit tests verify individual components in isolation:

| Test Area               | Coverage Target         | Tools                          |
| ----------------------- | ----------------------- | ------------------------------ |
| SQL Translation         | 100% of rules           | Go testing, table-driven tests |
| Placeholder conversion  | All edge cases          | Go testing                     |
| Database selector logic | All config combinations | Go testing                     |
| Connection retry logic  | Exponential backoff     | Go testing, mocks              |
| Pool configuration      | All env var overrides   | Go testing                     |

### Integration Tests

Integration tests verify database interactions:

| Test Area             | Coverage                 | Tools             |
| --------------------- | ------------------------ | ----------------- |
| PostgreSQL connection | Connection lifecycle     | testcontainers-go |
| Schema creation       | All tables and indexes   | testcontainers-go |
| Query execution       | CRUD operations          | testcontainers-go |
| Connection pooling    | Pool behavior under load | testcontainers-go |
| Health check          | Failure scenarios        | testcontainers-go |

### Migration Tests

Migration tests verify data integrity:

| Test Area                      | Coverage           | Tools             |
| ------------------------------ | ------------------ | ----------------- |
| SQLite to PostgreSQL migration | All tables         | testcontainers-go |
| Foreign key preservation       | All relationships  | testcontainers-go |
| ID preservation                | Auto-increment IDs | testcontainers-go |
| Data type conversion           | All column types   | testcontainers-go |

### Performance Tests

Performance tests compare query execution:

| Test Area               | Metrics                   | Tools           |
| ----------------------- | ------------------------- | --------------- |
| Query latency           | P50, P95, P99             | Go benchmarking |
| Connection pool scaling | Throughput vs connections | Go benchmarking |
| Concurrent writes       | Write throughput          | Go benchmarking |
| Memory usage            | Heap allocation           | pprof           |

### Test Configuration

```go
// TestDatabase provides isolated test databases
type TestDatabase struct {
    t          *testing.T
    postgres   *testcontainers.Container
    sqliteFile string
    db         DatabaseAdapter
}

func NewTestDatabase(t *testing.T, dbType string) *TestDatabase {
    switch dbType {
    case "postgres":
        return newPostgresTestDB(t)
    case "sqlite":
        return newSQLiteTestDB(t)
    default:
        t.Fatalf("unknown db type: %s", dbType)
        return nil
    }
}
```

## Configuration Management

### Environment Variables

| Variable            | Description                              | Default       | Required    |
| ------------------- | ---------------------------------------- | ------------- | ----------- |
| `DATABASE_TYPE`     | Explicit database selection              | Auto-detected | No          |
| `DATABASE_URL`      | Full PostgreSQL connection URL           | None          | Conditional |
| `PG_HOST`           | PostgreSQL host                          | None          | Conditional |
| `PG_PORT`           | PostgreSQL port                          | 5432          | No          |
| `PG_USER`           | PostgreSQL username                      | None          | Conditional |
| `PG_PASSWORD`       | PostgreSQL password                      | None          | Conditional |
| `PG_DATABASE`       | Database name                            | None          | Conditional |
| `PG_SSLMODE`        | SSL mode (disable, require, verify-full) | require       | No          |
| `PG_MAX_CONNS`      | Maximum connections                      | 25            | No          |
| `PG_MAX_IDLE_CONNS` | Maximum idle connections                 | 25            | No          |

### Configuration Priority

1. **Explicit DATABASE_TYPE**: If set to "sqlite" or "postgres", use that
2. **DATABASE_URL**: If present, parse and use PostgreSQL
3. **Individual PG\_\* vars**: If PG_HOST is set, use PostgreSQL
4. **Fallback**: Use SQLite

### Example Configurations

**PostgreSQL via URL:**

```bash
DATABASE_URL=postgresql://user:password@localhost:5432/waveflix?sslmode=require
```

**PostgreSQL via individual vars:**

```bash
PG_HOST=localhost
PG_PORT=5432
PG_USER=waveflix
PG_PASSWORD=secret
PG_DATABASE=waveflix
PG_SSLMODE=require
PG_MAX_CONNS=50
```

**SQLite (explicit):**

```bash
DATABASE_TYPE=sqlite
```

## Rollback Procedure

### Rollback Design

The rollback procedure allows reverting to SQLite if PostgreSQL migration issues occur.

```mermaid
graph LR
    A[PostgreSQL Issue Detected] --> B[Decision: Rollback?]
    B -->|Yes| C[Set DATABASE_TYPE=sqlite]
    C --> D[Restart Application]
    D --> E[Verify SQLite Connectivity]
    E --> F[Monitor for Issues]
    B -->|No| G[Troubleshoot PostgreSQL]
```

### Rollback Steps

1. **Stop traffic to PostgreSQL instances**

   ```bash
   # Update load balancer to point to maintenance page
   # Or use blue-green deployment
   ```

2. **Set environment variable to SQLite**

   ```bash
   export DATABASE_TYPE=sqlite
   # Or update .env file
   echo "DATABASE_TYPE=sqlite" >> backend/.env
   ```

3. **Restart the application**

   ```bash
   # Using systemd
   systemctl restart waveflix-backend

   # Or using Docker
   docker restart waveflix-backend
   ```

4. **Verify database connectivity**

   ```bash
   curl http://localhost:8080/health
   # Expected: {"status":"ok","database":"sqlite"}
   ```

5. **Verify functionality**
   - Test user login
   - Test watchlist operations
   - Test profile operations

### Rollback Verification Checklist

- [ ] Application starts successfully
- [ ] Health check returns "ok"
- [ ] Health check shows "database": "sqlite"
- [ ] User authentication works
- [ ] Watchlist operations work
- [ ] Favorites operations work
- [ ] History operations work
- [ ] Profile operations work

### Data Synchronization (Optional)

If data changes occurred in PostgreSQL during the issue, synchronize back to SQLite:

```go
// SyncFromPostgreSQL copies data from PostgreSQL to SQLite
func SyncFromPostgreSQL(ctx context.Context, pgDB, sqliteDB *sql.DB) error {
    // This is a one-way sync, use with caution
    // Only needed if data was modified in PostgreSQL during the issue

    tables := []string{"users", "profiles", "watchlist", "favorites", "history"}

    for _, table := range tables {
        if err := syncTable(ctx, pgDB, sqliteDB, table); err != nil {
            return fmt.Errorf("sync %s: %w", table, err)
        }
    }

    return nil
}
```

## Performance Considerations

### Connection Pool Tuning

| Parameter         | Formula                        | Description                               |
| ----------------- | ------------------------------ | ----------------------------------------- |
| `MaxOpenConns`    | CPU cores \* 2 + disk spindles | General formula for write-heavy workloads |
| `MaxIdleConns`    | Equal to MaxOpenConns          | Prevents connection thrashing             |
| `ConnMaxLifetime` | 30 minutes                     | Prevents stale connections                |
| `ConnMaxIdleTime` | 10 minutes                     | Closes unused connections                 |

### Query Optimization

1. **Prepared Statements**: Use prepared statements for repeated queries

   ```go
   // Cache prepared statements
   var stmtCache = make(map[string]*sql.Stmt)

   func prepareStatement(db *sql.DB, query string) (*sql.Stmt, error) {
       if stmt, ok := stmtCache[query]; ok {
           return stmt, nil
       }
       stmt, err := db.Prepare(query)
       if err != nil {
           return nil, err
       }
       stmtCache[query] = stmt
       return stmt, nil
   }
   ```

2. **Query Timeout**: Set statement timeout

   ```sql
   SET statement_timeout = '5s';
   ```

3. **Index Usage**: Verify index usage with EXPLAIN
   ```sql
   EXPLAIN ANALYZE SELECT * FROM watchlist WHERE user_id = 123;
   ```

### Benchmark Targets

| Metric                 | Target     | Measurement         |
| ---------------------- | ---------- | ------------------- |
| Query latency (P95)    | < 10ms     | Simple CRUD         |
| Query latency (P99)    | < 50ms     | Complex joins       |
| Connection acquisition | < 1ms      | From pool           |
| Write throughput       | > 1000 TPS | Single table insert |
| Concurrent connections | > 100      | Simultaneous        |

### Monitoring Queries

```sql
-- Active connections
SELECT count(*) FROM pg_stat_activity
WHERE datname = 'waveflix';

-- Long-running queries
SELECT pid, query, state, wait_event_type, wait_event
FROM pg_stat_activity
WHERE state = 'active' AND query_start < NOW() - INTERVAL '5 seconds';

-- Index usage
SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read
FROM pg_stat_user_indexes
ORDER BY idx_scan ASC;

-- Table bloat
SELECT schemaname, tablename, n_live_tup, n_dead_tup
FROM pg_stat_user_tables
ORDER BY n_dead_tup DESC;
```
