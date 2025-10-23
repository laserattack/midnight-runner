// Package storage: working with the database
package storage

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"sync"
)

// A Mu.ex for safe operation with a database stored on disk
var databaseFileMutex sync.RWMutex

//  NOTE: Database, metadata

type Metadata struct {
	UpdatedAt int64 `json:"updated_at"`
}

type Database struct {
	Mu       sync.RWMutex
	Version  string   `json:"version"`
	Metadata Metadata `json:"metadata"`
	Jobs     Jobs     `json:"jobs"`
}

//  TODO: Проверить остается ли в памяти мьютекс базы-донора

func UpdateDatabase(db, dbDonor *Database, slogLogger *slog.Logger) {
	db.Mu.Lock()
	defer db.Mu.Unlock()

	// Checking whether old data is being deleted
	// for jobName, job := range db.Jobs {
	// 	runtime.SetFinalizer(job, func(j *Job) {
	// 		slogLogger.Info("OLD JOB COLLECTED BY GC!",
	// 			"job_name", jobName,
	// 			"job_addr", fmt.Sprintf("%p", j))
	// 	})
	// }

	db.Version = dbDonor.Version
	db.Metadata = dbDonor.Metadata
	db.Jobs = dbDonor.Jobs

	// Run gc to delete old data immediately
	// go func() {
	// 	time.Sleep(2 * time.Second)
	// 	runtime.GC()
	// }()
}

//  NOTE: Serialize storage structure in byte array

func (db *Database) Serialize() ([]byte, error) {
	db.Mu.RLock()
	defer db.Mu.RUnlock()

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

	return Deserialize(data)
}

//  NOTE: Save database to file

func SaveToFile(db *Database, filepath string) error {
	databaseFileMutex.Lock()
	defer databaseFileMutex.Unlock()

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
