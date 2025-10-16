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
	var opts struct {
		DBPath string `short:"d" long:"db" description:"Database file path" required:"true"`
		Help   bool   `short:"h" long:"help" description:"Show this help message"`
	}

	parser := flags.NewParser(&opts, flags.Default)

	_, err := parser.Parse()
	if err != nil {
		// Checking err implements a *flags.Error type
		if _, ok := err.(*flags.Error); ok {
			parser.WriteHelp(os.Stdout)
		} else {
			// System problem
			slogLogger.Error("Failed to parse command line flags",
				"error", err,
			)
		}
		os.Exit(0)
	}

	//  NOTE: Setup signal's handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	//  NOTE: Load database from arg
	dbPath := opts.DBPath
	slogLogger.Info("Loading database", "file", dbPath)

	//  TODO: База в RAM должна обновляться если изменяется на диске
	// При этом изменение на диске может поломать базу (невалидный json)
	db, err := storage.LoadFromFile(dbPath)
	if err != nil {
		slogLogger.Error("Database load failed", "file", dbPath, "error", err)
		os.Exit(0)
	}

	slogLogger.Info("Database loaded successfully", "file", dbPath)

	//  NOTE: Setup scheduler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quartzLogger := logger.NewSlogLogger(ctx, slogLogger)
	scheduler, err := quartz.NewStdScheduler(quartz.WithLogger(quartzLogger))
	if err != nil {
		slogLogger.Error("Scheduler create failed", "error", err)
		os.Exit(0)
	}

	scheduler.Start(ctx)

	//  NOTE: Register jobs from db
	storage.RegisterJobs(scheduler, db, quartzLogger)

	//  NOTE: Shutdown
	<-sigChan
	quartzLogger.Info("Received shutdown signal")

	// Stop scheduler
	scheduler.Stop()
	// Wait for all workers to exit
	scheduler.Wait(ctx)

	quartzLogger.Info("Scheduler stopped")
}
