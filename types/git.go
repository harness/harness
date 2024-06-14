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
	"time"

	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/types/enum"
)

const NilSHA = "0000000000000000000000000000000000000000"

// PaginationFilter stores pagination query parameters.
type PaginationFilter struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// CommitFilter stores commit query parameters.
type CommitFilter struct {
	PaginationFilter
	After        string `json:"after"`
	Path         string `json:"path"`
	Since        int64  `json:"since"`
	Until        int64  `json:"until"`
	Committer    string `json:"committer"`
	IncludeStats bool   `json:"include_stats"`
}

// BranchFilter stores branch query parameters.
type BranchFilter struct {
	Query string                `json:"query"`
	Sort  enum.BranchSortOption `json:"sort"`
	Order enum.Order            `json:"order"`
	Page  int                   `json:"page"`
	Size  int                   `json:"size"`
}

// TagFilter stores commit tag query parameters.
type TagFilter struct {
	Query string             `json:"query"`
	Sort  enum.TagSortOption `json:"sort"`
	Order enum.Order         `json:"order"`
	Page  int                `json:"page"`
	Size  int                `json:"size"`
}

type ChangeStats struct {
	Insertions int64 `json:"insertions"`
	Deletions  int64 `json:"deletions"`
	Changes    int64 `json:"changes"`
}

type CommitFileStats struct {
	Path    string                 `json:"path"`
	OldPath string                 `json:"old_path,omitempty"`
	Status  gitenum.FileDiffStatus `json:"status"`
	ChangeStats
}

type CommitStats struct {
	Total ChangeStats       `json:"total,omitempty"`
	Files []CommitFileStats `json:"files,omitempty"`
}

type Commit struct {
	SHA        string       `json:"sha"`
	ParentSHAs []string     `json:"parent_shas,omitempty"`
	Title      string       `json:"title"`
	Message    string       `json:"message"`
	Author     Signature    `json:"author"`
	Committer  Signature    `json:"committer"`
	Stats      *CommitStats `json:"stats,omitempty"`
}

type Signature struct {
	Identity Identity  `json:"identity"`
	When     time.Time `json:"when"`
}

type Identity struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type RenameDetails struct {
	OldPath         string `json:"old_path"`
	NewPath         string `json:"new_path"`
	CommitShaBefore string `json:"commit_sha_before"`
	CommitShaAfter  string `json:"commit_sha_after"`
}

type ListCommitResponse struct {
	Commits       []Commit        `json:"commits"`
	RenameDetails []RenameDetails `json:"rename_details"`
	TotalCommits  int             `json:"total_commits,omitempty"`
}
