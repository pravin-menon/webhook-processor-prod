# 🚀 **Deployment Guide - Cloud-Native Setup**

This guide covers deploying the webhook processor using **MongoDB Atlas** and **CloudAMQP** for both development and production environments.

## 🏗️ **Architecture Overview**

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   MailerCloud   │────▶│  Your Domain +   │────▶│ Docker Compose  │
│   Webhooks      │    │  nginx + SSL     │    │  Application    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                                         │
                       ┌─────────────────┐               │
                       │  CloudAMQP      │◀──────────────┤
                       │  (RabbitMQ)     │               │
                       └─────────────────┘               │
                                                         │
                       ┌─────────────────┐               │
                       │  MongoDB Atlas  │◀──────────────┘
                       │  (Database)     │
                       └─────────────────┘
```

## ☁️ **Cloud Services Setup**

### **1. MongoDB Atlas Configuration**

#### **Create Cluster**
1. Sign up at [MongoDB Atlas](https://cloud.mongodb.com/)
2. Create a new cluster (free tier available)
3. Configure database access:
   - Create database user with read/write permissions
   - Whitelist your deployment IP addresses (or 0.0.0.0/0 for development)

#### **Get Connection String**
```bash
# Example connection string format:
MONGODB_URI=mongodb+srv://username:password@cluster0.xxxxx.mongodb.net/?retryWrites=true&w=majority
```

#### **Database Setup**
The application will automatically create:
- Database: `webhook_events` (configurable)
- Collection: `events` (configurable)
- Indexes: Created automatically on first run

### **2. CloudAMQP Configuration**

#### **Create Instance**
1. Sign up at [CloudAMQP](https://cloudamqp.com/)
2. Create a new instance (free tier available)
3. Note the connection details from the dashboard

#### **Get Connection String**
```bash
# Example connection string format:
CLOUDAMQP_URL=amqps://username:password@host.cloudamqp.com/vhost
```

#### **Exchange & Queue Setup**
The application will automatically create:
- Exchange: `webhook_events` (configurable)
- Queue: `webhook_queue` (configurable)
- Bindings: Automatically configured

## 🔧 **Environment Configuration**

### **Development Environment**

#### **Create `.env.development`**
```bash
# Copy template
cp .env.example .env.development

# Edit with your values
APP_ENV=development
APP_PORT=8080
LOG_LEVEL=debug
WEBHOOK_DEBUG=true

# MongoDB Atlas
MONGODB_URI=mongodb+srv://dev-user:password@dev-cluster.xxxxx.mongodb.net/?retryWrites=true&w=majority
MONGODB_DATABASE=webhook_events_dev
MONGODB_COLLECTION=events

# CloudAMQP  
CLOUDAMQP_URL=amqps://dev-user:password@dev-host.cloudamqp.com/dev-vhost
RABBITMQ_EXCHANGE=webhook_events_dev
RABBITMQ_QUEUE=webhook_queue_dev

# Security
MAILERCLOUD_API_KEY=your-development-api-key

# Development Monitoring
GRAFANA_DEV_USER=admin
GRAFANA_DEV_PASSWORD=admin
```

#### **Start Development**
```bash
# Start core services
docker-compose -f docker-compose.dev.yml --env-file .env.development up

# Start with monitoring
docker-compose -f docker-compose.dev.yml --env-file .env.development --profile monitoring up

# Background mode
docker-compose -f docker-compose.dev.yml --env-file .env.development up -d
```

### **Production Environment**

#### **Create `.env.production`**
```bash
# Copy template
cp .env.example .env.production

# Edit with your values
APP_ENV=production
LOG_LEVEL=info
WEBHOOK_DEBUG=false

# Domain & SSL
DOMAIN=api.yourdomain.com
LETSENCRYPT_EMAIL=admin@yourdomain.com

# MongoDB Atlas (Production Cluster)
MONGODB_URI=mongodb+srv://prod-user:secure-password@prod-cluster.xxxxx.mongodb.net/?retryWrites=true&w=majority
MONGODB_DATABASE=webhook_events_prod
MONGODB_COLLECTION=events

# CloudAMQP (Production Instance)
CLOUDAMQP_URL=amqps://prod-user:secure-password@prod-host.cloudamqp.com/prod-vhost
RABBITMQ_EXCHANGE=webhook_events_prod
RABBITMQ_QUEUE=webhook_queue_prod

# Security
MAILERCLOUD_API_KEY=your-production-api-key

# Monitoring
GRAFANA_PASSWORD=your-secure-grafana-password

# Docker Registry (if using private registry)
DOCKER_REGISTRY=your-registry.com/webhook-processor
TAG=v1.0.0
```

#### **DNS Configuration**
Point your domain to your server:
```bash
# A record example
api.yourdomain.com.    300    IN    A    your.server.ip
```

#### **Deploy Production**
```bash
# Deploy core services
docker-compose -f docker-compose.prod.yml --env-file .env.production up -d

# Deploy with monitoring
docker-compose -f docker-compose.prod.yml --env-file .env.production --profile monitoring up -d

# Check status
docker-compose -f docker-compose.prod.yml ps
```

## 🔐 **Security Configuration**

### **API Key Management**

#### **Single Client**
```bash
MAILERCLOUD_API_KEY=your-single-api-key
```

#### **Multiple Clients**
```bash
# Comma-separated format
MAILERCLOUD_API_KEYS="client1:api-key-1,client2:api-key-2,client3:api-key-3"

# Or individual environment variables
CLIENT1_API_KEY=api-key-1
CLIENT2_API_KEY=api-key-2
CLIENT3_API_KEY=api-key-3
```

### **SSL Certificate Management**

Let's Encrypt certificates are automatically:
- **Requested** on first deployment
- **Renewed** before expiration
- **Stored** in Docker volumes

#### **Manual Certificate Check**
```bash
# Check certificate status
docker-compose -f docker-compose.prod.yml logs letsencrypt

# View certificate details
docker-compose -f docker-compose.prod.yml exec nginx-proxy \
  openssl x509 -in /etc/nginx/certs/yourdomain.com.crt -text -noout
```

## 📊 **Monitoring Setup**

### **Development Monitoring**
Access points when monitoring profile is enabled:
- **Application**: http://localhost:8080
- **Prometheus**: http://localhost:9091  
- **Grafana**: http://localhost:3000 (admin/admin)

### **Production Monitoring**
Access points when monitoring profile is enabled:
- **Application**: https://yourdomain.com
- **Prometheus**: Internal only (port 9091)
- **Grafana**: https://yourdomain.com/grafana/ (admin/your-password)

### **Grafana Setup**
1. Access Grafana dashboard
2. Default dashboards are pre-configured
3. Data sources are automatically configured
4. Custom alerts can be added via the UI

## 🧪 **Testing Deployment**

### **Health Checks**
```bash
# Development
curl http://localhost:8080/health

# Production  
curl https://yourdomain.com/health
```

### **Webhook Testing**
```bash
# Test webhook endpoint with API key
curl -H "X-API-Key: your-api-key" \
     -H "Content-Type: application/json" \
     -d '{"event":"test","email":"test@example.com","campaign_id":"test-123"}' \
     https://yourdomain.com/webhook
```

### **Service Verification**
```bash
# Check all containers are running
docker-compose -f docker-compose.prod.yml ps

# View logs
docker-compose -f docker-compose.prod.yml logs webhook-processor
docker-compose -f docker-compose.prod.yml logs webhook-worker

# Check metrics
curl https://yourdomain.com/metrics
```

## 🚀 **MailerCloud Integration**

### **Webhook URL Configuration**

#### **Development with ngrok**
```bash
# Start ngrok tunnel
ngrok http 8080

# Use the provided HTTPS URL in MailerCloud
# Example: https://abc123.ngrok.io/webhook
```

#### **Update Development Webhooks**
```bash
cd scripts
export MAILERCLOUD_API_KEYS="your-client:your-api-key"
go run update_webhooks.go
```

#### **Production Webhook Setup**
```bash
# Configure MailerCloud webhooks to:
https://yourdomain.com/webhook

# Update production webhooks
cd scripts/production
export DOMAIN=yourdomain.com
export MAILERCLOUD_API_KEYS="your-client:your-api-key"
go run update_webhooks.go
```

## 🔄 **Maintenance & Updates**

### **Application Updates**
```bash
# Pull latest changes
git pull origin main

# Rebuild and redeploy
docker-compose -f docker-compose.prod.yml build
docker-compose -f docker-compose.prod.yml up -d
```

### **Scaling Workers**
```bash
# Scale worker processes
docker-compose -f docker-compose.prod.yml up -d --scale webhook-worker=3
```

### **Backup Considerations**

#### **MongoDB Atlas**
- Automatic backups included in Atlas
- Point-in-time recovery available
- Manual exports can be configured

#### **Configuration Backup**
```bash
# Backup environment and configs
tar -czf webhook-backup-$(date +%Y%m%d).tar.gz \
  .env.production \
  nginx/ \
  monitoring/ \
  docker-compose.prod.yml
```

## 🚨 **Troubleshooting**

### **Common Issues**

#### **SSL Certificate Problems**
```bash
# Check Let's Encrypt logs
docker-compose -f docker-compose.prod.yml logs letsencrypt

# Verify DNS propagation
nslookup yourdomain.com
dig yourdomain.com

# Force certificate renewal
docker-compose -f docker-compose.prod.yml restart letsencrypt
```

#### **Database Connection Issues**
```bash
# Test MongoDB Atlas connectivity
docker run --rm mongo:latest mongosh "your-mongodb-uri" --eval "db.adminCommand('ping')"

# Check application logs
docker-compose -f docker-compose.prod.yml logs webhook-processor | grep -i mongo
```

#### **Queue Connection Issues**
```bash
# Test CloudAMQP connectivity
docker-compose -f docker-compose.prod.yml logs webhook-worker | grep -i rabbit

# Check queue status in CloudAMQP dashboard
# Monitor queue depth and connection status
```

### **Performance Monitoring**
```bash
# Check resource usage
docker stats

# Monitor queue depth
curl -s https://yourdomain.com/metrics | grep webhook_queue

# Check processing times
curl -s https://yourdomain.com/metrics | grep webhook_processing_time
```

## ✅ **Deployment Checklist**

### **Pre-Deployment**
- [ ] MongoDB Atlas cluster created and configured
- [ ] CloudAMQP instance created and configured
- [ ] Domain DNS configured
- [ ] Environment variables set
- [ ] API keys configured
- [ ] SSL email configured

### **Post-Deployment**
- [ ] Health check responds
- [ ] SSL certificate issued
- [ ] Webhook endpoint accessible
- [ ] Monitoring dashboards loading
- [ ] MailerCloud webhooks updated
- [ ] Test webhook events processed

### **Production Readiness**
- [ ] Monitoring and alerting configured
- [ ] Backup procedures documented
- [ ] Log retention configured
- [ ] Security review completed
- [ ] Performance testing completed

---

## 🎯 **Next Steps**

1. **Complete cloud service setup** (MongoDB Atlas + CloudAMQP)
2. **Deploy development environment** for testing
3. **Configure MailerCloud webhooks** for development
4. **Test webhook processing end-to-end**
5. **Deploy production environment** with your domain
6. **Update production webhooks** and go live!

Your webhook processor is now ready for cloud-native deployment! 🚀

## **Run update webhooks Go script**
- go run -mod=mod ./scripts/update_webhooks.go

## **Network "Resource is still in use" Issue**

When `docker-compose down` shows "Network resource is still in use", it's usually because:
1. **Profile containers still running**: ngrok or other profile-based containers weren't stopped
2. **Orphaned containers**: Containers not managed by current compose command

### **Solutions:**
```bash
# Method 1: Use same profile for down (Recommended)
docker-compose -f docker-compose.dev.yml --env-file .env.development --profile ngrok down

# Method 2: Remove orphaned containers
docker-compose -f docker-compose.dev.yml --env-file .env.development down --remove-orphans

# Method 3: Check what's using the network
docker network inspect webhook-processor-prod_webhook-dev
docker stop $(docker ps -q)  # Stop orphaned containers
docker-compose down          # Then run normal down
```

## Moderate Force:
# Force kill containers with compose
docker-compose -f docker-compose.dev.yml --env-file .env.development kill

# Force kill all running containers
docker kill $(docker ps -q)

# Force remove specific container
docker rm -f <container_id>

## Strong Force:
# Force remove all containers
docker rm -f $(docker ps -aq)

# Force remove all networks
docker network prune -f

# Force remove all volumes
docker volume prune -f

## Nuclear option:
# Remove everything (containers, networks, volumes, images, build cache)
docker system prune -af --volumes

# If Docker daemon is completely stuck, restart Docker Desktop
# On Mac: Docker Desktop > Restart
# On Linux: sudo systemctl restart docker

## When Docker Daemon is Completely Stuck:
# On Mac with Docker Desktop
pkill Docker

# On Linux
sudo systemctl stop docker
sudo systemctl start docker

## Methods to Refresh ngrok URLs Without Restarting Container

**Method 1**: ngrok API (Fastest & Most Efficient)

# Delete existing tunnel
curl -X DELETE http://localhost:4040/api/tunnels/webhook
# Create new tunnel (gets new URL automatically)
curl -X POST http://localhost:4040/api/tunnels \  -H "Content-Type: application/json" \  -d '{    "name": "webhook",    "addr": "webhook-processor:8080",     "proto": "http"  }'
# Check new URL
curl -s http://localhost:4040/api/tunnels | jq '.tunnels[] | {name: .name, public_url: .public_url}'

**Method 2**: Restart ngrok Container (Quick)
# Restart just the ngrok service
docker-compose -f docker-compose.dev.yml --env-file .env.development restart ngrok

# Wait a moment and check new URL
sleep 3 && curl -s http://localhost:4040/api/tunnels | jq '.tunnels[] | {name: .name, public_url: .public_url}'

**Method 3**: Create Additional Tunnels - You can also create multiple tunnels with different names:

curl -X POST http://localhost:4040/api/tunnels \  -H "Content-Type: application/json" \  -d '{    "name": "webhook-new",    "addr": "webhook-processor:8080",    "proto": "http"  }'

**Advantages of Each Method**:
- API Method: Fastest, no container restart, keeps other services running
- Container Restart: Simple, clean restart, preserves configuration
- Multiple Tunnels: Allows testing multiple URLs simultaneously

**Configure Local DNS Resolution for nginx dev setup:** - You'll need to add the development domain to your local hosts file:
echo "127.0.0.1 webhook-dev.local" | sudo tee -a /etc/hosts

docker-compose -f docker-compose.dev.yml --env-file .env.development --profile nginx down