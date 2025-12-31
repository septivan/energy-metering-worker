package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	ServiceName string
	ServicePort int
	Database    DatabaseConfig
	RabbitMQ    RabbitMQConfig
	Validation  ValidationConfig
	Anomaly     AnomalyConfig
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	URL string
}

// RabbitMQConfig holds RabbitMQ connection and queue settings
type RabbitMQConfig struct {
	URL              string
	IngestExchange   string
	IngestQueue      string
	IngestRoutingKey string
	WorkerExchange   string
	WorkerRoutingKey string
	DLQQueue         string
	PrefetchCount    int
}

// ValidationConfig holds validation settings
type ValidationConfig struct {
	TimestampToleranceMinutes int
}

// AnomalyConfig holds anomaly detection settings
type AnomalyConfig struct {
	SpikeThreshold            float64
	MinDataPointsForDetection int
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		ServiceName: getEnv("SERVICE_NAME", "energy-metering-worker"),
		ServicePort: getEnvAsInt("SERVICE_PORT", 8081),
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", ""),
		},
		RabbitMQ: RabbitMQConfig{
			URL:              getEnv("RABBITMQ_URL", ""),
			IngestExchange:   getEnv("RABBITMQ_INGEST_EXCHANGE", "energy-metering.ingest.exchange"),
			IngestQueue:      getEnv("RABBITMQ_INGEST_QUEUE", "energy-metering.ingest.queue"),
			IngestRoutingKey: getEnv("RABBITMQ_INGEST_ROUTING_KEY", "meter.reading.raw"),
			WorkerExchange:   getEnv("RABBITMQ_WORKER_EXCHANGE", "energy-metering.worker.events.exchange"),
			WorkerRoutingKey: getEnv("RABBITMQ_WORKER_ROUTING_KEY", "meter.reading.accepted"),
			DLQQueue:         getEnv("RABBITMQ_DLQ_QUEUE", "energy-metering.ingest.dlq"),
			PrefetchCount:    getEnvAsInt("RABBITMQ_PREFETCH", 10),
		},
		Validation: ValidationConfig{
			TimestampToleranceMinutes: getEnvAsInt("VALIDATION_TIMESTAMP_TOLERANCE_MINUTES", 10080),
		},
		Anomaly: AnomalyConfig{
			SpikeThreshold:            getEnvAsFloat("ANOMALY_SPIKE_THRESHOLD", 3.0),
			MinDataPointsForDetection: getEnvAsInt("ANOMALY_MIN_DATA_POINTS", 3),
		},
	}

	// Validate required fields
	if cfg.Database.URL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required but not set in environment variables")
	}
	if cfg.RabbitMQ.URL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is required but not set in environment variables")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue
	}
	return value
}
