package storage

import (
	"context"
	"log/slog"
	"time"

	"midnight-runner/extjob"

	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/quartz"
)

func RegisterJobs(
	// quartz.Scheduler = &StdScheduler, 8 bytes
	scheduler quartz.Scheduler,
	db *Database,
	logger *slog.Logger,
) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// j - *Job, 8 bytes (cheap copying)
	for jk, j := range db.Jobs {
		if j.Type == TypeShell {
			//  TODO: Не уверен что хороший вариант так обрывать регистрацию
			err := registerShellJob(scheduler, db, jk, j, logger)
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
	logger *slog.Logger,
) error {
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

	afterExec := func(ctx context.Context, qj *job.ShellJob) {
		db.mu.Lock()
		j.Config.Status = StatusEnable
		db.mu.Unlock()

		status := qj.JobStatus()

		switch status {
		case job.StatusOK:
			logger.Info("Command completed successfully", logFields...)
		case job.StatusFailure:
			select {
			case <-ctx.Done():
				logger.Error("Command timeout exceeded", logFields...)
			default:
				logger.Error("Command failed", logFields...)
			}
		}
	}

	beforeExec := func(ctx context.Context, qj *job.ShellJob) {
		db.mu.Lock()
		j.Config.Status = StatusActive
		db.mu.Unlock()

		logger.Info("Start command execution", logFields...)
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
