// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

type Stage struct {
	ID          int64             `json:"id" db:"stage_id"`
	ExecutionID int64             `json:"execution_id" db:"stage_execution_id"`
	Number      int               `json:"number" db:"stage_number"`
	Name        string            `json:"name" db:"stage_name"`
	Kind        string            `json:"kind,omitempty" db:"stage_kind"`
	Type        string            `json:"type,omitempty" db:"stage_type"`
	Status      string            `json:"status" db:"stage_status"`
	Error       string            `json:"error,omitempty" db:"stage_error"`
	ErrIgnore   bool              `json:"errignore" db:"stage_errignore"`
	ExitCode    int               `json:"exit_code" db:"stage_exit_code"`
	Machine     string            `json:"machine,omitempty" db:"stage_machine"`
	OS          string            `json:"os" db:"stage_os"`
	Arch        string            `json:"arch" db:"stage_arch"`
	Variant     string            `json:"variant,omitempty" db:"stage_variant"`
	Kernel      string            `json:"kernel,omitempty" db:"stage_kernel"`
	Limit       int               `json:"limit,omitempty" db:"stage_limit"`
	LimitRepo   int               `json:"throttle,omitempty" db:"stage_limit_repo"`
	Started     int64             `json:"started" db:"stage_started"`
	Stopped     int64             `json:"stopped" db:"stage_stopped"`
	Created     int64             `json:"created" db:"stage_created"`
	Updated     int64             `json:"updated" db:"stage_updated"`
	Version     int64             `json:"version" db:"stage_version"`
	OnSuccess   bool              `json:"on_success" db:"stage_on_success"`
	OnFailure   bool              `json:"on_failure" db:"stage_on_failure"`
	DependsOn   []string          `json:"depends_on,omitempty" db:"stage_depends_on"`
	Labels      map[string]string `json:"labels,omitempty" db:"stage_labels"`
	Steps       []*Step           `json:"steps,omitempty" db:"-"`
}
