# WaveFlix Hub - PostgreSQL Automated Backup Script (PowerShell)
# This script performs comprehensive database backups with retention management

param(
    [string]$Command = "backup",
    [string]$BackupType = "daily",
    [string]$BackupFile = "",
    [string]$TargetDB = ""
)

# Configuration from environment variables with defaults
$PostgresHost = $env:POSTGRES_HOST ?? "localhost"
$PostgresPort = $env:POSTGRES_PORT ?? "5432"
$PostgresDB = $env:POSTGRES_DB ?? "waveflix"
$PostgresUser = $env:POSTGRES_USER ?? "waveflix"
if ([string]::IsNullOrWhiteSpace($env:POSTGRES_PASSWORD)) {
    throw "POSTGRES_PASSWORD must be set before running this script"
}
$PostgresPassword = $env:POSTGRES_PASSWORD

# Backup configuration
$BackupDir = $env:BACKUP_DIR ?? ".\backups"
$BackupRetentionDays = [int]($env:BACKUP_RETENTION_DAYS ?? 30)
$BackupRetentionWeeks = [int]($env:BACKUP_RETENTION_WEEKS ?? 12)
$BackupRetentionMonths = [int]($env:BACKUP_RETENTION_MONTHS ?? 12)

# Cloud backup configuration
$S3Bucket = $env:S3_BUCKET ?? ""
$S3Prefix = $env:S3_PREFIX ?? "waveflix-backups"
$AWSRegion = $env:AWS_REGION ?? "us-east-1"

# Notification configuration
$WebhookURL = $env:WEBHOOK_URL ?? ""
$SlackWebhook = $env:SLACK_WEBHOOK ?? ""

# Ensure backup directory exists
if (-not (Test-Path $BackupDir)) {
    New-Item -ItemType Directory -Path $BackupDir -Force | Out-Null
}

$LogFile = Join-Path $BackupDir "backup.log"

# Function to log messages
function Write-Log {
    param(
        [string]$Level,
        [string]$Message
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logEntry = "[$timestamp] [$Level] $Message"
    
    switch ($Level) {
        "INFO"    { Write-Host "[INFO] $Message" -ForegroundColor Blue }
        "SUCCESS" { Write-Host "[SUCCESS] $Message" -ForegroundColor Green }
        "WARNING" { Write-Host "[WARNING] $Message" -ForegroundColor Yellow }
        "ERROR"   { Write-Host "[ERROR] $Message" -ForegroundColor Red }
    }
    
    Add-Content -Path $LogFile -Value $logEntry
}

# Function to send notifications
function Send-Notification {
    param(
        [string]$Status,
        [string]$Message,
        [string]$BackupFile = ""
    )
    
    $emoji = "✅"
    $color = "#36a64f"
    
    switch ($Status) {
        "error"   { $emoji = "❌"; $color = "#ff0000" }
        "warning" { $emoji = "⚠️"; $color = "#ffaa00" }
    }
    
    # Slack notification
    if ($SlackWebhook) {
        try {
            $slackPayload = @{
                text = "$emoji WaveFlix Hub Database Backup"
                attachments = @(@{
                    color = $color
                    fields = @(
                        @{ title = "Status"; value = $Message; short = $true },
                        @{ title = "Timestamp"; value = (Get-Date).ToString(); short = $true }
                    )
                })
            }
            
            if ($BackupFile) {
                $slackPayload.attachments[0].fields += @{ title = "File"; value = $BackupFile; short = $true }
            }
            
            $json = $slackPayload | ConvertTo-Json -Depth 10
            Invoke-RestMethod -Uri $SlackWebhook -Method Post -Body $json -ContentType "application/json" | Out-Null
        }
        catch {
            Write-Log "WARNING" "Failed to send Slack notification: $_"
        }
    }
    
    # Generic webhook notification
    if ($WebhookURL) {
        try {
            $webhookPayload = @{
                service = "waveflix-hub"
                component = "database-backup"
                status = $Status
                message = $Message
                timestamp = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
                backup_file = $BackupFile
            }
            
            $json = $webhookPayload | ConvertTo-Json
            Invoke-RestMethod -Uri $WebhookURL -Method Post -Body $json -ContentType "application/json" | Out-Null
        }
        catch {
            Write-Log "WARNING" "Failed to send webhook notification: $_"
        }
    }
}

# Function to check PostgreSQL connectivity
function Test-PostgresConnection {
    Write-Log "INFO" "Checking PostgreSQL connectivity..."
    
    $env:PGPASSWORD = $PostgresPassword
    
    try {
        $result = pg_isready -h $PostgresHost -p $PostgresPort -U $PostgresUser -d $PostgresDB 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Log "SUCCESS" "PostgreSQL connectivity verified"
            return $true
        }
        else {
            Write-Log "ERROR" "PostgreSQL is not accessible"
            Send-Notification "error" "PostgreSQL connectivity check failed"
            return $false
        }
    }
    catch {
        Write-Log "ERROR" "PostgreSQL connectivity check failed: $_"
        Send-Notification "error" "PostgreSQL connectivity check failed"
        return $false
    }
}

# Function to create backup
function New-DatabaseBackup {
    param([string]$Type)
    
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    $backupFilename = "waveflix_${Type}_${timestamp}.sql"
    $backupPath = Join-Path $BackupDir $backupFilename
    $compressedPath = "$backupPath.gz"
    
    Write-Log "INFO" "Starting $Type backup: $backupFilename"
    
    $env:PGPASSWORD = $PostgresPassword
    
    try {
        # Create backup with pg_dump
        $pgDumpArgs = @(
            "-h", $PostgresHost
            "-p", $PostgresPort
            "-U", $PostgresUser
            "-d", $PostgresDB
            "--verbose"
            "--clean"
            "--if-exists"
            "--create"
            "--format=plain"
            "--no-password"
            "--no-privileges"
            "--no-owner"
            "--exclude-table-data=cache.*"
            "--exclude-table-data=analytics.page_views"
            "--exclude-table-data=analytics.api_requests"
        )
        
        $pgDumpOutput = & pg_dump @pgDumpArgs 2>&1
        
        if ($LASTEXITCODE -eq 0) {
            # Save output to file
            $pgDumpOutput | Out-File -FilePath $backupPath -Encoding UTF8
            
            # Compress the backup using 7-Zip or built-in compression
            if (Get-Command "7z" -ErrorAction SilentlyContinue) {
                & 7z a -tgzip "$compressedPath" "$backupPath" | Out-Null
                Remove-Item $backupPath
            }
            elseif (Get-Command "Compress-Archive" -ErrorAction SilentlyContinue) {
                Compress-Archive -Path $backupPath -DestinationPath "$backupPath.zip"
                Remove-Item $backupPath
                $compressedPath = "$backupPath.zip"
            }
            else {
                # Keep uncompressed if no compression available
                $compressedPath = $backupPath
                Write-Log "WARNING" "No compression tool available, backup saved uncompressed"
            }
            
            # Verify backup file
            if (Test-Path $compressedPath) {
                $backupSize = (Get-Item $compressedPath).Length
                $backupSizeFormatted = "{0:N2} MB" -f ($backupSize / 1MB)
                Write-Log "SUCCESS" "$Type backup completed: $compressedPath ($backupSizeFormatted)"
                
                # Create checksum
                $hash = Get-FileHash -Path $compressedPath -Algorithm SHA256
                $hash.Hash | Out-File -FilePath "$compressedPath.sha256" -Encoding UTF8
                
                # Upload to S3 if configured
                if ($S3Bucket) {
                    Send-BackupToS3 -BackupFile $compressedPath -BackupType $Type
                }
                
                Send-Notification "success" "$Type backup completed successfully" $compressedPath
                return $compressedPath
            }
            else {
                Write-Log "ERROR" "$Type backup verification failed"
                Send-Notification "error" "$Type backup verification failed"
                return $null
            }
        }
        else {
            Write-Log "ERROR" "$Type backup failed with exit code $LASTEXITCODE"
            Write-Log "ERROR" "pg_dump output: $pgDumpOutput"
            Send-Notification "error" "$Type backup failed"
            return $null
        }
    }
    catch {
        Write-Log "ERROR" "$Type backup failed: $_"
        Send-Notification "error" "$Type backup failed"
        return $null
    }
}

# Function to upload backup to S3
function Send-BackupToS3 {
    param(
        [string]$BackupFile,
        [string]$BackupType
    )
    
    if (-not (Get-Command "aws" -ErrorAction SilentlyContinue)) {
        Write-Log "WARNING" "AWS CLI not installed, skipping S3 upload"
        return
    }
    
    $s3Path = "s3://$S3Bucket/$S3Prefix/$BackupType/$(Split-Path $BackupFile -Leaf)"
    
    Write-Log "INFO" "Uploading backup to S3: $s3Path"
    
    try {
        aws s3 cp $BackupFile $s3Path --region $AWSRegion --storage-class STANDARD_IA
        
        if ($LASTEXITCODE -eq 0) {
            # Upload checksum as well
            aws s3 cp "$BackupFile.sha256" "$s3Path.sha256" --region $AWSRegion --storage-class STANDARD_IA
            Write-Log "SUCCESS" "Backup uploaded to S3: $s3Path"
        }
        else {
            Write-Log "ERROR" "Failed to upload backup to S3"
            Send-Notification "warning" "S3 upload failed for $BackupType backup"
        }
    }
    catch {
        Write-Log "ERROR" "S3 upload error: $_"
        Send-Notification "warning" "S3 upload failed for $BackupType backup"
    }
}

# Function to clean up old backups
function Remove-OldBackups {
    Write-Log "INFO" "Starting backup cleanup..."
    
    try {
        # Clean daily backups
        $cutoffDaily = (Get-Date).AddDays(-$BackupRetentionDays)
        Get-ChildItem -Path $BackupDir -Filter "waveflix_daily_*" | Where-Object { 
            $_.LastWriteTime -lt $cutoffDaily 
        } | Remove-Item -Force
        
        # Clean weekly backups
        $cutoffWeekly = (Get-Date).AddDays(-($BackupRetentionWeeks * 7))
        Get-ChildItem -Path $BackupDir -Filter "waveflix_weekly_*" | Where-Object { 
            $_.LastWriteTime -lt $cutoffWeekly 
        } | Remove-Item -Force
        
        # Clean monthly backups
        $cutoffMonthly = (Get-Date).AddDays(-($BackupRetentionMonths * 30))
        Get-ChildItem -Path $BackupDir -Filter "waveflix_monthly_*" | Where-Object { 
            $_.LastWriteTime -lt $cutoffMonthly 
        } | Remove-Item -Force
        
        # Clean old logs
        $cutoffLogs = (Get-Date).AddDays(-30)
        Get-ChildItem -Path $BackupDir -Filter "backup.log.*" | Where-Object { 
            $_.LastWriteTime -lt $cutoffLogs 
        } | Remove-Item -Force
        
        Write-Log "SUCCESS" "Backup cleanup completed"
    }
    catch {
        Write-Log "ERROR" "Backup cleanup failed: $_"
    }
}

# Function to verify backup integrity
function Test-BackupIntegrity {
    param([string]$BackupFilePath)
    
    Write-Log "INFO" "Verifying backup integrity: $(Split-Path $BackupFilePath -Leaf)"
    
    $checksumFile = "$BackupFilePath.sha256"
    
    if (-not (Test-Path $checksumFile)) {
        Write-Log "WARNING" "Checksum file not found for $(Split-Path $BackupFilePath -Leaf)"
        return $false
    }
    
    try {
        $originalHash = Get-Content $checksumFile
        $currentHash = (Get-FileHash -Path $BackupFilePath -Algorithm SHA256).Hash
        
        if ($originalHash.Trim() -eq $currentHash.Trim()) {
            Write-Log "SUCCESS" "Backup integrity verified: $(Split-Path $BackupFilePath -Leaf)"
            return $true
        }
        else {
            Write-Log "ERROR" "Backup integrity check failed: $(Split-Path $BackupFilePath -Leaf)"
            Send-Notification "error" "Backup integrity check failed" $BackupFilePath
            return $false
        }
    }
    catch {
        Write-Log "ERROR" "Backup integrity check error: $_"
        return $false
    }
}

# Function to create backup report
function New-BackupReport {
    $reportFile = Join-Path $BackupDir "backup_report_$(Get-Date -Format 'yyyyMMdd').md"
    
    $dailyBackups = Get-ChildItem -Path $BackupDir -Filter "waveflix_daily_*" | Where-Object { 
        $_.LastWriteTime -gt (Get-Date).AddDays(-7) 
    } | ForEach-Object { 
        "- $($_.Name) ($([math]::Round($_.Length / 1MB, 2)) MB - $($_.LastWriteTime))"
    }
    
    $weeklyBackups = Get-ChildItem -Path $BackupDir -Filter "waveflix_weekly_*" | Where-Object { 
        $_.LastWriteTime -gt (Get-Date).AddDays(-30) 
    } | ForEach-Object { 
        "- $($_.Name) ($([math]::Round($_.Length / 1MB, 2)) MB - $($_.LastWriteTime))"
    }
    
    $monthlyBackups = Get-ChildItem -Path $BackupDir -Filter "waveflix_monthly_*" | Where-Object { 
        $_.LastWriteTime -gt (Get-Date).AddDays(-365) 
    } | ForEach-Object { 
        "- $($_.Name) ($([math]::Round($_.Length / 1MB, 2)) MB - $($_.LastWriteTime))"
    }
    
    $totalSize = (Get-ChildItem -Path $BackupDir -Recurse | Measure-Object -Property Length -Sum).Sum
    $totalSizeFormatted = "{0:N2} MB" -f ($totalSize / 1MB)
    
    $reportContent = @"
# WaveFlix Hub - Backup Report

**Date:** $(Get-Date)
**Status:** Completed Successfully ✅

## Backup Summary

### Daily Backups
$($dailyBackups -join "`n")

### Weekly Backups  
$($weeklyBackups -join "`n")

### Monthly Backups
$($monthlyBackups -join "`n")

## Storage Usage

**Local Storage:** $totalSizeFormatted

## Configuration

- **Retention Policy:**
  - Daily: $BackupRetentionDays days
  - Weekly: $BackupRetentionWeeks weeks  
  - Monthly: $BackupRetentionMonths months

- **Database:** $PostgresHost`:$PostgresPort/$PostgresDB
- **Backup Location:** $BackupDir
$(if ($S3Bucket) { "- **S3 Bucket:** s3://$S3Bucket/$S3Prefix/" })

## Next Scheduled Backups

- **Daily:** Every day at 2:00 AM
- **Weekly:** Every Sunday at 3:00 AM  
- **Monthly:** 1st day of month at 4:00 AM

---
Generated by WaveFlix Hub Backup System
"@

    $reportContent | Out-File -FilePath $reportFile -Encoding UTF8
    Write-Log "SUCCESS" "Backup report created: $reportFile"
}

# Function to restore from backup
function Restore-DatabaseBackup {
    param(
        [string]$BackupFilePath,
        [string]$TargetDatabase = $PostgresDB
    )
    
    if (-not (Test-Path $BackupFilePath)) {
        Write-Log "ERROR" "Backup file not found: $BackupFilePath"
        return $false
    }
    
    Write-Log "WARNING" "Starting database restore from: $(Split-Path $BackupFilePath -Leaf)"
    Write-Log "WARNING" "Target database: $TargetDatabase"
    
    # Verify backup before restore
    if (-not (Test-BackupIntegrity -BackupFilePath $BackupFilePath)) {
        Write-Log "ERROR" "Backup verification failed, aborting restore"
        return $false
    }
    
    # Confirmation prompt
    $confirmation = Read-Host "This will OVERWRITE the database '$TargetDatabase'. Are you sure? (yes/no)"
    if ($confirmation -ne "yes") {
        Write-Log "INFO" "Restore cancelled by user"
        return $false
    }
    
    $env:PGPASSWORD = $PostgresPassword
    
    try {
        # Determine if file is compressed and decompress if needed
        $sqlFile = $BackupFilePath
        if ($BackupFilePath -match '\.(gz|zip)$') {
            $tempFile = [System.IO.Path]::GetTempFileName() + ".sql"
            
            if ($BackupFilePath -match '\.gz$') {
                # Decompress gzip (requires 7-Zip or similar)
                if (Get-Command "7z" -ErrorAction SilentlyContinue) {
                    & 7z e $BackupFilePath "-o$(Split-Path $tempFile -Parent)" -y | Out-Null
                    $sqlFile = $tempFile
                }
            }
            elseif ($BackupFilePath -match '\.zip$') {
                Expand-Archive -Path $BackupFilePath -DestinationPath (Split-Path $tempFile -Parent) -Force
                $extractedFiles = Get-ChildItem -Path (Split-Path $tempFile -Parent) -Filter "*.sql"
                if ($extractedFiles) {
                    $sqlFile = $extractedFiles[0].FullName
                }
            }
        }
        
        # Restore database
        Get-Content $sqlFile | psql -h $PostgresHost -p $PostgresPort -U $PostgresUser -d $TargetDatabase
        
        if ($LASTEXITCODE -eq 0) {
            Write-Log "SUCCESS" "Database restore completed successfully"
            Send-Notification "success" "Database restore completed" $BackupFilePath
            
            # Clean up temp file if created
            if ($sqlFile -ne $BackupFilePath -and (Test-Path $sqlFile)) {
                Remove-Item $sqlFile -Force
            }
            
            return $true
        }
        else {
            Write-Log "ERROR" "Database restore failed with exit code $LASTEXITCODE"
            Send-Notification "error" "Database restore failed" $BackupFilePath
            return $false
        }
    }
    catch {
        Write-Log "ERROR" "Database restore failed: $_"
        Send-Notification "error" "Database restore failed" $BackupFilePath
        return $false
    }
}

# Main function
function Main {
    switch ($Command.ToLower()) {
        "backup" {
            Write-Log "INFO" "Starting WaveFlix Hub PostgreSQL backup ($BackupType)"
            
            if (-not (Test-PostgresConnection)) {
                exit 1
            }
            
            $backupFile = New-DatabaseBackup -Type $BackupType
            if ($backupFile) {
                Test-BackupIntegrity -BackupFilePath $backupFile | Out-Null
                Remove-OldBackups
                New-BackupReport
                Write-Log "SUCCESS" "Backup process completed successfully"
            }
            else {
                exit 1
            }
        }
        
        "restore" {
            if (-not $BackupFile) {
                Write-Host "Usage: .\backup_postgres.ps1 -Command restore -BackupFile <path> [-TargetDB <name>]"
                exit 1
            }
            
            $success = Restore-DatabaseBackup -BackupFilePath $BackupFile -TargetDatabase ($TargetDB ? $TargetDB : $PostgresDB)
            if (-not $success) {
                exit 1
            }
        }
        
        "verify" {
            if (-not $BackupFile) {
                Write-Host "Usage: .\backup_postgres.ps1 -Command verify -BackupFile <path>"
                exit 1
            }
            
            $isValid = Test-BackupIntegrity -BackupFilePath $BackupFile
            if (-not $isValid) {
                exit 1
            }
        }
        
        "cleanup" {
            Remove-OldBackups
        }
        
        "report" {
            New-BackupReport
        }
        
        default {
            Write-Host @"
Usage: .\backup_postgres.ps1 -Command <command> [options]

Commands:
  backup [-BackupType <daily|weekly|monthly>]  - Create database backup
  restore -BackupFile <path> [-TargetDB <name>] - Restore database from backup
  verify -BackupFile <path>                     - Verify backup integrity
  cleanup                                       - Clean old backups
  report                                        - Generate backup report

Environment Variables:
  POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD
  BACKUP_DIR, BACKUP_RETENTION_DAYS, S3_BUCKET, SLACK_WEBHOOK
"@
            exit 1
        }
    }
}

# Run main function
try {
    Main
}
catch {
    Write-Log "ERROR" "Script execution failed: $_"
    exit 1
}
