// Package log provides a single structured logger setup for all services.
//
// Output is JSON (machine-parseable for Loki/OpenSearch) and includes the
// service name on every record.
package log

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a JSON slog.Logger annotated with the service name.
func New(service, level string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(level),
	})
	return slog.New(h).With(slog.String("service", service))
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
