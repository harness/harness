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
	ID          int64  `db:"repo_id"              json:"id"`
	Version     int64  `db:"repo_version"         json:"version"`
	ParentID    int64  `db:"repo_parent_id"       json:"parent_id"`
	UID         string `db:"repo_uid"             json:"uid"`
	Path        string `db:"repo_path"            json:"path"`
	Description string `db:"repo_description"     json:"description"`
	IsPublic    bool   `db:"repo_is_public"       json:"is_public"`
	CreatedBy   int64  `db:"repo_created_by"      json:"created_by"`
	Created     int64  `db:"repo_created"         json:"created"`
	Updated     int64  `db:"repo_updated"         json:"updated"`

	// Forking (omit isFork ... ForkID <= 0 is giving the same information)
	GitUID        string `db:"repo_git_uid"            json:"-"`
	DefaultBranch string `db:"repo_default_branch"     json:"default_branch"`
	ForkID        int64  `db:"repo_fork_id"            json:"fork_id"`

	// TODO: Check if we want to keep those values here
	NumForks       int `db:"repo_num_forks"            json:"num_forks"`
	NumPulls       int `db:"repo_num_pulls"            json:"num_pulls"`
	NumClosedPulls int `db:"repo_num_closed_pulls"     json:"num_closed_pulls"`
	NumOpenPulls   int `db:"repo_num_open_pulls"       json:"num_open_pulls"`

	// git urls
	GitURL string `db:"-" json:"git_url"`
}

// RepoFilter stores repo query parameters.
type RepoFilter struct {
	Page  int           `json:"page"`
	Size  int           `json:"size"`
	Query string        `json:"query"`
	Sort  enum.RepoAttr `json:"sort"`
	Order enum.Order    `json:"order"`
}
