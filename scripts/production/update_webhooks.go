package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	mailercloudBaseURL = "https://cloudapi.mailercloud.com/v1"

	maxRetries    = 3
	retryInterval = 2 * time.Second

	// Webhook status constants
	statusActive   = "Active"
	statusInactive = "Inactive"
	statusEnabled  = "1"
	statusDisabled = "0"
)

type WebhookList struct {
	Data            []Webhook `json:"data"`
	MaxWebhookLimit int       `json:"max_webhook_limit"`
	Total           int       `json:"total"`
	WebhookCount    int       `json:"webhook_count"`
}

type WebhookDetailResponse struct {
	Webhook WebhookDetail `json:"webhook"`
}

type WebhookDetail struct {
	ID           string   `json:"id"`
	URL          string   `json:"url"`
	Status       string   `json:"status"`
	Name         string   `json:"name"`
	Event        []string `json:"event"`
	CreatedDate  string   `json:"created_date"`
	ModifiedDate string   `json:"modified_date"`
}

type Webhook struct {
	ID           string   `json:"id"`
	URL          string   `json:"url"`
	Status       int      `json:"status"`
	Name         string   `json:"name"`
	Event        []string `json:"event"`
	CreatedDate  string   `json:"created_date"`
	ModifiedDate string   `json:"modified_date"`
}

type SearchWebhooksRequest struct {
	Limit     int    `json:"limit"`
	Page      int    `json:"page"`
	Search    string `json:"search"`
	SortField string `json:"sort_field"`
	SortOrder string `json:"sort_order"`
}

type Client struct {
	ID      string
	APIKey  string
	BaseURL string
}

func (c *Client) makeRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("error reading request body: %v", err)
		}
	}

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			log.Printf("Retrying request (attempt %d/%d)", i+1, maxRetries)
			time.Sleep(retryInterval)
		}

		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequest(method, c.BaseURL+endpoint, bodyReader)
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Authorization", c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Only retry on 5xx errors
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %s", resp.Status)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %v", maxRetries, lastErr)
}

func getProductionWebhookURL() (string, error) {
	// Try to get the domain from environment variables
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		return "", fmt.Errorf("DOMAIN environment variable is not set")
	}

	// Construct the webhook URL
	webhookURL := fmt.Sprintf("https://%s/webhook", domain)

	// Validate that the URL is accessible
	log.Printf("Validating webhook URL: %s", webhookURL)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(webhookURL)
	if err != nil {
		return "", fmt.Errorf("webhook URL validation failed: %v", err)
	}
	defer resp.Body.Close()

	// We expect either 200 (if GET is allowed) or 405 (Method Not Allowed) for POST-only endpoints
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusMethodNotAllowed {
		return "", fmt.Errorf("webhook URL validation failed: unexpected status %d", resp.StatusCode)
	}

	log.Printf("Webhook URL validation successful")
	return webhookURL, nil
}

func (c *Client) getWebhooks() ([]Webhook, error) {
	searchReq := SearchWebhooksRequest{
		Limit:     100, // Get maximum possible webhooks
		Page:      1,
		Search:    "",
		SortField: "name",
		SortOrder: "asc",
	}

	jsonData, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling search request: %v", err)
	}

	resp, err := c.makeRequest("POST", "/webhooks/search", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var webhookList WebhookList
	if err := json.NewDecoder(resp.Body).Decode(&webhookList); err != nil {
		return nil, err
	}

	return webhookList.Data, nil
}

type UpdateWebhookRequest struct {
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

type UpdateWebhookResponse struct {
	Message string `json:"message"`
}

func (c *Client) updateWebhookURL(webhookID string, webhook *Webhook, newURL string) error {
	updateReq := UpdateWebhookRequest{
		Name:   webhook.Name,  // Preserve existing name
		URL:    newURL,        // Update URL
		Events: webhook.Event, // Preserve existing events
	}

	jsonData, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("error marshaling update request: %v", err)
	}

	resp, err := c.makeRequest("PUT", "/webhooks/"+webhookID, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update webhook URL: status=%s body=%s", resp.Status, string(bodyBytes))
	}

	var updateResp UpdateWebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
		return fmt.Errorf("error decoding response: %v", err)
	}

	log.Printf("Update webhook response: %s", updateResp.Message)
	return nil
}

type ToggleWebhookRequest struct {
	Status string `json:"status"`
}

type ToggleWebhookResponse struct {
	Message string `json:"message"`
}

func (c *Client) toggleWebhookStatus(webhookID string) error {
	// Always set to "1" since we're activating webhooks
	toggleReq := ToggleWebhookRequest{
		Status: "1",
	}

	jsonData, err := json.Marshal(toggleReq)
	if err != nil {
		return fmt.Errorf("error marshaling toggle request: %v", err)
	}

	resp, err := c.makeRequest("POST", "/webhooks/toggle/"+webhookID, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to toggle webhook status: status=%s body=%s", resp.Status, string(bodyBytes))
	}

	var toggleResp ToggleWebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&toggleResp); err != nil {
		return fmt.Errorf("error decoding response: %v", err)
	}

	log.Printf("Toggle webhook response: %s", toggleResp.Message)
	return nil
}

func (c *Client) getWebhookDetails(webhookID string) (*Webhook, error) {
	resp, err := c.makeRequest("GET", "/webhooks/detail/"+webhookID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var detailResp WebhookDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&detailResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	// Convert WebhookDetail to Webhook
	webhook := &Webhook{
		ID:           detailResp.Webhook.ID,
		URL:          detailResp.Webhook.URL,
		Name:         detailResp.Webhook.Name,
		Event:        detailResp.Webhook.Event,
		CreatedDate:  detailResp.Webhook.CreatedDate,
		ModifiedDate: detailResp.Webhook.ModifiedDate,
		Status:       convertStatus(detailResp.Webhook.Status),
	}

	return webhook, nil
}

func convertStatus(status string) int {
	if strings.EqualFold(status, "Active") {
		return 1
	}
	return 0
}

func processWebhooks(clientID, apiKey, webhookURL string) error {
	client := &Client{
		ID:      clientID,
		APIKey:  apiKey,
		BaseURL: mailercloudBaseURL,
	}

	log.Printf("Processing webhooks for client: %s", clientID)

	// Step 1: Get all webhooks
	webhooks, err := client.getWebhooks()
	if err != nil {
		return fmt.Errorf("failed to get webhooks: %v", err)
	}

	if len(webhooks) == 0 {
		log.Printf("No webhooks found for client %s", clientID)
		return nil
	}
	log.Printf("Found %d webhooks", len(webhooks))

	for _, webhook := range webhooks {
		log.Printf("-----------------------------------")
		log.Printf("Processing webhook:")
		log.Printf("  ID: %s", webhook.ID)
		log.Printf("  Name: %s", webhook.Name)
		log.Printf("  URL: %s", webhook.URL)
		log.Printf("  Status: %d (1=active, 0=inactive)", webhook.Status)
		log.Printf("  Events: %v", webhook.Event)
		log.Printf("  Created: %s", webhook.CreatedDate)
		log.Printf("  Modified: %s", webhook.ModifiedDate)

		// Step 2: Check and update URL if needed
		if webhook.URL != webhookURL {
			log.Printf("Current URL doesn't match expected URL (%s). Updating...", webhookURL)
			if err := client.updateWebhookURL(webhook.ID, &webhook, webhookURL); err != nil {
				log.Printf("Error updating webhook URL: %v", err)
				continue
			}
			log.Printf("Successfully updated webhook URL to: %s", webhookURL)
		} else {
			log.Printf("URL is correctly configured")
		}

		// Step 3: Get current details and check status
		details, err := client.getWebhookDetails(webhook.ID)
		if err != nil {
			log.Printf("Error getting webhook details: %v", err)
			continue
		}

		// Step 4: Activate if needed
		if details.Status != 1 {
			log.Printf("Webhook is not active. Activating...")
			if err := client.toggleWebhookStatus(webhook.ID); err != nil {
				log.Printf("Error activating webhook: %v", err)
				continue
			}

			// Verify the status change
			updated, err := client.getWebhookDetails(webhook.ID)
			if err != nil {
				log.Printf("Error verifying webhook status: %v", err)
				continue
			}

			if updated.Status != 1 {
				log.Printf("WARNING: Webhook is still not active after toggle attempt")
			} else {
				log.Printf("Successfully activated webhook")
			}
		} else {
			log.Printf("Webhook is already active")
		}

		log.Printf("Webhook processing completed successfully")
		log.Printf("-----------------------------------")
	}

	return nil
}

// loadAPIKeysFromEnv loads API keys from various environment variable patterns
func loadAPIKeysFromEnv() map[string]string {
	apiKeys := make(map[string]string)

	// First, check for the MAILERCLOUD_API_KEYS format (client1:key1,client2:key2)
	if envAPIKeys := os.Getenv("MAILERCLOUD_API_KEYS"); envAPIKeys != "" {
		for _, config := range strings.Split(envAPIKeys, ",") {
			parts := strings.Split(config, ":")
			if len(parts) == 2 {
				clientID := strings.TrimSpace(parts[0])
				apiKey := strings.TrimSpace(parts[1])
				if clientID != "" && apiKey != "" {
					apiKeys[clientID] = apiKey
				}
			}
		}
	}

	// Also load individual API keys from environment variables
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		envName := parts[0]
		envValue := parts[1]

		// Look for *_API_KEY pattern
		if strings.HasSuffix(envName, "_API_KEY") && envValue != "" {
			// Convert CLIENT_NAME_API_KEY to client_name
			clientName := strings.ToLower(strings.TrimSuffix(envName, "_API_KEY"))
			apiKeys[clientName] = envValue
		}
	}

	return apiKeys
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Load environment variables - try production first, then development
	envFiles := []string{".env.production", ".env", "../.env.production", "../.env", "../../.env.production", "../../.env"}
	envLoaded := false
	for _, envFile := range envFiles {
		if err := godotenv.Load(envFile); err == nil {
			log.Printf("Loaded environment from: %s", envFile)
			envLoaded = true
			break
		}
	}
	if !envLoaded {
		log.Printf("Warning: Could not load environment file from any location")
	}

	// Get production webhook URL from domain
	log.Println("Constructing production webhook URL...")
	webhookURL, err := getProductionWebhookURL()
	if err != nil {
		log.Fatalf("Error getting production webhook URL: %v", err)
	}
	log.Printf("Production Webhook URL: %s", webhookURL)

	// Load API keys from environment
	apiKeys := loadAPIKeysFromEnv()
	if len(apiKeys) == 0 {
		log.Fatal("No API keys found. Set MAILERCLOUD_API_KEYS or individual *_API_KEY environment variables")
	}

	log.Printf("Found API keys for %d clients", len(apiKeys))

	// Process each client's webhooks
	for clientID, apiKey := range apiKeys {
		log.Printf("\n========================================")
		log.Printf("Processing client: %s", clientID)
		log.Printf("========================================")

		if err := processWebhooks(clientID, apiKey, webhookURL); err != nil {
			log.Printf("Error processing webhooks for client %s: %v", clientID, err)
		} else {
			log.Printf("Successfully processed webhooks for client: %s", clientID)
		}
	}

	log.Println("\n========================================")
	log.Println("Production webhook synchronization completed")
	log.Println("========================================")
}
