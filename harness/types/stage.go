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

import "github.com/harness/gitness/types/enum"

type Stage struct {
	ID          int64             `json:"-"`
	ExecutionID int64             `json:"execution_id"`
	RepoID      int64             `json:"repo_id"`
	Number      int64             `json:"number"`
	Name        string            `json:"name"`
	Kind        string            `json:"kind,omitempty"`
	Type        string            `json:"type,omitempty"`
	Status      enum.CIStatus     `json:"status"`
	Error       string            `json:"error,omitempty"`
	ErrIgnore   bool              `json:"errignore,omitempty"`
	ExitCode    int               `json:"exit_code"`
	Machine     string            `json:"machine,omitempty"`
	OS          string            `json:"os,omitempty"`
	Arch        string            `json:"arch,omitempty"`
	Variant     string            `json:"variant,omitempty"`
	Kernel      string            `json:"kernel,omitempty"`
	Limit       int               `json:"limit,omitempty"`
	LimitRepo   int               `json:"throttle,omitempty"`
	Started     int64             `json:"started,omitempty"`
	Stopped     int64             `json:"stopped,omitempty"`
	Created     int64             `json:"-"`
	Updated     int64             `json:"-"`
	Version     int64             `json:"-"`
	OnSuccess   bool              `json:"on_success"`
	OnFailure   bool              `json:"on_failure"`
	DependsOn   []string          `json:"depends_on,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Steps       []*Step           `json:"steps,omitempty"`
}
