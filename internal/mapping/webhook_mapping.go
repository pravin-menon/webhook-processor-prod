package mapping

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
)

// WebhookMapping represents the mapping between webhook IDs and clients
type WebhookMapping struct {
	WebhookToClient map[string]string `json:"webhook_to_client"`
	ClientToAPIKey  map[string]string `json:"client_to_api_key"`
	LastUpdated     time.Time         `json:"last_updated"`
}

// WebhookMappingService handles webhook ID to client ID mapping
type WebhookMappingService struct {
	mapping *WebhookMapping
	logger  *zap.Logger
}

// MailerCloudWebhook represents webhook data from MailerCloud API
type MailerCloudWebhook struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// MailerCloudWebhookList represents the response from MailerCloud webhook search
type MailerCloudWebhookList struct {
	Data []MailerCloudWebhook `json:"data"`
}

// SearchWebhooksRequest for MailerCloud API
type SearchWebhooksRequest struct {
	Limit     int    `json:"limit"`
	Page      int    `json:"page"`
	Search    string `json:"search"`
	SortField string `json:"sort_field"`
	SortOrder string `json:"sort_order"`
}

// NewWebhookMappingService creates a new webhook mapping service
func NewWebhookMappingService(logger *zap.Logger) *WebhookMappingService {
	return &WebhookMappingService{
		mapping: &WebhookMapping{
			WebhookToClient: make(map[string]string),
			ClientToAPIKey:  make(map[string]string),
			LastUpdated:     time.Now(),
		},
		logger: logger,
	}
}

// LoadMappingFromEnvironment loads the webhook-to-client mapping on startup
func (wms *WebhookMappingService) LoadMappingFromEnvironment() error {
	wms.logger.Info("Loading webhook-to-client mapping from MailerCloud API")

	// Parse MAILERCLOUD_API_KEYS environment variable
	apiKeysEnv := os.Getenv("MAILERCLOUD_API_KEYS")
	if apiKeysEnv == "" {
		return fmt.Errorf("MAILERCLOUD_API_KEYS environment variable is not set")
	}

	clients := make(map[string]string) // client -> apiKey
	for _, config := range strings.Split(apiKeysEnv, ",") {
		parts := strings.Split(config, ":")
		if len(parts) != 2 {
			wms.logger.Warn("Invalid client config format", zap.String("config", config))
			continue
		}
		clientID, apiKey := parts[0], parts[1]
		clients[clientID] = apiKey
		wms.mapping.ClientToAPIKey[clientID] = apiKey
	}

	// For each client, fetch their webhooks from MailerCloud
	for clientID, apiKey := range clients {
		webhooks, err := wms.fetchWebhooksForClient(clientID, apiKey)
		if err != nil {
			wms.logger.Error("Failed to fetch webhooks for client",
				zap.String("client", clientID),
				zap.Error(err))
			continue
		}

		// Map webhook IDs to client
		for _, webhook := range webhooks {
			wms.mapping.WebhookToClient[webhook.ID] = clientID
			wms.logger.Info("Mapped webhook to client",
				zap.String("webhook_id", webhook.ID),
				zap.String("client_id", clientID),
				zap.String("webhook_name", webhook.Name))
		}
	}

	wms.mapping.LastUpdated = time.Now()
	wms.logger.Info("Webhook mapping loaded successfully",
		zap.Int("total_webhooks", len(wms.mapping.WebhookToClient)),
		zap.Int("total_clients", len(wms.mapping.ClientToAPIKey)))

	return nil
}

// fetchWebhooksForClient fetches webhooks for a specific client using MailerCloud API
func (wms *WebhookMappingService) fetchWebhooksForClient(clientID, apiKey string) ([]MailerCloudWebhook, error) {
	searchReq := SearchWebhooksRequest{
		Limit:     100,
		Page:      1,
		Search:    "",
		SortField: "name",
		SortOrder: "asc",
	}

	jsonData, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling search request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://cloudapi.mailercloud.com/v1/webhooks/search", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var webhookList MailerCloudWebhookList
	if err := json.NewDecoder(resp.Body).Decode(&webhookList); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return webhookList.Data, nil
}

// GetClientForWebhook returns the client ID for a given webhook ID
func (wms *WebhookMappingService) GetClientForWebhook(webhookID string) (string, bool) {
	clientID, exists := wms.mapping.WebhookToClient[webhookID]
	return clientID, exists
}

// GetAPIKeyForClient returns the API key for a given client ID
func (wms *WebhookMappingService) GetAPIKeyForClient(clientID string) (string, bool) {
	apiKey, exists := wms.mapping.ClientToAPIKey[clientID]
	return apiKey, exists
}

// GetMappingStats returns statistics about the current mapping
func (wms *WebhookMappingService) GetMappingStats() map[string]interface{} {
	return map[string]interface{}{
		"total_webhooks":    len(wms.mapping.WebhookToClient),
		"total_clients":     len(wms.mapping.ClientToAPIKey),
		"last_updated":      wms.mapping.LastUpdated,
		"webhook_to_client": wms.mapping.WebhookToClient,
	}
}
