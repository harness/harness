// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

type Stage struct {
	ID          int64             `json:"-"`
	ExecutionID int64             `json:"execution_id"`
	Number      int               `json:"number"`
	Name        string            `json:"name"`
	Kind        string            `json:"kind,omitempty"`
	Type        string            `json:"type,omitempty"`
	Status      string            `json:"status"`
	Error       string            `json:"error,omitempty"`
	ErrIgnore   bool              `json:"errignore"`
	ExitCode    int               `json:"exit_code"`
	Machine     string            `json:"machine,omitempty"`
	OS          string            `json:"os"`
	Arch        string            `json:"arch"`
	Variant     string            `json:"variant,omitempty"`
	Kernel      string            `json:"kernel,omitempty"`
	Limit       int               `json:"limit,omitempty"`
	LimitRepo   int               `json:"throttle,omitempty"`
	Started     int64             `json:"started,omitempty"`
	Stopped     int64             `json:"stopped,omitempty"`
	Created     int64             `json:"-"`
	Updated     int64             `json:"-"`
	Version     int64             `json:"-"`
	OnSuccess   bool              `json:"on_success,omitempty"`
	OnFailure   bool              `json:"on_failure,omitempty"`
	DependsOn   []string          `json:"depends_on,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Steps       []*Step           `json:"steps,omitempty"`
}
