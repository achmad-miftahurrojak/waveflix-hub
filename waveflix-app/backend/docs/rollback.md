# PostgreSQL Migration Rollback Guide

This document provides step-by-step instructions for rolling back from PostgreSQL to SQLite if needed.

## When to Rollback

Consider rollback in these scenarios:
- Performance issues with PostgreSQL
- PostgreSQL server unavailability
- Data integrity problems after migration
- Need to return to simpler deployment model

## Quick Rollback (Environment Variable)

The fastest way to rollback is to change the environment variable:

1. Stop the application
2. Set `DATABASE_TYPE=sqlite` in your `.env` file
3. Restart the application

The application will automatically use SQLite with your existing data file.

## Complete Rollback Procedure

### Step 1: Stop Application
```bash
# Stop your application process
# Method depends on your deployment
```

### Step 2: Backup Current Data
```bash
# Create backup of current SQLite file (if exists)
cp waveflix.db waveflix.db.backup.$(date +%Y%m%d_%H%M%S)
```

### Step 3: Update Configuration
Edit your `.env` file:
```env
# Change from:
DATABASE_TYPE=postgres

# To:
DATABASE_TYPE=sqlite
```

### Step 4: Start Application
```bash
# Restart your application
# The app will automatically use SQLite
```

### Step 5: Verify Rollback
Check the application logs for confirmation:
```
[db] Using database: sqlite
[db] SQLite schema created successfully
[db] Database initialization completed successfully
```

## Rollback Verification Checklist

After rollback, verify these functions work:

- [ ] Application starts successfully
- [ ] Health check returns "ok" at `/health`
- [ ] Health check shows correct database type
- [ ] User authentication works
- [ ] User registration works
- [ ] Profile creation/management works
- [ ] Watchlist add/remove operations
- [ ] Favorites add/remove operations  
- [ ] Viewing history tracking
- [ ] All API endpoints respond correctly

## Data Synchronization (Optional)

If you need to sync data back from PostgreSQL to SQLite:

### Prerequisites
- PostgreSQL database is still accessible
- Have the migration tool available

### Reverse Migration Steps

1. Create a new SQLite file for the reverse migration:
```bash
# Remove or rename current SQLite file
mv waveflix.db waveflix.db.old

# The app will create a new empty SQLite file on next start
```

2. Use reverse migration (if implemented):
```bash
# This would need custom implementation
go run cmd/migrate/main.go -reverse \
  -source "postgresql://user:pass@host/db" \
  -dest "waveflix.db"
```

3. Verify data integrity after reverse migration

## Troubleshooting Rollback Issues

### Application Won't Start After Rollback

**Problem**: App crashes with database errors after setting `DATABASE_TYPE=sqlite`

**Solutions**:
1. Check if `waveflix.db` file exists and is readable
2. Verify `.env` file syntax is correct
3. Check application logs for specific error messages

### SQLite File Not Found

**Problem**: App creates new empty database instead of using existing data

**Solutions**:
1. Verify `waveflix.db` is in the correct directory (same as executable)
2. Check `SQLITE_FILE` environment variable if set
3. Ensure file permissions allow read/write access

### Data Missing After Rollback

**Problem**: Users, profiles, or content data not visible

**Solutions**:
1. Verify you're using the correct SQLite file
2. Check if data was actually migrated to PostgreSQL previously
3. Consider restoring from SQLite backup if available

### Performance Issues After Rollback

**Problem**: SQLite performance slower than expected

**Solutions**:
1. Verify PRAGMA settings are applied (WAL mode, etc.)
2. Check database file for corruption: `PRAGMA integrity_check`
3. Consider running `VACUUM` to optimize SQLite file

## Emergency Rollback

For critical production issues, use this minimal rollback:

1. **Immediate**: Change `DATABASE_TYPE=sqlite` and restart
2. **Verify**: Check `/health` endpoint shows sqlite
3. **Monitor**: Watch application logs for errors
4. **Communicate**: Notify users of temporary service restoration

## Re-migration Considerations

If you want to try PostgreSQL again later:

1. Keep your working SQLite file as backup
2. Address the issues that caused the rollback
3. Test thoroughly in staging environment first
4. Plan maintenance window for re-migration

## Support

If rollback procedures fail:

1. Check application logs for specific errors
2. Verify file permissions and paths
3. Ensure environment variables are set correctly
4. Test with a minimal configuration first

Remember: The SQLite fallback is designed to be reliable and always available as a safety net.
