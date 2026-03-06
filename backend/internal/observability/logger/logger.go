package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// LogConfig holds configuration for the structured logger.
type LogConfig struct {
	// Environment is "production" or "development".
	Environment string
	// Level is one of "debug", "info", "warn", "error".
	Level string
	// ServiceName identifies the service in every log entry.
	ServiceName string
	// Version is the service version (e.g., "1.0.0").
	Version string
	// DebugSampleRate controls 1-in-N sampling for debug-level logs in production.
	// Default: 100 (emit 1 out of every 100 debug logs).
	DebugSampleRate int
	// RedactedFields overrides the default list of field names to redact.
	// If nil, defaults are used.
	RedactedFields []string
}

// NewLogger creates a production-grade zerolog.Logger.
//
// Production mode: JSON output, configured level, caller info for warn+, debug sampling, redaction hook.
// Development mode: colored console, debug level, caller info on all levels, no sampling, no redaction.
//
// This function does NOT set the zerolog global logger — callers may do so explicitly.
func NewLogger(cfg LogConfig) zerolog.Logger {
	if cfg.DebugSampleRate <= 0 {
		cfg.DebugSampleRate = 100
	}

	var output io.Writer
	var lvl zerolog.Level

	isDev := cfg.Environment == "development" || cfg.Environment == "dev"

	if isDev {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		lvl = zerolog.DebugLevel
	} else {
		output = os.Stdout
		var err error
		lvl, err = zerolog.ParseLevel(cfg.Level)
		if err != nil {
			lvl = zerolog.InfoLevel
		}
	}

	hostname, _ := os.Hostname()

	ctx := zerolog.New(output).
		Level(lvl).
		With().
		Timestamp().
		Str("service", cfg.ServiceName).
		Str("version", cfg.Version).
		Str("env", cfg.Environment).
		Str("hostname", hostname).
		Int("pid", os.Getpid())

	// Add caller info: all levels in dev, warn+ in production.
	if isDev {
		ctx = ctx.Caller()
	}

	logger := ctx.Logger()

	// Apply redaction hook in production.
	if !isDev {
		fields := cfg.RedactedFields
		if len(fields) == 0 {
			fields = DefaultRedactedFields
		}
		hook := NewRedactionHook(fields)
		logger = logger.Hook(hook)
	}

	// Apply debug-level sampling in production.
	if !isDev {
		logger = logger.Sample(NewDebugSampler(cfg.DebugSampleRate))
	}

	return logger
}
