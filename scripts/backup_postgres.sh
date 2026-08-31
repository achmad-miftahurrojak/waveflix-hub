#!/bin/bash

# WaveFlix Hub - PostgreSQL Automated Backup Script
# This script performs comprehensive database backups with retention management

set -e  # Exit on any error

# Configuration from environment variables with defaults
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-waveflix}"
POSTGRES_USER="${POSTGRES_USER:-waveflix}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-waveflix_secure_password_2024}"

# Backup configuration
BACKUP_DIR="${BACKUP_DIR:-./backups}"
BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
BACKUP_RETENTION_WEEKS="${BACKUP_RETENTION_WEEKS:-12}"
BACKUP_RETENTION_MONTHS="${BACKUP_RETENTION_MONTHS:-12}"

# S3 backup configuration (optional)
S3_BUCKET="${S3_BUCKET:-}"
S3_PREFIX="${S3_PREFIX:-waveflix-backups}"
AWS_REGION="${AWS_REGION:-us-east-1}"

# Notification configuration
WEBHOOK_URL="${WEBHOOK_URL:-}"
SLACK_WEBHOOK="${SLACK_WEBHOOK:-}"
EMAIL_TO="${EMAIL_TO:-}"

# Colors and logging
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

LOG_FILE="${BACKUP_DIR}/backup.log"

# Ensure backup directory exists
mkdir -p "$BACKUP_DIR"

# Function to log messages
log() {
    local level=$1
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    case $level in
        INFO)  echo -e "${BLUE}[INFO]${NC} $message" ;;
        SUCCESS) echo -e "${GREEN}[SUCCESS]${NC} $message" ;;
        WARNING) echo -e "${YELLOW}[WARNING]${NC} $message" ;;
        ERROR) echo -e "${RED}[ERROR]${NC} $message" ;;
    esac
    
    echo "[$timestamp] [$level] $message" >> "$LOG_FILE"
}

# Function to send notifications
notify() {
    local status=$1
    local message=$2
    local backup_file=$3
    
    local emoji="✅"
    local color="#36a64f"
    
    if [ "$status" = "error" ]; then
        emoji="❌"
        color="#ff0000"
    elif [ "$status" = "warning" ]; then
        emoji="⚠️"
        color="#ffaa00"
    fi
    
    # Slack notification
    if [ -n "$SLACK_WEBHOOK" ]; then
        local slack_payload='{
            "text": "'"$emoji WaveFlix Hub Database Backup"'",
            "attachments": [{
                "color": "'"$color"'",
                "fields": [{
                    "title": "Status",
                    "value": "'"$message"'",
                    "short": true
                }, {
                    "title": "Timestamp",
                    "value": "'"$(date)"'",
                    "short": true
                }]
            }]
        }'
        
        if [ -n "$backup_file" ]; then
            slack_payload=$(echo "$slack_payload" | jq --arg file "$backup_file" '.attachments[0].fields += [{"title": "File", "value": $file, "short": true}]')
        fi
        
        curl -X POST -H 'Content-type: application/json' \
             --data "$slack_payload" \
             "$SLACK_WEBHOOK" 2>/dev/null || log WARNING "Failed to send Slack notification"
    fi
    
    # Generic webhook notification
    if [ -n "$WEBHOOK_URL" ]; then
        local webhook_payload='{
            "service": "waveflix-hub",
            "component": "database-backup",
            "status": "'"$status"'",
            "message": "'"$message"'",
            "timestamp": "'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'",
            "backup_file": "'"$backup_file"'"
        }'
        
        curl -X POST -H 'Content-type: application/json' \
             --data "$webhook_payload" \
             "$WEBHOOK_URL" 2>/dev/null || log WARNING "Failed to send webhook notification"
    fi
}

# Function to check PostgreSQL connectivity
check_postgres() {
    log INFO "Checking PostgreSQL connectivity..."
    
    export PGPASSWORD="$POSTGRES_PASSWORD"
    
    if ! pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null 2>&1; then
        log ERROR "PostgreSQL is not accessible"
        notify error "PostgreSQL connectivity check failed"
        exit 1
    fi
    
    log SUCCESS "PostgreSQL connectivity verified"
}

# Function to create backup
create_backup() {
    local backup_type=$1  # daily, weekly, monthly
    local timestamp=$(date '+%Y%m%d_%H%M%S')
    local backup_filename="waveflix_${backup_type}_${timestamp}.sql"
    local backup_path="$BACKUP_DIR/$backup_filename"
    local compressed_path="${backup_path}.gz"
    
    log INFO "Starting $backup_type backup: $backup_filename"
    
    export PGPASSWORD="$POSTGRES_PASSWORD"
    
    # Create backup with pg_dump
    if pg_dump -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
               --verbose --clean --if-exists --create --format=plain \
               --no-password --no-privileges --no-owner \
               --exclude-table-data="cache.*" \
               --exclude-table-data="analytics.page_views" \
               --exclude-table-data="analytics.api_requests" \
               > "$backup_path" 2>>"$LOG_FILE"; then
        
        # Compress the backup
        gzip "$backup_path"
        
        # Verify compressed backup
        if [ -f "$compressed_path" ] && [ -s "$compressed_path" ]; then
            local backup_size=$(du -h "$compressed_path" | cut -f1)
            log SUCCESS "$backup_type backup completed: $compressed_path ($backup_size)"
            
            # Create checksum
            sha256sum "$compressed_path" > "${compressed_path}.sha256"
            
            # Upload to S3 if configured
            if [ -n "$S3_BUCKET" ]; then
                upload_to_s3 "$compressed_path" "$backup_type"
            fi
            
            notify success "$backup_type backup completed successfully" "$compressed_path"
            echo "$compressed_path"
        else
            log ERROR "$backup_type backup verification failed"
            notify error "$backup_type backup verification failed"
            exit 1
        fi
    else
        log ERROR "$backup_type backup failed"
        notify error "$backup_type backup failed"
        exit 1
    fi
}

# Function to upload backup to S3
upload_to_s3() {
    local backup_file=$1
    local backup_type=$2
    
    if ! command -v aws &> /dev/null; then
        log WARNING "AWS CLI not installed, skipping S3 upload"
        return
    fi
    
    local s3_path="s3://$S3_BUCKET/$S3_PREFIX/$backup_type/$(basename "$backup_file")"
    
    log INFO "Uploading backup to S3: $s3_path"
    
    if aws s3 cp "$backup_file" "$s3_path" --region "$AWS_REGION" --storage-class STANDARD_IA; then
        # Upload checksum as well
        aws s3 cp "${backup_file}.sha256" "${s3_path}.sha256" --region "$AWS_REGION" --storage-class STANDARD_IA
        log SUCCESS "Backup uploaded to S3: $s3_path"
    else
        log ERROR "Failed to upload backup to S3"
        notify warning "S3 upload failed for $backup_type backup"
    fi
}

# Function to clean up old backups
cleanup_backups() {
    log INFO "Starting backup cleanup..."
    
    # Clean daily backups (keep last N days)
    find "$BACKUP_DIR" -name "waveflix_daily_*.sql.gz" -mtime +$BACKUP_RETENTION_DAYS -delete 2>/dev/null || true
    find "$BACKUP_DIR" -name "waveflix_daily_*.sql.gz.sha256" -mtime +$BACKUP_RETENTION_DAYS -delete 2>/dev/null || true
    
    # Clean weekly backups (keep last N weeks)
    local weeks_in_days=$((BACKUP_RETENTION_WEEKS * 7))
    find "$BACKUP_DIR" -name "waveflix_weekly_*.sql.gz" -mtime +$weeks_in_days -delete 2>/dev/null || true
    find "$BACKUP_DIR" -name "waveflix_weekly_*.sql.gz.sha256" -mtime +$weeks_in_days -delete 2>/dev/null || true
    
    # Clean monthly backups (keep last N months)
    local months_in_days=$((BACKUP_RETENTION_MONTHS * 30))
    find "$BACKUP_DIR" -name "waveflix_monthly_*.sql.gz" -mtime +$months_in_days -delete 2>/dev/null || true
    find "$BACKUP_DIR" -name "waveflix_monthly_*.sql.gz.sha256" -mtime +$months_in_days -delete 2>/dev/null || true
    
    # Clean old logs (keep last 30 days)
    find "$BACKUP_DIR" -name "backup.log.*" -mtime +30 -delete 2>/dev/null || true
    
    log SUCCESS "Backup cleanup completed"
}

# Function to verify backup integrity
verify_backup() {
    local backup_file=$1
    
    log INFO "Verifying backup integrity: $(basename "$backup_file")"
    
    # Check if checksum file exists
    if [ ! -f "${backup_file}.sha256" ]; then
        log WARNING "Checksum file not found for $(basename "$backup_file")"
        return 1
    fi
    
    # Verify checksum
    if sha256sum -c "${backup_file}.sha256" >/dev/null 2>&1; then
        log SUCCESS "Backup integrity verified: $(basename "$backup_file")"
        return 0
    else
        log ERROR "Backup integrity check failed: $(basename "$backup_file")"
        notify error "Backup integrity check failed" "$backup_file"
        return 1
    fi
}

# Function to create backup report
create_backup_report() {
    local report_file="$BACKUP_DIR/backup_report_$(date '+%Y%m%d').md"
    
    cat > "$report_file" << EOF
# WaveFlix Hub - Backup Report

**Date:** $(date)
**Status:** Completed Successfully ✅

## Backup Summary

### Daily Backups
$(find "$BACKUP_DIR" -name "waveflix_daily_*.sql.gz" -mtime -7 -exec ls -lh {} \; | awk '{print "- " $9 " (" $5 " - " $6 " " $7 " " $8 ")"}')

### Weekly Backups  
$(find "$BACKUP_DIR" -name "waveflix_weekly_*.sql.gz" -mtime -30 -exec ls -lh {} \; | awk '{print "- " $9 " (" $5 " - " $6 " " $7 " " $8 ")"}')

### Monthly Backups
$(find "$BACKUP_DIR" -name "waveflix_monthly_*.sql.gz" -mtime -365 -exec ls -lh {} \; | awk '{print "- " $9 " (" $5 " - " $6 " " $7 " " $8 ")"}')

## Storage Usage

**Local Storage:** $(du -sh "$BACKUP_DIR" | cut -f1)

$(if [ -n "$S3_BUCKET" ]; then
    echo "**S3 Storage:** $(aws s3 ls s3://$S3_BUCKET/$S3_PREFIX/ --recursive --human-readable --summarize 2>/dev/null | tail -1 | awk '{print $3}' || echo 'N/A')"
fi)

## Configuration

- **Retention Policy:**
  - Daily: $BACKUP_RETENTION_DAYS days
  - Weekly: $BACKUP_RETENTION_WEEKS weeks  
  - Monthly: $BACKUP_RETENTION_MONTHS months

- **Database:** $POSTGRES_HOST:$POSTGRES_PORT/$POSTGRES_DB
- **Backup Location:** $BACKUP_DIR
$(if [ -n "$S3_BUCKET" ]; then echo "- **S3 Bucket:** s3://$S3_BUCKET/$S3_PREFIX/"; fi)

## Next Scheduled Backups

- **Daily:** Every day at 2:00 AM
- **Weekly:** Every Sunday at 3:00 AM  
- **Monthly:** 1st day of month at 4:00 AM

---
Generated by WaveFlix Hub Backup System
EOF

    log SUCCESS "Backup report created: $report_file"
}

# Function to restore from backup
restore_backup() {
    local backup_file=$1
    local target_db=${2:-"$POSTGRES_DB"}
    
    if [ ! -f "$backup_file" ]; then
        log ERROR "Backup file not found: $backup_file"
        exit 1
    fi
    
    log WARNING "Starting database restore from: $(basename "$backup_file")"
    log WARNING "Target database: $target_db"
    
    # Verify backup before restore
    if ! verify_backup "$backup_file"; then
        log ERROR "Backup verification failed, aborting restore"
        exit 1
    fi
    
    # Confirmation prompt
    read -p "This will OVERWRITE the database '$target_db'. Are you sure? (yes/no): " -r
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        log INFO "Restore cancelled by user"
        exit 0
    fi
    
    export PGPASSWORD="$POSTGRES_PASSWORD"
    
    # Restore database
    if gunzip -c "$backup_file" | psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$target_db" >/dev/null 2>>"$LOG_FILE"; then
        log SUCCESS "Database restore completed successfully"
        notify success "Database restore completed" "$backup_file"
    else
        log ERROR "Database restore failed"
        notify error "Database restore failed" "$backup_file"
        exit 1
    fi
}

# Main function
main() {
    local command=${1:-backup}
    
    case $command in
        backup)
            local backup_type=${2:-daily}
            log INFO "Starting WaveFlix Hub PostgreSQL backup ($backup_type)"
            check_postgres
            backup_file=$(create_backup "$backup_type")
            verify_backup "$backup_file"
            cleanup_backups
            create_backup_report
            log SUCCESS "Backup process completed successfully"
            ;;
            
        restore)
            local backup_file=$2
            if [ -z "$backup_file" ]; then
                log ERROR "Usage: $0 restore <backup_file> [target_db]"
                exit 1
            fi
            restore_backup "$backup_file" "$3"
            ;;
            
        verify)
            local backup_file=$2
            if [ -z "$backup_file" ]; then
                log ERROR "Usage: $0 verify <backup_file>"
                exit 1
            fi
            verify_backup "$backup_file"
            ;;
            
        cleanup)
            cleanup_backups
            ;;
            
        report)
            create_backup_report
            ;;
            
        *)
            echo "Usage: $0 {backup|restore|verify|cleanup|report} [options]"
            echo ""
            echo "Commands:"
            echo "  backup [daily|weekly|monthly]  - Create database backup"
            echo "  restore <file> [target_db]     - Restore database from backup"
            echo "  verify <file>                  - Verify backup integrity"
            echo "  cleanup                        - Clean old backups"
            echo "  report                         - Generate backup report"
            echo ""
            echo "Environment Variables:"
            echo "  POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD"
            echo "  BACKUP_DIR, BACKUP_RETENTION_DAYS, S3_BUCKET, SLACK_WEBHOOK"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"