package storage

import (
	"encoding/json"
	"fmt"
)

//  NOTE: Job type type

type JobType int

const (
	TypeShell JobType = iota
)

func (jt JobType) String() string {
	switch jt {
	case TypeShell:
		return "shell"
	default:
		return "unknown"
	}
}

func (jt JobType) MarshalJSON() ([]byte, error) {
	return json.Marshal(jt.String())
}

func (jt *JobType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch s {
	case "shell":
		*jt = TypeShell
	default:
		return fmt.Errorf("invalid JobType: %s", s)
	}
	return nil
}
