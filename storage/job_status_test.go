package storage

import (
	"encoding/json"
	"testing"
)

func TestJobStatusJSONUnmarshal(t *testing.T) {
	tests := []struct {
		name        string
		jsonInput   string
		expected    JobStatus
		expectError bool
	}{
		{
			name:      "enable status",
			jsonInput: `"enable"`,
			expected:  StatusEnable,
		},
		{
			name:      "disable status",
			jsonInput: `"disable"`,
			expected:  StatusDisable,
		},
		{
			name:      "active status",
			jsonInput: `"active"`,
			expected:  StatusActive,
		},
		{
			name:        "invalid status",
			jsonInput:   `"invalid"`,
			expectError: true,
		},
		{
			name:        "empty status",
			jsonInput:   `""`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var status JobStatus
			err := json.Unmarshal([]byte(tt.jsonInput), &status)

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

			if status != tt.expected {
				t.Errorf(
					"For input %s: expected %v, got %v",
					tt.jsonInput,
					tt.expected,
					status,
				)
			}
		})
	}
}

func TestJobStatusJSONMarshal(t *testing.T) {
	tests := []struct {
		name     string
		status   JobStatus
		expected string
	}{
		{
			name:     "enable status",
			status:   StatusEnable,
			expected: `"enable"`,
		},
		{
			name:     "disable status",
			status:   StatusDisable,
			expected: `"disable"`,
		},
		{
			name:     "active status",
			status:   StatusActive,
			expected: `"active"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.status)
			if err != nil {
				t.Fatalf("Failed to marshal %v: %v", tt.status, err)
			}

			if string(data) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(data))
			}
		})
	}
}

func TestJobStatusJSONRoundTrip(t *testing.T) {
	tests := []JobStatus{StatusEnable, StatusDisable, StatusActive}

	for _, original := range tests {
		t.Run(original.String(), func(t *testing.T) {
			data, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var restored JobStatus
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
