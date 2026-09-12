# SSL Certificates

This directory contains SSL certificates for WaveFlix Hub.

## Development

For development, self-signed certificates will be generated automatically by the nginx container.

## Production

Replace the auto-generated certificates with proper SSL certificates from a Certificate Authority:

1. **Let's Encrypt (Recommended for free certificates):**
   ```bash
   # Install certbot
   sudo apt-get install certbot

   # Get certificates
   sudo certbot certonly --standalone -d yourdomain.com -d www.yourdomain.com

   # Copy certificates
   sudo cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem ./nginx/ssl/waveflix.crt
   sudo cp /etc/letsencrypt/live/yourdomain.com/privkey.pem ./nginx/ssl/waveflix.key
   ```

2. **Commercial Certificate:**
   - Purchase SSL certificate from a trusted CA
   - Copy certificate files:
     - `waveflix.crt` - Certificate file (including intermediate certificates)
     - `waveflix.key` - Private key file

3. **Update permissions:**
   ```bash
   chmod 600 nginx/ssl/waveflix.key
   chmod 644 nginx/ssl/waveflix.crt
   ```

## Security Notes

- Never commit private key files to version control
- Use environment variables or Docker secrets for production deployments
- Regularly renew certificates (Let's Encrypt expires every 90 days)
- Consider using automated certificate management tools like cert-manager for Kubernetes

## Certificate Renewal

For Let's Encrypt, set up automatic renewal:

```bash
# Add to crontab
0 12 * * * /usr/bin/certbot renew --quiet && docker-compose restart nginx
```