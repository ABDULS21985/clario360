package observability

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type loggerKey struct{}

// NewLogger creates a configured zerolog.Logger.
func NewLogger(level, format, serviceName string) zerolog.Logger {
	var output io.Writer

	if format == "console" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	} else {
		output = os.Stdout
	}

	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.DebugLevel
	}

	return zerolog.New(output).
		Level(lvl).
		With().
		Timestamp().
		Str("service", serviceName).
		Caller().
		Logger()
}

// WithLogger stores a logger in the context.
func WithLogger(ctx context.Context, logger zerolog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// LoggerFromContext retrieves the logger from context, or returns a default.
func LoggerFromContext(ctx context.Context) zerolog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(zerolog.Logger); ok {
		return logger
	}
	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}
