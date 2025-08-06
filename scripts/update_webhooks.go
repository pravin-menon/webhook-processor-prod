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
	ngrokAPIURL        = "http://127.0.0.1:4040/api/tunnels"

	maxRetries    = 3
	retryInterval = 2 * time.Second

	// Webhook status constants
	statusActive   = "Active"
	statusInactive = "Inactive"
	statusEnabled  = "1"
	statusDisabled = "0"
)

type NgrokTunnels struct {
	Tunnels []struct {
		PublicURL string `json:"public_url"`
	} `json:"tunnels"`
}

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

func getNgrokURL() (string, error) {
	resp, err := http.Get(ngrokAPIURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tunnels NgrokTunnels
	if err := json.NewDecoder(resp.Body).Decode(&tunnels); err != nil {
		return "", err
	}

	if len(tunnels.Tunnels) == 0 {
		return "", fmt.Errorf("no ngrok tunnels found")
	}

	return tunnels.Tunnels[0].PublicURL, nil
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

func processWebhooks(clientID, apiKey, ngrokURL string) error {
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

	expectedURL := ngrokURL + "/webhook"
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
		if webhook.URL != expectedURL {
			log.Printf("Current URL doesn't match expected URL (%s). Updating...", expectedURL)
			if err := client.updateWebhookURL(webhook.ID, &webhook, expectedURL); err != nil {
				log.Printf("Error updating webhook URL: %v", err)
				continue
			}
			log.Printf("Successfully updated webhook URL to: %s", expectedURL)
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

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Load environment variables from .env.development
	envFiles := []string{".env.development", "scripts/../.env.development"}
	envLoaded := false
	for _, envFile := range envFiles {
		if err := godotenv.Load(envFile); err == nil {
			envLoaded = true
			break
		}
	}
	if !envLoaded {
		log.Printf("Warning: Could not load .env.development file from any location")
	}

	// Get ngrok URL
	log.Println("Fetching ngrok public URL...")
	ngrokURL, err := getNgrokURL()
	if err != nil {
		log.Fatalf("Error getting ngrok URL: %v", err)
	}
	log.Printf("Ngrok URL: %s", ngrokURL)

	// Get API keys from environment
	apiKeys := os.Getenv("MAILERCLOUD_API_KEYS")
	if apiKeys == "" {
		log.Fatal("MAILERCLOUD_API_KEYS environment variable is not set")
	}

	// Process each client's webhooks
	for _, config := range strings.Split(apiKeys, ",") {
		parts := strings.Split(config, ":")
		if len(parts) != 2 {
			log.Printf("Invalid client config format: %s", config)
			continue
		}

		clientID, apiKey := parts[0], parts[1]
		if err := processWebhooks(clientID, apiKey, ngrokURL); err != nil {
			log.Printf("Error processing webhooks for client %s: %v", clientID, err)
		}
	}

	log.Println("Webhook synchronization completed")
}
