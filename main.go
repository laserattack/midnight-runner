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

	slogLogger.Info("Program started with configuration",
		"database", dbPath,
		"database-reload-interval", dbReloadInterval,
		"max-update-attempts", dbUpdateAttemptMaxCount,
	)

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
	var dbUpdateAttemptCounter atomic.Uint32
	dbUpdateTickerStopChan := utils.Ticker(func() {
		// Protection against startup after the start of app shutdown
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Exit if database reload has failed many
		// times in a row - likely a persistent issue
		if dbUpdateAttemptCounter.Load() >= dbUpdateAttemptMaxCount {
			slogLogger.Error(
				"Persistent database reload failures - shutting down",
			)
			cancel()
			return
		}

		// Начало процесса обновления базы в RAM
		slogLogger.Info("Database updating in RAM")

		dbDonor, err := storage.LoadFromFile(dbPath)
		if err != nil {
			slogLogger.Warn("Database load failed",
				"file", dbPath,
				"error", err,
			)
			dbUpdateAttemptCounter.Add(1)
			return
		}

		// Если мы тут, то загрузка актуальной базы из файла прошла успешно

		// updated_at field in json should be updated with any change in db
		if db.UpdatedAtIsEqual(dbDonor.Metadata.UpdatedAt) &&
			dbUpdateAttemptCounter.Load() == 0 {
			slogLogger.Info("Database does not require updating")
			return
		}

		// Сюда заходим если база в RAM неактуальная или если
		// какие то ошибки были во время предыдущих попыток обновления
		// эти ошибки могут означать что
		// 1. Планировщик не смог корректно очиститься от старых работ
		// 2. Не все новые работы были зарегистрированы

		// Т.е. после ошибки надо перерегистрировать задачи,
		// даже если данные не изменились

		slogLogger.Info("Database needs to be updated")
		storage.UpdateDatabase(db, dbDonor)

		// На этом этапе база данных в RAM уже новая

		if err = scheduler.Clear(); err != nil {
			// В базе данных уже новые работы, но планировщик не очищается.
			// Продолжает содержать старые работы => они будут выполняться
			slogLogger.Warn("Scheduler clear failed",
				"error", err,
			)
			dbUpdateAttemptCounter.Add(1)
			return
		}

		err = storage.RegisterJobs(scheduler, db, quartzLogger)
		if err != nil {
			// Не все новые работы зареганы
			slogLogger.Warn("Jobs register failed",
				"error", err,
			)
			dbUpdateAttemptCounter.Add(1)
			return
		}

		dbUpdateAttemptCounter.Store(0)
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
