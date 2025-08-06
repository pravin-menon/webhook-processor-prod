package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"webhook-processor/config"
	"webhook-processor/internal/queue"
	"webhook-processor/internal/storage"
	"webhook-processor/internal/worker"
	"webhook-processor/pkg/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := logger.NewLogger(cfg.LogLevel)

	// Initialize RabbitMQ connection
	amqpConn, err := queue.NewRabbitMQConnection(cfg.RabbitMQ.URL)
	if err != nil {
		logger.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer amqpConn.Close()

	// Create a channel
	ch, err := amqpConn.Channel()
	if err != nil {
		logger.Fatalf("Failed to open channel: %v", err)
	}
	defer ch.Close()

	// Declare exchange
	err = ch.ExchangeDeclare(
		cfg.RabbitMQ.Exchange, // name
		"direct",              // type
		true,                  // durable
		false,                 // auto-deleted
		false,                 // internal
		false,                 // no-wait
		nil,                   // arguments
	)
	if err != nil {
		logger.Fatalf("Failed to declare exchange: %v", err)
	}

	// Declare queue
	q, err := ch.QueueDeclare(
		cfg.RabbitMQ.QueueName, // name
		true,                   // durable
		false,                  // delete when unused
		false,                  // exclusive
		false,                  // no-wait
		nil,                    // arguments
	)
	if err != nil {
		logger.Fatalf("Failed to declare queue: %v", err)
	}

	// Bind queue to exchange
	err = ch.QueueBind(
		q.Name,                // queue name
		"",                    // routing key
		cfg.RabbitMQ.Exchange, // exchange
		false,
		nil,
	)
	if err != nil {
		logger.Fatalf("Failed to bind queue: %v", err)
	}

	// Initialize MongoDB connection
	db, err := storage.NewMongoDB(cfg.MongoDB.URI, cfg.MongoDB.Database, cfg.MongoDB.Collection, logger.Desugar())
	if err != nil {
		logger.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Initialize worker
	w := worker.NewWorker(ch, db, logger.Desugar())

	// Start consuming messages
	if err := w.Start(context.Background(), q.Name); err != nil {
		logger.Fatalf("Failed to start worker: %v", err)
	}

	logger.Info("Worker started successfully")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Worker shutting down")
}
