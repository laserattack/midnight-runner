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
) error {
	db.Mu.RLock()
	defer db.Mu.RUnlock()

	// j - *Job, 8 bytes (cheap copying)
	for jk, j := range db.Jobs {
		if j.Type == TypeShell {
			//  TODO: Не уверен что хороший вариант так обрывать регистрацию
			err := registerShellJob(scheduler, db, jk, j, quartzLogger)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func registerShellJob(
	// quartz.Scheduler = &StdScheduler, 8 bytes
	scheduler quartz.Scheduler,
	db *Database,
	jobKey string,
	// instead of a heavy structure pass a pointer
	j *Job,
	quartzLogger *logger.SlogLogger,
) error {
	description := j.Description
	command := j.Config.Command
	maxRetries := j.Config.MaxRetries
	retryInterval := j.Config.RetryInterval
	cronExpression := j.Config.CronExpression
	timeout := j.Config.Timeout

	logFields := func() []any {
		return []any{
			"description", description,
			"command", command,
			"cron_expression", cronExpression,
		}
	}

	afterExec := func(ctx context.Context, qj *job.ShellJob) {
		db.Mu.Lock()
		j.Config.Status = StatusEnable
		db.Mu.Unlock()

		status := qj.JobStatus()
		fields := logFields()

		switch status {
		case job.StatusOK:
			quartzLogger.Info("Command completed successfully", fields...)
		case job.StatusFailure:
			select {
			case <-ctx.Done():
				quartzLogger.Error("Command timeout exceeded", fields...)
			default:
				quartzLogger.Error("Command failed", fields...)
			}
		}
	}

	beforeExec := func(ctx context.Context, qj *job.ShellJob) {
		db.Mu.Lock()
		j.Config.Status = StatusActive
		db.Mu.Unlock()

		fields := logFields()
		quartzLogger.Info("Start command execution", fields...)
	}

	quartzJob := extjob.NewShellJobWithCallbacks(
		command,
		time.Duration(timeout)*time.Second,
		beforeExec,
		afterExec,
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

	quartzCronTrigger, err := quartz.NewCronTrigger(cronExpression)
	if err != nil {
		return err
	}

	err = scheduler.ScheduleJob(quartzJobDetail, quartzCronTrigger)
	if err != nil {
		return err
	}

	return nil
}
