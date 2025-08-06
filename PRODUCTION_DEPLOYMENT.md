# 🚀 Production Deployment Checklist

## Pre-Deployment Requirements

### ☁️ Cloud Services Setup
- [ ] **MongoDB Atlas cluster** configured and accessible
- [ ] **CloudAMQP instance** configured and accessible  
- [ ] **Domain name** configured with DNS pointing to your server
- [ ] **SSL certificate email** configured for Let's Encrypt

### 🔐 Security Configuration
- [ ] **Production API keys** generated using `scripts/generate_secrets.sh`
- [ ] **Environment variables** configured in `.env.production`
- [ ] **Firewall rules** configured (ports 80, 443 open)
- [ ] **Server access** secured (SSH keys, etc.)

### 🐳 Docker Environment
- [ ] **Docker** and **Docker Compose** installed on production server
- [ ] **Docker registry** access configured (if using private registry)
- [ ] **Sufficient disk space** for containers and volumes

## Deployment Steps

### 1. Environment Configuration
```bash
# Copy and configure production environment
cp .env.production.example .env.production
# Edit .env.production with your actual values
```

**Required Variables:**
- `DOMAIN=your-domain.com`
- `LETSENCRYPT_EMAIL=your-email@domain.com`
- `MONGODB_URI=mongodb+srv://...`
- `CLOUDAMQP_URL=amqps://...`
- `MAILERCLOUD_API_KEYS=client1:key1,client2:key2`
- `GRAFANA_PASSWORD=secure-password`

### 2. Deploy Production Stack
```bash
# Basic deployment (webhook processor + nginx + SSL)
docker-compose -f docker-compose.prod.yml --env-file .env.production up -d

# With monitoring (includes Prometheus + Grafana)
docker-compose -f docker-compose.prod.yml --env-file .env.production --profile monitoring up -d
```

### 3. Verify Deployment
```bash
# Check all containers are running
docker-compose -f docker-compose.prod.yml ps

# Check logs for any errors
docker-compose -f docker-compose.prod.yml logs webhook-processor
docker-compose -f docker-compose.prod.yml logs nginx-proxy

# Test endpoints
curl https://your-domain.com/health
curl -H "X-API-Key: your-api-key" https://your-domain.com/webhook
```

### 4. Configure MailerCloud Webhooks
```bash
# Update MailerCloud webhook URLs to production domain
cd scripts/production
go run update_webhooks.go
```

## Post-Deployment Verification

### ✅ Health Checks
- [ ] Application responds at `https://your-domain.com/health`
- [ ] SSL certificate is valid and auto-renewing
- [ ] Webhook endpoint accepts requests at `https://your-domain.com/webhook`
- [ ] Database connectivity verified
- [ ] Message queue connectivity verified

### 📊 Monitoring (if enabled)
- [ ] Prometheus metrics accessible at `http://server-ip:9091`
- [ ] Grafana dashboards loading at `https://your-domain.com/grafana/`
- [ ] Webhook metrics being collected and displayed
- [ ] Alerts configured and functioning

### 🔒 Security Verification
- [ ] HTTP redirects to HTTPS
- [ ] Security headers present
- [ ] Rate limiting functional
- [ ] Metrics endpoint restricted to internal networks
- [ ] No sensitive data in logs

## Environment-Specific Configurations

### Production Optimizations
```yaml
# In docker-compose.prod.yml these are already configured:
- restart: unless-stopped         # Auto-restart containers
- healthcheck: enabled           # Container health monitoring  
- resource limits               # Memory and CPU limits
- security: non-root user      # Security hardening
- logging: structured JSON     # Production logging
```

### Performance Settings
```bash
# Nginx optimizations (already configured)
NGINX_WORKER_PROCESSES=auto
NGINX_WORKER_CONNECTIONS=1024
NGINX_WEBHOOK_RATE_LIMIT=50r/s
NGINX_API_RATE_LIMIT=500r/m

# Application timeouts
SERVER_READ_TIMEOUT=10s
SERVER_WRITE_TIMEOUT=15s
```

## Maintenance Commands

### Updates and Rollbacks
```bash
# Update to new version
docker-compose -f docker-compose.prod.yml --env-file .env.production pull
docker-compose -f docker-compose.prod.yml --env-file .env.production up -d

# View logs
docker-compose -f docker-compose.prod.yml logs -f webhook-processor

# Restart specific service
docker-compose -f docker-compose.prod.yml restart webhook-processor

# Scale workers (if needed)
docker-compose -f docker-compose.prod.yml up -d --scale webhook-worker=3
```

### Backup Important Data
```bash
# Export environment configuration
cp .env.production .env.production.backup

# MongoDB backup (using MongoDB Atlas built-in backups)
# CloudAMQP monitoring (using CloudAMQP console)

# SSL certificates backup (automatically handled by Let's Encrypt)
docker volume inspect webhook-processor-prod_certs
```

## Troubleshooting

### Common Issues
1. **SSL Certificate Issues**: Check DNS propagation and Let's Encrypt logs
2. **Database Connection**: Verify MongoDB Atlas IP whitelist and credentials
3. **Message Queue**: Check CloudAMQP connection and plan limits
4. **Rate Limiting**: Monitor nginx logs for rate limit triggers

### Debug Commands
```bash
# Check container status
docker-compose -f docker-compose.prod.yml ps

# View detailed logs
docker-compose -f docker-compose.prod.yml logs webhook-processor -f

# Test internal connectivity
docker-compose -f docker-compose.prod.yml exec webhook-processor wget -qO- http://localhost:8080/health

# Check SSL status
curl -I https://your-domain.com
```

## 🎉 Production Ready!

Your webhook processor is now deployed with:
- ✅ **Cloud-native architecture** (MongoDB Atlas + CloudAMQP)
- ✅ **SSL automation** with Let's Encrypt
- ✅ **Production monitoring** with Prometheus & Grafana
- ✅ **Security hardening** with rate limiting and headers
- ✅ **High availability** with health checks and auto-restart
- ✅ **Scalable infrastructure** ready for growth

**Monitor your deployment and scale as needed!** 🚀
