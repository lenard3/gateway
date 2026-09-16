package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New creates a new logger with the level set in .env
// If level string can't be unmarshalled, defaults to info
func New(s string) *slog.Logger {
	var level slog.Level
	err := level.UnmarshalText([]byte(strings.ToLower(s)))
	if err != nil {
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
