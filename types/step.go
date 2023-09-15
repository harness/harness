// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import (
	"encoding/json"
	"fmt"

	"github.com/harness/gitness/types/enum"
)

type Step struct {
	ID        int64         `json:"-"`
	StageID   int64         `json:"-"`
	Number    int64         `json:"number"`
	Name      string        `json:"name"`
	Status    enum.CIStatus `json:"status"`
	Error     string        `json:"error,omitempty"`
	ErrIgnore bool          `json:"errignore,omitempty"`
	ExitCode  int           `json:"exit_code"`
	Started   int64         `json:"started,omitempty"`
	Stopped   int64         `json:"stopped,omitempty"`
	Version   int64         `json:"-" db:"step_version"`
	DependsOn []string      `json:"depends_on,omitempty"`
	Image     string        `json:"image,omitempty"`
	Detached  bool          `json:"detached"`
	Schema    string        `json:"schema,omitempty"`
}

// Pretty print a step
func (s Step) String() string {
	// Convert the Step struct to JSON
	jsonStr, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error converting to JSON: %v", err)
	}
	return string(jsonStr)
}
