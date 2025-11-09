package storage

import (
	"context"
	"log/slog"
	"time"

	"midnight-runner/extjob"

	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/quartz"
)

//  WARN: BEFORE CALLING THIS, PLS THINK ABOUT TAKE DB MUTEX

func RegisterJobs(
	// quartz.Scheduler = &StdScheduler, 8 bytes
	scheduler quartz.Scheduler,
	db *Database,
	logger *slog.Logger,
) error {
	// j - *Job, 8 bytes (cheap copying)
	for jk, j := range db.Jobs {
		if j.Config.Status != StatusDisable {
			err := registerShellJob(scheduler, db, jk, logger)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func registerShellJob(
	scheduler quartz.Scheduler,
	db *Database,
	jobKey string,
	logger *slog.Logger,
) error {
	j := db.Jobs[jobKey]

	description := j.Description
	command := j.Config.Command
	maxRetries := j.Config.MaxRetries
	retryInterval := j.Config.RetryInterval
	cronExpression := j.Config.CronExpression
	timeout := j.Config.Timeout

	logFields := []any{
		"name", jobKey,
		"description", description,
		"command", command,
		"cron_expression", cronExpression,
	}

	beforeExec := createBeforeExecCallback(db, j, logger, logFields)
	afterExec := createAfterExecCallback(db, j, logger, logFields)

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

func createBeforeExecCallback(
	db *Database,
	j *Job,
	logger *slog.Logger,
	logFields []any,
) func(context.Context, *job.ShellJob) {
	return func(ctx context.Context, qj *job.ShellJob) {
		db.Mu.Lock()
		switch j.Config.Status {
		case StatusEnable:
			j.Config.Status = StatusActiveDuringEnable
		case StatusDisable:
			j.Config.Status = StatusActiveDuringDisable
		}
		db.Mu.Unlock()

		logger.Info("Start command execution", logFields...)
	}
}

func createAfterExecCallback(
	db *Database,
	j *Job,
	logger *slog.Logger,
	logFields []any,
) func(context.Context, *job.ShellJob) {
	return func(ctx context.Context, qj *job.ShellJob) {
		db.Mu.Lock()
		switch j.Config.Status {
		case StatusActiveDuringDisable:
			j.Config.Status = StatusDisable
		case StatusActiveDuringEnable:
			j.Config.Status = StatusEnable
		}
		db.Mu.Unlock()

		status := qj.JobStatus()

		switch status {
		case job.StatusOK:
			logger.Info("Command completed successfully", logFields...)
		case job.StatusFailure:
			select {
			case <-ctx.Done():
				logger.Warn("Command timeout exceeded", logFields...)
			default:
				logger.Warn("Command failed", logFields...)
			}
		}
	}
}
