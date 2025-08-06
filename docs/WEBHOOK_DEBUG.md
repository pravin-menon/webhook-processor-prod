# Webhook Debug Instructions

## Overview
This document explains how to capture and analyze raw MailerCloud webhook data to understand client identification and event deduplication strategies.

## Debug Mode Setup

### 1. Enable Debug Mode
Set the environment variable to enable detailed webhook logging:
```bash
export WEBHOOK_DEBUG=true
```

### 2. Use Debug Handler
Replace the standard webhook handler with the debug version in your router configuration.

### 3. Monitor Raw Data
Debug mode will:
- Save raw webhook data to timestamped JSON files
- Log detailed analysis of potential client identifiers
- Log extracted event data for verification
- Analyze all incoming fields for pattern recognition

## Raw Data Collection

### Debug Files
Raw webhook data will be saved to:
```
raw_webhook_data_[timestamp].json
```

### Log Analysis
Look for these log entries:
```
=== RAW MAILERCLOUD WEBHOOK DATA ===
=== WEBHOOK DATA ANALYSIS ===
=== EXTRACTED EVENT DATA ===
```

## Client Identification Strategies

The debug handler tests multiple strategies to identify clients:

### Strategy 1: Authorization Header
- Checks for API key in `Authorization` header
- Maps API keys to client IDs
- Most reliable for multi-tenant scenarios

### Strategy 2: Payload Fields
Searches for client identification in these fields:
- `client_id`
- `customer_id` 
- `account_id`
- `user_id`
- `tenant_id`
- `sender_id`

### Strategy 3: Domain-Based
- Extracts domain from email addresses
- Useful when clients are domain-based

### Strategy 4: IP-Based (Fallback)
- Uses remote IP address
- Last resort identification method

## Event Deduplication

### Unique ID Generation
The debug handler tests these approaches:

### Strategy 1: Existing IDs
Looks for existing unique identifiers:
- `webhook_id`
- `message_id`
- `event_id`
- `delivery_id`
- `tracking_id`

### Strategy 2: Composite Keys
Generates unique IDs from combinations:
- `campaign_id` + `email` + `timestamp` + `event`
- Ensures uniqueness across similar events

### Strategy 3: Timestamp Fallback
- Uses nanosecond timestamp
- Guarantees uniqueness but not idempotency

## Fields to Analyze

### Standard MailerCloud Fields
Expected fields in webhook payloads:
- `event`: Event type (sent, delivered, opened, clicked, etc.)
- `campaign_name`: Human-readable campaign name
- `campaign_id` / `camp_id`: Campaign identifier
- `tag_name`: Campaign tag/category
- `date_event`: Event date string
- `ts`: Unix timestamp
- `ts_event`: Event-specific timestamp
- `email`: Recipient email address
- `url`: Clicked URL (for click events)
- `reason`: Bounce/unsubscribe reason
- `list_id`: Mailing list identifier
- `emails`: Array of emails (for bulk events)

### Client-Specific Fields
Look for additional fields that might indicate:
- Account/client ownership
- API key associations
- Sender domain information
- Custom tracking parameters

## Usage Instructions

### 1. Deploy Debug Version
```bash
# In your router setup, replace:
# handler := handlers.NewMailerCloudWebhookHandler(logger, publisher)
# with:
handler := handlers.NewDebugMailerCloudWebhookHandler(logger, publisher)
```

### 2. Trigger Test Webhooks
- Use MailerCloud's webhook testing feature
- Send actual campaigns to generate real data
- Monitor multiple client accounts if available

### 3. Analyze Results
- Review generated JSON files. Find all debug JSON files in the webhook processor container:
docker exec webhook-processor-prod-webhook-processor-1 find /app -name "raw_webhook_data_*.json" -type f
docker exec webhook-processor-prod-webhook-processor-1 ls -la /app/raw_webhook_data_*.json
- Copy first debug file to host
docker cp webhook-processor-prod-webhook-processor-1:/app/raw_webhook_data_1754415031340068555.json ./debug_files/webhook_debug_1.json
- Check logs for identification patterns
docker logs webhook-processor-prod-webhook-processor-1 | grep -A 20 -B 5 "RAW MAILERCLOUD WEBHOOK"
docker logs webhook-processor-prod-webhook-processor-1 | grep -A 10 "EXTRACTED EVENT DATA" | tail -15 
docker logs webhook-processor-prod-webhook-processor-1 | grep -A 10 "WEBHOOK DATA ANALYSIS" | tail -15 
- Look for consistent unique identifiers
- Verify client segregation methods
🎉 Debug File Setup Complete!
📁 Your Debug Files Are Ready:
✅ Location: debug_files

webhook_debug_1.json - First test webhook (MailerCloud User-Agent)
webhook_debug_2.json - Second test webhook (realistic payload)
webhook_debug_3.json - Third test webhook (permission test)
🔧 Tools Available:
✅ Analysis Script: analyze_debug_files.sh

Comprehensive analysis of all debug files
Client identification strategy recommendations
Field mapping and pattern analysis
✅ Monitor Script: monitor_debug_files.sh

Real-time monitoring for new webhook files
Automatic copying and organization
Run in background to capture live data
✅ Status Check: check_debug_status.sh

Quick check for new files
Sync status between container and local
Recent activity summary
🎯 Key Client Identification Findings:
✅ PRIMARY Strategy: API Key Mapping

Currently Working: Your X-API-Key header maps to clients
Reliability: HIGH ✅
Implementation: Already in place
✅ FALLBACK Strategy: Email Domain

Pattern: Extract domain from email field (user@client-domain.com → domain_client-domain.com)
Reliability: MEDIUM
Use Case: When API key mapping isn't sufficient
📝 To Continue Monitoring:
Keep debug mode running - webhooks will generate new JSON files
Run monitoring: monitor_debug_files.sh in a separate terminal
Periodic analysis: analyze_debug_files.sh to review patterns
Quick checks: check_debug_status.sh for status updates
🚀 What to Look For in Real MailerCloud Data:
Different User-Agent patterns (currently seeing Go-http-client/2.0)
Additional payload fields that might identify clients
Campaign ID patterns that could be client-specific
URL patterns or tracking parameters
Custom headers from MailerCloud
Your debug setup is perfect for capturing and analyzing real webhook patterns! 🎯

### 4. Update Production Code
Based on findings, update the production webhook handler:
- Implement reliable client identification
- Add proper deduplication logic
- Enhance event processing based on actual field structure

## Security Considerations

### Debug Mode Restrictions
- Only enable in development/testing environments
- Raw webhook data may contain sensitive information
- Disable debug mode in production
- Clean up debug files regularly

### Data Privacy
- Raw webhook files may contain email addresses
- Consider data retention policies
- Implement proper access controls
- Anonymize data when possible

## Expected Outcomes

After running debug mode, you should have:
1. **Complete field mapping** of MailerCloud webhook structure
2. **Client identification strategy** based on actual data
3. **Deduplication approach** using reliable unique identifiers
4. **Event processing optimization** based on field analysis
5. **Multi-tenant support** validation for different clients

## Production Implementation

Use debug findings to enhance:
- `api/handlers/webhook.go`: Update client ID extraction
- `internal/models/webhook.go`: Add missing fields
- `internal/worker/processor.go`: Improve deduplication
- Authentication middleware for API key mapping
