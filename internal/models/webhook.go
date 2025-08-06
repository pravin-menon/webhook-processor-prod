package models

import (
	"time"
)

// WebhookEvent represents the base webhook event structure
type WebhookEvent struct {
	WebhookID      string `json:"webhook_id" bson:"webhook_id"`     // From Webhook-Id header
	WebhookType    string `json:"webhook_type" bson:"webhook_type"` // From Webhook-Type header
	Event          string `json:"event" bson:"event"`
	CampaignName   string `json:"campaign_name" bson:"campaign_name"`
	CampaignID     string `json:"campaign_id" bson:"campaign_id"`
	TagName        string `json:"tag_name" bson:"tag_name"`
	DateEvent      string `json:"date_event" bson:"date_event"`
	Timestamp      int64  `json:"ts" bson:"ts"`
	TimestampEvent int64  `json:"ts_event" bson:"ts_event"`

	// Optional fields based on event type
	Emails []string `json:"emails,omitempty" bson:"emails,omitempty"`
	Email  string   `json:"email,omitempty" bson:"email,omitempty"`
	URL    string   `json:"URL,omitempty" bson:"url,omitempty"`
	ListID any      `json:"list_id,omitempty" bson:"list_id,omitempty"` // Can be string or array
	Reason string   `json:"reason,omitempty" bson:"reason,omitempty"`

	// Metadata
	ClientID   string    `json:"-" bson:"client_id"`
	ReceivedAt time.Time `json:"-" bson:"received_at"`
	UpdatedAt  time.Time `json:"-" bson:"updated_at"`
	RetryCount int       `json:"-" bson:"retry_count"`
	Status     string    `json:"-" bson:"status"`
}

// EventStatus represents the possible states of a webhook event
type EventStatus string

const (
	EventStatusPending   EventStatus = "pending"
	EventStatusProcessed EventStatus = "processed"
	EventStatusFailed    EventStatus = "failed"
	EventStatusRetrying  EventStatus = "retrying"
)
