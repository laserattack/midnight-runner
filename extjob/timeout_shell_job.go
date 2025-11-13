// Package extjob: go-quartz extensions
package extjob

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/reugn/go-quartz/quartz"
)

type Status int8

const (
	StatusNA Status = iota
	StatusOK
	StatusFailure
)

type ShellJob struct {
	mtx        sync.Mutex
	cmd        string
	exitCode   int
	stdout     string
	stderr     string
	jobStatus  Status
	timeout    time.Duration
	beforeExec func(context.Context, *ShellJob)
	afterExec  func(context.Context, *ShellJob)
}

var _ quartz.Job = (*ShellJob)(nil)

func NewShellJob(cmd string) *ShellJob {
	return &ShellJob{
		cmd:       cmd,
		jobStatus: StatusNA,
	}
}

func NewShellJobWithCallbacks(
	cmd string,
	timeout time.Duration,
	beforeExec func(ctx context.Context, j *ShellJob),
	afterExec func(ctx context.Context, j *ShellJob),
) *ShellJob {
	return &ShellJob{
		cmd:        cmd,
		jobStatus:  StatusNA,
		timeout:    timeout,
		beforeExec: beforeExec,
		afterExec:  afterExec,
	}
}

func (sh *ShellJob) Description() string {
	return fmt.Sprintf("ShellJob%s%s", quartz.Sep, sh.cmd)
}

var (
	shellOnce = sync.Once{}
	shellPath string
	shellArgs []string
)

func getShell() (string, []string) {
	shellOnce.Do(func() {
		switch runtime.GOOS {
		case "windows":
			shellPath = "cmd"
			shellArgs = []string{"/c"}
		default:
			// Try bash first, fallback to sh
			shell := "sh"
			if _, err := exec.LookPath("bash"); err == nil {
				shell = "bash"
			}
			shellPath = shell
			shellArgs = []string{"-c"}
		}
	})
	return shellPath, shellArgs
}

func (j *ShellJob) execute(ctx context.Context) error {
	shell, args := getShell()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, shell, append(args, j.cmd)...)
	cmd.Stdout = io.Writer(&stdout)
	cmd.Stderr = io.Writer(&stderr)

	err := cmd.Run()

	j.mtx.Lock()
	j.stdout, j.stderr = stdout.String(), stderr.String()
	j.exitCode = cmd.ProcessState.ExitCode()

	if err != nil {
		j.jobStatus = StatusFailure
	} else {
		j.jobStatus = StatusOK
	}
	j.mtx.Unlock()

	return err
}

func (j *ShellJob) Execute(ctx context.Context) error {
	if j.beforeExec != nil {
		j.beforeExec(ctx, j)
	}

	var err error
	if j.timeout <= 0 {
		err = j.execute(ctx)
	} else {
		timeoutCtx, cancel := context.WithTimeout(ctx, j.timeout)
		defer cancel()
		err = j.execute(timeoutCtx)
	}

	if j.afterExec != nil {
		j.afterExec(ctx, j)
	}

	return err
}

func (sh *ShellJob) ExitCode() int {
	sh.mtx.Lock()
	defer sh.mtx.Unlock()
	return sh.exitCode
}

func (sh *ShellJob) Stdout() string {
	sh.mtx.Lock()
	defer sh.mtx.Unlock()
	return sh.stdout
}

func (sh *ShellJob) Stderr() string {
	sh.mtx.Lock()
	defer sh.mtx.Unlock()
	return sh.stderr
}

func (sh *ShellJob) JobStatus() Status {
	sh.mtx.Lock()
	defer sh.mtx.Unlock()
	return sh.jobStatus
}
