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
	for jk, jv := range db.Jobs {
		if jv.Type == TypeShell {
			registerShellJob(scheduler, jk, &jv, quartzLogger)
		}
	}
}

func registerShellJob(
	// quartz.Scheduler = &StdScheduler, 8 bytes
	scheduler quartz.Scheduler,
	jobKey string,
	// instead of a heavy structure pass a pointer
	jv *Job,
	quartzLogger *logger.SlogLogger,
) {
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
		quartz.NewJobKey(jobKey),
		quartzJobOpts,
	)

	quartzCronTrigger, _ := quartz.NewCronTrigger(cronExpression)

	_ = scheduler.ScheduleJob(
		quartzJobDetail,
		quartzCronTrigger,
	)
}
