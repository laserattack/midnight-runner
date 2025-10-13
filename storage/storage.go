// Package storage: working with the database
package storage

import (
	"bytes"
	"encoding/json"
)

type JobStatus string

const (
	StatusEnable  JobStatus = "enable"
	StatusDisable JobStatus = "disable"
	StatusActive  JobStatus = "active"
)

type JobType string

const (
	TypeShell JobType = "shell"
)

type Metadata struct {
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

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

type Jobs map[string]Job

type Database struct {
	Version  string   `json:"version"`
	Metadata Metadata `json:"metadata"`
	Jobs     Jobs     `json:"jobs"`
}

func (db *Database) Serialize() ([]byte, error) {
	return json.MarshalIndent(db, "", "    ")
}

func Deserialize(data []byte) (*Database, error) {
	var db Database
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&db)
	if err != nil {
		return nil, err
	}
	return &db, nil
}
