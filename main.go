package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"servant/storage"

	"github.com/jessevdk/go-flags"
	"github.com/reugn/go-quartz/logger"
	"github.com/reugn/go-quartz/quartz"
)

func main() {
	//  NOTE: Setup logger
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	//  NOTE: Parse args
	var flagOpts struct {
		DatabasePath string `short:"d" long:"database" description:"Database file path" required:"true"`
	}

	parser := flags.NewParser(&flagOpts, flags.Default)

	_, err := parser.Parse()
	if err != nil {
		// Checking err implements a *flags.Error type
		if _, ok := err.(*flags.Error); ok {
			return
		} else {
			// System problem
			slogLogger.Error("Failed to parse command line flags",
				"error", err,
			)
		}
		return
	}

	//  NOTE: Setup signal's handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	//  NOTE: Load database from arg
	dbPath := flagOpts.DatabasePath
	slogLogger.Info("Loading database",
		"file", dbPath,
	)

	//  TODO: База в RAM должна обновляться если изменяется на диске
	// При этом изменение на диске может поломать базу (невалидный json)
	db, err := storage.LoadFromFile(dbPath)
	if err != nil {
		slogLogger.Error("Database load failed",
			"file", dbPath,
			"error", err,
		)
		return
	}

	slogLogger.Info("Database loaded successfully", "file", dbPath)

	//  NOTE: Setup scheduler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quartzLogger := logger.NewSlogLogger(ctx, slogLogger)
	scheduler, err := quartz.NewStdScheduler(quartz.WithLogger(quartzLogger))
	if err != nil {
		slogLogger.Error("Scheduler create failed",
			"error", err,
		)
		return
	}

	scheduler.Start(ctx)
	defer func() {
		scheduler.Stop()
		scheduler.Wait(ctx)
		quartzLogger.Info("Scheduler stopped")
	}()

	//  NOTE: Register jobs from db
	err = storage.RegisterJobs(scheduler, db, quartzLogger)
	if err != nil {
		slogLogger.Error("Jobs register failed",
			"error", err,
		)
		return
	}

	//  NOTE: Shutdown
	<-sigChan
	quartzLogger.Info("Received shutdown signal")
}
