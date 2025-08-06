package storage

import (
	"context"
	"time"

	"webhook-processor/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type MongoDB struct {
	client     *mongo.Client
	collection *mongo.Collection
	logger     *zap.Logger
}

func NewMongoDB(uri, database, collection string, logger *zap.Logger) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// MongoDB Atlas specific client options
	clientOptions := options.Client().ApplyURI(uri).
		SetMaxPoolSize(100).
		SetMaxConnIdleTime(30 * time.Second).
		SetConnectTimeout(10 * time.Second).
		SetSocketTimeout(30 * time.Second).
		SetServerSelectionTimeout(10 * time.Second)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	logger.Info("Successfully connected to MongoDB",
		zap.String("database", database),
		zap.String("collection", collection),
	)

	coll := client.Database(database).Collection(collection)

	// Create indexes
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "webhook_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "client_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "received_at", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "campaign_id", Value: 1},
				{Key: "client_id", Value: 1},
				{Key: "event", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "client_id", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "email", Value: 1},
				{Key: "campaign_id", Value: 1},
			},
		},
	}

	_, err = coll.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return nil, err
	}

	return &MongoDB{
		client:     client,
		collection: coll,
		logger:     logger,
	}, nil
}

func (m *MongoDB) InsertEvent(ctx context.Context, event *models.WebhookEvent) error {
	// Initialize event status if not set
	if event.Status == "" {
		event.Status = string(models.EventStatusPending)
	}

	doc := bson.M{
		"webhook_id":   event.WebhookID,
		"webhook_type": event.WebhookType,
		"client_id":    event.ClientID,
		"event":        event.Event,
		"received_at":  event.ReceivedAt,
		"status":       event.Status,
		"retry_count":  event.RetryCount,
	}

	// Add optional fields only if they have values
	if event.CampaignID != "" {
		doc["campaign_id"] = event.CampaignID
	}
	if event.CampaignName != "" {
		doc["campaign_name"] = event.CampaignName
	}
	if event.TagName != "" {
		doc["tag_name"] = event.TagName
	}
	if event.DateEvent != "" {
		doc["date_event"] = event.DateEvent
	}
	if event.URL != "" {
		doc["url"] = event.URL
	}
	if event.Email != "" {
		doc["email"] = event.Email
	}
	if len(event.Emails) > 0 {
		doc["emails"] = event.Emails
	}
	if event.ListID != nil {
		doc["list_id"] = event.ListID
	}
	if event.Reason != "" {
		doc["reason"] = event.Reason
	}

	_, err := m.collection.InsertOne(ctx, doc)
	if err != nil {
		m.logger.Error("Failed to insert event",
			zap.Error(err),
			zap.String("client_id", event.ClientID),
			zap.String("webhook_id", event.WebhookID))
		return err
	}
	return nil
}

func (m *MongoDB) UpdateEventStatus(ctx context.Context, event *models.WebhookEvent, status models.EventStatus) error {
	filter := bson.M{
		"webhook_id": event.WebhookID,
	}

	update := bson.M{
		"$set": bson.M{
			"status":      status,
			"retry_count": event.RetryCount,
			"updated_at":  time.Now().UTC(),
		},
	}

	_, err := m.collection.UpdateOne(ctx, filter, update)
	return err
}

func (m *MongoDB) GetFailedEvents(ctx context.Context, clientID string) ([]*models.WebhookEvent, error) {
	filter := bson.M{
		"client_id": clientID,
		"status":    models.EventStatusFailed,
	}

	cursor, err := m.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []*models.WebhookEvent
	if err = cursor.All(ctx, &events); err != nil {
		return nil, err
	}

	return events, nil
}

func (m *MongoDB) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}
