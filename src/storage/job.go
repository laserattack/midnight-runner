package storage

import (
	"time"

	"github.com/reugn/go-quartz/quartz"
)

// NOTE: Job structure

type JobConfig struct {
	Command        string    `json:"command"`
	CronExpression string    `json:"cron_expression"`
	Status         JobStatus `json:"status"`
	Timeout        uint      `json:"timeout"`
	MaxRetries     uint      `json:"max_retries"`
	RetryInterval  uint      `json:"retry_interval"`
}

type Job struct {
	Type        JobType   `json:"type"`
	Description string    `json:"description"`
	Config      JobConfig `json:"config"`
	Metadata    Metadata  `json:"metadata"`
}

type Jobs map[string]*Job

func ShellJob(
	description, command, cronExpression string,
	timeout, maxRetries, retryInterval uint,
) (*Job, error) {
	if err := quartz.ValidateCronExpression(cronExpression); err != nil {
		return nil, err
	}

	return &Job{
		Type:        TypeShell,
		Description: description,
		Config: JobConfig{
			Command:        command,
			CronExpression: cronExpression,
			Status:         StatusEnable,
			Timeout:        timeout,
			MaxRetries:     maxRetries,
			RetryInterval:  retryInterval,
		},
		Metadata: Metadata{
			UpdatedAt: time.Now().Unix(),
		},
	}, nil
}
