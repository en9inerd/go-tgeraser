package log

import (
	"log/slog"
	"os"
)

// NewLogger returns a logger that emits logs at INFO level by default, or DEBUG when verbose is true.
func NewLogger(verbose bool) *slog.Logger {
	var level slog.Level

	if verbose {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(handler)
}
