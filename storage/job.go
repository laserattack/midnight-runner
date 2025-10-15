package storage

//  NOTE: Job structure

//  TODO: mb Expression -> CronExpression

type JobConfig struct {
	Command       string    `json:"command"`
	Expression    string    `json:"expression"`
	Status        JobStatus `json:"status"`
	Timeout       int       `json:"timeout"`
	MaxRetries    int       `json:"max_retries"`
	RetryInterval int       `json:"retry_interval"`
}

type Job struct {
	Type        JobType   `json:"type"`
	Description string    `json:"description"`
	Config      JobConfig `json:"config"`
	Metadata    Metadata  `json:"metadata"`
}

type Jobs map[string]*Job
