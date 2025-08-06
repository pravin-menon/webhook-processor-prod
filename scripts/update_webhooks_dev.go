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
)

type WebhookConfig struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

type WebhookResponse struct {
	ID     string   `json:"id"`
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Status string   `json:"status"`
}

func main() {
	// Get API keys from environment
	apiKeysEnv := os.Getenv("MAILERCLOUD_API_KEYS")
	if apiKeysEnv == "" {
		log.Fatal("MAILERCLOUD_API_KEYS environment variable is required")
	}

	// Get development domain configuration
	devDomain := os.Getenv("DEV_DOMAIN")
	if devDomain == "" {
		devDomain = "webhook-dev.local"
	}

	devPort := os.Getenv("DEV_PORT")
	if devPort == "" {
		devPort = "8080"
	}

	// Construct webhook URL
	webhookURL := fmt.Sprintf("http://%s:%s/webhook", devDomain, devPort)

	log.Printf("🔧 Development Webhook Updater")
	log.Printf("📍 Target URL: %s", webhookURL)
	log.Printf("🔑 Processing API keys...")

	// Parse API keys
	apiKeys := strings.Split(apiKeysEnv, ",")

	for _, keyPair := range apiKeys {
		parts := strings.Split(strings.TrimSpace(keyPair), ":")
		if len(parts) != 2 {
			log.Printf("❌ Invalid API key format: %s", keyPair)
			continue
		}

		clientID := parts[0]
		apiKey := parts[1]

		log.Printf("\n📊 Processing client: %s", clientID)

		if err := updateWebhook(clientID, apiKey, webhookURL); err != nil {
			log.Printf("❌ Failed to update webhook for %s: %v", clientID, err)
		} else {
			log.Printf("✅ Successfully updated webhook for %s", clientID)
		}
	}

	log.Printf("\n🎉 Development webhook configuration complete!")
	log.Printf("📝 Test your webhook with:")
	log.Printf("   curl -H \"X-API-Key: your-api-key\" \\")
	log.Printf("        -H \"Content-Type: application/json\" \\")
	log.Printf("        -d '{\"event\":\"test\",\"email\":\"test@example.com\"}' \\")
	log.Printf("        %s", webhookURL)
}

func updateWebhook(clientID, apiKey, webhookURL string) error {
	// MailerCloud webhook configuration
	config := WebhookConfig{
		URL: webhookURL,
		Events: []string{
			"subscriber.subscribed",
			"subscriber.unsubscribed",
			"email.sent",
			"email.delivered",
			"email.bounced",
			"email.opened",
			"email.clicked",
			"campaign.sent",
		},
	}

	jsonData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook config: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.mailercloud.com/v1/webhooks", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	log.Printf("📤 Response Status: %s", resp.Status)
	log.Printf("📥 Response Body: %s", string(body))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var webhookResp WebhookResponse
		if err := json.Unmarshal(body, &webhookResp); err == nil {
			log.Printf("🆔 Webhook ID: %s", webhookResp.ID)
			log.Printf("🌐 Configured URL: %s", webhookResp.URL)
			log.Printf("📋 Events: %v", webhookResp.Events)
		}
		return nil
	}

	return fmt.Errorf("API request failed with status %s: %s", resp.Status, string(body))
}
