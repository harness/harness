// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

type Step struct {
	ID        int64    `json:"id" db:"step_id"`
	StageID   int64    `json:"stage_id" db:"step_stage_id"`
	Number    int      `json:"number" db:"step_number"`
	Name      string   `json:"name" db:"step_name"`
	Status    string   `json:"status" db:"step_status"`
	Error     string   `json:"error,omitempty" db:"step_error"`
	ErrIgnore bool     `json:"errignore,omitempty" db:"step_errignore"`
	ExitCode  int      `json:"exit_code" db:"step_exit_code"`
	Started   int64    `json:"started,omitempty" db:"step_started"`
	Stopped   int64    `json:"stopped,omitempty" db:"step_stopped"`
	Version   int64    `json:"version" db:"step_version"`
	DependsOn []string `json:"depends_on,omitempty" db:"step_depends_on"`
	Image     string   `json:"image,omitempty" db:"step_image"`
	Detached  bool     `json:"detached,omitempty" db:"step_detached"`
	Schema    string   `json:"schema,omitempty" db:"step_schema"`
}
