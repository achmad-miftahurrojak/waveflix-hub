#!/bin/sh
set -e

echo "🔧 Starting Nginx configuration setup..."

# Set default domain if not provided
export DOMAIN_NAME=${DOMAIN_NAME:-localhost}

echo "📋 Domain: $DOMAIN_NAME"

# Generate self-signed SSL certificate if none exists
if [ ! -f "/etc/ssl/certs/waveflix.crt" ] || [ ! -f "/etc/ssl/certs/waveflix.key" ]; then
    echo "🔒 Generating self-signed SSL certificate for $DOMAIN_NAME..."
    
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout /etc/ssl/certs/waveflix.key \
        -out /etc/ssl/certs/waveflix.crt \
        -subj "/C=US/ST=State/L=City/O=Organization/OU=OrgUnit/CN=$DOMAIN_NAME/emailAddress=admin@$DOMAIN_NAME" \
        -addext "subjectAltName=DNS:$DOMAIN_NAME,DNS:www.$DOMAIN_NAME,DNS:api.$DOMAIN_NAME,DNS:cdn.$DOMAIN_NAME"
    
    echo "✅ Self-signed certificate generated"
    echo "⚠️  Remember to replace with proper SSL certificates in production!"
fi

# Substitute environment variables in nginx config
echo "🔄 Processing nginx configuration templates..."
envsubst '${DOMAIN_NAME}' < /etc/nginx/nginx.conf > /tmp/nginx.conf && mv /tmp/nginx.conf /etc/nginx/nginx.conf
envsubst '${DOMAIN_NAME}' < /etc/nginx/conf.d/default.conf > /tmp/default.conf && mv /tmp/default.conf /etc/nginx/conf.d/default.conf

# Test nginx configuration
echo "🧪 Testing nginx configuration..."
nginx -t

if [ $? -eq 0 ]; then
    echo "✅ Nginx configuration test passed"
else
    echo "❌ Nginx configuration test failed"
    exit 1
fi

# Create htpasswd file for staging auth if not exists
if [ ! -f "/etc/nginx/.htpasswd" ]; then
    echo "🔐 Creating basic auth file for staging..."
    # Default: admin/admin123 (change in production)
    echo "admin:\$2y\$10\$8K1p/a0dJnkjrpZT9ZgUu.YMFvUfq8W5K8sVw5V9K8aN6GbU6b9yG" > /etc/nginx/.htpasswd
fi

echo "🚀 Starting nginx..."
exec "$@"