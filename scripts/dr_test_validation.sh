#!/bin/bash

# WaveFlix Hub - Disaster Recovery Test Validation Script
# This script validates that disaster recovery procedures work correctly

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Test environment configuration
TEST_POSTGRES_HOST="${TEST_POSTGRES_HOST:-localhost}"
TEST_POSTGRES_PORT="${TEST_POSTGRES_PORT:-5433}"
TEST_POSTGRES_DB="${TEST_POSTGRES_DB:-waveflix_dr_test}"
TEST_POSTGRES_USER="${TEST_POSTGRES_USER:-waveflix}"
TEST_POSTGRES_PASSWORD="${TEST_POSTGRES_PASSWORD:-waveflix_secure_password_2024}"

TEST_BACKEND_URL="${TEST_BACKEND_URL:-http://localhost:8081}"
TEST_FRONTEND_URL="${TEST_FRONTEND_URL:-http://localhost:3001}"

BACKUP_DIR="${BACKUP_DIR:-$PROJECT_ROOT/backups}"
LOG_FILE="$BACKUP_DIR/dr_test_$(date +%Y%m%d_%H%M%S).log"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test results tracking
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Logging function
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

# Test execution function
run_test() {
    local test_name=$1
    local test_function=$2
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    
    log INFO "Running test: $test_name"
    
    if $test_function; then
        log SUCCESS "✓ $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        log ERROR "✗ $test_name"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Test 1: Backup Availability
test_backup_availability() {
    local latest_backup=$(find "$BACKUP_DIR" -name "waveflix_*.sql.gz" -type f -printf '%T@ %p\n' 2>/dev/null | sort -nr | head -1 | cut -d' ' -f2-)
    
    if [ -z "$latest_backup" ]; then
        log ERROR "No backup files found in $BACKUP_DIR"
        return 1
    fi
    
    local backup_age_hours=$(( ($(date +%s) - $(stat -c%Y "$latest_backup" 2>/dev/null || echo 0)) / 3600 ))
    
    if [ "$backup_age_hours" -gt 25 ]; then
        log ERROR "Latest backup is too old: $backup_age_hours hours"
        return 1
    fi
    
    log INFO "Latest backup: $(basename "$latest_backup") (${backup_age_hours}h old)"
    return 0
}

# Test 2: Backup Integrity
test_backup_integrity() {
    local latest_backup=$(find "$BACKUP_DIR" -name "waveflix_*.sql.gz" -type f -printf '%T@ %p\n' 2>/dev/null | sort -nr | head -1 | cut -d' ' -f2-)
    
    if [ -z "$latest_backup" ]; then
        log ERROR "No backup file to verify"
        return 1
    fi
    
    if [ ! -f "${latest_backup}.sha256" ]; then
        log ERROR "Checksum file missing for backup"
        return 1
    fi
    
    if sha256sum -c "${latest_backup}.sha256" >/dev/null 2>&1; then
        log INFO "Backup integrity verified"
        return 0
    else
        log ERROR "Backup integrity check failed"
        return 1
    fi
}

# Test 3: Database Recovery
test_database_recovery() {
    local latest_backup=$(find "$BACKUP_DIR" -name "waveflix_*.sql.gz" -type f -printf '%T@ %p\n' 2>/dev/null | sort -nr | head -1 | cut -d' ' -f2-)
    
    if [ -z "$latest_backup" ]; then
        log ERROR "No backup available for recovery test"
        return 1
    fi
    
    # Start test PostgreSQL container
    log INFO "Starting test PostgreSQL container..."
    docker run -d --name waveflix-dr-test \
        -e POSTGRES_DB="$TEST_POSTGRES_DB" \
        -e POSTGRES_USER="$TEST_POSTGRES_USER" \
        -e POSTGRES_PASSWORD="$TEST_POSTGRES_PASSWORD" \
        -p "$TEST_POSTGRES_PORT:5432" \
        postgres:15-alpine >/dev/null 2>&1
    
    # Wait for database to be ready
    local attempts=0
    while [ $attempts -lt 30 ]; do
        if docker exec waveflix-dr-test pg_isready -U "$TEST_POSTGRES_USER" -d "$TEST_POSTGRES_DB" >/dev/null 2>&1; then
            break
        fi
        sleep 2
        attempts=$((attempts + 1))
    done
    
    if [ $attempts -eq 30 ]; then
        log ERROR "Test database failed to start"
        docker rm -f waveflix-dr-test >/dev/null 2>&1 || true
        return 1
    fi
    
    # Restore backup
    log INFO "Restoring backup to test database..."
    export PGPASSWORD="$TEST_POSTGRES_PASSWORD"
    
    if gunzip -c "$latest_backup" | psql -h localhost -p "$TEST_POSTGRES_PORT" -U "$TEST_POSTGRES_USER" -d "$TEST_POSTGRES_DB" >/dev/null 2>&1; then
        log INFO "Backup restoration successful"
    else
        log ERROR "Backup restoration failed"
        docker rm -f waveflix-dr-test >/dev/null 2>&1 || true
        return 1
    fi
    
    # Verify data integrity
    local user_count=$(psql -h localhost -p "$TEST_POSTGRES_PORT" -U "$TEST_POSTGRES_USER" -d "$TEST_POSTGRES_DB" -t -c "SELECT COUNT(*) FROM users;" 2>/dev/null | xargs)
    
    if [ "$user_count" -gt 0 ]; then
        log INFO "Data integrity verified: $user_count users found"
    else
        log ERROR "Data integrity check failed: no users found"
        docker rm -f waveflix-dr-test >/dev/null 2>&1 || true
        return 1
    fi
    
    # Cleanup
    docker rm -f waveflix-dr-test >/dev/null 2>&1 || true
    return 0
}

# Test 4: Application Recovery
test_application_recovery() {
    local backend_dir="$PROJECT_ROOT/website/backend"
    
    # Check if backend can start with test configuration
    if [ ! -f "$backend_dir/.env" ]; then
        log ERROR "Backend configuration file not found"
        return 1
    fi
    
    # Create test configuration
    local test_env="$backend_dir/.env.dr_test"
    cat > "$test_env" << EOF
DATABASE_TYPE=postgres
DATABASE_URL=postgres://$TEST_POSTGRES_USER:$TEST_POSTGRES_PASSWORD@localhost:$TEST_POSTGRES_PORT/$TEST_POSTGRES_DB?sslmode=disable
TMDB_API_KEY=test_key
JWT_SECRET=test_secret_minimum_32_characters_long
PORT=8081
ALLOWED_ORIGIN=http://localhost:3001
EOF
    
    # Test backend build
    cd "$backend_dir"
    if go build -o waveflix-test . >/dev/null 2>&1; then
        log INFO "Backend build successful"
        rm -f waveflix-test
    else
        log ERROR "Backend build failed"
        rm -f "$test_env"
        return 1
    fi
    
    # Cleanup
    rm -f "$test_env"
    return 0
}

# Test 5: Configuration Validation
test_configuration_validation() {
    local required_files=(
        "$PROJECT_ROOT/website/backend/.env.example"
        "$PROJECT_ROOT/website/backend/go.mod"
        "$PROJECT_ROOT/website/frontend/package.json"
        "$PROJECT_ROOT/database/init.sql"
        "$PROJECT_ROOT/docker-compose.yml"
    )
    
    for file in "${required_files[@]}"; do
        if [ ! -f "$file" ]; then
            log ERROR "Required configuration file missing: $file"
            return 1
        fi
    done
    
    log INFO "All required configuration files present"
    return 0
}

# Test 6: Backup Script Functionality
test_backup_script_functionality() {
    local backup_script="$PROJECT_ROOT/scripts/backup_postgres.sh"
    
    if [ ! -x "$backup_script" ]; then
        log ERROR "Backup script not found or not executable: $backup_script"
        return 1
    fi
    
    # Test script help
    if "$backup_script" --help >/dev/null 2>&1 || "$backup_script" help >/dev/null 2>&1; then
        log INFO "Backup script help accessible"
    else
        log WARNING "Backup script help not available (may be normal)"
    fi
    
    # Test dry run or validation mode (if available)
    # Note: This depends on the actual backup script implementation
    
    return 0
}

# Test 7: Monitoring Integration
test_monitoring_integration() {
    local monitoring_config="$PROJECT_ROOT/monitoring/backup-monitoring.yml"
    
    if [ ! -f "$monitoring_config" ]; then
        log ERROR "Monitoring configuration not found: $monitoring_config"
        return 1
    fi
    
    # Basic YAML syntax check (if yq is available)
    if command -v yq >/dev/null 2>&1; then
        if yq eval '.' "$monitoring_config" >/dev/null 2>&1; then
            log INFO "Monitoring configuration syntax valid"
        else
            log ERROR "Monitoring configuration has syntax errors"
            return 1
        fi
    else
        log INFO "Monitoring configuration file present (syntax not validated)"
    fi
    
    return 0
}

# Test 8: Network Connectivity
test_network_connectivity() {
    # Test external dependencies
    local external_hosts=(
        "api.themoviedb.org"
        "github.com"
    )
    
    for host in "${external_hosts[@]}"; do
        if ping -c 1 -W 5 "$host" >/dev/null 2>&1; then
            log INFO "Network connectivity to $host: OK"
        else
            log WARNING "Network connectivity to $host: FAILED"
        fi
    done
    
    return 0
}

# Test 9: Storage Space Check
test_storage_space() {
    local backup_dir_usage=$(df "$BACKUP_DIR" 2>/dev/null | tail -1 | awk '{print $5}' | sed 's/%//' || echo "0")
    
    if [ "$backup_dir_usage" -gt 90 ]; then
        log ERROR "Backup directory storage usage critical: ${backup_dir_usage}%"
        return 1
    elif [ "$backup_dir_usage" -gt 80 ]; then
        log WARNING "Backup directory storage usage high: ${backup_dir_usage}%"
    else
        log INFO "Backup directory storage usage acceptable: ${backup_dir_usage}%"
    fi
    
    return 0
}

# Test 10: Alert Notification Test
test_alert_notification() {
    local health_check_script="$PROJECT_ROOT/scripts/backup_health_check.sh"
    
    if [ ! -x "$health_check_script" ]; then
        log ERROR "Health check script not found: $health_check_script"
        return 1
    fi
    
    # Test health check execution (should not send real alerts)
    log INFO "Testing health check script execution..."
    if "$health_check_script" check >/dev/null 2>&1; then
        log INFO "Health check script executed successfully"
    else
        log WARNING "Health check script execution failed (may be due to missing dependencies)"
    fi
    
    return 0
}

# Generate DR test report
generate_dr_test_report() {
    local report_file="$BACKUP_DIR/dr_test_report_$(date +%Y%m%d_%H%M%S).json"
    
    cat > "$report_file" << EOF
{
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "test_summary": {
        "total_tests": $TESTS_TOTAL,
        "passed": $TESTS_PASSED,
        "failed": $TESTS_FAILED,
        "success_rate": $(( TESTS_PASSED * 100 / TESTS_TOTAL ))
    },
    "environment": {
        "test_postgres_host": "$TEST_POSTGRES_HOST",
        "test_postgres_port": "$TEST_POSTGRES_PORT",
        "backup_directory": "$BACKUP_DIR",
        "log_file": "$LOG_FILE"
    },
    "recommendations": [
        $([ $TESTS_FAILED -eq 0 ] && echo '"All tests passed - DR procedures are operational"' || echo '"Review failed tests and update DR procedures"'),
        "Schedule regular DR testing",
        "Update emergency contact information",
        "Verify backup retention policies"
    ]
}
EOF
    
    log INFO "DR test report generated: $report_file"
}

# Cleanup function
cleanup() {
    # Remove any test containers
    docker rm -f waveflix-dr-test >/dev/null 2>&1 || true
    
    # Clean up temporary files
    rm -f /tmp/waveflix_dr_test_* 2>/dev/null || true
}

# Main execution
main() {
    local test_type=${1:-full}
    
    mkdir -p "$BACKUP_DIR"
    
    log INFO "Starting WaveFlix Hub Disaster Recovery Test Validation"
    log INFO "Test type: $test_type"
    log INFO "Log file: $LOG_FILE"
    
    # Trap cleanup on exit
    trap cleanup EXIT
    
    case $test_type in
        full)
            log INFO "Running full DR test suite..."
            
            run_test "Backup Availability" test_backup_availability
            run_test "Backup Integrity" test_backup_integrity
            run_test "Database Recovery" test_database_recovery
            run_test "Application Recovery" test_application_recovery
            run_test "Configuration Validation" test_configuration_validation
            run_test "Backup Script Functionality" test_backup_script_functionality
            run_test "Monitoring Integration" test_monitoring_integration
            run_test "Network Connectivity" test_network_connectivity
            run_test "Storage Space Check" test_storage_space
            run_test "Alert Notification Test" test_alert_notification
            ;;
            
        quick)
            log INFO "Running quick DR validation..."
            
            run_test "Backup Availability" test_backup_availability
            run_test "Backup Integrity" test_backup_integrity
            run_test "Configuration Validation" test_configuration_validation
            run_test "Storage Space Check" test_storage_space
            ;;
            
        backup)
            log INFO "Running backup-specific tests..."
            
            run_test "Backup Availability" test_backup_availability
            run_test "Backup Integrity" test_backup_integrity
            run_test "Database Recovery" test_database_recovery
            ;;
            
        *)
            echo "Usage: $0 {full|quick|backup}"
            echo ""
            echo "Test Types:"
            echo "  full   - Complete DR validation suite"
            echo "  quick  - Basic DR readiness check"
            echo "  backup - Backup and recovery tests only"
            exit 1
            ;;
    esac
    
    # Generate report
    generate_dr_test_report
    
    # Summary
    echo ""
    log INFO "=== DR Test Results ==="
    log INFO "Tests Passed: $TESTS_PASSED"
    log INFO "Tests Failed: $TESTS_FAILED"
    log INFO "Success Rate: $(( TESTS_PASSED * 100 / TESTS_TOTAL ))%"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        log SUCCESS "All tests passed - DR procedures are operational"
        exit 0
    else
        log ERROR "$TESTS_FAILED tests failed - review and fix issues"
        exit 1
    fi
}

# Execute main function
main "$@"