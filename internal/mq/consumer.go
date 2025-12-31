package mq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// MessageHandler is a function that processes a message
type MessageHandler func(ctx context.Context, body []byte) error

// Consumer handles message consumption from RabbitMQ
type Consumer struct {
	conn             *Connection
	channel          *amqp.Channel
	queue            string
	dlqQueue         string
	exchange         string
	routingKey       string
	prefetchCount    int
	logger           *zap.Logger
	messageProcessor MessageHandler
}

// ConsumerConfig holds consumer configuration
type ConsumerConfig struct {
	Connection       *Connection
	Queue            string
	DLQQueue         string
	Exchange         string
	RoutingKey       string
	PrefetchCount    int
	Logger           *zap.Logger
	MessageProcessor MessageHandler
}

// NewConsumer creates a new RabbitMQ consumer
func NewConsumer(cfg ConsumerConfig) (*Consumer, error) {
	ch, err := cfg.Connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	// Set QoS (prefetch)
	err = ch.Qos(cfg.PrefetchCount, 0, false)
	if err != nil {
		ch.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	// Declare exchange
	err = ch.ExchangeDeclare(
		cfg.Exchange,
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

	// Declare main queue
	// Try to declare with DLX, if fails due to precondition, try without DLX
	args := map[string]interface{}{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": cfg.DLQQueue,
	}
	_, err = ch.QueueDeclare(
		cfg.Queue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		args,
	)
	if err != nil {
		// If queue already exists with different args, try without DLX
		cfg.Logger.Warn("failed to declare queue with DLX, trying without DLX",
			zap.Error(err))
		_, err = ch.QueueDeclare(
			cfg.Queue,
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			nil,   // no arguments
		)
		if err != nil {
			ch.Close()
			return nil, fmt.Errorf("failed to declare queue: %w", err)
		}
	}

	// Declare DLQ
	_, err = ch.QueueDeclare(
		cfg.DLQQueue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		ch.Close()
		return nil, fmt.Errorf("failed to declare DLQ: %w", err)
	}

	// Bind queue to exchange
	err = ch.QueueBind(
		cfg.Queue,
		cfg.RoutingKey,
		cfg.Exchange,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	return &Consumer{
		conn:             cfg.Connection,
		channel:          ch,
		queue:            cfg.Queue,
		dlqQueue:         cfg.DLQQueue,
		exchange:         cfg.Exchange,
		routingKey:       cfg.RoutingKey,
		prefetchCount:    cfg.PrefetchCount,
		logger:           cfg.Logger,
		messageProcessor: cfg.MessageProcessor,
	}, nil
}

// Start starts consuming messages
func (c *Consumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		c.queue,
		"",    // consumer tag
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	c.logger.Info("consumer started",
		zap.String("queue", c.queue),
		zap.Int("prefetch", c.prefetchCount),
	)

	go func() {
		for {
			select {
			case <-ctx.Done():
				c.logger.Info("consumer context cancelled, stopping")
				return
			case msg, ok := <-msgs:
				if !ok {
					c.logger.Warn("message channel closed")
					return
				}
				c.processMessage(ctx, msg)
			}
		}
	}()

	return nil
}

func (c *Consumer) processMessage(ctx context.Context, msg amqp.Delivery) {
	c.logger.Info("received message from queue",
		zap.String("queue", c.queue),
		zap.String("routing_key", msg.RoutingKey),
		zap.Int("body_size", len(msg.Body)),
	)

	// Process message with message processor
	err := c.messageProcessor(ctx, msg.Body)
	if err != nil {
		c.logger.Error("failed to process message",
			zap.Error(err),
			zap.String("routing_key", msg.RoutingKey),
		)

		// NACK with requeue=false sends to DLQ
		if nackErr := msg.Nack(false, false); nackErr != nil {
			c.logger.Error("failed to NACK message", zap.Error(nackErr))
		}
		return
	}

	// ACK message after successful processing
	if ackErr := msg.Ack(false); ackErr != nil {
		c.logger.Error("failed to ACK message", zap.Error(ackErr))
	} else {
		c.logger.Info("message processed and acknowledged successfully",
			zap.String("routing_key", msg.RoutingKey),
		)
	}
}

// Close closes the consumer channel
func (c *Consumer) Close() error {
	if c.channel != nil {
		return c.channel.Close()
	}
	return nil
}

// RegisterLifecycle registers the consumer with Fx lifecycle
func (c *Consumer) RegisterLifecycle(lc fx.Lifecycle, ctx context.Context) {
	lc.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			return c.Start(ctx)
		},
		OnStop: func(stopCtx context.Context) error {
			if err := c.channel.Close(); err != nil {
				c.logger.Error("failed to close consumer channel", zap.Error(err))
				return err
			}
			c.logger.Info("consumer stopped")
			return nil
		},
	})
}
