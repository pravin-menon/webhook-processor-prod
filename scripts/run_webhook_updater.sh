#!/bin/bash

# Script to run the webhook updater
# Usage: ./scripts/run_webhook_updater.sh

set -e

echo "Webhook Updater Script"
echo "======================"

# Check if ngrok is running
if curl -s http://localhost:4040/api/tunnels > /dev/null 2>&1; then
    echo "✅ ngrok is running - will update webhooks automatically"
    echo "Running webhook updater..."
    go run -mod=mod ./scripts/update_webhooks.go
else
    echo "❌ ngrok is not running on port 4040"
    echo ""
    echo "To fix this:"
    echo "1. Get your ngrok authtoken from https://ngrok.com"
    echo "2. Update NGROK_AUTHTOKEN in .env.development"
    echo "3. Restart Docker with ngrok profile:"
    echo "   docker-compose -f docker-compose.dev.yml --env-file .env.development --profile ngrok up -d"
    echo ""
    echo "Once ngrok is running, you can:"
    echo "- View ngrok dashboard: http://localhost:4040"
    echo "- Run this script again to update webhooks"
    exit 1
fi
