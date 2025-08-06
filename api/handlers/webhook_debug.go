package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"webhook-processor/internal/mapping"
	"webhook-processor/internal/models"
	"webhook-processor/internal/queue"
	"webhook-processor/pkg/metrics"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type DebugMailerCloudWebhookHandler struct {
	logger        *zap.Logger
	publisher     queue.Publisher
	rateLimiter   *RateLimiter
	debugMode     bool
	webhookMapper *mapping.WebhookMappingService
}

type RawWebhookData struct {
	Timestamp time.Time              `json:"timestamp"`
	Method    string                 `json:"method"`
	Headers   map[string][]string    `json:"headers"`
	Body      map[string]interface{} `json:"body"`
	UserAgent string                 `json:"user_agent"`
	RemoteIP  string                 `json:"remote_ip"`
}

func NewDebugMailerCloudWebhookHandler(logger *zap.Logger, publisher queue.Publisher, webhookMapper *mapping.WebhookMappingService) *DebugMailerCloudWebhookHandler {
	debugMode := os.Getenv("WEBHOOK_DEBUG") == "true"
	return &DebugMailerCloudWebhookHandler{
		logger:        logger,
		publisher:     publisher,
		rateLimiter:   NewRateLimiter(),
		debugMode:     debugMode,
		webhookMapper: webhookMapper,
	}
}

func (h *DebugMailerCloudWebhookHandler) saveRawWebhookData(c *gin.Context, data map[string]interface{}) {
	if !h.debugMode {
		return
	}

	rawData := RawWebhookData{
		Timestamp: time.Now().UTC(),
		Method:    c.Request.Method,
		Headers:   c.Request.Header,
		Body:      data,
		UserAgent: c.GetHeader("User-Agent"),
		RemoteIP:  c.ClientIP(),
	}

	// Save to file for analysis
	filename := fmt.Sprintf("raw_webhook_data_%d.json", time.Now().UnixNano())
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		h.logger.Error("Failed to create debug file", zap.Error(err))
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(rawData); err != nil {
		h.logger.Error("Failed to write debug data", zap.Error(err))
	}

	// Also log detailed information
	h.logger.Info("=== RAW MAILERCLOUD WEBHOOK DATA ===",
		zap.String("timestamp", rawData.Timestamp.Format(time.RFC3339)),
		zap.String("method", rawData.Method),
		zap.String("user_agent", rawData.UserAgent),
		zap.String("remote_ip", rawData.RemoteIP),
		zap.Any("headers", rawData.Headers),
		zap.Any("body", rawData.Body),
	)
}

func (h *DebugMailerCloudWebhookHandler) analyzeClientIdentification(data map[string]interface{}) map[string]interface{} {
	analysis := map[string]interface{}{
		"potential_client_identifiers": []string{},
		"potential_unique_identifiers": []string{},
		"all_fields":                   []string{},
	}

	potentialClientFields := []string{
		"client_id", "customer_id", "account_id", "user_id", "tenant_id",
		"api_key", "auth_token", "organization_id", "workspace_id",
		"sender_id", "source_id", "domain_id", "brand_id",
	}

	potentialUniqueFields := []string{
		"event_id", "message_id", "webhook_id", "delivery_id", "tracking_id",
		"transaction_id", "uuid", "guid", "hash", "signature",
		"ts", "timestamp", "created_at", "sent_at", "delivered_at",
	}

	clientIdentifiers := []string{}
	uniqueIdentifiers := []string{}
	allFields := []string{}

	for key, value := range data {
		allFields = append(allFields, key)

		// Check for client identification fields
		for _, clientField := range potentialClientFields {
			if key == clientField {
				clientIdentifiers = append(clientIdentifiers, fmt.Sprintf("%s: %v", key, value))
			}
		}

		// Check for unique identification fields
		for _, uniqueField := range potentialUniqueFields {
			if key == uniqueField {
				uniqueIdentifiers = append(uniqueIdentifiers, fmt.Sprintf("%s: %v", key, value))
			}
		}
	}

	analysis["potential_client_identifiers"] = clientIdentifiers
	analysis["potential_unique_identifiers"] = uniqueIdentifiers
	analysis["all_fields"] = allFields

	return analysis
}

func (h *DebugMailerCloudWebhookHandler) HandleWebhook(c *gin.Context) {
	// Handle GET requests for URL validation
	if c.Request.Method == "GET" {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Debug Webhook endpoint is valid"})
		return
	}

	// Read the request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("Failed to read request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Reset the request body for further processing
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Parse the JSON data
	var data map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		h.logger.Error("Failed to parse webhook payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}

	// Save raw webhook data for analysis
	h.saveRawWebhookData(c, data)

	// Analyze potential client and unique identifiers
	analysis := h.analyzeClientIdentification(data)
	h.logger.Info("=== WEBHOOK DATA ANALYSIS ===", zap.Any("analysis", analysis))

	// For test requests from MailerCloud
	if c.Request.UserAgent() == "MailerCloud" {
		h.logger.Info("Handling MailerCloud test request")
		metrics.WebhookReceived.WithLabelValues("test", "verification").Inc()
		c.JSON(http.StatusOK, gin.H{
			"message": "Debug Webhook URL verified",
			"success": true,
		})
		return
	}

	// Extract client ID from multiple potential sources
	clientID := h.extractClientID(c, data)

	// Log client identification process
	h.logger.Info("=== CLIENT IDENTIFICATION ===",
		zap.String("extracted_client_id", clientID),
		zap.String("webhook_id_header", c.GetHeader("Webhook-Id")),
		zap.String("webhook_type_header", c.GetHeader("Webhook-Type")),
	)

	// Check rate limits
	if !h.rateLimiter.AllowRequest(clientID) {
		metrics.RateLimitExceeded.WithLabelValues(clientID, "requests").Inc()
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
		return
	}

	// Create webhook event with enhanced identification
	event := models.WebhookEvent{
		WebhookID:   h.generateWebhookID(data),
		WebhookType: "email_event",
		ClientID:    clientID,
		ReceivedAt:  time.Now().UTC(),
		Status:      string(models.EventStatusPending),
	}

	// Extract all available fields from the payload
	h.extractAllFields(&event, data)

	// Log extracted event for debugging
	h.logger.Info("=== EXTRACTED EVENT DATA ===",
		zap.String("webhook_id", event.WebhookID),
		zap.String("client_id", event.ClientID),
		zap.String("event", event.Event),
		zap.String("campaign_name", event.CampaignName),
		zap.String("campaign_id", event.CampaignID),
		zap.String("email", event.Email),
		zap.Int64("timestamp", event.Timestamp),
		zap.String("date_event", event.DateEvent),
	)

	// Record the received event metric
	metrics.WebhookReceived.WithLabelValues(event.ClientID, event.Event).Inc()

	// Send the event to the message queue
	if err := h.publisher.Publish(event); err != nil {
		metrics.WebhookProcessed.WithLabelValues(event.ClientID, event.Event, "failed").Inc()
		h.logger.Error("Failed to publish event", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process event"})
		return
	}

	metrics.WebhookProcessed.WithLabelValues(event.ClientID, event.Event, "success").Inc()

	c.JSON(http.StatusOK, gin.H{
		"message":    "Event accepted",
		"webhook_id": event.WebhookID,
		"client_id":  event.ClientID,
		"debug":      h.debugMode,
	})
}

func (h *DebugMailerCloudWebhookHandler) extractClientID(c *gin.Context, data map[string]interface{}) string {
	// Primary Strategy: Use Webhook-Id header to lookup client via mapping service
	webhookID := c.GetHeader("Webhook-Id")
	if webhookID != "" && h.webhookMapper != nil {
		h.logger.Info("Attempting to lookup client via webhook ID", zap.String("webhook_id", webhookID))

		if clientID, found := h.webhookMapper.GetClientForWebhook(webhookID); found {
			h.logger.Info("Successfully mapped webhook ID to client",
				zap.String("webhook_id", webhookID),
				zap.String("client_id", clientID))
			return clientID
		}

		h.logger.Warn("Webhook ID not found in mapping, falling back to webhook ID",
			zap.String("webhook_id", webhookID))
	}

	// Fallback 1: Use webhook ID as client identifier if available
	if webhookID != "" {
		h.logger.Info("Using webhook ID as client identifier", zap.String("webhook_id", webhookID))
		return webhookID
	}

	// Fallback 2: Check for explicit client identification fields in payload
	clientFields := []string{"client_id", "customer_id", "account_id", "user_id", "tenant_id", "sender_id"}
	for _, field := range clientFields {
		if val, ok := data[field].(string); ok && val != "" {
			h.logger.Info("Found client ID in payload", zap.String("field", field), zap.String("value", val))
			return val
		}
	}

	// Final fallback: Unknown client
	h.logger.Warn("No client identification available, using unknown client")
	return "unknown"
}

func (h *DebugMailerCloudWebhookHandler) generateWebhookID(data map[string]interface{}) string {
	// Strategy 1: Use existing webhook/message ID if available
	idFields := []string{"webhook_id", "message_id", "event_id", "delivery_id", "tracking_id"}
	for _, field := range idFields {
		if val, ok := data[field].(string); ok && val != "" {
			return val
		}
	}

	// Strategy 2: Generate based on combination of fields for uniqueness
	var components []string

	if val, ok := data["campaign_id"].(string); ok && val != "" {
		components = append(components, val)
	}
	if val, ok := data["email"].(string); ok && val != "" {
		components = append(components, val)
	}
	if val, ok := data["ts"].(float64); ok {
		components = append(components, fmt.Sprintf("%.0f", val))
	}
	if val, ok := data["event"].(string); ok && val != "" {
		components = append(components, val)
	}

	if len(components) > 0 {
		return fmt.Sprintf("mc_%x", components)
	}

	// Strategy 3: Fallback to timestamp-based ID
	return fmt.Sprintf("mc_%d", time.Now().UnixNano())
}

func (h *DebugMailerCloudWebhookHandler) extractAllFields(event *models.WebhookEvent, data map[string]interface{}) {
	// Extract standard fields with type assertions and error handling
	if val, ok := data["event"].(string); ok {
		event.Event = val
	}

	// Campaign name variations
	if val, ok := data["campaign_name"].(string); ok {
		event.CampaignName = val
	} else if val, ok := data["campaign name"].(string); ok {
		event.CampaignName = val
	}

	// Campaign ID variations
	if val, ok := data["campaign_id"].(string); ok {
		event.CampaignID = val
	} else if val, ok := data["camp_id"].(string); ok {
		event.CampaignID = val
	}

	// Tag name variations
	if val, ok := data["tag_name"].(string); ok {
		event.TagName = val
	} else if val, ok := data["tag"].(string); ok {
		event.TagName = val
	}

	if val, ok := data["date_event"].(string); ok {
		event.DateEvent = val
	}
	if val, ok := data["ts"].(float64); ok {
		event.Timestamp = int64(val)
	}
	if val, ok := data["ts_event"].(float64); ok {
		event.TimestampEvent = int64(val)
	}
	if val, ok := data["email"].(string); ok {
		event.Email = val
	}

	// URL field variations (for click events)
	if val, ok := data["URL"].(string); ok {
		event.URL = val
	} else if val, ok := data["url"].(string); ok {
		event.URL = val
	} else if val, ok := data["click_url"].(string); ok {
		event.URL = val
	}

	// Reason field (for bounce, spam, campaign_error events)
	if val, ok := data["reason"].(string); ok {
		event.Reason = val
	}

	// Handle list_id which can be string, number, or array (for unsubscribe events)
	if val, exists := data["list_id"]; exists {
		event.ListID = val
	}

	// Handle emails array
	if val, ok := data["emails"].([]interface{}); ok {
		emails := make([]string, 0, len(val))
		for _, email := range val {
			if emailStr, ok := email.(string); ok {
				emails = append(emails, emailStr)
			}
		}
		event.Emails = emails
	}

	// Event-specific field validation and logging
	h.logEventSpecificFields(event, data)
}

// logEventSpecificFields logs event-specific field validation and processing
func (h *DebugMailerCloudWebhookHandler) logEventSpecificFields(event *models.WebhookEvent, data map[string]interface{}) {
	eventType := strings.ToLower(event.Event)

	switch eventType {
	case "clicked", "click":
		if event.URL != "" {
			h.logger.Info("=== CLICK EVENT PROCESSING ===",
				zap.String("event", event.Event),
				zap.String("url", event.URL),
				zap.String("email", event.Email))
		} else {
			h.logger.Warn("CLICK EVENT MISSING URL",
				zap.String("event", event.Event),
				zap.Any("raw_data", data))
		}

	case "bounced", "bounce", "hard_bounce", "soft_bounce":
		if event.Reason != "" {
			h.logger.Info("=== BOUNCE EVENT PROCESSING ===",
				zap.String("event", event.Event),
				zap.String("reason", event.Reason),
				zap.String("email", event.Email))
		} else {
			h.logger.Warn("BOUNCE EVENT MISSING REASON",
				zap.String("event", event.Event),
				zap.Any("raw_data", data))
		}

	case "spam":
		if event.Reason != "" {
			h.logger.Info("=== SPAM EVENT PROCESSING ===",
				zap.String("event", event.Event),
				zap.String("reason", event.Reason),
				zap.String("email", event.Email))
		} else {
			h.logger.Warn("SPAM EVENT MISSING REASON",
				zap.String("event", event.Event),
				zap.Any("raw_data", data))
		}

	case "campaign_error":
		if event.Reason != "" {
			h.logger.Info("=== CAMPAIGN ERROR PROCESSING ===",
				zap.String("event", event.Event),
				zap.String("reason", event.Reason),
				zap.String("campaign_id", event.CampaignID))
		} else {
			h.logger.Warn("CAMPAIGN ERROR MISSING REASON",
				zap.String("event", event.Event),
				zap.Any("raw_data", data))
		}

	case "unsubscribe", "unsubscribed":
		if event.ListID != nil {
			h.logger.Info("=== UNSUBSCRIBE EVENT PROCESSING ===",
				zap.String("event", event.Event),
				zap.Any("list_id", event.ListID),
				zap.String("email", event.Email))
		} else {
			h.logger.Warn("UNSUBSCRIBE EVENT MISSING LIST_ID",
				zap.String("event", event.Event),
				zap.Any("raw_data", data))
		}

	default:
		h.logger.Info("=== STANDARD EVENT PROCESSING ===",
			zap.String("event", event.Event),
			zap.String("email", event.Email),
			zap.String("campaign_id", event.CampaignID))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
