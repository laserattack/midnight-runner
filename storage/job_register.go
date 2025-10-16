package storage

import (
	"context"
	"time"

	"servant/extjob"

	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/logger"
	"github.com/reugn/go-quartz/quartz"
)

func RegisterJobs(
	// quartz.Scheduler = &StdScheduler, 8 bytes
	scheduler quartz.Scheduler,
	db *Database,
	quartzLogger *logger.SlogLogger,
) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// j - *Job, 8 bytes (cheap copying)
	for jk, j := range db.Jobs {
		if j.Type == TypeShell {
			registerShellJob(scheduler, jk, j, quartzLogger)
		}
	}
}

func registerShellJob(
	// quartz.Scheduler = &StdScheduler, 8 bytes
	scheduler quartz.Scheduler,
	jobKey string,
	// instead of a heavy structure pass a pointer
	j *Job,
	quartzLogger *logger.SlogLogger,
) {
	description := j.Description
	command := j.Config.Command
	maxRetries := j.Config.MaxRetries
	retryInterval := j.Config.RetryInterval
	cronExpression := j.Config.CronExpression
	timeout := j.Config.Timeout

	quartzJob := extjob.NewShellJobWithCallbackAndTimeout(
		command,
		time.Duration(timeout)*time.Second,
		func(ctx context.Context, j *job.ShellJob) {
			status := j.JobStatus()
			switch status {
			case job.StatusOK:
				quartzLogger.Info("Command completed successfully",
					"description", description,
					"command", command,
					"cron_expression", cronExpression,
					"exit_code", j.ExitCode(),
				)
			case job.StatusFailure:
				select {
				case <-ctx.Done():
					quartzLogger.Error("Command timeout exceeded",
						"description", description,
						"command", command,
						"cron_expression", cronExpression,
						"exit_code", j.ExitCode(),
					)
				default:
					quartzLogger.Error("Command failed",
						"description", description,
						"command", command,
						"cron_expression", cronExpression,
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
		quartz.NewJobKey(jobKey),
		quartzJobOpts,
	)

	//  TODO: Обработка ошибок
	quartzCronTrigger, _ := quartz.NewCronTrigger(cronExpression)

	//  TODO: Обработка ошибок
	_ = scheduler.ScheduleJob(
		quartzJobDetail,
		quartzCronTrigger,
	)
}
