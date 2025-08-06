#!/bin/bash

# Generate API Keys for different clients
MAILERCLOUD_API_KEY=$(openssl rand -hex 32)

echo "=== Webhook Processor Secrets Generator ==="
echo ""
echo "Generated secrets for your .env file:"
echo ""
echo "# API Keys"
echo "MAILERCLOUD_API_KEY=${MAILERCLOUD_API_KEY}"
echo ""
echo "# MongoDB Atlas Connection String Format:"
echo "# MONGODB_URI=mongodb+srv://username:password@cluster0.xxxxx.mongodb.net/?retryWrites=true&w=majority"
echo ""
echo "# CloudAMQP Connection String Format:"
echo "# CLOUDAMQP_URL=amqps://username:password@cougar.rmq.cloudamqp.com/vhost"
echo ""
echo "=== Configuration Notes ==="
echo "1. Replace placeholder values in MongoDB and CloudAMQP URLs with your actual credentials"
echo "2. Add additional client API keys using the pattern: CLIENT_NAME_API_KEY=generated_key"
echo "3. For production, use a proper secrets management system"
echo ""

# Create production environment template
cat > .env.production << EOL
# Application
APP_ENV=production
APP_PORT=8080
LOG_LEVEL=info

# MongoDB Atlas
MONGODB_URI=mongodb+srv://username:password@cluster0.xxxxx.mongodb.net/?retryWrites=true&w=majority
MONGODB_DATABASE=webhook_events
MONGODB_COLLECTION=events

# CloudAMQP (RabbitMQ in the cloud)
CLOUDAMQP_URL=amqps://username:password@cougar.rmq.cloudamqp.com/vhost
RABBITMQ_EXCHANGE=webhook_events
RABBITMQ_QUEUE=webhook_queue

# Security - API Keys for webhook authentication
API_KEY_HEADER=X-API-Key
MAILERCLOUD_API_KEY=${MAILERCLOUD_API_KEY}

# Monitoring
PROMETHEUS_PORT=9090
METRICS_PATH=/metrics

# Production Domain & SSL
DOMAIN=your-domain.com
LETSENCRYPT_EMAIL=your-email@domain.com

# Optional: Docker Registry
DOCKER_REGISTRY=your-registry.com/webhook-processor
TAG=latest

# Optional: Grafana Admin Password
GRAFANA_PASSWORD=secure-password-here
EOL

echo "Created .env.production template with generated API key."
echo "Update the MongoDB and CloudAMQP URLs with your actual credentials."
