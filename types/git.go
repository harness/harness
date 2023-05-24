// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import (
	"time"

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
	After     string `json:"after"`
	Path      string `json:"path"`
	Since     int64  `json:"since"`
	Until     int64  `json:"until"`
	Committer string `json:"committer"`
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

type Commit struct {
	SHA       string    `json:"sha"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Author    Signature `json:"author"`
	Committer Signature `json:"committer"`
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
}
