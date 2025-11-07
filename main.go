package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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
	DatabasePath                string `short:"d" long:"database" description:"Database file path"`
	WebServerPort               uint16 `short:"p" long:"port" description:"Web server port" default:"3777"`
	DatabaseSyncInterval        uint   `long:"sync-interval" description:"Database sync interval in seconds" default:"1"`
	DatabaseSyncAttemptMaxCount uint32 `long:"max-sync-attempts" description:"Max consecutive database sync attempts before shutdown" default:"10"`
	WebServerShutdownTimeout    uint   `long:"server-shutdown-timeout" description:"The time in seconds that the web server gives all connections to complete before it terminates them harshly" default:"10"`
	MemStatsInterval            uint   `long:"mem-stats-interval" description:"Interval in seconds for printing memory statistics (for leak detection)" default:"0"`
	HTTPLog                     bool   `long:"http-log" description:"Log messages about HTTP connections"`
}

//  TODO: По каждой джобе должна быть возможность посмотреть ее логи
//  TODO: Потестить граничные входные данные в полях ввода

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
	if dbPath == "" {
		dbDir, err := os.UserConfigDir()
		if err != nil {
			logger.Error("The -d / --database flag is not specified" +
				" and the default config directory could not be determined")
			return
		}

		dbPath = filepath.Join(dbDir, "midnight-runner-database.json")

		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			db := storage.New()
			if err = db.SaveToFile(dbPath); err != nil {
				logger.Error("Failed to save database", "error", err)
				return
			}
		}
	}

	dbSyncInterval := fo.DatabaseSyncInterval
	dbSyncAttemptMaxCount := fo.DatabaseSyncAttemptMaxCount
	webServerPort := fmt.Sprint(fo.WebServerPort)
	webServerShutdownTimeout := fo.WebServerShutdownTimeout
	memStatsInterval := fo.MemStatsInterval
	HTTPLog := fo.HTTPLog

	if dbSyncInterval == 0 {
		logger.Error("Database reload interval must be positive")
		return
	}

	if dbSyncAttemptMaxCount == 0 {
		logger.Error("Maximum database update attempts must be positive")
		return
	}

	logger.Info("Program started with configuration",
		"database", dbPath,
		"sync-interval", dbSyncInterval,
		"max-sync-attempts", dbSyncAttemptMaxCount,
		"port", webServerPort,
		"server-shutdown-timeout", webServerShutdownTimeout,
		"mem-stats-interval", memStatsInterval,
		"http-log", HTTPLog,
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

	//  NOTE: Save db to file

	var dbSyncFailureCount atomic.Uint32
	var prevUpdatedAt atomic.Int64
	prevUpdatedAt.Store(db.Metadata.UpdatedAt)

	dbSyncTickerStopChan := utils.Ticker(func() {
		// Protection against startup after the start of app shutdown
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Exit if database reload has failed many
		// times in a row - likely a persistent issue
		if dbSyncFailureCount.Load() >= dbSyncAttemptMaxCount {
			logger.Error(
				"Persistent database reload failures - shutting down",
			)
			cancel()
			return
		}

		db.Mu.RLock()
		defer db.Mu.RUnlock()

		if db.Metadata.UpdatedAt <= prevUpdatedAt.Load() {
			return
		}

		if err = scheduler.Clear(); err != nil {
			logger.Warn("Scheduler clear failed", "error", err)
			dbSyncFailureCount.Add(1)
			return
		}

		err = storage.RegisterJobs(scheduler, db, logger)
		if err != nil {
			logger.Warn("Jobs register failed", "error", err)
			dbSyncFailureCount.Add(1)
			return
		}

		if err := db.SaveToFile(dbPath); err != nil {
			logger.Warn("Save database to file failed", "error", err)
			dbSyncFailureCount.Add(1)
			return
		}

		prevUpdatedAt.Store(db.Metadata.UpdatedAt)
		dbSyncFailureCount.Store(0)
	}, time.Second*time.Duration(dbSyncInterval))
	defer close(dbSyncTickerStopChan)

	//  NOTE: Start Web Server

	httpLogger := utils.MaybeLogger(logger, HTTPLog)
	server := gui.CreateWebServer(
		webServerPort,
		httpLogger,
		logger,
		db,
		ctx,
	)
	go func() {
		logger.Info("Starting web server", "port", webServerPort)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Error("Web server error", "error", err)
		}
	}()

	//  NOTE: Shutdown

	{
		select {
		case <-sigChan:
			logger.Info("Received shutdown signal")
		case <-ctx.Done():
			logger.Info("Shutdown triggered by internal error")
		}
	}

	{
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
}
