#!/bin/bash
# WaveFlix Hub Load Balancer Setup Script

set -e

echo "🔧 Setting up WaveFlix Hub Load Balancer..."

# Configuration
NGINX_DIR="./nginx"
SSL_DIR="$NGINX_DIR/ssl"
DOMAIN_NAME=${DOMAIN_NAME:-localhost}

# Create directories
mkdir -p "$SSL_DIR"
mkdir -p "./nginx/cache"
mkdir -p "./nginx/logs"

echo "📁 Created load balancer directories"

# Generate SSL certificate for development
if [ ! -f "$SSL_DIR/waveflix.crt" ]; then
    echo "🔒 Generating SSL certificate for $DOMAIN_NAME..."
    
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout "$SSL_DIR/waveflix.key" \
        -out "$SSL_DIR/waveflix.crt" \
        -subj "/C=US/ST=State/L=City/O=WaveFlix/CN=$DOMAIN_NAME" \
        -addext "subjectAltName=DNS:$DOMAIN_NAME,DNS:www.$DOMAIN_NAME,DNS:api.$DOMAIN_NAME,DNS:cdn.$DOMAIN_NAME"
    
    echo "✅ SSL certificate generated"
fi

# Set proper permissions
chmod 600 "$SSL_DIR/waveflix.key" 2>/dev/null || echo "⚠️  Could not set key permissions (normal on Windows)"
chmod 644 "$SSL_DIR/waveflix.crt" 2>/dev/null || echo "⚠️  Could not set certificate permissions (normal on Windows)"

# Test nginx configuration
echo "🧪 Testing nginx configuration..."

# Build nginx image first
docker build -t waveflix-nginx "$NGINX_DIR"

# Test configuration
docker run --rm -v "$(pwd)/$NGINX_DIR:/etc/nginx" waveflix-nginx nginx -t

if [ $? -eq 0 ]; then
    echo "✅ Nginx configuration test passed"
else
    echo "❌ Nginx configuration test failed"
    exit 1
fi

# Setup load balancer monitoring
echo "📊 Setting up load balancer monitoring..."

# Create htpasswd for basic auth (staging)
if [ ! -f "$NGINX_DIR/.htpasswd" ]; then
    echo "🔐 Creating basic auth file..."
    # Create admin/admin123 (change in production)
    echo "admin:\$2y\$10\$8K1p/a0dJnkjrpZT9ZgUu.YMFvUfq8W5K8sVw5V9K8aN6GbU6b9yG" > "$NGINX_DIR/.htpasswd"
    echo "   Username: admin"
    echo "   Password: admin123"
    echo "   ⚠️  Change these credentials in production!"
fi

# Display configuration summary
echo ""
echo "📋 Load Balancer Configuration Summary:"
echo "   Domain: $DOMAIN_NAME"
echo "   HTTP Port: 80 (redirects to HTTPS)"
echo "   HTTPS Port: 443"
echo "   SSL Certificate: $SSL_DIR/waveflix.crt"
echo "   Nginx Config: $NGINX_DIR/nginx.conf"
echo ""
echo "🎯 Backend Upstream: backend:8080"
echo "🖥️  Frontend Upstream: frontend:3000"
echo ""
echo "⚡ Features Enabled:"
echo "   - SSL/TLS termination"
echo "   - Gzip compression" 
echo "   - API response caching"
echo "   - Rate limiting"
echo "   - Security headers"
echo "   - Health checks"
echo "   - Load balancing (least_conn)"
echo ""
echo "🚀 Start the load balancer:"
echo "   docker-compose up -d nginx"
echo ""
echo "🔍 Monitor nginx:"
echo "   docker-compose logs -f nginx"
echo "   curl -I https://$DOMAIN_NAME/health"