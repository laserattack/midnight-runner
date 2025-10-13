package storage

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

const jsonData = `{
	"version": "1.1",
	"metadata": {
		"created_at": 123,
		"update_at": 123
	},
	"jobs": {
		"shellJob": {
			"type": "shell",
			"description": "List directory contents every 5 seconds",
			"config": {
				"command": "ls -la",
				"expression": "*/5 * * * * *",
				"status": "enable",
				"timeout": 10,
				"max_retries": 0,
				"retry_interval": 1
			},
			"metadata": {
				"created_at": 123,
				"updated_at": 123
			}
		}
	}
}`

func TestSerialize(t *testing.T) {
	db := &Database{
		Version: "1.1",
		Metadata: Metadata{
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
		},
		Jobs: Jobs{
			"shellJob": {
				Type:        TypeShell,
				Description: "List directory contents every 5 seconds",
				Config: JobConfig{
					Command:       "ls -la",
					Expression:    "*/5 * * * * *",
					Status:        StatusEnable,
					Timeout:       10,
					MaxRetries:    0,
					RetryInterval: 1,
				},
				Metadata: Metadata{
					CreatedAt: time.Now().Unix(),
					UpdatedAt: time.Now().Unix(),
				},
			},
		},
	}

	jsonData, err := db.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	var parsedData map[string]any
	if err := json.Unmarshal(jsonData, &parsedData); err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	if version, exists := parsedData["version"]; !exists || version != "1.1" {
		t.Errorf("Expected version '1.1', got '%v'", version)
	}

	if jobs, exists := parsedData["jobs"]; !exists {
		t.Error("Jobs field missing in serialized JSON")
	} else if jobsMap, ok := jobs.(map[string]any); !ok {
		t.Error("Jobs field is not a map")
	} else if _, jobExists := jobsMap["shellJob"]; !jobExists {
		t.Error("shellJob missing in serialized jobs")
	}
}

func TestDeserialize(t *testing.T) {
	db, err := Deserialize([]byte(jsonData))
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if db.Version != "1.1" {
		t.Errorf("Expected version '1.1', got '%s'", db.Version)
	}

	if len(db.Jobs) != 1 {
		t.Errorf("Expected 1 job, got %d", len(db.Jobs))
	}

	job, exists := db.Jobs["shellJob"]
	if !exists {
		t.Fatal("shellJob not found in deserialized data")
	}

	if job.Type != TypeShell {
		t.Errorf("Expected job type 'shell', got '%s'", job.Type)
	}

	if job.Description != "List directory contents every 5 seconds" {
		t.Errorf("Unexpected job description: %s", job.Description)
	}

	if job.Config.Command != "ls -la" {
		t.Errorf("Expected command 'ls -la', got '%s'", job.Config.Command)
	}

	if job.Config.Status != StatusEnable {
		t.Errorf("Expected status 'enable', got '%s'", job.Config.Status)
	}
}

func TestSerializeDeserializeRoundTrip(t *testing.T) {
	original := &Database{
		Version: "1.1",
		Metadata: Metadata{
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
		},
		Jobs: Jobs{
			"testJob": {
				Type:        TypeShell,
				Description: "Test job",
				Config: JobConfig{
					Command:       "echo test",
					Expression:    "*/10 * * * * *",
					Status:        StatusDisable,
					Timeout:       5,
					MaxRetries:    3,
					RetryInterval: 2,
				},
				Metadata: Metadata{
					CreatedAt: time.Now().Unix(),
					UpdatedAt: time.Now().Unix(),
				},
			},
		},
	}

	jsonData, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	restored, err := Deserialize(jsonData)
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if !reflect.DeepEqual(original, restored) {
		t.Errorf("Round-trip failed: original != restored")
	}
}

func TestDeserializeInvalidJSON(t *testing.T) {
	invalidJSON := `{invalid json}`

	_, err := Deserialize([]byte(invalidJSON))
	if err == nil {
		t.Error("Expected error for invalid JSON, but got none")
	}
}
