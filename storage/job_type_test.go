package storage

import (
	"encoding/json"
	"testing"
)

func TestJobTypeJSONUnmarshal(t *testing.T) {
	tests := []struct {
		name        string
		jsonInput   string
		expected    JobType
		expectError bool
	}{
		{
			name:      "shell type",
			jsonInput: `"shell"`,
			expected:  TypeShell,
		},
		{
			name:        "invalid type",
			jsonInput:   `"invalid"`,
			expectError: true,
		},
		{
			name:        "empty type",
			jsonInput:   `""`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jobType JobType
			err := json.Unmarshal([]byte(tt.jsonInput), &jobType)

			if tt.expectError {
				if err == nil {
					t.Errorf(
						"Expected error for input %s, but got none",
						tt.jsonInput,
					)
				}
				return
			}

			if err != nil {
				t.Errorf(
					"Unexpected error for input %s: %v",
					tt.jsonInput,
					err,
				)
				return
			}

			if jobType != tt.expected {
				t.Errorf(
					"For input %s: expected %v, got %v",
					tt.jsonInput,
					tt.expected,
					jobType,
				)
			}
		})
	}
}

func TestJobTypeJSONMarshal(t *testing.T) {
	tests := []struct {
		name     string
		jobType  JobType
		expected string
	}{
		{
			name:     "shell type",
			jobType:  TypeShell,
			expected: `"shell"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.jobType)
			if err != nil {
				t.Fatalf("Failed to marshal %v: %v", tt.jobType, err)
			}

			if string(data) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(data))
			}
		})
	}
}

func TestJobTypeJSONRoundTrip(t *testing.T) {
	tests := []JobType{TypeShell}

	for _, original := range tests {
		t.Run(original.String(), func(t *testing.T) {
			data, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var restored JobType
			err = json.Unmarshal(data, &restored)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if restored != original {
				t.Errorf(
					"Round trip failed: original %v, restored %v",
					original,
					restored,
				)
			}
		})
	}
}
