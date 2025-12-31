package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/septivank/energy-metering-worker/internal/anomaly"
	"github.com/septivank/energy-metering-worker/internal/config"
	"github.com/septivank/energy-metering-worker/internal/db"
	"github.com/septivank/energy-metering-worker/internal/logging"
	"github.com/septivank/energy-metering-worker/internal/mq"
	"github.com/septivank/energy-metering-worker/internal/repository"
	"github.com/septivank/energy-metering-worker/internal/validator"
	"go.uber.org/zap"
)

// IngestMessage represents the incoming message from RabbitMQ
type IngestMessage struct {
	RequestID         string    `json:"request_id"`
	ClientFingerprint string    `json:"client_fingerprint"`
	IPAddress         string    `json:"ip_address"`
	UserAgent         string    `json:"user_agent"`
	ReceivedAt        time.Time `json:"received_at"`
	Payload           Payload   `json:"payload"`
}

// Payload represents the meter reading payload
type Payload struct {
	PM []PMData `json:"PM"`
}

// PMData represents a single power meter reading
type PMData struct {
	Date string `json:"date"`
	Data string `json:"data"`
	Name string `json:"name"`
}

// ProcessorService handles message processing logic
type ProcessorService struct {
	repo      *repository.Repository
	publisher *mq.Publisher
	detector  *anomaly.Detector
	validator *validator.Validator
	cfg       *config.Config
	logger    *zap.Logger
}

// NewProcessorService creates a new processor service
func NewProcessorService(
	repo *repository.Repository,
	publisher *mq.Publisher,
	detector *anomaly.Detector,
	validator *validator.Validator,
	cfg *config.Config,
	logger *zap.Logger,
) *ProcessorService {
	return &ProcessorService{
		repo:      repo,
		publisher: publisher,
		detector:  detector,
		validator: validator,
		cfg:       cfg,
		logger:    logger,
	}
}

// ProcessMessage processes an incoming meter reading message
func (s *ProcessorService) ProcessMessage(ctx context.Context, body []byte) error {
	// Parse incoming message
	var msg IngestMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Add request_id to logger context
	reqLogger := logging.WithRequestID(s.logger, msg.RequestID)
	reqLogger.Info("processing message",
		zap.String("client_fingerprint", msg.ClientFingerprint),
		zap.Int("pm_count", len(msg.Payload.PM)),
	)

	// Get or create client with IP address and user agent
	var userAgent *string
	if msg.UserAgent != "" {
		userAgent = &msg.UserAgent
	}

	client, err := s.repo.GetOrCreateClient(ctx, msg.ClientFingerprint, msg.IPAddress, userAgent)
	if err != nil {
		reqLogger.Error("failed to get or create client", zap.Error(err))
		return fmt.Errorf("failed to get or create client: %w", err)
	}

	// Process each PM reading in a transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		reqLogger.Error("failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	reqLogger.Debug("client resolved", zap.String("client_id", client.ID.String()))

	var processedEvents []mq.ProcessedEvent

	for _, pm := range msg.Payload.PM {
		event, err := s.processSingleReading(ctx, tx, client.ID, pm, msg.ReceivedAt, body, reqLogger)
		if err != nil {
			reqLogger.Error("failed to process reading",
				zap.Error(err),
				zap.String("metric_name", pm.Name),
			)
			return fmt.Errorf("failed to process reading: %w", err)
		}
		if event != nil {
			processedEvents = append(processedEvents, *event)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		reqLogger.Error("failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Publish events after successful commit
	for _, event := range processedEvents {
		if err := s.publisher.PublishProcessedEvent(ctx, event, s.cfg.RabbitMQ.WorkerRoutingKey); err != nil {
			// Log error but don't fail the entire message processing
			reqLogger.Error("failed to publish event",
				zap.Error(err),
				zap.String("client_id", event.ClientID),
				zap.String("metric_name", event.MetricName),
			)
		}
	}

	reqLogger.Info("message processed successfully",
		zap.Int("readings_count", len(processedEvents)),
	)

	return nil
}

func (s *ProcessorService) processSingleReading(
	ctx context.Context,
	tx repository.Tx,
	clientID uuid.UUID,
	pm PMData,
	receivedAt time.Time,
	rawPayload []byte,
	logger *zap.Logger,
) (*mq.ProcessedEvent, error) {
	// Convert to validator format
	metricData := validator.MetricData{
		Date: pm.Date,
		Data: pm.Data,
		Name: pm.Name,
	}

	// Validate metric data
	value, readingTime, validationResult := s.validator.ValidateMetricData(metricData, receivedAt)

	// If timestamp parsing failed, use receivedAt as fallback
	if readingTime.IsZero() {
		readingTime = receivedAt
	}

	validationStatus := "valid"
	var anomalyReason *string

	if !validationResult.IsValid {
		validationStatus = "invalid"
		anomalyReason = &validationResult.AnomalyReason
	} else {
		// Only do anomaly detection for valid readings
		// Get recent readings for this client and metric
		historicalValues, err := s.repo.GetRecentReadingsForClient(ctx, clientID, pm.Name, 10)
		if err != nil {
			logger.Warn("failed to get historical readings for anomaly detection",
				zap.Error(err),
				zap.String("metric_name", pm.Name),
			)
		} else {
			isAnomaly, reason := s.detector.DetectAnomaly(value, historicalValues)
			if isAnomaly {
				validationStatus = "invalid"
				anomalyReason = &reason
				logger.Debug("anomaly detected",
					zap.String("metric_name", pm.Name),
					zap.Float64("value", value),
					zap.String("reason", reason),
				)
			}
		}
	}

	// Insert reading into database
	reading := &db.MeterReading{
		ClientID:         clientID,
		MetricName:       pm.Name,
		MetricValue:      value,
		ReadingTimestamp: readingTime,
		ReceivedAt:       receivedAt,
		ValidationStatus: validationStatus,
		AnomalyReason:    anomalyReason,
		RawPayload:       rawPayload,
	}

	if err := s.repo.InsertMeterReadingTx(ctx, tx, reading); err != nil {
		return nil, fmt.Errorf("failed to insert reading: %w", err)
	}

	// Create processed event
	event := &mq.ProcessedEvent{
		ClientID:         clientID.String(),
		MetricName:       pm.Name,
		MetricValue:      value,
		ReadingTimestamp: readingTime.Format(time.RFC3339),
		ValidationStatus: validationStatus,
	}

	return event, nil
}
