package logging

import (
	"go.uber.org/zap"
)

// NewLogger creates a new structured logger
func NewLogger(serviceName string) (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.InitialFields = map[string]interface{}{
		"service": serviceName,
	}

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// WithRequestID returns a logger with request_id field
func WithRequestID(logger *zap.Logger, requestID string) *zap.Logger {
	return logger.With(zap.String("request_id", requestID))
}
