package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the global logger instance
var Log *zap.Logger

// Init initializes the global logger
// isDevelopment: true for colorful console output, false for JSON structured logging
func Init(isDevelopment bool) error {
	var err error
	var config zap.Config

	if isDevelopment {
		// Development: colorful console output with debug level
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		// Production: JSON structured logging with info level
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	Log, err = config.Build(
		zap.AddCaller(),       // Add caller information (file:line)
		zap.AddStacktrace(zap.ErrorLevel), // Add stack trace for errors
	)

	if err != nil {
		return err
	}

	return nil
}

// Sync flushes any buffered log entries
// Should be called before application exits
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
