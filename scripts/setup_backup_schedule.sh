#!/bin/bash

# WaveFlix Hub - Backup Schedule Setup Script
# This script sets up automated backup scheduling using cron

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Function to setup cron jobs
setup_cron_jobs() {
    print_info "Setting up cron jobs for automated backups..."
    
    local backup_script="$SCRIPT_DIR/backup_postgres.sh"
    local log_dir="$PROJECT_ROOT/backups"
    
    # Make scripts executable
    chmod +x "$backup_script"
    
    # Create backup directory if it doesn't exist
    mkdir -p "$log_dir"
    
    # Create temporary cron file
    local temp_cron=$(mktemp)
    
    # Get existing cron jobs (excluding our backup jobs)
    crontab -l 2>/dev/null | grep -v "waveflix-backup" > "$temp_cron" || true
    
    # Add our backup jobs with unique identifiers
    cat >> "$temp_cron" << EOF

# WaveFlix Hub Database Backup Jobs - waveflix-backup
# Daily backup at 2:00 AM
0 2 * * * cd "$PROJECT_ROOT" && "$backup_script" backup daily >> "$log_dir/cron.log" 2>&1 # waveflix-backup

# Weekly backup every Sunday at 3:00 AM  
0 3 * * 0 cd "$PROJECT_ROOT" && "$backup_script" backup weekly >> "$log_dir/cron.log" 2>&1 # waveflix-backup

# Monthly backup on the 1st day of each month at 4:00 AM
0 4 1 * * cd "$PROJECT_ROOT" && "$backup_script" backup monthly >> "$log_dir/cron.log" 2>&1 # waveflix-backup

# Cleanup old backups daily at 5:00 AM
0 5 * * * cd "$PROJECT_ROOT" && "$backup_script" cleanup >> "$log_dir/cron.log" 2>&1 # waveflix-backup

EOF
    
    # Install the new cron jobs
    crontab "$temp_cron"
    rm "$temp_cron"
    
    print_success "Cron jobs installed successfully"
    
    # Display installed jobs
    print_info "Installed backup schedule:"
    crontab -l | grep "waveflix-backup" | while read -r line; do
        echo "  $line"
    done
}

# Function to setup systemd timers (alternative to cron)
setup_systemd_timers() {
    print_info "Setting up systemd timers for automated backups..."
    
    local backup_script="$SCRIPT_DIR/backup_postgres.sh"
    local service_dir="/etc/systemd/system"
    
    # Check if we have sudo access
    if ! sudo -n true 2>/dev/null; then
        print_error "sudo access required for systemd timer setup"
        return 1
    fi
    
    # Create backup service
    sudo tee "$service_dir/waveflix-backup@.service" > /dev/null << EOF
[Unit]
Description=WaveFlix Hub Database Backup (%i)
After=network.target postgresql.service

[Service]
Type=oneshot
User=$USER
WorkingDirectory=$PROJECT_ROOT
Environment=BACKUP_DIR=$PROJECT_ROOT/backups
ExecStart=$backup_script backup %i
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

    # Create daily backup timer
    sudo tee "$service_dir/waveflix-backup-daily.timer" > /dev/null << EOF
[Unit]
Description=WaveFlix Hub Daily Database Backup
Requires=waveflix-backup@daily.service

[Timer]
OnCalendar=daily
Persistent=true
RandomizedDelaySec=300

[Install]
WantedBy=timers.target
EOF

    # Create weekly backup timer
    sudo tee "$service_dir/waveflix-backup-weekly.timer" > /dev/null << EOF
[Unit]
Description=WaveFlix Hub Weekly Database Backup
Requires=waveflix-backup@weekly.service

[Timer]
OnCalendar=weekly
Persistent=true
RandomizedDelaySec=600

[Install]
WantedBy=timers.target
EOF

    # Create monthly backup timer
    sudo tee "$service_dir/waveflix-backup-monthly.timer" > /dev/null << EOF
[Unit]
Description=WaveFlix Hub Monthly Database Backup
Requires=waveflix-backup@monthly.service

[Timer]
OnCalendar=monthly
Persistent=true
RandomizedDelaySec=900

[Install]
WantedBy=timers.target
EOF

    # Create cleanup service
    sudo tee "$service_dir/waveflix-backup-cleanup.service" > /dev/null << EOF
[Unit]
Description=WaveFlix Hub Database Backup Cleanup
After=network.target

[Service]
Type=oneshot
User=$USER
WorkingDirectory=$PROJECT_ROOT
ExecStart=$backup_script cleanup
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

    # Create cleanup timer
    sudo tee "$service_dir/waveflix-backup-cleanup.timer" > /dev/null << EOF
[Unit]
Description=WaveFlix Hub Database Backup Cleanup
Requires=waveflix-backup-cleanup.service

[Timer]
OnCalendar=daily
Persistent=true
RandomizedDelaySec=1800

[Install]
WantedBy=timers.target
EOF

    # Reload systemd and enable timers
    sudo systemctl daemon-reload
    sudo systemctl enable waveflix-backup-daily.timer
    sudo systemctl enable waveflix-backup-weekly.timer  
    sudo systemctl enable waveflix-backup-monthly.timer
    sudo systemctl enable waveflix-backup-cleanup.timer
    
    # Start timers
    sudo systemctl start waveflix-backup-daily.timer
    sudo systemctl start waveflix-backup-weekly.timer
    sudo systemctl start waveflix-backup-monthly.timer
    sudo systemctl start waveflix-backup-cleanup.timer
    
    print_success "Systemd timers setup successfully"
    
    # Display timer status
    print_info "Timer status:"
    sudo systemctl list-timers --no-pager | grep waveflix-backup
}

# Function to create backup environment configuration
create_backup_env() {
    print_info "Creating backup environment configuration..."
    
    local env_file="$PROJECT_ROOT/.env.backup"
    
    cat > "$env_file" << EOF
# WaveFlix Hub - Backup Configuration
# Copy this to your environment or source before running backup scripts

# Database Configuration
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_DB=waveflix
export POSTGRES_USER=waveflix
# export POSTGRES_PASSWORD=replace-with-a-strong-secret

# Backup Configuration
export BACKUP_DIR=$PROJECT_ROOT/backups
export BACKUP_RETENTION_DAYS=30
export BACKUP_RETENTION_WEEKS=12
export BACKUP_RETENTION_MONTHS=12

# Cloud Storage (Optional)
# export S3_BUCKET=your-backup-bucket
# export S3_PREFIX=waveflix-backups
# export AWS_REGION=us-east-1

# Notifications (Optional)
# export SLACK_WEBHOOK=https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK
# export WEBHOOK_URL=https://your-monitoring-webhook.com/endpoint

# Monitoring (Optional)
# export EMAIL_TO=admin@yourdomain.com
EOF

    print_success "Backup environment configuration created: $env_file"
    print_info "Edit $env_file with your specific configuration"
}

# Function to test backup setup
test_backup_setup() {
    print_info "Testing backup setup..."
    
    local backup_script="$SCRIPT_DIR/backup_postgres.sh"
    
    # Source environment if available
    if [ -f "$PROJECT_ROOT/.env.backup" ]; then
        source "$PROJECT_ROOT/.env.backup"
    fi
    
    # Test PostgreSQL connection
    print_info "Testing PostgreSQL connection..."
    if command -v pg_isready &> /dev/null; then
        if pg_isready -h "${POSTGRES_HOST:-localhost}" -p "${POSTGRES_PORT:-5432}" -U "${POSTGRES_USER:-waveflix}" >/dev/null 2>&1; then
            print_success "PostgreSQL connection test passed"
        else
            print_warning "PostgreSQL connection test failed - backup jobs may fail"
        fi
    else
        print_warning "pg_isready not found - install PostgreSQL client tools"
    fi
    
    # Test backup script
    print_info "Testing backup script permissions..."
    if [ -x "$backup_script" ]; then
        print_success "Backup script is executable"
    else
        print_error "Backup script is not executable"
        return 1
    fi
    
    # Test backup directory
    local backup_dir="${BACKUP_DIR:-$PROJECT_ROOT/backups}"
    if [ -w "$backup_dir" ]; then
        print_success "Backup directory is writable: $backup_dir"
    else
        print_warning "Backup directory is not writable: $backup_dir"
    fi
    
    print_success "Backup setup test completed"
}

# Function to remove backup schedule
remove_backup_schedule() {
    print_info "Removing backup schedule..."
    
    # Remove from cron
    local temp_cron=$(mktemp)
    crontab -l 2>/dev/null | grep -v "waveflix-backup" > "$temp_cron" || true
    crontab "$temp_cron"
    rm "$temp_cron"
    
    # Remove systemd timers if they exist
    if command -v systemctl &> /dev/null; then
        for timer in waveflix-backup-daily waveflix-backup-weekly waveflix-backup-monthly waveflix-backup-cleanup; do
            sudo systemctl stop "${timer}.timer" 2>/dev/null || true
            sudo systemctl disable "${timer}.timer" 2>/dev/null || true
        done
        
        # Remove systemd files
        sudo rm -f /etc/systemd/system/waveflix-backup* 2>/dev/null || true
        sudo systemctl daemon-reload 2>/dev/null || true
    fi
    
    print_success "Backup schedule removed"
}

# Function to show backup status
show_backup_status() {
    print_info "Backup Schedule Status:"
    
    # Check cron jobs
    local cron_jobs=$(crontab -l 2>/dev/null | grep "waveflix-backup" | wc -l)
    if [ "$cron_jobs" -gt 0 ]; then
        print_success "Cron jobs: $cron_jobs active"
        crontab -l | grep "waveflix-backup" | sed 's/^/  /'
    else
        print_warning "Cron jobs: none found"
    fi
    
    # Check systemd timers
    if command -v systemctl &> /dev/null; then
        local timer_count=$(systemctl list-timers --no-pager --no-legend | grep waveflix-backup | wc -l)
        if [ "$timer_count" -gt 0 ]; then
            print_success "Systemd timers: $timer_count active"
            systemctl list-timers --no-pager | grep waveflix-backup | sed 's/^/  /'
        else
            print_warning "Systemd timers: none found"
        fi
    fi
    
    # Check recent backups
    local backup_dir="${BACKUP_DIR:-$PROJECT_ROOT/backups}"
    if [ -d "$backup_dir" ]; then
        local recent_backups=$(find "$backup_dir" -name "waveflix_*.sql*" -mtime -7 | wc -l)
        print_info "Recent backups (last 7 days): $recent_backups files"
        
        if [ "$recent_backups" -gt 0 ]; then
            print_info "Latest backups:"
            find "$backup_dir" -name "waveflix_*.sql*" -mtime -7 -printf "  %TY-%Tm-%Td %TH:%TM %s bytes %f\n" | sort -r | head -5
        fi
    else
        print_warning "Backup directory not found: $backup_dir"
    fi
}

# Main function
main() {
    local command=${1:-setup}
    
    case $command in
        setup)
            print_info "Setting up WaveFlix Hub automated backup system..."
            create_backup_env
            
            if command -v systemctl &> /dev/null && [ -d "/etc/systemd/system" ]; then
                read -p "Use systemd timers instead of cron? (recommended) [y/N]: " -n 1 -r
                echo
                if [[ $REPLY =~ ^[Yy]$ ]]; then
                    setup_systemd_timers
                else
                    setup_cron_jobs
                fi
            else
                setup_cron_jobs
            fi
            
            test_backup_setup
            print_success "Backup system setup completed!"
            ;;
            
        remove)
            remove_backup_schedule
            ;;
            
        status)
            show_backup_status
            ;;
            
        test)
            test_backup_setup
            ;;
            
        *)
            echo "Usage: $0 {setup|remove|status|test}"
            echo ""
            echo "Commands:"
            echo "  setup   - Setup automated backup schedule"
            echo "  remove  - Remove backup schedule"
            echo "  status  - Show backup schedule status"
            echo "  test    - Test backup configuration"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
