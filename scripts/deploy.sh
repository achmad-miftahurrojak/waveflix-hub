#!/bin/bash
# WaveFlix Hub Production Deployment Script

set -e

echo "🚀 Starting WaveFlix Hub deployment..."

# Configuration
COMPOSE_FILE="docker-compose.yml"
ENV_FILE=".env.production"
BACKUP_DIR="./backups"
LOG_DIR="./logs"

# Create necessary directories
mkdir -p "$BACKUP_DIR" "$LOG_DIR"

# Load environment variables
if [ -f "$ENV_FILE" ]; then
    echo "📋 Loading environment from $ENV_FILE"
    export $(grep -v '^#' "$ENV_FILE" | xargs)
else
    echo "❌ Environment file $ENV_FILE not found!"
    exit 1
fi

# Validate required environment variables
required_vars=("DOMAIN_NAME" "POSTGRES_PASSWORD" "REDIS_PASSWORD" "TMDB_API_KEY" "JWT_SECRET")
for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        echo "❌ Required environment variable $var is not set!"
        exit 1
    fi
done

echo "✅ Environment validation passed"

# Database backup (if exists)
if docker-compose ps postgres | grep -q "Up"; then
    echo "💾 Creating database backup..."
    timestamp=$(date +%Y%m%d_%H%M%S)
    docker-compose exec -T postgres pg_dump -U waveflix waveflix > "$BACKUP_DIR/waveflix_$timestamp.sql"
    echo "✅ Database backed up to $BACKUP_DIR/waveflix_$timestamp.sql"
fi

# Pull latest images
echo "📥 Pulling latest Docker images..."
docker-compose pull

# Build custom images
echo "🔨 Building application images..."
docker-compose build --no-cache

# Deploy with zero-downtime rolling update
echo "🔄 Performing rolling deployment..."

# Start new services
docker-compose up -d --scale backend=6  # Scale up before switching
docker-compose up -d --remove-orphans

# Wait for health checks
echo "🩺 Waiting for health checks..."
max_attempts=30
attempt=0

while [ $attempt -lt $max_attempts ]; do
    if docker-compose exec backend wget --quiet --tries=1 --spider http://localhost:8080/health; then
        echo "✅ Backend health check passed"
        break
    fi
    
    attempt=$((attempt + 1))
    echo "⏳ Health check attempt $attempt/$max_attempts..."
    sleep 10
done

if [ $attempt -eq $max_attempts ]; then
    echo "❌ Health checks failed after $max_attempts attempts"
    echo "🔄 Rolling back..."
    docker-compose down
    exit 1
fi

# Scale back to normal replica count
docker-compose up -d --scale backend=3

# Clean up old images
echo "🧹 Cleaning up old Docker images..."
docker image prune -f

# Display deployment status
echo "📊 Deployment Status:"
docker-compose ps

echo "✅ WaveFlix Hub deployment completed successfully!"
echo "🌐 Application should be available at: https://$DOMAIN_NAME"
echo "📈 Monitoring available at: https://$DOMAIN_NAME:3001 (Grafana)"
echo "📜 Logs: docker-compose logs -f"