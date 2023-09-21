// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import (
	"github.com/harness/gitness/types/enum"
)

// Repository represents a code repository.
type Repository struct {
	// TODO: int64 ID doesn't match DB
	ID          int64  `json:"id"`
	Version     int64  `json:"-"`
	ParentID    int64  `json:"parent_id"`
	UID         string `json:"uid"`
	Path        string `json:"path"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
	CreatedBy   int64  `json:"created_by"`
	Created     int64  `json:"created"`
	Updated     int64  `json:"updated"`

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

func (r Repository) GetGitUID() string {
	return r.GitUID
}

// RepoFilter stores repo query parameters.
type RepoFilter struct {
	Page  int           `json:"page"`
	Size  int           `json:"size"`
	Query string        `json:"query"`
	Sort  enum.RepoAttr `json:"sort"`
	Order enum.Order    `json:"order"`
}

// RepositoryGitInfo holds git info for a repository.
type RepositoryGitInfo struct {
	ID       int64
	ParentID int64
	GitUID   string
}
