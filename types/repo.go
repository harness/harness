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
	ID          int64  `json:"id"`
	Version     int64  `json:"-"`
	ParentID    int64  `json:"parent_id"`
	Identifier  string `json:"identifier"`
	Path        string `json:"path"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
	CreatedBy   int64  `json:"created_by"`
	Created     int64  `json:"created"`
	Updated     int64  `json:"updated"`
	Deleted     *int64 `json:"deleted,omitempty"`

	Size        int64 `json:"size"`
	SizeUpdated int64 `json:"size_updated"`

	GitUID        string `json:"-"`
	DefaultBranch string `json:"default_branch"`
	ForkID        int64  `json:"fork_id"`
	PullReqSeq    int64  `json:"-"`

	NumForks       int `json:"num_forks"`
	NumPulls       int `json:"num_pulls"`
	NumClosedPulls int `json:"num_closed_pulls"`
	NumOpenPulls   int `json:"num_open_pulls"`
	NumMergedPulls int `json:"num_merged_pulls"`

	Importing bool `json:"importing"`

	// git urls
	GitURL string `json:"git_url"`
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
	ID          int64  `json:"id"`
	GitUID      string `json:"git_uid"`
	Size        int64  `json:"size"`
	SizeUpdated int64  `json:"size_updated"`
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
	DeletedBeforeOrAt *int64        `json:"deleted_before_or_at,omitempty"`
	Recursive         bool
}

// RepositoryGitInfo holds git info for a repository.
type RepositoryGitInfo struct {
	ID       int64
	ParentID int64
	GitUID   string
}
