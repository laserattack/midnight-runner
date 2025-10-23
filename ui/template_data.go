package ui

import (
	"strconv"
	"time"

	"servant/storage"
)

// the data structure passed to the template

type TemplateData struct {
	Title           string
	RenderTimestamp int64
	Database        *TemplateDatabase
}

type TemplateDatabase struct {
	Version  string
	Metadata TemplateMetadata
	Jobs     TemplateJobs
}

type TemplateMetadata struct {
	UpdatedAt string
}

type TemplateJobs map[string]*TemplateJob

type TemplateJob struct {
	Type        string
	Description string
	Config      TemplateJobConfig
	Metadata    TemplateMetadata
}

type TemplateJobConfig struct {
	Command        string
	CronExpression string
	Status         string
	Timeout        string
	MaxRetries     string
	RetryInterval  string
}

func convertDatabase(db *storage.Database) *TemplateDatabase {
	db.Mu.RLock()
	defer db.Mu.RUnlock()

	templateDB := &TemplateDatabase{
		Version: db.Version,
		Metadata: TemplateMetadata{
			UpdatedAt: formatTimestamp(db.Metadata.UpdatedAt),
		},
		Jobs: make(TemplateJobs),
	}

	for name, job := range db.Jobs {
		templateDB.Jobs[name] = convertJob(job)
	}

	return templateDB
}

func convertJob(job *storage.Job) *TemplateJob {
	var jobStatus string
	switch job.Config.Status {
	case storage.StatusActive:
		jobStatus = "🟡"
	case storage.StatusEnable:
		jobStatus = "🟢"
	case storage.StatusDisable:
		jobStatus = "🔴"
	}

	jobTimeout := formatInt(job.Config.Timeout) + " sec"
	jobRetryInterval := formatInt(job.Config.RetryInterval) + " sec"

	return &TemplateJob{
		Type:        job.Type.String(),
		Description: job.Description,
		Config: TemplateJobConfig{
			Command:        job.Config.Command,
			CronExpression: job.Config.CronExpression,
			Status:         jobStatus,
			Timeout:        jobTimeout,
			MaxRetries:     formatInt(job.Config.MaxRetries),
			RetryInterval:  jobRetryInterval,
		},
		Metadata: TemplateMetadata{
			UpdatedAt: formatTimestamp(job.Metadata.UpdatedAt),
		},
	}
}

func formatTimestamp(timestamp int64) string {
	if timestamp <= 0 {
		return "Never"
	}
	return time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
}

func formatInt(value int) string {
	return strconv.Itoa(value)
}
