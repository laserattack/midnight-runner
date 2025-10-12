package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"servant/extjob"

	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/logger"
	"github.com/reugn/go-quartz/quartz"
)

func main() {
	//  NOTE: setup context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//  NOTE: setup logger
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	quartzLogger := logger.NewSlogLogger(ctx, slogLogger)

	//  NOTE: setup scheduler
	scheduler, _ := quartz.NewStdScheduler(
		quartz.WithLogger(quartzLogger),
	)
	scheduler.Start(ctx)

	//  NOTE: setup signal's handler
	sigChan := make(chan os.Signal, 1)

	// from https://pkg.go.dev/os#Signal:
	// The only signal values guaranteed to be present in the os package
	// on all systems are os.Interrupt (send the process an interrupt)
	// and os.Kill (force the process to exit)

	// but os.Kill can not be trapped
	signal.Notify(sigChan, os.Interrupt)

	//  NOTE: create jobs with timeout
	command := "sleep 4"
	cronTrigger, _ := quartz.NewCronTrigger("*/10 * * * * *")

	shellJob := extjob.NewShellJobWithCallbackAndTimeout(
		command,
		2*time.Second,
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

	//  NOTE: job options
	opts := quartz.NewDefaultJobDetailOptions()
	opts.MaxRetries = 1
	opts.RetryInterval = 1 * time.Second
	opts.Replace = false
	opts.Suspended = false

	jobDetail := quartz.NewJobDetailWithOptions(
		shellJob,
		quartz.NewJobKey("shellJob"),
		opts,
	)

	//  NOTE: start jobs
	_ = scheduler.ScheduleJob(jobDetail, cronTrigger)

	//  NOTE: shutdown
	<-sigChan
	quartzLogger.Info("Received shutdown signal")

	// stop the scheduler
	scheduler.Stop()
	// wait for all workers to exit
	scheduler.Wait(ctx)

	quartzLogger.Info("Scheduler stopped")
}
