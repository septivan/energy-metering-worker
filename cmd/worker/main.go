package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/septivank/energy-metering-worker/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	// Load .env file - flexible path for both Linux (pods/containers) and Windows
	envPaths := []string{
		".env",                     // Current working directory (works in pods/containers)
		"../../.env",               // If running from bin/ subdirectory
		filepath.Join(".", ".env"), // Explicit current dir
	}

	// Try to find .env file starting from current directory and moving up
	if workDir, err := os.Getwd(); err == nil {
		// Add parent directories for flexibility
		parentDir := filepath.Dir(workDir)
		grandParentDir := filepath.Dir(parentDir)

		envPaths = append(envPaths,
			filepath.Join(workDir, ".env"),
			filepath.Join(parentDir, ".env"),
			filepath.Join(grandParentDir, ".env"),
		)
	}

	envLoaded := false
	for _, envPath := range envPaths {
		// Check if file exists first to avoid unnecessary errors
		if _, err := os.Stat(envPath); err == nil {
			if err := godotenv.Load(envPath); err == nil {
				absPath, _ := filepath.Abs(envPath)
				fmt.Printf("Loaded environment from: %s\n", absPath)
				envLoaded = true
				break
			}
		}
	}

	if !envLoaded {
		fmt.Println("No .env file found, using system environment variables (OK for pods/containers)")
	}

	app := fx.New(
		fx.Provide(
			config.Load,
			newLogger,
			ProvideDBPool,
			ProvideRepository,
			ProvideAnomalyDetector,
			ProvideValidator,
			ProvideMQConnection,
			ProvidePublisher,
			ProvideProcessorService,
		),
		fx.Invoke(startWorker),
	)

	// Setup signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create a temporary logger for startup error messages
	tempLogger, _ := newLogger(&config.Config{ServiceName: "energy-metering-worker"})
	tempLogger.Info("starting application...", zap.String("timeout", "30s"))

	startCtx, startCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer startCancel()

	if err := app.Start(startCtx); err != nil {
		// Check if it's a timeout error
		if startCtx.Err() == context.DeadlineExceeded {
			tempLogger.Error("APPLICATION START TIMEOUT: Failed to start within 30 seconds. This usually means a dependency (Database or RabbitMQ) is not accessible. Check the error messages above for specific connection failures.")
		}
		panic(err)
	}

	// Wait for interrupt signal
	<-ctx.Done()

	// Stop application gracefully
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()
	if err := app.Stop(stopCtx); err != nil {
		fmt.Println("error stopping app:", err)
	}
}
