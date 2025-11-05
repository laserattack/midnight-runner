package utils

import (
	"io"
	"log/slog"
)

func MaybeLogger(logger *slog.Logger, enabled bool) *slog.Logger {
	if !enabled {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return logger
}
