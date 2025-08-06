# Configuration Guide

This guide provides comprehensive configuration details for deploying the Webhook Processor in both development and production environments using cloud services.

## 🔧 Environment Configuration

### Required Cloud Services

1. **MongoDB Atlas** - Managed MongoDB hosting
2. **CloudAMQP** - Managed RabbitMQ hosting  
3. **Domain with SSL** - For production deployment

### Core Environment Variables

```bash
# Application Settings
APP_ENV=production
APP_PORT=8080
LOG_LEVEL=info

# MongoDB Atlas Configuration
MONGODB_URI=mongodb+srv://username:password@cluster0.xxxxx.mongodb.net/?retryWrites=true&w=majority
MONGODB_DATABASE=webhook_events
MONGODB_COLLECTION=events

# CloudAMQP Configuration
CLOUDAMQP_URL=amqps://username:password@cougar.rmq.cloudamqp.com/vhost
RABBITMQ_EXCHANGE=webhook_events
RABBITMQ_QUEUE=webhook_queue

# Security Configuration
API_KEY_HEADER=X-API-Key
MAILERCLOUD_API_KEY=your-generated-api-key

# Production Domain & SSL
DOMAIN=your-domain.com
LETSENCRYPT_EMAIL=your-email@domain.com

# Monitoring
PROMETHEUS_PORT=9090
METRICS_PATH=/metrics

# Optional: Docker Registry
DOCKER_REGISTRY=your-registry.com/webhook-processor
TAG=latest
```

## 🛠️ Development Setup

### Prerequisites
- Docker and Docker Compose
- Git
- Text editor

### Local Development with Cloud Services

```bash
# 1. Clone and setup
git clone <repository-url>
cd webhook-processor-prod
cp .env.example .env

# 2. Generate API keys
chmod +x scripts/generate_secrets.sh
./scripts/generate_secrets.sh

# 3. Configure MongoDB Atlas
# - Create cluster at https://cloud.mongodb.com
# - Get connection string and update MONGODB_URI in .env

# 4. Configure CloudAMQP
# - Create instance at https://cloudamqp.com
# - Get AMQP URL and update CLOUDAMQP_URL in .env

# 5. Start development environment
docker-compose -f docker-compose.dev.yml up --build -d

# 6. For webhook testing with ngrok
docker-compose -f docker-compose.dev.yml --profile ngrok up -d
```

### Development Services

| Service | URL | Purpose |
|---------|-----|---------|
| Webhook API | http://localhost:8080 | Main application |
| Prometheus | http://localhost:9091 | Metrics collection |
| Grafana | http://localhost:3000 | Metrics visualization |
| ngrok Web UI | http://localhost:4040 | Tunnel management |

## 🏭 Production Deployment

### Server Prerequisites

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install docker.io docker-compose git

# CentOS/RHEL
sudo yum install docker docker-compose git
sudo systemctl start docker
sudo systemctl enable docker
```

### Production Deployment Steps

1. **Server Setup**:
   ```bash
   # Clone repository
   git clone <repository-url>
   cd webhook-processor-prod
   
   # Setup environment
   cp .env.example .env.production
   ./scripts/generate_secrets.sh
   ```

2. **Configure Environment**:
   Edit `.env.production` with your actual values:
   ```bash
   # Update these with your actual cloud service credentials
   MONGODB_URI=mongodb+srv://user:pass@cluster.mongodb.net/...
   CLOUDAMQP_URL=amqps://user:pass@instance.cloudamqp.com/vhost
   MAILERCLOUD_API_KEY=your-actual-api-key
   DOMAIN=webhooks.yourcompany.com
   LETSENCRYPT_EMAIL=admin@yourcompany.com
   ```

3. **Deploy Application**:
   ```bash
   # Production with monitoring
   docker-compose -f docker-compose.prod.yml --profile monitoring --env-file .env.production up -d
   
   # Production without monitoring
   docker-compose -f docker-compose.prod.yml --env-file .env.production up -d
   ```

4. **Verify Deployment**:
   ```bash
   # Check service status
   docker-compose -f docker-compose.prod.yml ps
   
   # Check logs
   docker-compose -f docker-compose.prod.yml logs webhook-processor
   
   # Test health endpoint
   curl https://your-domain.com/health
   ```

## 🔒 Security Configuration

### SSL/TLS Setup

The application automatically handles SSL certificates using Let's Encrypt:

```yaml
# Automatic SSL configuration via docker-compose
letsencrypt:
  image: nginxproxy/acme-companion:latest
  environment:
    - DEFAULT_EMAIL=${LETSENCRYPT_EMAIL}
```

### Nginx Security Configuration

Located in `nginx/custom.conf`:

```nginx
# Rate limiting
limit_req_zone $binary_remote_addr zone=webhook:10m rate=10r/s;

# Security headers
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
add_header X-Frame-Options DENY;
add_header X-Content-Type-Options nosniff;

# Webhook endpoint protection
location /webhook {
    limit_req zone=webhook burst=5 nodelay;
}
```

### API Key Management

```bash
# Generate new API key for a client
openssl rand -hex 32

# Add to environment variables
NEWCLIENT_API_KEY=generated-key-here

# Restart services to apply
docker-compose restart webhook-processor
```

## 🔧 Service Configuration

### MongoDB Atlas Setup

1. **Create Cluster**:
   - Go to https://cloud.mongodb.com
   - Create new cluster
   - Choose region closest to your servers

2. **Database User**:
   - Create database user with read/write permissions
   - Use strong password

3. **Network Access**:
   - Add your server IP to whitelist
   - Or use 0.0.0.0/0 for all IPs (less secure)

4. **Connection String**:
   ```
   mongodb+srv://username:password@cluster0.xxxxx.mongodb.net/?retryWrites=true&w=majority
   ```

### CloudAMQP Setup

1. **Create Instance**:
   - Go to https://cloudamqp.com
   - Choose plan (Little Lemur for development, larger for production)
   - Select region

2. **Connection Details**:
   - Get AMQP URL from dashboard
   - Format: `amqps://username:password@host.cloudamqp.com/vhost`

3. **Queue Configuration**:
   - Queues are created automatically by the application
   - Monitor via CloudAMQP dashboard

## 📊 Monitoring Configuration

### Prometheus Setup

Configuration in `monitoring/prometheus/prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'webhook-processor'
    static_configs:
      - targets: ['webhook-processor:9090']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

### Grafana Setup

Default credentials:
- Username: `admin`
- Password: `admin` (change in production)

Dashboard configuration:
- Automatic provisioning from `monitoring/grafana/provisioning/`
- Dashboards for events, performance, and errors
- Alerts for high error rates and queue backlog

### Available Dashboards

1. **Events Dashboard**:
   - Real-time event processing
   - Success/failure rates
   - Processing latency

2. **Performance Dashboard**:
   - Request throughput
   - Response times
   - Resource utilization

3. **Error Dashboard**:
   - Error rates by client
   - Failed events
   - Retry statistics

## 🐳 Container Configuration

### Resource Limits

For production, add resource limits to docker-compose.prod.yml:

```yaml
webhook-processor:
  deploy:
    resources:
      limits:
        cpus: '1.0'
        memory: 1G
      reservations:
        cpus: '0.5'
        memory: 512M
```

### Health Checks

All services include health checks:

```yaml
healthcheck:
  test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
  interval: 30s
  timeout: 10s
  retries: 3
```

## 🔄 Backup & Recovery

### Database Backup

```bash
# MongoDB Atlas automatic backups are included
# For manual backup:
mongodump --uri="mongodb+srv://..." --out=/backup/$(date +%Y%m%d)
```

### Configuration Backup

```bash
# Backup environment files
tar -czf config-backup-$(date +%Y%m%d).tar.gz .env.production nginx/
```

## 🚨 Troubleshooting

### Common Issues

1. **SSL Certificate Errors**:
   ```bash
   # Check certificate status
   docker-compose logs letsencrypt
   
   # Force certificate renewal
   docker-compose exec letsencrypt certbot renew --force-renewal
   ```

2. **Database Connection Issues**:
   ```bash
   # Test MongoDB connection
   docker-compose exec webhook-processor wget -qO- http://localhost:9090/metrics | grep mongodb_up
   ```

3. **Queue Connection Issues**:
   ```bash
   # Check RabbitMQ status
   docker-compose logs webhook-processor | grep "rabbitmq"
   ```

### Performance Tuning

1. **Database Performance**:
   - Enable MongoDB Atlas auto-scaling
   - Create appropriate indexes
   - Monitor slow queries

2. **Queue Performance**:
   - Upgrade CloudAMQP plan if needed
   - Adjust worker concurrency
   - Monitor queue depth

3. **Application Performance**:
   - Increase container resources
   - Adjust rate limits
   - Enable HTTP/2

### Scaling

1. **Horizontal Scaling**:
   ```yaml
   webhook-processor:
     deploy:
       replicas: 3
   ```

2. **Database Scaling**:
   - MongoDB Atlas auto-scaling
   - Read replicas for analytics

3. **Queue Scaling**:
   - CloudAMQP plan upgrades
   - Multiple worker instances

## 📞 Support Contacts

- **MongoDB Atlas**: https://support.mongodb.com
- **CloudAMQP**: https://support.cloudamqp.com
- **Let's Encrypt**: https://letsencrypt.org/docs/
- **Nginx**: https://nginx.org/en/docs/
