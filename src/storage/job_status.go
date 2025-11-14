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
		return "E"
	case StatusDisable:
		return "D"
	case StatusActiveDuringEnable:
		return "AE"
	case StatusActiveDuringDisable:
		return "AD"
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
	case "E":
		*js = StatusEnable
	case "D":
		*js = StatusDisable
	case "AE":
		*js = StatusActiveDuringEnable
	case "AD":
		*js = StatusActiveDuringDisable
	default:
		return fmt.Errorf("invalid JobStatus: %s", s)
	}
	return nil
}
