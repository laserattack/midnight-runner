package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"time"

	"midnight-runner/gui"
	"midnight-runner/storage"
	"midnight-runner/utils"

	"github.com/jessevdk/go-flags"
	qLogger "github.com/reugn/go-quartz/logger"
	"github.com/reugn/go-quartz/quartz"
)

type flagOpts struct {
	DatabasePath                  string `short:"d" long:"database" description:"Database file path" required:"true"`
	DatabaseReloadInterval        uint   `short:"r" long:"database-reload-interval" description:"Reload database interval in seconds" default:"10"`
	DatabaseUpdateAttemptMaxCount uint32 `short:"m" long:"max-update-attempts" description:"Max consecutive database reload attempts before shutdown" default:"10"`
	WebServerPort                 uint16 `short:"p" long:"port" description:"Web server port" default:"3777"`
	WebServerShutdownTimeout      uint   `long:"server-shutdown-timeout" description:"The time in seconds that the web server gives all connections to complete before it terminates them harshly" default:"10"`
	MemStatsInterval              uint   `long:"mem-stats-interval" description:"Interval in seconds for printing memory statistics (for leak detection)" default:"0"`
	ServerLog                     bool   `long:"server-log" description:"Log messages from HTTP server"`
}

func main() {
	//  NOTE: Setup logger

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	//  NOTE: Parse args

	var fo flagOpts
	parser := flags.NewParser(&fo, flags.Default)

	_, err := parser.Parse()
	if err != nil {
		if _, ok := err.(*flags.Error); ok {
			return
		}
		logger.Error("Failed to parse command line flags", "error", err)
		return
	}

	dbPath := fo.DatabasePath
	dbReloadInterval := fo.DatabaseReloadInterval
	dbUpdateAttemptMaxCount := fo.DatabaseUpdateAttemptMaxCount
	webServerPort := fmt.Sprint(fo.WebServerPort)
	webServerShutdownTimeout := fo.WebServerShutdownTimeout
	memStatsInterval := fo.MemStatsInterval
	serverLog := fo.ServerLog

	if dbReloadInterval == 0 {
		logger.Error("Database reload interval must be positive")
		return
	}
	if dbUpdateAttemptMaxCount == 0 {
		logger.Error("Maximum database update attempts must be positive")
		return
	}

	logger.Info("Program started with configuration",
		"database", dbPath,
		"database-reload-interval", dbReloadInterval,
		"max-update-attempts", dbUpdateAttemptMaxCount,
		"port", webServerPort,
		"server-shutdown-timeout", webServerShutdownTimeout,
		"mem-stats-interval", memStatsInterval,
		"server-log", serverLog,
	)

	//  NOTE: Setup signal's handler

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	//  NOTE: Load database

	logger.Info("Loading database", "file", dbPath)
	db, err := storage.LoadFromFile(dbPath)
	if err != nil {
		logger.Error("Database load failed",
			"file", dbPath,
			"error", err,
		)
		return
	}
	logger.Info("Database loaded successfully", "file", dbPath)

	//  NOTE: Setup context

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//  NOTE: Setup scheduler

	scheduler, err := quartz.NewStdScheduler(
		quartz.WithLogger(qLogger.NewSlogLogger(ctx, logger)),
	)
	if err != nil {
		logger.Error("Scheduler create failed", "error", err)
		return
	}

	scheduler.Start(ctx)
	defer func() {
		scheduler.Stop()
		scheduler.Wait(ctx)
		logger.Info("Scheduler stopped")
	}()

	//  NOTE: Register jobs from db

	err = storage.RegisterJobs(scheduler, db, logger)
	if err != nil {
		logger.Error("Jobs register failed",
			"error", err,
		)
		return
	}

	//  NOTE: Memory monitor

	if memStatsInterval != 0 {
		memMonitorStopChan := utils.Ticker(func() {
			select {
			case <-ctx.Done():
				return
			default:
			}
			runtime.GC()
			utils.LogMemStats(logger)
		}, time.Second*time.Duration(memStatsInterval))
		defer close(memMonitorStopChan)
	}

	//  NOTE: Updating the database in RAM

	// Without atomic, a situation is possible where
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
			logger.Error(
				"Persistent database reload failures - shutting down",
			)
			cancel()
			return
		}

		logger.Info("Database actualizing...")

		dbDonor, err := storage.LoadFromFile(dbPath)
		if err != nil {
			logger.Warn("Database load failed",
				"file", dbPath,
				"error", err,
			)
			dbUpdateAttemptCounter.Add(1)
			return
		}

		needRestartScheduler, err := storage.ActualizeDatabase(
			db,
			dbDonor,
			logger,
		)
		// Не актуализировалась
		if err != nil {
			logger.Warn("Actualize database error", "error", err)
			dbUpdateAttemptCounter.Add(1)
			return
		}

		// Актуализировалась
		// (на этом моменте базы в файле и в RAM идентичны)

		// При актуализации НЕ обновилась в RAM?
		// тогда выходим в случае если попытка нулевая
		// (не было ошибок)
		if !needRestartScheduler && dbUpdateAttemptCounter.Load() == 0 {
			logger.Info("Database is up-to-date. No need to restart scheduler")
			return
		}

		// Обновилась в RAM => надо шедулер перезапустить

		if err = scheduler.Clear(); err != nil {
			logger.Warn("Scheduler clear failed",
				"error", err,
			)
			dbUpdateAttemptCounter.Add(1)
			return
		}

		err = storage.RegisterJobs(scheduler, db, logger)
		if err != nil {
			logger.Warn("Jobs register failed",
				"error", err,
			)
			dbUpdateAttemptCounter.Add(1)
			return
		}

		dbUpdateAttemptCounter.Store(0)
		logger.Info("Database successfully updated in RAM")
	}, time.Second*time.Duration(dbReloadInterval))
	defer close(dbUpdateTickerStopChan)

	//  NOTE: Start Web Server

	serverLogger := logger
	if !serverLog {
		serverLogger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	server := gui.CreateWebServer(webServerPort, serverLogger, db)
	go func() {
		logger.Info("Starting web server", "port", webServerPort)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Error("Web server error", "error", err)
		}
	}()

	//  NOTE: Shutdown

	select {
	case <-sigChan:
		logger.Info("Received shutdown signal")
	case <-ctx.Done():
		logger.Info("Shutdown triggered by internal error")
	}

	//  NOTE: Shutdown Web Server

	serverCtx, serverCancel := context.WithTimeout(
		context.Background(),
		time.Duration(webServerShutdownTimeout)*time.Second,
	)
	defer serverCancel()

	if err := server.Shutdown(serverCtx); err != nil {
		logger.Error("Web server shutdown error", "error", err)
	} else {
		logger.Info("Web server stopped")
	}
}
