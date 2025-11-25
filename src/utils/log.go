package utils

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

// Swappable Writer

type SwappableWriter struct {
	mu sync.RWMutex
	w  io.Writer
}

func NewSwappableWriter(w io.Writer) *SwappableWriter {
	return &SwappableWriter{
		w: w,
	}
}

func (sw *SwappableWriter) Write(p []byte) (n int, err error) {
	sw.mu.RLock()
	defer sw.mu.RUnlock()
	return sw.w.Write(p)
}

func (sw *SwappableWriter) Set(w io.Writer) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.w = w
}

// Buffered handler for slog

type SlogBufferedHandler struct {
	slog.Handler
	mu         sync.RWMutex
	buffer     []slog.Record
	MaxRecords int
}

func NewSlogBufferedHandler(
	h slog.Handler,
	maxRecords int,
) *SlogBufferedHandler {
	return &SlogBufferedHandler{
		Handler:    h,
		buffer:     make([]slog.Record, 0, maxRecords),
		MaxRecords: maxRecords,
	}
}

func (bh *SlogBufferedHandler) Handle(
	ctx context.Context,
	r slog.Record,
) error {
	bh.mu.Lock()
	if len(bh.buffer) >= bh.MaxRecords {
		bh.buffer = bh.buffer[1:]
	}
	bh.buffer = append(bh.buffer, r.Clone())
	bh.mu.Unlock()

	return bh.Handler.Handle(ctx, r)
}

func (bh *SlogBufferedHandler) GetLastRecords(n int) []slog.Record {
	bh.mu.RLock()
	defer bh.mu.RUnlock()
	start := 0
	if len(bh.buffer) > n {
		start = len(bh.buffer) - n
	}
	return bh.buffer[start:]
}

// Some recipes

func MaybeLogger(logger *slog.Logger, enabled bool) *slog.Logger {
	if !enabled {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return logger
}

func OpenLogFile(path string, maxSizeBytes int64) (*os.File, error) {
	if info, err := os.Stat(path); err == nil {
		if info.Size() >= maxSizeBytes {
			if err := os.WriteFile(path, nil, 0o644); err != nil {
				return nil, err
			}
		}
	}
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
}
