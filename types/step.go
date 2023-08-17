// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

type Step struct {
	ID        int64    `json:"-"`
	StageID   int64    `json:"-"`
	Number    int64    `json:"number"`
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	Error     string   `json:"error,omitempty"`
	ErrIgnore bool     `json:"errignore,omitempty"`
	ExitCode  int      `json:"exit_code"`
	Started   int64    `json:"started,omitempty"`
	Stopped   int64    `json:"stopped,omitempty"`
	Version   int64    `json:"-" db:"step_version"`
	DependsOn []string `json:"depends_on"`
	Image     string   `json:"image,omitempty"`
	Detached  bool     `json:"detached"`
	Schema    string   `json:"schema,omitempty"`
}
