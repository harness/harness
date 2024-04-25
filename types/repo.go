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

	"github.com/harness/gitness/types/enum"
)

// Repository represents a code repository.
type Repository struct {
	// TODO: int64 ID doesn't match DB
	ID          int64  `json:"id" yaml:"id"`
	Version     int64  `json:"-" yaml:"-"`
	ParentID    int64  `json:"parent_id" yaml:"parent_id"`
	Identifier  string `json:"identifier" yaml:"identifier"`
	Path        string `json:"path" yaml:"path"`
	Description string `json:"description" yaml:"description"`
	IsPublic    bool   `json:"is_public" yaml:"is_public"`
	CreatedBy   int64  `json:"created_by" yaml:"created_by"`
	Created     int64  `json:"created" yaml:"created"`
	Updated     int64  `json:"updated" yaml:"updated"`
	Deleted     *int64 `json:"deleted,omitempty" yaml:"deleted"`

	// Size of the repository in KiB.
	Size int64 `json:"size" yaml:"size"`
	// SizeUpdated is the time when the Size was last updated.
	SizeUpdated int64 `json:"size_updated" yaml:"size_updated"`

	GitUID        string `json:"-" yaml:"-"`
	DefaultBranch string `json:"default_branch" yaml:"default_branch"`
	ForkID        int64  `json:"fork_id" yaml:"fork_id"`
	PullReqSeq    int64  `json:"-" yaml:"-"`

	NumForks       int `json:"num_forks" yaml:"num_forks"`
	NumPulls       int `json:"num_pulls" yaml:"num_pulls"`
	NumClosedPulls int `json:"num_closed_pulls" yaml:"num_closed_pulls"`
	NumOpenPulls   int `json:"num_open_pulls" yaml:"num_open_pulls"`
	NumMergedPulls int `json:"num_merged_pulls" yaml:"num_merged_pulls"`

	Importing bool `json:"importing" yaml:"-"`
	IsEmpty   bool `json:"is_empty,omitempty" yaml:"is_empty"`

	// git urls
	GitURL string `json:"git_url" yaml:"git_url"`
}

// Clone makes deep copy of repository object.
func (r Repository) Clone() Repository {
	var deleted *int64
	if r.Deleted != nil {
		id := *r.Deleted
		deleted = &id
	}
	r.Deleted = deleted
	return r
}

// TODO [CODE-1363]: remove after identifier migration.
func (r Repository) MarshalJSON() ([]byte, error) {
	// alias allows us to embed the original object while avoiding an infinite loop of marshaling.
	type alias Repository
	return json.Marshal(&struct {
		alias
		UID string `json:"uid"`
	}{
		alias: (alias)(r),
		UID:   r.Identifier,
	})
}

type RepositorySizeInfo struct {
	ID     int64  `json:"id"`
	GitUID string `json:"git_uid"`
	// Size of the repository in KiB.
	Size int64 `json:"size"`
	// SizeUpdated is the time when the Size was last updated.
	SizeUpdated int64 `json:"size_updated"`
}

func (r Repository) GetGitUID() string {
	return r.GitUID
}

// RepoFilter stores repo query parameters.
type RepoFilter struct {
	Page              int           `json:"page"`
	Size              int           `json:"size"`
	Query             string        `json:"query"`
	Sort              enum.RepoAttr `json:"sort"`
	Order             enum.Order    `json:"order"`
	DeletedAt         *int64        `json:"deleted_at,omitempty"`
	DeletedBeforeOrAt *int64        `json:"deleted_before_or_at,omitempty"`
	Recursive         bool
}

// RepositoryGitInfo holds git info for a repository.
type RepositoryGitInfo struct {
	ID       int64
	ParentID int64
	GitUID   string
}
