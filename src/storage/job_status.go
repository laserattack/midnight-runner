package storage

import (
	"encoding/json"
	"fmt"
)

//  NOTE: Job status type

type JobStatus int

const (
	StatusEnable JobStatus = iota
	StatusDisable
	StatusActiveDuringEnable
	StatusActiveDuringDisable
)

func (js JobStatus) String() string {
	switch js {
	case StatusEnable:
		return "💚"
	case StatusDisable:
		return "🩶"
	case StatusActiveDuringEnable:
		return "🚀💚"
	case StatusActiveDuringDisable:
		return "🚀🩶"
	default:
		return "unknown"
	}
}

func (js JobStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(js.String())
}

func (js *JobStatus) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch s {
	case "💚":
		*js = StatusEnable
	case "🩶":
		*js = StatusDisable
	case "🚀💚":
		*js = StatusActiveDuringEnable
	case "🚀🩶":
		*js = StatusActiveDuringDisable
	default:
		return fmt.Errorf("invalid JobStatus: %s", s)
	}
	return nil
}
