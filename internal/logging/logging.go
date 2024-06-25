package logging

import (
	"github.com/sajoniks/GoShort/internal/config"
	"go.uber.org/zap"
)

func ConfigureLogger(env string, cfg *config.AppConfig) (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	fields := zap.Fields(
		zap.String("host", cfg.Server.Host),
		zap.String("env", config.GetEnvironment()),
	)
	switch env {
	case "dev":
		logger, err = zap.NewDevelopment(fields)
	default:
		logger, err = zap.NewProduction(fields)
	}

	return logger, err
}
