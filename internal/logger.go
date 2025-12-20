package internal

import (
	"log/slog"
	"os"
)

// NewLogger creates a configured slog.Logger based on the environment.
// For DEV environment (isDev=true):
//   - Uses text handler for human-readable output
//   - Sets log level to DEBUG for detailed logging
//
// For PROD environment (isDev=false):
//   - Uses JSON handler for structured logging
//   - Sets log level to INFO for production use
func NewLogger(isDev bool) *slog.Logger {
	var handler slog.Handler
	if isDev {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	return slog.New(handler)
}
