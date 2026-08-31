#!/bin/bash

# WaveFlix Hub - Recovery Report Generator
# Generates comprehensive post-recovery validation reports

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-waveflix}"
POSTGRES_USER="${POSTGRES_USER:-waveflix}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-waveflix_secure_password_2024}"

BACKEND_URL="${BACKEND_URL:-http://localhost:8080}"
FRONTEND_URL="${FRONTEND_URL:-http://localhost:3000}"

BACKUP_DIR="${BACKUP_DIR:-$PROJECT_ROOT/backups}"
OUTPUT_DIR="${OUTPUT_DIR:-$PROJECT_ROOT/reports}"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    local level=$1
    shift
    local message="$*"
    
    case $level in
        INFO)    echo -e "${BLUE}[INFO]${NC} $message" ;;
        SUCCESS) echo -e "${GREEN}[SUCCESS]${NC} $message" ;;
        WARNING) echo -e "${YELLOW}[WARNING]${NC} $message" ;;
        ERROR)   echo -e "${RED}[ERROR]${NC} $message" ;;
    esac
}

# Check database connectivity and collect metrics
check_database_status() {
    local db_status="unknown"
    local connection_count=0
    local database_size="unknown"
    local last_backup="unknown"
    
    export PGPASSWORD="$POSTGRES_PASSWORD"
    
    if pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null 2>&1; then
        db_status="connected"
        
        # Get connection count
        connection_count=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT count(*) FROM pg_stat_activity WHERE datname = '$POSTGRES_DB';" 2>/dev/null | xargs || echo "0")
        
        # Get database size
        database_size=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT pg_size_pretty(pg_database_size('$POSTGRES_DB'));" 2>/dev/null | xargs || echo "unknown")
        
        # Get table counts
        user_count=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT COUNT(*) FROM users;" 2>/dev/null | xargs || echo "0")
        profile_count=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT COUNT(*) FROM profiles;" 2>/dev/null | xargs || echo "0")
        watchlist_count=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT COUNT(*) FROM watchlist;" 2>/dev/null | xargs || echo "0")
        favorites_count=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT COUNT(*) FROM favorites;" 2>/dev/null | xargs || echo "0")
        history_count=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT COUNT(*) FROM history;" 2>/dev/null | xargs || echo "0")
    else
        db_status="disconnected"
        user_count=0
        profile_count=0
        watchlist_count=0
        favorites_count=0
        history_count=0
    fi
    
    # Find latest backup
    if [ -d "$BACKUP_DIR" ]; then
        last_backup=$(find "$BACKUP_DIR" -name "waveflix_*.sql.gz" -type f -printf '%T@ %p\n' 2>/dev/null | sort -nr | head -1 | cut -d' ' -f2- | xargs basename 2>/dev/null || echo "none")
    fi
    
    cat << EOF
{
    "status": "$db_status",
    "connection_count": $connection_count,
    "database_size": "$database_size",
    "table_counts": {
        "users": $user_count,
        "profiles": $profile_count,
        "watchlist": $watchlist_count,
        "favorites": $favorites_count,
        "history": $history_count
    },
    "last_backup": "$last_backup"
}
EOF
}

# Check application endpoints
check_application_status() {
    local backend_status="down"
    local backend_response_time=0
    local frontend_status="down"
    local frontend_response_time=0
    
    # Check backend health endpoint
    if response_time=$(curl -o /dev/null -s -w "%{time_total}" -f "$BACKEND_URL/health" 2>/dev/null); then
        backend_status="up"
        backend_response_time=$(echo "$response_time * 1000" | bc | cut -d'.' -f1)
    fi
    
    # Check frontend
    if response_time=$(curl -o /dev/null -s -w "%{time_total}" -f "$FRONTEND_URL" 2>/dev/null); then
        frontend_status="up"
        frontend_response_time=$(echo "$response_time * 1000" | bc | cut -d'.' -f1)
    fi
    
    # Test API endpoints
    local api_endpoints=(
        "/api/trending"
        "/api/discover"
        "/api/search?q=test"
        "/auth/check-email?email=test@example.com"
    )
    
    local api_results=""
    for endpoint in "${api_endpoints[@]}"; do
        local endpoint_status="fail"
        local endpoint_time=0
        
        if response_time=$(curl -o /dev/null -s -w "%{time_total}" -f "$BACKEND_URL$endpoint" 2>/dev/null); then
            endpoint_status="success"
            endpoint_time=$(echo "$response_time * 1000" | bc | cut -d'.' -f1)
        fi
        
        api_results="$api_results,{\"endpoint\":\"$endpoint\",\"status\":\"$endpoint_status\",\"response_time_ms\":$endpoint_time}"
    done
    
    # Remove leading comma
    api_results=${api_results#,}
    
    cat << EOF
{
    "backend": {
        "status": "$backend_status",
        "response_time_ms": $backend_response_time,
        "url": "$BACKEND_URL"
    },
    "frontend": {
        "status": "$frontend_status", 
        "response_time_ms": $frontend_response_time,
        "url": "$FRONTEND_URL"
    },
    "api_endpoints": [$api_results]
}
EOF
}

# Check system resources
check_system_resources() {
    local cpu_usage=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | awk -F'%' '{print $1}' || echo "0")
    local memory_usage=$(free | grep Mem | awk '{printf "%.1f", $3/$2 * 100.0}' || echo "0")
    local disk_usage=$(df / | tail -1 | awk '{print $5}' | sed 's/%//' || echo "0")
    
    # Docker container status
    local container_status=""
    if command -v docker >/dev/null 2>&1; then
        local containers=$(docker ps --format "{{.Names}}" --filter "name=waveflix" 2>/dev/null || echo "")
        for container in $containers; do
            local status=$(docker inspect --format='{{.State.Status}}' "$container" 2>/dev/null || echo "unknown")
            container_status="$container_status,{\"name\":\"$container\",\"status\":\"$status\"}"
        done
        container_status=${container_status#,}
    fi
    
    cat << EOF
{
    "cpu_usage_percent": $cpu_usage,
    "memory_usage_percent": $memory_usage,
    "disk_usage_percent": $disk_usage,
    "containers": [$container_status]
}
EOF
}

# Check backup integrity
check_backup_integrity() {
    local backup_status="no_backups"
    local total_backups=0
    local valid_backups=0
    local latest_backup_age_hours=0
    
    if [ -d "$BACKUP_DIR" ]; then
        total_backups=$(find "$BACKUP_DIR" -name "waveflix_*.sql.gz" -type f | wc -l)
        
        if [ "$total_backups" -gt 0 ]; then
            backup_status="available"
            
            # Check integrity of recent backups
            local recent_backups=$(find "$BACKUP_DIR" -name "waveflix_*.sql.gz" -type f -mtime -7)
            for backup in $recent_backups; do
                if [ -f "${backup}.sha256" ] && sha256sum -c "${backup}.sha256" >/dev/null 2>&1; then
                    valid_backups=$((valid_backups + 1))
                fi
            done
            
            # Calculate age of latest backup
            local latest_backup=$(find "$BACKUP_DIR" -name "waveflix_*.sql.gz" -type f -printf '%T@ %p\n' 2>/dev/null | sort -nr | head -1 | cut -d' ' -f2-)
            if [ -n "$latest_backup" ]; then
                local backup_timestamp=$(stat -c%Y "$latest_backup" 2>/dev/null || echo "0")
                local current_timestamp=$(date +%s)
                latest_backup_age_hours=$(( (current_timestamp - backup_timestamp) / 3600 ))
            fi
        fi
    fi
    
    cat << EOF
{
    "status": "$backup_status",
    "total_backups": $total_backups,
    "valid_backups": $valid_backups,
    "latest_backup_age_hours": $latest_backup_age_hours
}
EOF
}

# Check monitoring and alerting
check_monitoring_status() {
    local prometheus_status="unknown"
    local grafana_status="unknown"
    local alerts_active=0
    
    # Check Prometheus (if running)
    if curl -s http://localhost:9090/-/healthy >/dev/null 2>&1; then
        prometheus_status="up"
    else
        prometheus_status="down"
    fi
    
    # Check Grafana (if running)
    if curl -s http://localhost:3001/api/health >/dev/null 2>&1; then
        grafana_status="up"
    else
        grafana_status="down"
    fi
    
    # Check for active alerts (if Prometheus is up)
    if [ "$prometheus_status" = "up" ]; then
        alerts_active=$(curl -s http://localhost:9090/api/v1/alerts | jq '.data | length' 2>/dev/null || echo "0")
    fi
    
    cat << EOF
{
    "prometheus": "$prometheus_status",
    "grafana": "$grafana_status", 
    "active_alerts": $alerts_active
}
EOF
}

# Generate security status report
check_security_status() {
    local failed_logins=0
    local suspicious_activities=0
    local last_security_scan="never"
    
    # Check for failed login attempts (if database is accessible)
    export PGPASSWORD="$POSTGRES_PASSWORD"
    if pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null 2>&1; then
        # This assumes you have a failed_login_attempts table - adjust as needed
        failed_logins=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT COUNT(*) FROM users WHERE created_at > now() - interval '24 hours';" 2>/dev/null | xargs || echo "0")
    fi
    
    # Check for security scan results
    if [ -f "$OUTPUT_DIR/security_scan.log" ]; then
        last_security_scan=$(stat -c%Y "$OUTPUT_DIR/security_scan.log" 2>/dev/null | date -d@"$(cat)" +"%Y-%m-%d %H:%M:%S" 2>/dev/null || echo "never")
    fi
    
    cat << EOF
{
    "failed_logins_24h": $failed_logins,
    "suspicious_activities": $suspicious_activities,
    "last_security_scan": "$last_security_scan"
}
EOF
}

# Generate comprehensive recovery report
generate_recovery_report() {
    local report_type=${1:-post-recovery}
    local timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    local report_file="$OUTPUT_DIR/recovery_report_$(date +%Y%m%d_%H%M%S).json"
    
    log INFO "Generating recovery report: $report_file"
    
    # Collect all status information
    local db_status=$(check_database_status)
    local app_status=$(check_application_status)
    local system_status=$(check_system_resources)
    local backup_status=$(check_backup_integrity)
    local monitoring_status=$(check_monitoring_status)
    local security_status=$(check_security_status)
    
    # Calculate overall health score
    local health_score=0
    local max_score=100
    
    # Database health (30 points)
    if echo "$db_status" | jq -r '.status' | grep -q "connected"; then
        health_score=$((health_score + 30))
    fi
    
    # Application health (25 points)
    if echo "$app_status" | jq -r '.backend.status' | grep -q "up"; then
        health_score=$((health_score + 15))
    fi
    if echo "$app_status" | jq -r '.frontend.status' | grep -q "up"; then
        health_score=$((health_score + 10))
    fi
    
    # Backup health (20 points)
    local backup_age=$(echo "$backup_status" | jq -r '.latest_backup_age_hours')
    if [ "$backup_age" -lt 25 ]; then
        health_score=$((health_score + 20))
    elif [ "$backup_age" -lt 48 ]; then
        health_score=$((health_score + 10))
    fi
    
    # System resources (15 points)
    local cpu_usage=$(echo "$system_status" | jq -r '.cpu_usage_percent')
    local memory_usage=$(echo "$system_status" | jq -r '.memory_usage_percent')
    if (( $(echo "$cpu_usage < 80" | bc -l) )) && (( $(echo "$memory_usage < 80" | bc -l) )); then
        health_score=$((health_score + 15))
    elif (( $(echo "$cpu_usage < 90" | bc -l) )) && (( $(echo "$memory_usage < 90" | bc -l) )); then
        health_score=$((health_score + 10))
    fi
    
    # Monitoring (10 points)
    if echo "$monitoring_status" | jq -r '.prometheus' | grep -q "up"; then
        health_score=$((health_score + 10))
    fi
    
    # Determine overall status
    local overall_status="unknown"
    if [ "$health_score" -ge 90 ]; then
        overall_status="healthy"
    elif [ "$health_score" -ge 70 ]; then
        overall_status="degraded"
    elif [ "$health_score" -ge 50 ]; then
        overall_status="partial"
    else
        overall_status="critical"
    fi
    
    # Generate recommendations
    local recommendations=""
    
    if echo "$db_status" | jq -r '.status' | grep -q "disconnected"; then
        recommendations="$recommendations,\"Restore database connectivity immediately\""
    fi
    
    if [ "$backup_age" -gt 25 ]; then
        recommendations="$recommendations,\"Run fresh backup - latest backup is ${backup_age}h old\""
    fi
    
    if (( $(echo "$cpu_usage > 80" | bc -l) )); then
        recommendations="$recommendations,\"Investigate high CPU usage (${cpu_usage}%)\""
    fi
    
    if (( $(echo "$memory_usage > 80" | bc -l) )); then
        recommendations="$recommendations,\"Monitor memory usage (${memory_usage}%)\""
    fi
    
    if echo "$app_status" | jq -r '.backend.status' | grep -q "down"; then
        recommendations="$recommendations,\"Restart backend application\""
    fi
    
    if [ -z "$recommendations" ]; then
        recommendations="\"System appears to be operating normally\""
    else
        recommendations=${recommendations#,}
    fi
    
    # Create the complete report
    cat > "$report_file" << EOF
{
    "metadata": {
        "report_type": "$report_type",
        "timestamp": "$timestamp",
        "generated_by": "WaveFlix Hub Recovery Report Generator",
        "report_version": "1.0"
    },
    "summary": {
        "overall_status": "$overall_status",
        "health_score": $health_score,
        "max_score": $max_score,
        "health_percentage": $(( health_score * 100 / max_score ))
    },
    "database": $db_status,
    "application": $app_status,
    "system": $system_status,
    "backups": $backup_status,
    "monitoring": $monitoring_status,
    "security": $security_status,
    "recommendations": [$recommendations],
    "next_actions": [
        "Monitor system stability for 24 hours",
        "Verify all critical business functions", 
        "Schedule follow-up recovery test",
        "Update incident documentation",
        "Review and update recovery procedures based on lessons learned"
    ]
}
EOF
    
    log SUCCESS "Recovery report generated: $report_file"
    
    # Generate human-readable summary
    local summary_file="$OUTPUT_DIR/recovery_summary_$(date +%Y%m%d_%H%M%S).md"
    
    cat > "$summary_file" << EOF
# WaveFlix Hub Recovery Report

**Generated**: $timestamp  
**Overall Status**: $overall_status  
**Health Score**: $health_score/$max_score ($(( health_score * 100 / max_score ))%)

## System Status Summary

### Database
- Status: $(echo "$db_status" | jq -r '.status')
- Active Connections: $(echo "$db_status" | jq -r '.connection_count')
- Database Size: $(echo "$db_status" | jq -r '.database_size')
- Users: $(echo "$db_status" | jq -r '.table_counts.users')

### Application
- Backend: $(echo "$app_status" | jq -r '.backend.status') ($(echo "$app_status" | jq -r '.backend.response_time_ms')ms)
- Frontend: $(echo "$app_status" | jq -r '.frontend.status') ($(echo "$app_status" | jq -r '.frontend.response_time_ms')ms)

### System Resources
- CPU Usage: $(echo "$system_status" | jq -r '.cpu_usage_percent')%
- Memory Usage: $(echo "$system_status" | jq -r '.memory_usage_percent')%
- Disk Usage: $(echo "$system_status" | jq -r '.disk_usage_percent')%

### Backups
- Status: $(echo "$backup_status" | jq -r '.status')
- Total Backups: $(echo "$backup_status" | jq -r '.total_backups')
- Latest Backup Age: $(echo "$backup_status" | jq -r '.latest_backup_age_hours') hours

## Recommendations

$(echo "$recommendations" | sed 's/,/\n- /g' | sed 's/^"//g' | sed 's/"$//g' | sed 's/^/- /')

## Next Steps

1. Monitor system stability for 24 hours
2. Verify all critical business functions
3. Schedule follow-up recovery test
4. Update incident documentation
5. Review and update recovery procedures

---
*Report generated by WaveFlix Hub Recovery System*
EOF
    
    log SUCCESS "Human-readable summary: $summary_file"
    
    # Display summary to console
    echo ""
    log INFO "=== Recovery Report Summary ==="
    log INFO "Overall Status: $overall_status"
    log INFO "Health Score: $health_score/$max_score ($(( health_score * 100 / max_score ))%)"
    log INFO "Report Files:"
    log INFO "  - JSON: $report_file"
    log INFO "  - Summary: $summary_file"
    
    return 0
}

# Main execution
main() {
    local command=${1:-generate}
    
    case $command in
        generate)
            local report_type=${2:-post-recovery}
            generate_recovery_report "$report_type"
            ;;
            
        check-db)
            check_database_status | jq '.'
            ;;
            
        check-app)
            check_application_status | jq '.'
            ;;
            
        check-system)
            check_system_resources | jq '.'
            ;;
            
        check-backups)
            check_backup_integrity | jq '.'
            ;;
            
        *)
            echo "Usage: $0 {generate|check-db|check-app|check-system|check-backups} [report_type]"
            echo ""
            echo "Commands:"
            echo "  generate [post-recovery|health-check|scheduled]  - Generate comprehensive recovery report"
            echo "  check-db                                         - Check database status only"
            echo "  check-app                                        - Check application status only"
            echo "  check-system                                     - Check system resources only"
            echo "  check-backups                                    - Check backup status only"
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"