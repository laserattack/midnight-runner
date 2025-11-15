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

// NOTE: Database, metadata

type Metadata struct {
	UpdatedAt int64 `json:"updated_at"`
}

type Database struct {
	Mu       sync.RWMutex
	Version  string   `json:"version"`
	Metadata Metadata `json:"metadata"`
	Jobs     Jobs     `json:"jobs"`
}

func New() *Database {
	return &Database{
		Version: "1.1",
		Metadata: Metadata{
			UpdatedAt: time.Now().Unix(),
		},
		Jobs: Jobs{},
	}
}

func (db *Database) ToggleJob(name string) {
	db.Mu.Lock()
	defer db.Mu.Unlock()

	var j *Job
	var exists bool

	if j, exists = db.Jobs[name]; !exists {
		return
	}

	switch j.Config.Status {
	case StatusActiveDuringEnable, StatusActiveDuringDisable, StatusEnable:
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
	db.Mu.RLock()
	defer db.Mu.RUnlock()

	if _, exists := db.Jobs[name]; !exists {
		return
	}

	j := db.Jobs[name]

	beforeExec := createBeforeExecCallback(db, name, logger)
	afterExec := createAfterExecCallback(db, name, logger)

	job := extjob.NewShellJobWithCallbacks(
		j.Config.Command,
		time.Duration(j.Config.Timeout)*time.Second,
		beforeExec,
		afterExec,
	)

	go func() {
		_ = job.Execute(ctx)
	}()
}

func (db *Database) DeleteJob(name string) {
	db.Mu.Lock()
	defer db.Mu.Unlock()

	if _, exists := db.Jobs[name]; !exists {
		return
	}

	delete(db.Jobs, name)
	db.Metadata.UpdatedAt = time.Now().Unix()
}

func (db *Database) SetJob(j *Job, k string) {
	db.Mu.Lock()
	defer db.Mu.Unlock()

	db.Jobs[k] = j
	db.Metadata.UpdatedAt = time.Now().Unix()
}

// NOTE: Serialize storage structure in byte array

func (db *Database) SerializeWithLock() ([]byte, error) {
	db.Mu.RLock()
	defer db.Mu.RUnlock()

	return db.Serialize()
}

func (db *Database) Serialize() ([]byte, error) {
	return json.MarshalIndent(db, "", "    ")
}

// NOTE: Deserialize byte array in storage structure

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

// NOTE: Load database from file

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

	return db, nil
}

// NOTE: Save database to file

// WARN: BEFORE CALLING THIS, PLS THINK ABOUT TAKE DB MUTEX

func (db *Database) SaveToFile(filepath string) error {
	databaseFileMutex.Lock()
	defer databaseFileMutex.Unlock()

	for jk := range db.Jobs {
		if db.Jobs[jk].Config.Status == StatusActiveDuringEnable ||
			db.Jobs[jk].Config.Status == StatusActiveDuringDisable {
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
