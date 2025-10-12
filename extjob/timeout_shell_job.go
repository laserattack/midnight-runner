// Package extjob: go-quartz extensions
package extjob

import (
	"context"
	"time"

	"github.com/reugn/go-quartz/job"
)

type TimeoutShellJob struct {
	command  string
	timeout  time.Duration
	shellJob *job.ShellJob
	callback func(ctx context.Context, j *job.ShellJob)
}

func (j *TimeoutShellJob) Execute(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, j.timeout)
	defer cancel()

	err := j.shellJob.Execute(timeoutCtx)

	if j.callback != nil {
		j.callback(ctx, j.shellJob)
	}

	return err
}

func (j *TimeoutShellJob) Description() string {
	return j.shellJob.Description()
}

func (j *TimeoutShellJob) JobStatus() job.Status {
	return j.shellJob.JobStatus()
}

func (j *TimeoutShellJob) ExitCode() int {
	return j.shellJob.ExitCode()
}

func (j *TimeoutShellJob) Stdout() string {
	return j.shellJob.Stdout()
}

func (j *TimeoutShellJob) Stderr() string {
	return j.shellJob.Stderr()
}

func NewShellJobWithCallbackAndTimeout(
	command string,
	timeout time.Duration,
	callback func(ctx context.Context, j *job.ShellJob),
) *TimeoutShellJob {
	shellJob := job.NewShellJobWithCallback(command, callback)

	return &TimeoutShellJob{
		command:  command,
		timeout:  timeout,
		shellJob: shellJob,
		callback: callback,
	}
}
