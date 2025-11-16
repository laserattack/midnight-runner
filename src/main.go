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

	"cronshroom/gui"
	"cronshroom/storage"
	"cronshroom/utils"

	"github.com/jessevdk/go-flags"
	qLogger "github.com/reugn/go-quartz/logger"
	"github.com/reugn/go-quartz/quartz"
)

const (
	defaultDatabaseName = "cronshroom-database.json"
	defaultLogFileName  = "cronshroom-log"
)

type flagOpts struct {
	DatabasePath                string `short:"d" long:"database" description:"Path to the database file (default: in system config directory)"`
	WebServerPort               uint16 `short:"p" long:"port" description:"Web server port" default:"3777"`
	DatabaseSyncInterval        uint   `long:"sync-interval" description:"Database sync interval in seconds" default:"1"`
	DatabaseSyncAttemptMaxCount uint32 `long:"max-sync-attempts" description:"Max consecutive database sync attempts before shutdown" default:"10"`
	WebServerShutdownTimeout    uint   `long:"server-shutdown-timeout" description:"The time in seconds that the web server gives all connections to complete before it terminates them harshly" default:"10"`
	MemStatsInterval            uint   `long:"mem-stats-interval" description:"Interval in seconds for logging memory statistics (for leak detection). It also causes garbage collection. Disable - 0 value" default:"1800"`
	HTTPLog                     bool   `long:"http-log" description:"Log messages about HTTP connections"`
	LogFileMaxSizeBytes         uint64 `long:"log-file-max-size" description:"Log file max size in bytes (if the max size is reached the file will be overwritten)" default:"10485760"`
	Cleanup                     bool   `long:"cleanup" description:"Delete all files created by the program in system config directory and shut down"`
}

// TODO: doc files https://github.com/reugn/go-quartz/blob/master/job/doc.go

func main() {
	// NOTE: Setup logger

	logWriter := utils.NewSwappableWriter(os.Stdout)
	logHandler := utils.NewSlogBufferedHandler(
		slog.NewTextHandler(logWriter, nil),
		1000,
	)
	logger := slog.New(logHandler)

	// NOTE: Parse args

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

	logFileMaxSizeBytes := fo.LogFileMaxSizeBytes
	dbPath := fo.DatabasePath
	dbSyncInterval := fo.DatabaseSyncInterval
	dbSyncAttemptMaxCount := fo.DatabaseSyncAttemptMaxCount
	webServerPort := fmt.Sprint(fo.WebServerPort)
	webServerShutdownTimeout := fo.WebServerShutdownTimeout
	memStatsInterval := fo.MemStatsInterval
	HTTPLog := fo.HTTPLog
	cleanup := fo.Cleanup

	logger.Info("Program started with flags",
		"database", dbPath,
		"sync-interval", dbSyncInterval,
		"max-sync-attempts", dbSyncAttemptMaxCount,
		"port", webServerPort,
		"server-shutdown-timeout", webServerShutdownTimeout,
		"mem-stats-interval", memStatsInterval,
		"http-log", HTTPLog,
		"log-file-max-size", logFileMaxSizeBytes,
		"cleanup", cleanup,
	)

	if dbSyncInterval == 0 {
		logger.Error("Database reload interval must be positive")
		return
	}

	if dbSyncAttemptMaxCount == 0 {
		logger.Error("Maximum database update attempts must be positive")
		return
	}

	//
	logFilePath, err := utils.ResolveFileInDefaultConfigDir(
		defaultLogFileName,
		func(fullPath string) error {
			return nil
		},
	)
	if err != nil {
		logger.Warn("Failed to resolve log file path", "error", err)
	} else if !cleanup {
		logger.Info("Loading log file", "file", logFilePath)
		logFile, err := utils.OpenLogFile(
			logFilePath,
			int64(logFileMaxSizeBytes),
		)
		if err != nil {
			logger.Error("Failed to open log file",
				"file", logFilePath,
				"error", err,
			)
		} else {
			logger.Info("Log file loaded successfully", "file", logFilePath)
			logWriter.Set(io.MultiWriter(os.Stdout, logFile))
			defer func() {
				if err := logFile.Close(); err != nil {
					slog.New(slog.NewTextHandler(os.Stdout, nil)).Error(
						"Failed to close log file",
						"file", logFilePath,
						"error", err,
					)
				}
			}()
		}
	} else if cleanup {
		if err := os.Remove(logFilePath); err != nil {
			logger.Warn("Failed to delete log file",
				"file", logFilePath,
				"error", err,
			)
		}
	}

	//

	if dbPath == "" {
		dbPath, err = utils.ResolveFileInDefaultConfigDir(
			defaultDatabaseName,
			func(fullPath string) error {
				return storage.New().SaveToFile(fullPath)
			},
		)
		if err != nil {
			logger.Error("Failed to resolve database file",
				"file", dbPath,
				"error", err,
			)
			return
		}
	}

	if cleanup {
		if err := os.Remove(dbPath); err != nil {
			logger.Warn("Failed to delete database file",
				"file", dbPath,
				"error", err,
			)
		}
		logger.Info("Cleanup done")
		return
	}

	// NOTE: Setup signal's handler

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// NOTE: Load database

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

	// NOTE: Setup context

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// NOTE: Setup scheduler

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

	// NOTE: Register jobs from db

	err = storage.RegisterJobs(scheduler, db, logger)
	if err != nil {
		logger.Error("Jobs register failed",
			"error", err,
		)
		return
	}

	// NOTE: Memory monitor

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

	// NOTE: Save db to file

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

		// it is important here that if there are any working jobs,
		// they will not be interrupted, but will be delete
		// from memory after the end of the work
		if err = scheduler.Clear(); err != nil {
			logger.Warn("Scheduler clear failed", "error", err)
			dbSyncFailureCount.Add(1)
			return
		}
		logger.Info("Scheduler is cleared")

		err = storage.RegisterJobs(scheduler, db, logger)
		if err != nil {
			logger.Warn("Jobs register failed", "error", err)
			dbSyncFailureCount.Add(1)
			return
		}
		logger.Info("Jobs are registered in scheduler")

		if err := db.SaveToFile(dbPath); err != nil {
			logger.Warn("Save database to file failed", "error", err)
			dbSyncFailureCount.Add(1)
			return
		}
		logger.Info("Database is saved to file")

		prevUpdatedAt.Store(db.Metadata.UpdatedAt)
		dbSyncFailureCount.Store(0)
	}, time.Second*time.Duration(dbSyncInterval))
	defer close(dbSyncTickerStopChan)

	// NOTE: Start Web Server

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
			cancel()
		}
	}()

	// NOTE: Shutdown

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
