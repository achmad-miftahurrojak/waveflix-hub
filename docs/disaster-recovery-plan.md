# WaveFlix Hub - Disaster Recovery Plan (DRP)

## Table of Contents

1. [Overview](#overview)
2. [Recovery Time Objectives (RTO) & Recovery Point Objectives (RPO)](#rto-rpo)
3. [Disaster Scenarios](#disaster-scenarios)
4. [Recovery Procedures](#recovery-procedures)
5. [Communication Plan](#communication-plan)
6. [Testing & Validation](#testing-validation)
7. [Post-Recovery Actions](#post-recovery-actions)

## Overview

This Disaster Recovery Plan outlines procedures for restoring WaveFlix Hub services in the event of various disaster scenarios. The plan focuses on minimizing downtime and data loss while ensuring business continuity.

### Key Components
- **Database**: PostgreSQL with automated backups
- **Application**: Go backend + Next.js frontend
- **Infrastructure**: Docker containers, Kubernetes orchestration
- **Data Storage**: Local backups + S3 cloud storage
- **Monitoring**: Prometheus, Grafana, custom alerting

### Disaster Recovery Team
- **Primary DR Coordinator**: Lead DevOps Engineer
- **Secondary DR Coordinator**: Senior Backend Developer  
- **Database Administrator**: PostgreSQL Expert
- **Infrastructure Team**: Kubernetes/Docker specialists
- **Communication Lead**: Product Manager

## RTO & RPO Targets {#rto-rpo}

| Service Component | RTO (Recovery Time) | RPO (Data Loss) | Priority |
|-------------------|---------------------|-----------------|----------|
| Database | 30 minutes | 1 hour | Critical |
| Backend API | 15 minutes | 0 minutes | Critical |
| Frontend Web App | 10 minutes | 0 minutes | High |
| User Authentication | 20 minutes | 1 hour | Critical |
| File Uploads | 60 minutes | 4 hours | Medium |
| Analytics/Logs | 120 minutes | 24 hours | Low |

## Disaster Scenarios

### Scenario 1: Database Server Failure

**Symptoms:**
- Database connection timeouts
- Backend API returning 500 errors
- Health checks failing

**Impact:** Complete service outage

**Recovery Procedure:** [DB-RECOVERY-001](#db-recovery-001)

### Scenario 2: Application Server Failure

**Symptoms:**
- Frontend cannot connect to backend
- API endpoints unreachable
- Load balancer showing unhealthy targets

**Impact:** Service degradation or outage

**Recovery Procedure:** [APP-RECOVERY-001](#app-recovery-001)

### Scenario 3: Complete Infrastructure Loss

**Symptoms:**
- All services unreachable
- Infrastructure monitoring alerts
- Physical or cloud provider outage

**Impact:** Complete service outage

**Recovery Procedure:** [INFRA-RECOVERY-001](#infra-recovery-001)

### Scenario 4: Data Corruption

**Symptoms:**
- Database integrity check failures
- Inconsistent data in application
- Backup verification failures

**Impact:** Data integrity issues

**Recovery Procedure:** [DATA-RECOVERY-001](#data-recovery-001)

### Scenario 5: Security Breach

**Symptoms:**
- Unauthorized database access
- Suspicious user activities
- Security monitoring alerts

**Impact:** Data security compromise

**Recovery Procedure:** [SEC-RECOVERY-001](#sec-recovery-001)

## Recovery Procedures

### Database Recovery {#db-recovery-001}

#### DB-RECOVERY-001: PostgreSQL Database Recovery

**Prerequisites:**
- Access to backup storage (local + S3)
- PostgreSQL server or ability to provision new instance
- Database administrator credentials

**Steps:**

1. **Assess Damage** (5 minutes)
   ```bash
   # Check if database is accessible
   ./scripts/backup_health_check.sh check
   
   # Try to connect to database
   psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB -c "SELECT version();"
   ```

2. **Identify Latest Backup** (2 minutes)
   ```bash
   # List available backups
   ls -la backups/waveflix_*.sql.gz | tail -5
   
   # Check backup integrity
   ./scripts/backup_postgres.sh verify backups/latest_backup.sql.gz
   ```

3. **Provision New Database** (10 minutes)
   ```bash
   # Using Docker
   docker run -d --name waveflix-postgres-recovery \
     -e POSTGRES_DB=waveflix \
     -e POSTGRES_USER=waveflix \
     -e POSTGRES_PASSWORD=$POSTGRES_PASSWORD \
     -p 5433:5432 \
     postgres:15-alpine
   
   # Wait for database to be ready
   docker exec waveflix-postgres-recovery pg_isready -U waveflix
   ```

4. **Restore from Backup** (10 minutes)
   ```bash
   # Restore latest backup
   ./scripts/backup_postgres.sh restore backups/latest_backup.sql.gz waveflix
   
   # Verify restoration
   psql -h localhost -p 5433 -U waveflix -d waveflix -c "SELECT COUNT(*) FROM users;"
   ```

5. **Update Application Configuration** (3 minutes)
   ```bash
   # Update database connection
   # Edit website/backend/.env
   DATABASE_URL=postgres://waveflix:$POSTGRES_PASSWORD@localhost:5433/waveflix
   
   # Restart backend services
   docker-compose restart backend
   ```

**Expected RTO:** 30 minutes  
**Expected RPO:** 1 hour (daily backup frequency)

### Application Recovery {#app-recovery-001}

#### APP-RECOVERY-001: Application Server Recovery

**Prerequisites:**
- Container registry access
- Application source code
- Configuration files

**Steps:**

1. **Diagnose Issue** (3 minutes)
   ```bash
   # Check container status
   docker ps -a | grep waveflix
   
   # Check logs
   docker logs waveflix-backend
   docker logs waveflix-frontend
   ```

2. **Quick Restart** (2 minutes)
   ```bash
   # Try restarting services
   docker-compose restart
   
   # Check if services recover
   curl -f http://localhost:8080/health
   ```

3. **Full Redeployment** (10 minutes)
   ```bash
   # Pull latest images
   docker-compose pull
   
   # Stop and recreate containers
   docker-compose down
   docker-compose up -d
   
   # Monitor startup
   docker-compose logs -f
   ```

**Expected RTO:** 15 minutes  
**Expected RPO:** 0 minutes (stateless application)

### Infrastructure Recovery {#infra-recovery-001}

#### INFRA-RECOVERY-001: Complete Infrastructure Recovery

**Prerequisites:**
- Cloud provider account access
- Infrastructure as Code templates
- DNS management access
- Backup storage access

**Steps:**

1. **Activate Alternate Site** (0 minutes)
   ```bash
   # This should be automated via monitoring
   # Manual activation if needed:
   
   # Update DNS to point to DR site
   # (This step varies by DNS provider)
   ```

2. **Provision New Infrastructure** (15 minutes)
   ```bash
   # Using Kubernetes
   kubectl apply -f k8s/namespace.yaml
   kubectl apply -f k8s/secrets.yaml
   kubectl apply -f k8s/postgres-deployment.yaml
   kubectl apply -f k8s/backend-deployment.yaml
   kubectl apply -f k8s/frontend-deployment.yaml
   
   # Wait for pods to be ready
   kubectl get pods -w
   ```

3. **Restore Database** (30 minutes)
   ```bash
   # Download latest backup from S3
   aws s3 cp s3://waveflix-backups/daily/latest.sql.gz ./
   
   # Restore to new database
   kubectl exec -it postgres-pod -- psql -U waveflix -d waveflix < latest.sql.gz
   ```

4. **Validate Services** (10 minutes)
   ```bash
   # Check all endpoints
   kubectl port-forward service/frontend 3000:80
   curl -f http://localhost:3000/api/health
   
   # Smoke test critical functions
   # - User login
   # - Content browsing  
   # - Watchlist operations
   ```

**Expected RTO:** 60 minutes  
**Expected RPO:** 4 hours

### Data Recovery {#data-recovery-001}

#### DATA-RECOVERY-001: Data Corruption Recovery

**Prerequisites:**
- Multiple backup versions available
- Database expertise
- Data validation tools

**Steps:**

1. **Identify Corruption Scope** (10 minutes)
   ```bash
   # Run data integrity checks
   psql -d waveflix -c "SELECT * FROM pg_stat_database WHERE datname='waveflix';"
   
   # Check specific tables
   psql -d waveflix -c "SELECT COUNT(*) FROM users WHERE created_at IS NULL;"
   ```

2. **Find Clean Backup** (5 minutes)
   ```bash
   # Test multiple backups
   for backup in backups/waveflix_daily_*.sql.gz; do
     echo "Testing $backup"
     ./scripts/backup_postgres.sh verify "$backup"
   done
   ```

3. **Selective Data Recovery** (20 minutes)
   ```bash
   # Create temporary database
   createdb waveflix_temp
   
   # Restore clean backup to temp
   gunzip -c backups/clean_backup.sql.gz | psql waveflix_temp
   
   # Extract clean data
   pg_dump waveflix_temp --table=users --data-only > clean_users.sql
   
   # Restore clean data to main database
   psql waveflix -c "TRUNCATE users CASCADE;"
   psql waveflix < clean_users.sql
   ```

**Expected RTO:** 45 minutes  
**Expected RPO:** Variable (depends on backup frequency)

### Security Breach Recovery {#sec-recovery-001}

#### SEC-RECOVERY-001: Security Incident Response

**Prerequisites:**
- Security incident response team
- Clean backup available
- New credentials prepared

**Steps:**

1. **Immediate Containment** (5 minutes)
   ```bash
   # Isolate affected systems
   docker-compose down
   
   # Block suspicious IP addresses
   iptables -A INPUT -s SUSPICIOUS_IP -j DROP
   
   # Disable compromised user accounts
   psql -d waveflix -c "UPDATE users SET is_active=false WHERE suspicious_activity=true;"
   ```

2. **Evidence Preservation** (10 minutes)
   ```bash
   # Capture logs
   docker logs waveflix-backend > incident_logs_$(date +%s).log
   
   # Database forensics
   pg_dump waveflix --schema-only > schema_forensics.sql
   ```

3. **Clean Recovery** (30 minutes)
   ```bash
   # Restore from known-clean backup
   # (Use backup from before incident timeline)
   
   # Update all credentials
   # - Database passwords
   # - API keys
   # - JWT secrets
   
   # Redeploy with new credentials
   ```

4. **Security Hardening** (15 minutes)
   ```bash
   # Update security configurations
   # Enable additional logging
   # Apply security patches
   # Review access controls
   ```

**Expected RTO:** 60 minutes  
**Expected RPO:** Variable (depends on incident timing)

## Communication Plan

### Alert Escalation Matrix

| Severity | Initial Response Time | Escalation Path |
|----------|---------------------|-----------------|
| Critical | 5 minutes | On-call → Team Lead → Management |
| High | 15 minutes | Team Member → Team Lead |
| Medium | 1 hour | Normal business hours |
| Low | 4 hours | Email notification |

### Communication Channels

1. **Internal Team**: Slack #waveflix-alerts
2. **Management**: Email + Phone calls for critical issues  
3. **Users**: Status page updates
4. **Stakeholders**: Email updates every 30 minutes during incidents

### Status Page Updates

```bash
# Update status page (example)
curl -X POST https://status.waveflix.com/api/incidents \
  -H "Authorization: Bearer $STATUS_PAGE_TOKEN" \
  -d '{
    "name": "Database connectivity issues",
    "status": "investigating", 
    "message": "We are investigating reports of slow response times."
  }'
```

## Testing & Validation {#testing-validation}

### Monthly DR Tests

1. **Backup Restoration Test**
   - Restore latest backup to test environment
   - Verify data integrity
   - Measure restoration time

2. **Failover Test**
   - Switch to secondary database
   - Test application functionality
   - Measure failover time

3. **Communication Test**
   - Test alert notifications
   - Verify contact information
   - Practice escalation procedures

### Quarterly Full DR Drills

1. **Complete Infrastructure Rebuild**
   - Provision new environment from scratch
   - Restore all services and data
   - Full application testing

2. **Team Training**
   - Update team on new procedures
   - Practice recovery scenarios
   - Review and update documentation

### Testing Checklist

```bash
# DR Test Validation Script
./scripts/dr_test_validation.sh

# Key validation points:
# ✓ Database connectivity
# ✓ API endpoints responding
# ✓ User authentication working
# ✓ Core functionality operational
# ✓ Data integrity confirmed
# ✓ Performance within acceptable limits
```

## Post-Recovery Actions

### Immediate Actions (0-2 hours)

1. **Service Validation**
   - Run comprehensive health checks
   - Verify all critical functions
   - Monitor system performance

2. **Communication Updates**
   - Notify stakeholders of resolution
   - Update status page
   - Internal team notification

### Short-term Actions (2-24 hours)

1. **Root Cause Analysis**
   - Document incident timeline
   - Identify failure points
   - Gather relevant logs and evidence

2. **Performance Monitoring**
   - Monitor system stability
   - Watch for residual issues
   - Verify backup processes

### Long-term Actions (1-30 days)

1. **Post-Incident Review**
   - Team retrospective meeting
   - Update procedures based on lessons learned
   - Improve monitoring and alerting

2. **Documentation Updates**
   - Update runbooks
   - Revise recovery procedures
   - Update contact information

3. **Process Improvements**
   - Implement preventive measures
   - Enhance monitoring coverage
   - Automate manual processes

### Post-Recovery Validation Script

```bash
#!/bin/bash
# Post-recovery validation

echo "=== Post-Recovery Validation ==="

# Check database
echo "✓ Database connectivity"
./scripts/backup_health_check.sh check

# Check application
echo "✓ Application health"
curl -f http://localhost:8080/health

# Check user functions
echo "✓ User authentication"
curl -X POST http://localhost:8080/api/auth/login -d '{"test":"user"}'

# Generate recovery report
./scripts/generate_recovery_report.sh

echo "=== Recovery Validation Complete ==="
```

---

## Emergency Contact Information

| Role | Name | Primary Phone | Secondary Contact | Email |
|------|------|---------------|------------------|-------|
| DR Coordinator | [Name] | +1-xxx-xxx-xxxx | Slack: @username | email@domain.com |
| Database Admin | [Name] | +1-xxx-xxx-xxxx | WhatsApp: +1-xxx | email@domain.com |
| Infrastructure Lead | [Name] | +1-xxx-xxx-xxxx | Teams: @username | email@domain.com |

## Document Control

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 1.0 | 2024-01-XX | Initial DRP creation | DevOps Team |
| 1.1 | 2024-XX-XX | Added security breach procedures | Security Team |

**Next Review Date**: Every 6 months  
**Document Owner**: DevOps Team Lead  
**Approval**: CTO

---

*This document contains sensitive information. Distribute only to authorized personnel.*