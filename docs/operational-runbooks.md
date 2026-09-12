# WaveFlix Hub - Operational Runbooks

## Table of Contents

1. [Database Operations](#database-operations)
2. [Backup Operations](#backup-operations)
3. [Application Deployment](#application-deployment)
4. [Monitoring & Alerting](#monitoring-alerting)
5. [Security Operations](#security-operations)
6. [Performance Troubleshooting](#performance-troubleshooting)

## Database Operations

### PostgreSQL Database Management

#### Daily Database Health Check

**Frequency**: Daily  
**Estimated Time**: 5 minutes  
**Prerequisites**: Database access, monitoring dashboard

```bash
# 1. Check database connectivity
./scripts/backup_health_check.sh check

# 2. Verify database performance
psql -d waveflix -c "
SELECT 
    datname, 
    numbackends, 
    xact_commit, 
    xact_rollback,
    blks_read,
    blks_hit,
    (blks_hit::float/(blks_read + blks_hit + 1)) * 100 AS cache_hit_ratio
FROM pg_stat_database 
WHERE datname = 'waveflix';
"

# 3. Check for long-running queries
psql -d waveflix -c "
SELECT 
    pid,
    now() - pg_stat_activity.query_start AS duration,
    query,
    state
FROM pg_stat_activity
WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes'
AND state = 'active';
"

# 4. Monitor database size growth
psql -d waveflix -c "
SELECT 
    pg_size_pretty(pg_database_size('waveflix')) as db_size;
"
```

#### Weekly Database Maintenance

**Frequency**: Weekly (Sunday 3 AM)  
**Estimated Time**: 30 minutes

```bash
# 1. Update table statistics
psql -d waveflix -c "ANALYZE;"

# 2. Vacuum tables to reclaim space
psql -d waveflix -c "VACUUM (ANALYZE, VERBOSE);"

# 3. Reindex if needed (check fragmentation first)
psql -d waveflix -c "REINDEX DATABASE waveflix;"

# 4. Check for unused indexes
psql -d waveflix -c "
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE idx_tup_read = 0
ORDER BY schemaname, tablename;
"
```

#### Database Connection Issues

**Symptoms**: Connection timeouts, max connections reached

```bash
# 1. Check current connections
psql -d waveflix -c "
SELECT 
    count(*) as connections,
    state,
    application_name
FROM pg_stat_activity 
WHERE datname = 'waveflix'
GROUP BY state, application_name
ORDER BY connections DESC;
"

# 2. Kill idle connections (if needed)
psql -d waveflix -c "
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity 
WHERE datname = 'waveflix' 
AND state = 'idle' 
AND query_start < now() - interval '1 hour';
"

# 3. Adjust connection settings
# Edit postgresql.conf:
# max_connections = 200
# shared_buffers = 256MB

# 4. Restart PostgreSQL service
sudo systemctl restart postgresql
```

## Backup Operations

### Backup Verification Process

**Frequency**: Daily after backup completion  
**Estimated Time**: 10 minutes

```bash
# 1. Verify latest backup exists
latest_backup=$(find ./backups -name "waveflix_daily_*.sql.gz" -mtime -1 | head -1)

if [ -z "$latest_backup" ]; then
    echo "ERROR: No recent backup found"
    # Send alert
    ./scripts/backup_health_check.sh check
    exit 1
fi

# 2. Verify backup integrity
./scripts/backup_postgres.sh verify "$latest_backup"

# 3. Test backup restoration (to temporary database)
./scripts/dr_test_validation.sh backup

# 4. Check backup size compared to previous backups
current_size=$(stat -c%s "$latest_backup")
avg_size=$(find ./backups -name "waveflix_daily_*.sql.gz" -mtime -7 -exec stat -c%s {} \; | awk '{sum+=$1} END {print sum/NR}')

size_diff=$(echo "scale=2; ($current_size - $avg_size) / $avg_size * 100" | bc)
echo "Backup size difference from 7-day average: ${size_diff}%"

if (( $(echo "$size_diff > 50" | bc -l) )); then
    echo "WARNING: Backup size significantly larger than average"
fi
```

### Backup Restoration Process

**When to Use**: Database corruption, data loss, disaster recovery  
**Estimated Time**: 15-30 minutes

```bash
# 1. Stop application services
docker-compose stop backend

# 2. Create backup of current database (if possible)
./scripts/backup_postgres.sh backup emergency

# 3. List available backups
ls -la backups/waveflix_*.sql.gz | tail -10

# 4. Select and verify backup to restore
backup_to_restore="backups/waveflix_daily_20241201_020000.sql.gz"
./scripts/backup_postgres.sh verify "$backup_to_restore"

# 5. Restore backup
./scripts/backup_postgres.sh restore "$backup_to_restore"

# 6. Verify restoration
psql -d waveflix -c "SELECT COUNT(*) FROM users;"
psql -d waveflix -c "SELECT MAX(created_at) FROM users;"

# 7. Restart application services
docker-compose start backend

# 8. Verify application functionality
curl -f http://localhost:8080/health
curl -f http://localhost:8080/api/auth/check-email?email=test@example.com
```

### Backup Storage Management

**Frequency**: Weekly  
**Estimated Time**: 15 minutes

```bash
# 1. Check backup directory size
du -sh backups/

# 2. Check available disk space
df -h /path/to/backups

# 3. Clean old backups (automated via script)
./scripts/backup_postgres.sh cleanup

# 4. Verify S3 backup sync (if configured)
if [ -n "$S3_BUCKET" ]; then
    aws s3 ls s3://$S3_BUCKET/waveflix-backups/ --recursive --human-readable --summarize
fi

# 5. Test backup download from S3
latest_s3_backup=$(aws s3 ls s3://$S3_BUCKET/waveflix-backups/daily/ | tail -1 | awk '{print $4}')
aws s3 cp "s3://$S3_BUCKET/waveflix-backups/daily/$latest_s3_backup" ./test-restore.sql.gz
./scripts/backup_postgres.sh verify ./test-restore.sql.gz
rm ./test-restore.sql.gz
```

## Application Deployment

### Standard Deployment Process

**Frequency**: As needed  
**Estimated Time**: 20 minutes  
**Prerequisites**: Code review completed, tests passing

```bash
# 1. Pre-deployment checks
git status
git pull origin main

# Run tests
cd website/backend && go test ./...
cd ../frontend && npm run build

# 2. Database migration (if needed)
# Review migration scripts first
ls database/migrations/

# Apply migrations
./scripts/migrate_to_postgres.sh

# 3. Build and deploy
docker-compose build
docker-compose up -d

# 4. Health check
sleep 30
curl -f http://localhost:8080/health
curl -f http://localhost:3000/

# 5. Smoke tests
./scripts/smoke_tests.sh

# 6. Monitor for errors
docker-compose logs --tail=50 -f
```

### Rollback Process

**When to Use**: Deployment issues, critical bugs discovered  
**Estimated Time**: 10 minutes

```bash
# 1. Quick rollback using previous Docker images
docker-compose down
docker-compose up -d --scale backend=0  # Stop new backend
docker run -d --name backend-rollback \
    previous-backend-image:tag
docker-compose up -d

# 2. Or rollback using Git
git log --oneline -5  # Find previous commit
git checkout <previous-commit>
docker-compose build --no-cache
docker-compose up -d

# 3. Database rollback (if schema changed)
# Restore from pre-deployment backup
backup_file="backups/pre_deployment_$(date +%Y%m%d).sql.gz"
./scripts/backup_postgres.sh restore "$backup_file"

# 4. Verify rollback
curl -f http://localhost:8080/health
./scripts/smoke_tests.sh
```

## Monitoring & Alerting

### Alert Response Procedures

#### Critical Database Alert

**Alert**: Database connectivity failure  
**Response Time**: Immediate (5 minutes)

```bash
# 1. Check database status
docker ps | grep postgres
docker logs postgres-container

# 2. Check database connectivity
pg_isready -h localhost -p 5432 -U waveflix

# 3. Check system resources
df -h
free -h
top -n 1

# 4. If database is down, restart
docker-compose restart postgres

# 5. If restart fails, check logs
docker logs postgres-container --tail=100

# 6. If corruption suspected, restore from backup
./scripts/backup_postgres.sh restore latest_backup.sql.gz

# 7. Update stakeholders
# Post to #waveflix-alerts channel
# Update status page
```

#### High API Error Rate

**Alert**: API error rate > 5%  
**Response Time**: 15 minutes

```bash
# 1. Check application logs
docker logs waveflix-backend --tail=100

# 2. Check specific error patterns
docker logs waveflix-backend 2>&1 | grep -i error | tail -20

# 3. Check resource utilization
docker stats --no-stream

# 4. Check database performance
psql -d waveflix -c "
SELECT query, calls, total_time, mean_time 
FROM pg_stat_statements 
ORDER BY total_time DESC 
LIMIT 10;
"

# 5. Scale up if needed
docker-compose up -d --scale backend=3

# 6. If persistent, check for database locks
psql -d waveflix -c "
SELECT 
    blocked_locks.pid AS blocked_pid,
    blocked_activity.usename AS blocked_user,
    blocking_locks.pid AS blocking_pid,
    blocking_activity.usename AS blocking_user,
    blocked_activity.query AS blocked_statement
FROM pg_catalog.pg_locks blocked_locks
JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
JOIN pg_catalog.pg_locks blocking_locks ON blocking_locks.locktype = blocked_locks.locktype
JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.GRANTED;
"
```

### Performance Monitoring

#### Daily Performance Review

**Frequency**: Daily  
**Estimated Time**: 10 minutes

```bash
# 1. Check response times
curl -w "@curl-format.txt" -o /dev/null -s http://localhost:8080/api/trending

# 2. Database performance metrics
psql -d waveflix -c "
SELECT 
    datname,
    tup_returned,
    tup_fetched,
    tup_inserted,
    tup_updated,
    tup_deleted
FROM pg_stat_database 
WHERE datname = 'waveflix';
"

# 3. Check slow queries
psql -d waveflix -c "
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    stddev_time,
    rows
FROM pg_stat_statements 
WHERE mean_time > 1000  -- queries slower than 1 second
ORDER BY mean_time DESC 
LIMIT 10;
"

# 4. Application metrics
curl -s http://localhost:8080/metrics | grep -E "(http_requests|database_connections)"

# 5. Resource utilization
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}"
```

## Security Operations

### Security Incident Response

#### Suspected Unauthorized Access

**Severity**: Critical  
**Response Time**: Immediate

```bash
# 1. Immediate containment
# Block suspicious IP addresses
iptables -A INPUT -s SUSPICIOUS_IP -j DROP

# 2. Audit database access
psql -d waveflix -c "
SELECT 
    usename,
    datname,
    pid,
    client_addr,
    application_name,
    query_start,
    state,
    query
FROM pg_stat_activity
WHERE datname = 'waveflix'
AND client_addr IS NOT NULL;
"

# 3. Check for suspicious user activities
psql -d waveflix -c "
SELECT 
    id,
    email,
    last_login,
    login_count,
    created_at
FROM users 
WHERE last_login > now() - interval '1 hour'
ORDER BY last_login DESC;
"

# 4. Rotate credentials immediately
# Generate new JWT secret
new_jwt_secret=$(openssl rand -base64 32)
# Update .env file with new secret
# Restart application to invalidate existing sessions

# 5. Enable additional logging
# Increase log level to debug
# Enable query logging in PostgreSQL

# 6. Document incident
# Create incident report
# Preserve evidence (logs, database dumps)
```

### Routine Security Tasks

#### Weekly Security Review

**Frequency**: Weekly  
**Estimated Time**: 20 minutes

```bash
# 1. Review application logs for anomalies
grep -i "failed\|error\|unauthorized" logs/application.log | tail -20

# 2. Check for failed authentication attempts
psql -d waveflix -c "
SELECT 
    email,
    COUNT(*) as failed_attempts,
    MAX(created_at) as last_attempt
FROM failed_login_attempts 
WHERE created_at > now() - interval '7 days'
GROUP BY email
HAVING COUNT(*) > 5
ORDER BY failed_attempts DESC;
"

# 3. Review database user privileges
psql -d waveflix -c "\\du"

# 4. Check for unused user accounts
psql -d waveflix -c "
SELECT 
    id,
    email,
    last_login,
    created_at
FROM users 
WHERE last_login < now() - interval '90 days'
OR last_login IS NULL
ORDER BY created_at DESC;
"

# 5. Verify backup encryption (if implemented)
# Check backup files for proper encryption
# Test decryption process

# 6. Update dependencies
cd website/backend && go mod tidy && go mod audit
cd ../frontend && npm audit
```

## Performance Troubleshooting

### High CPU Usage

**Symptoms**: CPU usage > 80%, slow response times

```bash
# 1. Identify the source
top -p $(pgrep -d',' -f waveflix)

# 2. Check for long-running queries
psql -d waveflix -c "
SELECT 
    pid,
    now() - pg_stat_activity.query_start AS duration,
    query,
    state
FROM pg_stat_activity
WHERE state = 'active'
AND now() - pg_stat_activity.query_start > interval '30 seconds'
ORDER BY duration DESC;
"

# 3. Check application profiling
# Enable profiling endpoint temporarily
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof

# 4. Scale horizontally if needed
docker-compose up -d --scale backend=3

# 5. Analyze slow queries and add indexes if needed
psql -d waveflix -c "
CREATE INDEX CONCURRENTLY idx_users_email_active 
ON users(email) WHERE is_active = true;
"
```

### High Memory Usage

**Symptoms**: Memory usage > 90%, OOM kills

```bash
# 1. Check memory usage by process
ps aux --sort=-%mem | head -20

# 2. Check database connections
psql -d waveflix -c "
SELECT 
    count(*) as connections,
    state
FROM pg_stat_activity 
WHERE datname = 'waveflix'
GROUP BY state;
"

# 3. Check for memory leaks in application
# Monitor Go garbage collection
curl http://localhost:8080/debug/pprof/heap > heap.prof

# 4. Adjust PostgreSQL memory settings
# Edit postgresql.conf:
# shared_buffers = 256MB
# work_mem = 4MB
# maintenance_work_mem = 64MB

# 5. Restart services with memory limits
docker-compose down
docker-compose up -d --memory=1g
```

### Database Performance Issues

**Symptoms**: Slow queries, connection timeouts

```bash
# 1. Enable query logging temporarily
psql -d waveflix -c "ALTER SYSTEM SET log_min_duration_statement = 1000;"
psql -d waveflix -c "SELECT pg_reload_conf();"

# 2. Analyze query performance
psql -d waveflix -c "
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    (total_time/calls) as avg_time_ms
FROM pg_stat_statements 
ORDER BY total_time DESC 
LIMIT 20;
"

# 3. Check for missing indexes
psql -d waveflix -c "
SELECT 
    schemaname,
    tablename,
    seq_scan,
    seq_tup_read,
    idx_scan,
    idx_tup_fetch
FROM pg_stat_user_tables
WHERE seq_scan > 1000
ORDER BY seq_tup_read DESC;
"

# 4. Check table bloat
psql -d waveflix -c "
SELECT 
    tablename,
    n_tup_ins,
    n_tup_upd,
    n_tup_del,
    n_live_tup,
    n_dead_tup
FROM pg_stat_user_tables
WHERE n_dead_tup > n_live_tup * 0.1
ORDER BY n_dead_tup DESC;
"

# 5. Vacuum and analyze affected tables
psql -d waveflix -c "VACUUM ANALYZE users;"

# 6. Consider partitioning for large tables
# For tables with time-based queries
```

---

## Emergency Contacts

| Role | Primary | Secondary | Escalation |
|------|---------|-----------|------------|
| On-call Engineer | +1-xxx-xxx-xxxx | Slack DM | Team Lead |
| Database Admin | +1-xxx-xxx-xxxx | Email | CTO |
| Security Team | security@domain.com | +1-xxx-xxx-xxxx | CISO |

## Useful Commands Quick Reference

```bash
# Database
psql -d waveflix -c "SELECT version();"
pg_isready -h localhost -p 5432 -U waveflix

# Application
docker-compose ps
docker logs -f waveflix-backend
curl -f http://localhost:8080/health

# System
df -h
free -h
netstat -tlnp | grep :5432

# Backups
./scripts/backup_postgres.sh backup daily
./scripts/backup_health_check.sh check
find backups/ -name "*.sql.gz" -mtime -1
```

---

*Last Updated: 2024-01-XX*  
*Document Owner: DevOps Team*