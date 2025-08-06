# 🚀 **Webhook Processor - Production Ready**

A cloud-native webhook processing system built with Go, designed for high-performance email event handling from MailerCloud with MongoDB Atlas and CloudAMQP integration.

## ✨ **Features**

### 🎯 **Core Functionality**
- **Multi-client webhook processing** with API key authentication
- **Real-time event streaming** via CloudAMQP message queues
- **MongoDB Atlas storage** with automatic scaling
- **Production-ready monitoring** with Prometheus & Grafana
- **SSL automation** with Let's Encrypt and nginx reverse proxy
- **Docker containerization** for easy deployment

### 🛡️ **Security & Performance**
- API key-based authentication for multiple clients
- Rate limiting and request validation
- Automatic SSL certificate management
- Health checks and graceful shutdowns
- Comprehensive error handling and retry logic

### 📊 **Monitoring & Observability**
- Prometheus metrics collection
- Grafana dashboards for visualization
- Structured JSON logging
- Real-time queue monitoring
- Custom alerting rules

## 🏗️ **Architecture**

```mermaid
graph TB
    MC[MailerCloud] -->|HTTPS POST| NX[nginx + SSL]
    NX --> WH[Webhook Handler]
    WH --> RMQ[CloudAMQP]
    RMQ --> WK[Worker Process]
    WK --> DB[MongoDB Atlas]
    
    WH --> PM[Prometheus Metrics]
    PM --> GF[Grafana Dashboards]
    
    subgraph "Cloud Services"
        RMQ
        DB
    end
    
    subgraph "Docker Containers"
        NX
        WH
        WK
        PM
        GF
    end
```

## 🚀 **Quick Start**

### **1. Prerequisites**
- Docker & Docker Compose
- MongoDB Atlas account
- CloudAMQP account
- Domain with DNS access (for production)

### **2. Environment Setup**
```bash
# Clone and setup
git clone <repository>
cd webhook-processor-prod

# Copy environment template
cp .env.example .env.development  # For development
cp .env.example .env.production   # For production

# Generate secure API keys
chmod +x scripts/generate_secrets.sh
./scripts/generate_secrets.sh
```

### **3. Configure Environment Variables**

#### **Required Settings:**
```bash
# MongoDB Atlas
MONGODB_URI=mongodb+srv://username:password@cluster.mongodb.net/?retryWrites=true&w=majority
MONGODB_DATABASE=webhook_events

# CloudAMQP
CLOUDAMQP_URL=amqps://username:password@host.cloudamqp.com/vhost

# Security
MAILERCLOUD_API_KEY=your-api-key-here

# Production Domain (production only)
DOMAIN=your-domain.com
LETSENCRYPT_EMAIL=your-email@domain.com
```

### **4. Development**
```bash
# Start development environment
docker-compose -f docker-compose.dev.yml --env-file .env.development up

# With monitoring
docker-compose -f docker-compose.dev.yml --profile monitoring up
```

### **5. Production Deployment**
```bash
# Deploy production stack
docker-compose -f docker-compose.prod.yml --env-file .env.production up -d

# With monitoring enabled
docker-compose -f docker-compose.prod.yml --profile monitoring up -d
```

## 📋 **Environment Reference**

### **🔧 Core Application**
| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `APP_ENV` | Environment mode | `development` | No |
| `APP_PORT` | Application port | `8080` | No |
| `LOG_LEVEL` | Logging level | `info` | No |

### **☁️ Cloud Services**
| Variable | Description | Example | Required |
|----------|-------------|---------|----------|
| `MONGODB_URI` | MongoDB Atlas connection | `mongodb+srv://...` | **Yes** |
| `CLOUDAMQP_URL` | CloudAMQP connection | `amqps://...` | **Yes** |
| `DOMAIN` | Production domain | `api.example.com` | Prod only |
| `LETSENCRYPT_EMAIL` | SSL certificate email | `admin@example.com` | Prod only |

### **🔐 Security**
| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `API_KEY_HEADER` | API key header name | `X-API-Key` | No |
| `MAILERCLOUD_API_KEY` | MailerCloud API key | - | **Yes** |

## 🎮 **Usage**

### **Webhook Configuration**
Configure MailerCloud to send webhooks to:
- **Development**: `https://your-ngrok-url.ngrok.io/webhook`
- **Production**: `https://your-domain.com/webhook`

### **API Endpoints**
| Endpoint | Method | Purpose | Auth Required |
|----------|--------|---------|---------------|
| `/webhook` | `POST` | Process webhook events | API Key |
| `/health` | `GET` | Health check | None |
| `/metrics` | `GET` | Prometheus metrics | IP restricted |

### **Webhook Scripts**
```bash
# Development - Update webhooks to ngrok URL
cd scripts && go run update_webhooks.go

# Production - Update webhooks to domain URL
cd scripts/production && go run update_webhooks.go
```

### **Monitoring Access**
| Service | Development | Production |
|---------|-------------|------------|
| Application | `http://localhost:8080` | `https://your-domain.com` |
| Metrics | `http://localhost:9090` | Internal only |
| Grafana | `http://localhost:3000` | `https://your-domain.com/grafana/` |

## 🔧 **Development Tools**

### **Debug Mode**
Enable webhook debugging to capture raw MailerCloud data:
```bash
# Enable debug mode
export WEBHOOK_DEBUG=true

# Use debug handler in router.go
handler := handlers.NewDebugMailerCloudWebhookHandler(logger, publisher)
```

### **Live Reloading**
Development containers use Air for automatic reloading:
- Main app: Watches Go files and restarts on changes
- Worker: Automatically rebuilds and restarts worker process

### **Local Testing**
```bash
# Health check
curl http://localhost:8080/health

# Send test webhook (with API key)
curl -H "X-API-Key: your-api-key" \
     -H "Content-Type: application/json" \
     -d '{"event":"test","email":"test@example.com"}' \
     http://localhost:8080/webhook
```

## 📊 **Monitoring**

### **Metrics Available**
- `webhook_requests_total` - Total webhook requests
- `webhook_processing_time` - Processing duration
- `webhook_queue_size` - Queue depth
- `webhook_retries_total` - Retry attempts

### **Grafana Dashboards**
- **Events Dashboard**: Real-time event processing metrics
- **System Dashboard**: Application performance and health
- **Queue Dashboard**: Message queue monitoring

### **Alerting**
Pre-configured alerts for:
- High error rates
- Queue depth thresholds
- Processing time degradation
- Service availability

## 🛠️ **Advanced Configuration**

### **Multi-Client Support**
Configure multiple MailerCloud accounts:
```bash
# Multiple API keys (comma-separated)
MAILERCLOUD_API_KEYS="client1:key1,client2:key2,client3:key3"

# Or individual keys
CLIENT1_API_KEY=your-client1-key
CLIENT2_API_KEY=your-client2-key
```

### **Performance Tuning**
```bash
# Nginx workers and connections
NGINX_WORKER_PROCESSES=auto
NGINX_WORKER_CONNECTIONS=1024

# Rate limiting
NGINX_WEBHOOK_RATE_LIMIT=10r/s
NGINX_API_RATE_LIMIT=100r/m

# RabbitMQ retry configuration
RABBITMQ_RETRY_COUNT=3
RABBITMQ_RETRY_DELAY=10s
RABBITMQ_MAX_RETRY_DELAY=300s
```

## 🚨 **Troubleshooting**

### **Common Issues**

#### **Connection Errors**
```bash
# Check MongoDB Atlas connectivity
docker-compose exec webhook-processor wget -qO- "your-mongodb-uri"

# Check CloudAMQP connectivity
docker-compose logs webhook-worker
```

#### **SSL Certificate Issues**
```bash
# Check Let's Encrypt status
docker-compose logs letsencrypt

# Verify domain DNS
nslookup your-domain.com
```

#### **Webhook Not Receiving Events**
1. Verify MailerCloud webhook URL configuration
2. Check API key authentication
3. Review nginx access logs
4. Verify firewall/security group settings

### **Log Analysis**
```bash
# Application logs
docker-compose logs webhook-processor

# Worker logs  
docker-compose logs webhook-worker

# Nginx logs
docker-compose logs nginx-proxy
```

## 📚 **Documentation**

Detailed documentation available in [`docs/`](./docs/):
- [Configuration Guide](./docs/CONFIG.md) - Detailed configuration options
- [Deployment Guide](./docs/DEPLOYMENT.md) - Production deployment steps
- [Development Guide](./docs/DEVELOPMENT.md) - Development setup and workflows
- [Monitoring Guide](./docs/MONITORING.md) - Monitoring and alerting setup
- [API Reference](./docs/API.md) - Complete API documentation
- [Troubleshooting](./docs/TROUBLESHOOTING.md) - Common issues and solutions

## 🤝 **Contributing**

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📄 **License**

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🎉 **Ready to Deploy!**

Your webhook processor is configured for:
- ✅ **MongoDB Atlas** integration
- ✅ **CloudAMQP** message queuing  
- ✅ **SSL automation** with Let's Encrypt
- ✅ **Production monitoring** with Prometheus/Grafana
- ✅ **Multi-client support** with API key authentication
- ✅ **Docker containerization** for easy deployment

**Start with development, then deploy to production with confidence!** 🚀
