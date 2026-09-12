FROM nginx:alpine

# Install additional tools
RUN apk add --no-cache \
    openssl \
    curl \
    envsubst

# Create cache directories
RUN mkdir -p /var/cache/nginx/api \
             /var/cache/nginx/images \
             /var/cache/nginx/static && \
    chown -R nginx:nginx /var/cache/nginx

# Create log directories
RUN mkdir -p /var/log/nginx && \
    chown -R nginx:nginx /var/log/nginx

# Copy configuration files
COPY nginx.conf /etc/nginx/nginx.conf
COPY conf.d/ /etc/nginx/conf.d/

# Copy SSL certificate placeholder (replace with real certs in production)
COPY ssl/ /etc/ssl/certs/

# Copy startup script
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/nginx-health || exit 1

EXPOSE 80 443 8080 9113

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["nginx", "-g", "daemon off;"]