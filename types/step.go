// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

// Pretty print a step.
func (s Step) String() string {
	// Convert the Step struct to JSON
	jsonStr, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error converting to JSON: %v", err)
	}
	return string(jsonStr)
}
