package utils

import (
	"fmt"
	"log/slog"
	"runtime"
)

func LogMemStats(logger *slog.Logger) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	bytesToMB := func(bytes uint64) float64 {
		return float64(bytes) / 1024 / 1024
	}

	logger.Info("Memory stats",
		"Heap allocated memory (MB)", fmt.Sprintf("%.2f", bytesToMB(m.HeapAlloc)),
		"Heap objects count", m.HeapObjects,
		"Garbage collections count", m.NumGC,
		"Active goroutines count", runtime.NumGoroutine(),
		"Next GC threshold (MB)", fmt.Sprintf("%.2f", bytesToMB(m.NextGC)),
	)
}
