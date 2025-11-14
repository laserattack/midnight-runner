package storage

import (
	"reflect"
	"testing"
	"time"
)

func TestStorageJSONRountTrip(t *testing.T) {
	tests := []struct {
		name        string
		database    *Database
		expectError bool
	}{
		{
			name: "database with jobs",
			database: &Database{
				Version: "1.0.0",
				Metadata: Metadata{
					UpdatedAt: time.Now().Unix(),
				},
				Jobs: Jobs{
					"job1": {
						Type:        TypeShell,
						Description: "Test job 1",
						Config: JobConfig{
							Command:        "echo hello",
							CronExpression: "* * * * *",
							Status:         StatusEnable,
							Timeout:        30,
							MaxRetries:     3,
							RetryInterval:  10,
						},
						Metadata: Metadata{
							UpdatedAt: time.Now().Unix(),
						},
					},
					"job2": {
						Type:        TypeShell,
						Description: "Test job 2",
						Config: JobConfig{
							Command:        "echo world",
							CronExpression: "0 * * * *",
							Status:         StatusDisable,
							Timeout:        60,
							MaxRetries:     5,
							RetryInterval:  5,
						},
						Metadata: Metadata{
							UpdatedAt: time.Now().Unix(),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.database.SerializeWithLock()
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			restored, err := Deserialize(data)
			if err != nil {
				t.Fatalf("Deserialize failed: %v", err)
			}

			if !reflect.DeepEqual(tt.database, restored) {
				t.Errorf("Database mismatch after round-trip serialization")
			}
		})
	}
}

func TestStorageJSONInvalid(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
	}{
		{
			name:      "invalid JSON",
			jsonInput: `{invalid json}`,
		},
		{
			name:      "unknown field",
			jsonInput: `{"version": "1.0.0", "unknown_field": "value"}`,
		},
		{
			name:      "invalid job status",
			jsonInput: `{"version": "1.0.0", "metadata": {"created_at": 123, "updated_at": 456}, "jobs": {"test": {"type": "shell", "description": "test", "config": {"command": "echo", "cron_expression": "* * * * *", "status": "invalid_status", "timeout": 30, "max_retries": 3, "retry_interval": 10}, "metadata": {"created_at": 123, "updated_at": 456}}}}`,
		},
		{
			name:      "invalid job type",
			jsonInput: `{"version": "1.0.0", "metadata": {"created_at": 123, "updated_at": 456}, "jobs": {"test": {"type": "invalid_type", "description": "test", "config": {"command": "echo", "cron_expression": "* * * * *", "status": "💚", "timeout": 30, "max_retries": 3, "retry_interval": 10}, "metadata": {"created_at": 123, "updated_at": 456}}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Deserialize([]byte(tt.jsonInput))
			if err == nil {
				t.Errorf("Expected error for input %s, but got none", tt.jsonInput)
			}
		})
	}
}
