#!/bin/bash

# WaveFlix Hub - Backup Health Check Script
# This script monitors backup health and sends alerts if issues are detected

set -e

# Configuration
BACKUP_DIR="${BACKUP_DIR:-./backups}"
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-waveflix}"
POSTGRES_USER="${POSTGRES_USER:-waveflix}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-waveflix_secure_password_2024}"

# Alert thresholds
DAILY_BACKUP_MAX_AGE_HOURS=25    # Alert if daily backup is older than 25 hours
WEEKLY_BACKUP_MAX_AGE_DAYS=8     # Alert if weekly backup is older than 8 days
MONTHLY_BACKUP_MAX_AGE_DAYS=32   # Alert if monthly backup is older than 32 days
MAX_BACKUP_DURATION_MINUTES=60   # Alert if backup takes longer than 60 minutes
MIN_BACKUP_SIZE_MB=1             # Alert if backup is smaller than 1MB
STORAGE_USAGE_THRESHOLD=85       # Alert if backup storage usage exceeds 85%

# Notification settings
WEBHOOK_URL="${WEBHOOK_URL:-}"
SLACK_WEBHOOK="${SLACK_WEBHOOK:-}"
EMAIL_TO="${EMAIL_TO:-}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging
LOG_FILE="$BACKUP_DIR/health_check.log"
mkdir -p "$BACKUP_DIR"

log() {
    local level=$1
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    case $level in
        INFO)    echo -e "${BLUE}[INFO]${NC} $message" ;;
        SUCCESS) echo -e "${GREEN}[SUCCESS]${NC} $message" ;;
        WARNING) echo -e "${YELLOW}[WARNING]${NC} $message" ;;
        ERROR)   echo -e "${RED}[ERROR]${NC} $message" ;;
    esac
    
    echo "[$timestamp] [$level] $message" >> "$LOG_FILE"
}

# Function to send alerts
send_alert() {
    local severity=$1
    local title=$2
    local message=$3
    local details=$4
    
    local emoji="🔍"
    local color="#36a64f"
    
    case $severity in
        critical) emoji="🚨"; color="#ff0000" ;;
        warning)  emoji="⚠️"; color="#ffaa00" ;;
        info)     emoji="ℹ️"; color="#0099cc" ;;
    esac
    
    log "$(echo "$severity" | tr '[:lower:]' '[:upper:]')" "$title: $message"
    
    # Slack notification
    if [ -n "$SLACK_WEBHOOK" ]; then
        local slack_payload='{
            "text": "'"$emoji WaveFlix Hub Backup Health Check"'",
            "attachments": [{
                "color": "'"$color"'",
                "title": "'"$title"'",
                "text": "'"$message"'",
                "fields": [{
                    "title": "Severity",
                    "value": "'"$severity"'",
                    "short": true
                }, {
                    "title": "Timestamp",
                    "value": "'"$(date)"'",
                    "short": true
                }],
                "footer": "WaveFlix Hub Monitoring"
            }]
        }'
        
        if [ -n "$details" ]; then
            slack_payload=$(echo "$slack_payload" | jq --arg details "$details" '.attachments[0].fields += [{"title": "Details", "value": $details, "short": false}]')
        fi
        
        curl -X POST -H 'Content-type: application/json' \
             --data "$slack_payload" \
             "$SLACK_WEBHOOK" 2>/dev/null || log WARNING "Failed to send Slack notification"
    fi
    
    # Generic webhook notification
    if [ -n "$WEBHOOK_URL" ]; then
        local webhook_payload='{
            "service": "waveflix-hub",
            "component": "backup-health-check",
            "severity": "'"$severity"'",
            "title": "'"$title"'",
            "message": "'"$message"'",
            "details": "'"$details"'",
            "timestamp": "'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"
        }'
        
        curl -X POST -H 'Content-type: application/json' \
             --data "$webhook_payload" \
             "$WEBHOOK_URL" 2>/dev/null || log WARNING "Failed to send webhook notification"
    fi
}

# Function to check backup age
check_backup_age() {
    log INFO "Checking backup age..."
    
    local current_time=$(date +%s)
    local issues=0
    
    # Check daily backups
    local latest_daily=$(find "$BACKUP_DIR" -name "waveflix_daily_*.sql*" -type f -printf '%T@ %p\n' 2>/dev/null | sort -nr | head -1)
    if [ -n "$latest_daily" ]; then
        local daily_timestamp=$(echo "$latest_daily" | cut -d' ' -f1 | cut -d'.' -f1)
        local daily_age_hours=$(( (current_time - daily_timestamp) / 3600 ))
        
        if [ "$daily_age_hours" -gt "$DAILY_BACKUP_MAX_AGE_HOURS" ]; then
            send_alert "critical" "Daily Backup Missing" "Latest daily backup is $daily_age_hours hours old" "Expected: max $DAILY_BACKUP_MAX_AGE_HOURS hours"
            issues=$((issues + 1))
        else
            log SUCCESS "Daily backup is current ($daily_age_hours hours old)"
        fi
    else
        send_alert "critical" "Daily Backup Missing" "No daily backups found in backup directory"
        issues=$((issues + 1))
    fi
    
    # Check weekly backups
    local latest_weekly=$(find "$BACKUP_DIR" -name "waveflix_weekly_*.sql*" -type f -printf '%T@ %p\n' 2>/dev/null | sort -nr | head -1)
    if [ -n "$latest_weekly" ]; then
        local weekly_timestamp=$(echo "$latest_weekly" | cut -d' ' -f1 | cut -d'.' -f1)
        local weekly_age_days=$(( (current_time - weekly_timestamp) / 86400 ))
        
        if [ "$weekly_age_days" -gt "$WEEKLY_BACKUP_MAX_AGE_DAYS" ]; then
            send_alert "warning" "Weekly Backup Overdue" "Latest weekly backup is $weekly_age_days days old" "Expected: max $WEEKLY_BACKUP_MAX_AGE_DAYS days"
            issues=$((issues + 1))
        else
            log SUCCESS "Weekly backup is current ($weekly_age_days days old)"
        fi
    else
        send_alert "warning" "Weekly Backup Missing" "No weekly backups found in backup directory"
        issues=$((issues + 1))
    fi
    
    # Check monthly backups
    local latest_monthly=$(find "$BACKUP_DIR" -name "waveflix_monthly_*.sql*" -type f -printf '%T@ %p\n' 2>/dev/null | sort -nr | head -1)
    if [ -n "$latest_monthly" ]; then
        local monthly_timestamp=$(echo "$latest_monthly" | cut -d' ' -f1 | cut -d'.' -f1)
        local monthly_age_days=$(( (current_time - monthly_timestamp) / 86400 ))
        
        if [ "$monthly_age_days" -gt "$MONTHLY_BACKUP_MAX_AGE_DAYS" ]; then
            send_alert "warning" "Monthly Backup Overdue" "Latest monthly backup is $monthly_age_days days old" "Expected: max $MONTHLY_BACKUP_MAX_AGE_DAYS days"
            issues=$((issues + 1))
        else
            log SUCCESS "Monthly backup is current ($monthly_age_days days old)"
        fi
    else
        send_alert "info" "Monthly Backup Missing" "No monthly backups found in backup directory" "This may be normal for new installations"
    fi
    
    return $issues
}

# Function to check backup sizes
check_backup_sizes() {
    log INFO "Checking backup sizes..."
    
    local issues=0
    
    # Check recent backups for size anomalies
    local recent_backups=$(find "$BACKUP_DIR" -name "waveflix_*.sql*" -type f -mtime -7)
    
    for backup in $recent_backups; do
        local size_bytes=$(stat -c%s "$backup" 2>/dev/null || echo "0")
        local size_mb=$(( size_bytes / 1024 / 1024 ))
        
        if [ "$size_mb" -lt "$MIN_BACKUP_SIZE_MB" ]; then
            send_alert "critical" "Backup Size Too Small" "Backup $(basename "$backup") is only ${size_mb}MB" "Expected: min ${MIN_BACKUP_SIZE_MB}MB"
            issues=$((issues + 1))
        fi
    done
    
    if [ $issues -eq 0 ]; then
        log SUCCESS "All backup sizes are within expected ranges"
    fi
    
    return $issues
}

# Function to check storage usage
check_storage_usage() {
    log INFO "Checking backup storage usage..."
    
    local backup_dir_usage=$(df "$BACKUP_DIR" | tail -1 | awk '{print $5}' | sed 's/%//')
    
    if [ "$backup_dir_usage" -gt "$STORAGE_USAGE_THRESHOLD" ]; then
        local available_space=$(df -h "$BACKUP_DIR" | tail -1 | awk '{print $4}')
        send_alert "warning" "Backup Storage Usage High" "Backup directory is ${backup_dir_usage}% full" "Available space: $available_space"
        return 1
    else
        log SUCCESS "Storage usage is acceptable (${backup_dir_usage}%)"
        return 0
    fi
}

# Function to verify database connectivity
check_database_connectivity() {
    log INFO "Checking database connectivity for backups..."
    
    export PGPASSWORD="$POSTGRES_PASSWORD"
    
    if ! pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null 2>&1; then
        send_alert "critical" "Database Connectivity Failed" "Cannot connect to PostgreSQL database" "Host: $POSTGRES_HOST:$POSTGRES_PORT, DB: $POSTGRES_DB"
        return 1
    else
        log SUCCESS "Database connectivity verified"
        return 0
    fi
}

# Function to verify backup integrity (sample check)
check_backup_integrity() {
    log INFO "Checking backup integrity..."
    
    local issues=0
    
    # Check checksums for recent backups
    local recent_backups=$(find "$BACKUP_DIR" -name "waveflix_*.sql*.sha256" -type f -mtime -1)
    
    for checksum_file in $recent_backups; do
        local backup_file=$(echo "$checksum_file" | sed 's/.sha256$//')
        
        if [ -f "$backup_file" ]; then
            if ! sha256sum -c "$checksum_file" >/dev/null 2>&1; then
                send_alert "critical" "Backup Integrity Check Failed" "Checksum verification failed for $(basename "$backup_file")" "File may be corrupted"
                issues=$((issues + 1))
            fi
        else
            send_alert "warning" "Backup File Missing" "Backup file missing for checksum $(basename "$checksum_file")" "Orphaned checksum file"
            issues=$((issues + 1))
        fi
    done
    
    if [ $issues -eq 0 ]; then
        log SUCCESS "All backup integrity checks passed"
    fi
    
    return $issues
}

# Function to check cron job status
check_backup_schedule() {
    log INFO "Checking backup schedule..."
    
    local cron_jobs=$(crontab -l 2>/dev/null | grep "waveflix-backup" | wc -l)
    local systemd_timers=0
    
    if command -v systemctl >/dev/null 2>&1; then
        systemd_timers=$(systemctl list-timers --no-pager --no-legend 2>/dev/null | grep waveflix-backup | wc -l)
    fi
    
    if [ "$cron_jobs" -eq 0 ] && [ "$systemd_timers" -eq 0 ]; then
        send_alert "warning" "Backup Schedule Missing" "No backup cron jobs or systemd timers found" "Automated backups may not be configured"
        return 1
    else
        log SUCCESS "Backup schedule configured (cron: $cron_jobs, systemd: $systemd_timers)"
        return 0
    fi
}

# Function to generate health report
generate_health_report() {
    local total_issues=$1
    local report_file="$BACKUP_DIR/health_report_$(date '+%Y%m%d_%H%M%S').json"
    
    local backup_count_daily=$(find "$BACKUP_DIR" -name "waveflix_daily_*.sql*" -type f | wc -l)
    local backup_count_weekly=$(find "$BACKUP_DIR" -name "waveflix_weekly_*.sql*" -type f | wc -l)
    local backup_count_monthly=$(find "$BACKUP_DIR" -name "waveflix_monthly_*.sql*" -type f | wc -l)
    
    local storage_usage=$(df "$BACKUP_DIR" | tail -1 | awk '{print $5}' | sed 's/%//')
    local storage_available=$(df -h "$BACKUP_DIR" | tail -1 | awk '{print $4}')
    
    local latest_backup=$(find "$BACKUP_DIR" -name "waveflix_*.sql*" -type f -printf '%T@ %p\n' 2>/dev/null | sort -nr | head -1 | cut -d' ' -f2-)
    local latest_backup_age=""
    
    if [ -n "$latest_backup" ]; then
        local backup_timestamp=$(stat -c%Y "$latest_backup" 2>/dev/null)
        local current_timestamp=$(date +%s)
        local age_hours=$(( (current_timestamp - backup_timestamp) / 3600 ))
        latest_backup_age="${age_hours}h ago"
    fi
    
    cat > "$report_file" << EOF
{
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "status": "$([ $total_issues -eq 0 ] && echo "healthy" || echo "issues_detected")",
    "total_issues": $total_issues,
    "backup_counts": {
        "daily": $backup_count_daily,
        "weekly": $backup_count_weekly,
        "monthly": $backup_count_monthly
    },
    "storage": {
        "usage_percent": $storage_usage,
        "available": "$storage_available"
    },
    "latest_backup": {
        "file": "$(basename "$latest_backup" 2>/dev/null)",
        "age": "$latest_backup_age"
    },
    "checks_performed": [
        "backup_age",
        "backup_sizes", 
        "storage_usage",
        "database_connectivity",
        "backup_integrity",
        "backup_schedule"
    ]
}
EOF
    
    log INFO "Health report generated: $report_file"
    
    if [ $total_issues -eq 0 ]; then
        send_alert "info" "Backup Health Check Completed" "All checks passed - backup system is healthy" "Report: $report_file"
    else
        send_alert "warning" "Backup Health Check Completed" "$total_issues issues detected in backup system" "Report: $report_file"
    fi
}

# Main function
main() {
    local command=${1:-check}
    
    case $command in
        check)
            log INFO "Starting WaveFlix Hub backup health check..."
            
            local total_issues=0
            
            check_database_connectivity || total_issues=$((total_issues + 1))
            check_backup_age || total_issues=$((total_issues + $?))
            check_backup_sizes || total_issues=$((total_issues + $?))
            check_storage_usage || total_issues=$((total_issues + 1))
            check_backup_integrity || total_issues=$((total_issues + $?))
            check_backup_schedule || total_issues=$((total_issues + 1))
            
            generate_health_report $total_issues
            
            if [ $total_issues -eq 0 ]; then
                log SUCCESS "Health check completed - no issues detected"
                exit 0
            else
                log WARNING "Health check completed - $total_issues issues detected"
                exit 1
            fi
            ;;
            
        monitor)
            # Continuous monitoring mode
            log INFO "Starting continuous backup monitoring..."
            while true; do
                main check
                sleep 3600  # Check every hour
            done
            ;;
            
        *)
            echo "Usage: $0 {check|monitor}"
            echo ""
            echo "Commands:"
            echo "  check   - Run backup health check once"
            echo "  monitor - Run continuous monitoring (hourly checks)"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"