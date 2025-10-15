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
		DBName string `short:"d" long:"db" description:"Database file name" required:"true"`
		Help   bool   `short:"h" long:"help" description:"Show this help message"`
	}

	parser := flags.NewParser(&opts, flags.Default)

	_, err := parser.Parse()
	if err != nil {
		if _, ok := err.(*flags.Error); ok {
			parser.WriteHelp(os.Stdout)
		} else {
			// system problem
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
	//  TODO: Тут должен быть путь а не имя
	dbName := opts.DBName
	slogLogger.Info("Loading database", "file", dbName)

	db, err := storage.LoadFromFile(dbName)
	if err != nil {
		slogLogger.Error("Database load failed",
			"file", dbName,
			"error", err,
		)
		os.Exit(0)
	}

	//  NOTE: Setup scheduler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quartzLogger := logger.NewSlogLogger(ctx, slogLogger)
	scheduler, _ := quartz.NewStdScheduler(
		quartz.WithLogger(quartzLogger),
	)
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
