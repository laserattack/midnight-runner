package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

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
	signal.Notify(sigChan, os.Interrupt)

	//  NOTE: start jobs
	cronTrigger, _ := quartz.NewCronTrigger("*/5 * * * * *")
	shellJob := job.NewShellJobWithCallback("ls -la",
		func(ctx context.Context, j *job.ShellJob) {
			status := j.JobStatus()
			switch status {
			case job.StatusOK:
				quartzLogger.Info("Command completed successfully",
					"command", "ls -la",
					"exit_code", j.ExitCode(),
					//"stdout", j.Stdout(),
				)
			case job.StatusFailure:
				quartzLogger.Error("Command failed",
					"command", "ls -la",
					"exit_code", j.ExitCode(),
					"stderr", j.Stderr(),
				)
			}
		},
	)

	_ = scheduler.ScheduleJob(
		quartz.NewJobDetail(shellJob, quartz.NewJobKey("shellJob")),
		cronTrigger,
	)

	//  NOTE: shutdown
	<-sigChan
	quartzLogger.Info("Received shutdown signal")

	// stop the scheduler
	scheduler.Stop()
	// wait for all workers to exit
	scheduler.Wait(ctx)

	quartzLogger.Info("Scheduler stopped")
}
