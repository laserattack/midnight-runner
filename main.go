package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"servant/storage"

	"github.com/reugn/go-quartz/logger"
	"github.com/reugn/go-quartz/quartz"
)

func main() {
	//  NOTE: setup signal's handler and logger
	sigChan := make(chan os.Signal, 1)
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// from https://pkg.go.dev/os#Signal:
	// The only signal values guaranteed to be present in the os package
	// on all systems are os.Interrupt (send the process an interrupt)
	// and os.Kill (force the process to exit)

	// but os.Kill can not be trapped
	signal.Notify(sigChan, os.Interrupt)

	dbName := "database_example.json"
	db, err := storage.LoadFromFile(dbName)
	if err != nil {
		slog.Error("Database load failed",
			"file", dbName,
			"error", err,
		)
		os.Exit(1)
	}

	//  NOTE: setup scheduler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quartzLogger := logger.NewSlogLogger(ctx, slogLogger)
	scheduler, _ := quartz.NewStdScheduler(
		quartz.WithLogger(quartzLogger),
	)
	scheduler.Start(ctx)

	//  NOTE: schedule jobs
	storage.RegisterJobs(scheduler, db, quartzLogger)

	//  NOTE: shutdown
	<-sigChan
	quartzLogger.Info("Received shutdown signal")

	// stop the scheduler
	scheduler.Stop()
	// wait for all workers to exit
	scheduler.Wait(ctx)

	quartzLogger.Info("Scheduler stopped")
}
