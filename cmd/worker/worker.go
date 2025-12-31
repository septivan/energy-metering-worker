package main

import (
	"context"

	"github.com/septivank/energy-metering-worker/internal/anomaly"
	"github.com/septivank/energy-metering-worker/internal/config"
	"github.com/septivank/energy-metering-worker/internal/db"
	"github.com/septivank/energy-metering-worker/internal/mq"
	"github.com/septivank/energy-metering-worker/internal/repository"
	"github.com/septivank/energy-metering-worker/internal/service"
	"github.com/septivank/energy-metering-worker/internal/validator"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func startWorker(
	lc fx.Lifecycle,
	conn *mq.Connection,
	cfg *config.Config,
	logger *zap.Logger,
	processor *service.ProcessorService,
) (*mq.Consumer, error) {
	// Create context for consumer that will be cancelled on shutdown
	ctx, cancel := context.WithCancel(context.Background())

	consumer, err := mq.NewConsumer(mq.ConsumerConfig{
		Connection:       conn,
		Queue:            cfg.RabbitMQ.IngestQueue,
		DLQQueue:         cfg.RabbitMQ.DLQQueue,
		Exchange:         cfg.RabbitMQ.IngestExchange,
		RoutingKey:       cfg.RabbitMQ.IngestRoutingKey,
		PrefetchCount:    cfg.RabbitMQ.PrefetchCount,
		Logger:           logger,
		MessageProcessor: processor.ProcessMessage,
	})
	if err != nil {
		cancel()
		return nil, err
	}

	// Register lifecycle hooks
	lc.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			logger.Info("starting worker consumer",
				zap.String("queue", cfg.RabbitMQ.IngestQueue),
				zap.Int("prefetch", cfg.RabbitMQ.PrefetchCount))
			return consumer.Start(ctx)
		},
		OnStop: func(stopCtx context.Context) error {
			cancel()
			if err := consumer.Close(); err != nil {
				logger.Error("failed to close consumer", zap.Error(err))
				return err
			}
			logger.Info("worker stopped gracefully")
			return nil
		},
	})

	return consumer, nil
}

// ProvideRepository creates a new repository instance
func ProvideRepository(pool *db.Pool) *repository.Repository {
	return repository.NewRepository(pool)
}

// ProvideAnomalyDetector creates a new anomaly detector instance
func ProvideAnomalyDetector(cfg *config.Config) *anomaly.Detector {
	return anomaly.NewDetector(cfg.Anomaly.SpikeThreshold, cfg.Anomaly.MinDataPointsForDetection)
}

// ProvideValidator creates a new validator instance
func ProvideValidator(cfg *config.Config) *validator.Validator {
	return validator.NewValidator(cfg.Validation.TimestampToleranceMinutes)
}

// ProvidePublisher creates a new publisher instance
func ProvidePublisher(conn *mq.Connection, cfg *config.Config, logger *zap.Logger) (*mq.Publisher, error) {
	return mq.NewPublisher(conn, cfg.RabbitMQ.WorkerExchange, logger)
}

// ProvideProcessorService creates a new processor service instance
func ProvideProcessorService(
	repo *repository.Repository,
	publisher *mq.Publisher,
	detector *anomaly.Detector,
	validator *validator.Validator,
	cfg *config.Config,
	logger *zap.Logger,
) *service.ProcessorService {
	return service.NewProcessorService(repo, publisher, detector, validator, cfg, logger)
}

// ProvideDBPool creates a new database pool instance
func ProvideDBPool(lc fx.Lifecycle, logger *zap.Logger, cfg *config.Config) (*db.Pool, error) {
	return db.NewPool(lc, logger, cfg.Database.URL)
}

// ProvideMQConnection creates a new RabbitMQ connection instance
func ProvideMQConnection(lc fx.Lifecycle, logger *zap.Logger, cfg *config.Config) (*mq.Connection, error) {
	return mq.NewConnection(lc, logger, cfg.RabbitMQ.URL)
}
