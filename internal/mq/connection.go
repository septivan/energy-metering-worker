package mq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Connection wraps RabbitMQ connection
type Connection struct {
	conn *amqp.Connection
}

// NewConnection creates a new RabbitMQ connection
func NewConnection(lc fx.Lifecycle, logger *zap.Logger, url string) (*Connection, error) {
	logger.Info("attempting to connect to RabbitMQ...")

	conn, err := amqp.Dial(url)
	if err != nil {
		logger.Error("rabbitmq connection failed", zap.Error(err))
		return nil, fmt.Errorf("[RABBITMQ CONNECTION FAILED] cannot connect to RabbitMQ. Please check: 1) RabbitMQ is running, 2) RABBITMQ_URL is correct, 3) Credentials are valid. Error: %w", err)
	}

	mqConn := &Connection{conn: conn}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("rabbitmq connection established successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := conn.Close(); err != nil {
				logger.Error("failed to close rabbitmq connection", zap.Error(err))
				return err
			}
			logger.Info("rabbitmq connection closed")
			return nil
		},
	})

	return mqConn, nil
}

// Channel creates a new RabbitMQ channel
func (c *Connection) Channel() (*amqp.Channel, error) {
	return c.conn.Channel()
}
