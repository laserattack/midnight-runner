package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"servant/extjob"
	"servant/storage"

	"github.com/reugn/go-quartz/job"
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
	//  TODO: put it in jobs package ??
	for jk, jv := range db.Jobs {
		if jv.Type == storage.TypeShell {

			command := jv.Config.Command
			maxRetries := jv.Config.MaxRetries
			retryInterval := jv.Config.RetryInterval
			cronExpression := jv.Config.Expression
			timeout := jv.Config.Timeout

			quartzJob := extjob.NewShellJobWithCallbackAndTimeout(
				command,
				time.Duration(timeout)*time.Second,
				func(ctx context.Context, j *job.ShellJob) {
					status := j.JobStatus()
					switch status {
					case job.StatusOK:
						quartzLogger.Info("Command completed successfully",
							"command", command,
							"exit_code", j.ExitCode(),
						)
					case job.StatusFailure:
						select {
						case <-ctx.Done():
							quartzLogger.Error("Command timeout exceeded",
								"command", command,
								"exit_code", j.ExitCode(),
							)
						default:
							quartzLogger.Error("Command failed",
								"command", command,
								"exit_code", j.ExitCode(),
							)
						}
					}
				},
			)

			quartzJobOpts := &quartz.JobDetailOptions{
				MaxRetries:    maxRetries,
				RetryInterval: time.Duration(retryInterval) * time.Second,
				Replace:       false,
				Suspended:     false,
			}

			quartzJobDetail := quartz.NewJobDetailWithOptions(
				quartzJob,
				quartz.NewJobKey(jk),
				quartzJobOpts,
			)

			quartzCronTrigger, _ := quartz.NewCronTrigger(cronExpression)

			_ = scheduler.ScheduleJob(
				quartzJobDetail,
				quartzCronTrigger,
			)
		}
	}

	//  NOTE: shutdown
	<-sigChan
	quartzLogger.Info("Received shutdown signal")

	// stop the scheduler
	scheduler.Stop()
	// wait for all workers to exit
	scheduler.Wait(ctx)

	quartzLogger.Info("Scheduler stopped")
}
