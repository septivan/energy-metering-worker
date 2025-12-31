package mq

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// Publisher handles message publishing to RabbitMQ
type Publisher struct {
	conn     *Connection
	channel  *amqp.Channel
	exchange string
	logger   *zap.Logger
}

// NewPublisher creates a new RabbitMQ publisher
func NewPublisher(conn *Connection, exchange string, logger *zap.Logger) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	// Declare exchange
	err = ch.ExchangeDeclare(
		exchange,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		ch.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	return &Publisher{
		conn:     conn,
		channel:  ch,
		exchange: exchange,
		logger:   logger,
	}, nil
}

// ProcessedEvent represents the event published after processing
type ProcessedEvent struct {
	ClientID         string  `json:"client_id"`
	MetricName       string  `json:"metric_name"`
	MetricValue      float64 `json:"metric_value"`
	ReadingTimestamp string  `json:"reading_timestamp"`
	ValidationStatus string  `json:"validation_status"`
}

// PublishProcessedEvent publishes a processed meter reading event
func (p *Publisher) PublishProcessedEvent(ctx context.Context, event ProcessedEvent, routingKey string) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = p.channel.PublishWithContext(
		ctx,
		p.exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	p.logger.Debug("published processed event",
		zap.String("routing_key", routingKey),
		zap.String("client_id", event.ClientID),
		zap.String("metric_name", event.MetricName),
	)

	return nil
}

// Close closes the publisher channel
func (p *Publisher) Close() error {
	if p.channel != nil {
		return p.channel.Close()
	}
	return nil
}
