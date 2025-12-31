package main

import (
	"github.com/septivank/energy-metering-worker/internal/config"
	"github.com/septivank/energy-metering-worker/internal/logging"
	"go.uber.org/zap"
)

func newLogger(cfg *config.Config) (*zap.Logger, error) {
	return logging.NewLogger(cfg.ServiceName)
}
