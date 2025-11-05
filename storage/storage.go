// Package storage: working with the database
package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"

	"midnight-runner/extjob"
)

// A Mutex for safe operation with a database stored on disk
var databaseFileMutex sync.RWMutex

//  NOTE: Database, metadata

type Metadata struct {
	UpdatedAt int64 `json:"updated_at"`
}

type Database struct {
	mu       sync.RWMutex
	filepath string
	Version  string   `json:"version"`
	Metadata Metadata `json:"metadata"`
	Jobs     Jobs     `json:"jobs"`
}

func (db *Database) ToggleJob(name string) {
	db.mu.Lock()
	defer db.mu.Unlock()

	var j *Job
	var exists bool

	if j, exists = db.Jobs[name]; !exists {
		return
	}

	switch j.Config.Status {
	case StatusActive, StatusEnable:
		j.Config.Status = StatusDisable
	case StatusDisable:
		j.Config.Status = StatusEnable
	}

	db.Metadata.UpdatedAt = time.Now().Unix()
}

func (db *Database) ExecJob(
	name string,
	ctx context.Context,
	logger *slog.Logger,
) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if _, exists := db.Jobs[name]; !exists {
		return
	}

	j := db.Jobs[name]

	if j.Config.Status == StatusDisable {
		return
	}

	logFields := []any{
		"name", name,
		"description", j.Description,
		"command", j.Config.Command,
		"cron_expression", j.Config.CronExpression,
	}

	beforeExec := createBeforeExecCallback(db, j, logger, logFields)
	afterExec := createAfterExecCallback(db, j, logger, logFields)

	job := extjob.NewShellJobWithCallbacks(
		j.Config.Command,
		time.Duration(j.Config.Timeout)*time.Second,
		beforeExec,
		afterExec,
	)

	go job.Execute(ctx)
}

func (db *Database) DeleteJob(name string) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.Jobs[name]; !exists {
		return
	}

	delete(db.Jobs, name)
	db.Metadata.UpdatedAt = time.Now().Unix()
}

func (db *Database) SetJob(j *Job, k string) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.Jobs[k] = j
	db.Metadata.UpdatedAt = time.Now().Unix()
}

func ActualizeDatabase(
	db,
	dbDonor *Database,
	logger *slog.Logger,
) (needRestartScheduler bool, err error) {
	db.mu.Lock()

	dbDonor.mu.RLock()
	defer dbDonor.mu.RUnlock()

	// No change
	if db.Metadata.UpdatedAt == dbDonor.Metadata.UpdatedAt {
		db.mu.Unlock()
		return false, nil
	}

	// Actualize in ram
	if db.Metadata.UpdatedAt < dbDonor.Metadata.UpdatedAt {
		db.Version = dbDonor.Version
		db.Metadata = dbDonor.Metadata
		db.Jobs = dbDonor.Jobs
		db.mu.Unlock()
		return true, nil
	}

	// Actualize in file
	db.mu.Unlock()
	if err := SaveToFile(db, db.filepath); err != nil {
		return false, err
	}

	return true, nil
}

//  NOTE: Serialize storage structure in byte array

func (db *Database) Serialize() ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return json.MarshalIndent(db, "", "    ")
}

//  NOTE: Deserialize byte array in storage structure

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

//  NOTE: Load database from file

func LoadFromFile(filepath string) (*Database, error) {
	databaseFileMutex.RLock()
	defer databaseFileMutex.RUnlock()

	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	db, err := Deserialize(data)
	if err != nil {
		return nil, err
	}

	db.filepath = filepath
	return db, nil
}

//  NOTE: Save database to file

func SaveToFile(db *Database, filepath string) error {
	databaseFileMutex.Lock()
	defer databaseFileMutex.Unlock()

	for jk := range db.Jobs {
		if db.Jobs[jk].Config.Status == StatusActive {
			db.Jobs[jk].Config.Status = StatusEnable
		}
	}

	data, err := db.Serialize()
	if err != nil {
		return err
	}

	// Write to temporary file first
	tmpFilepath := filepath + ".tmp"
	err = os.WriteFile(tmpFilepath, data, 0o644)
	if err != nil {
		return err
	}

	// Rename temporary file to actual file (atomic operation)
	return os.Rename(tmpFilepath, filepath)
}
