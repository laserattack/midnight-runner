package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"servant/storage"
	"servant/utils"

	"github.com/jessevdk/go-flags"
	"github.com/reugn/go-quartz/logger"
	"github.com/reugn/go-quartz/quartz"
)

func main() {
	//  NOTE: Setup logger
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	//  NOTE: Parse args
	var flagOpts struct {
		DatabasePath                  string `short:"d" long:"database" description:"Database file path" required:"true"`
		DatabaseReloadInterval        uint   `short:"r" long:"database-reload-interval" description:"Reload database interval in seconds" required:"true"`
		DatabaseUpdateAttemptMaxCount uint32 `short:"m" long:"max-update-attempts" description:"Max consecutive database reload attempts before shutdown" default:"10"`
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

	dbPath := flagOpts.DatabasePath
	dbReloadInterval := flagOpts.DatabaseReloadInterval
	dbUpdateAttemptMaxCount := flagOpts.DatabaseUpdateAttemptMaxCount

	if dbReloadInterval == 0 {
		slogLogger.Error("Database reload interval must be positive")
		return
	}

	if dbUpdateAttemptMaxCount == 0 {
		slogLogger.Error("Maximum database update attempts must be positive")
		return
	}

	//  NOTE: Setup signal's handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	//  NOTE: Load database from arg
	slogLogger.Info("Loading database", "file", dbPath)
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
		slogLogger.Error("Scheduler create failed", "error", err)
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

	//  NOTE: Updating the database in RAM

	// Without atomic, a situation is possible (im not sure) where
	// 2 goroutines increment a variable at the same time and
	// it increases by 1 instead of 2
	var dbUpdateAttemptCount atomic.Uint32
	dbUpdateTickerStopChan := utils.Ticker(func() {
		// Protection against startup after the start of app shutdown
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Exit if database reload has failed many
		// times in a row - likely a persistent issue
		if dbUpdateAttemptCount.Load() >= dbUpdateAttemptMaxCount {
			slogLogger.Error(
				"Persistent database reload failures - shutting down",
			)
			cancel()
			return
		}
		slogLogger.Info("Database updating in RAM")

		dbDonor, err := storage.LoadFromFile(dbPath)
		if err != nil {
			slogLogger.Error("Database load failed",
				"file", dbPath,
				"error", err,
			)
			dbUpdateAttemptCount.Add(1)
			return
		}

		// updated_at field in json should be updated with any change in db
		if db.UpdatedAtIsEqual(dbDonor.Metadata.UpdatedAt) {
			slogLogger.Info("No changes in database")
			dbUpdateAttemptCount.Store(0)
			return
		}

		slogLogger.Info("Changes detected in database")
		storage.UpdateDatabase(db, dbDonor)

		if err = scheduler.Clear(); err != nil {
			slogLogger.Error("Scheduler clear failed",
				"error", err,
			)
			dbUpdateAttemptCount.Add(1)
			return
		}

		err = storage.RegisterJobs(scheduler, db, quartzLogger)
		if err != nil {
			slogLogger.Error("Jobs register failed",
				"error", err,
			)
			dbUpdateAttemptCount.Add(1)
			return
		}

		dbUpdateAttemptCount.Store(0)
		slogLogger.Info("Database successfully updated in RAM")
	}, time.Second*time.Duration(dbReloadInterval))
	defer close(dbUpdateTickerStopChan)

	//  NOTE: Shutdown
	select {
	case <-sigChan:
		quartzLogger.Info("Received shutdown signal")
	case <-ctx.Done():
		quartzLogger.Info("Shutdown triggered by internal error")
	}
}
