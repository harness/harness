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
	ParentID    int64  `db:"repo_parentId"        json:"parentId"`
	UID         string `db:"repo_uid"             json:"uid"`
	Path        string `db:"repo_path"            json:"path"`
	Description string `db:"repo_description"     json:"description"`
	IsPublic    bool   `db:"repo_isPublic"        json:"isPublic"`
	CreatedBy   int64  `db:"repo_createdBy"       json:"createdBy"`
	Created     int64  `db:"repo_created"         json:"created"`
	Updated     int64  `db:"repo_updated"         json:"updated"`

	// Forking (omit isFork ... ForkID <= 0 is giving the same information)
	GitUID        string `db:"repo_gitUid"             json:"-"`
	DefaultBranch string `db:"repo_defaultBranch"      json:"defaultBranch"`
	ForkID        int64  `db:"repo_forkId"             json:"forkId"`

	// TODO: Check if we want to keep those values here
	NumForks       int `db:"repo_numForks"             json:"numForks"`
	NumPulls       int `db:"repo_numPulls"             json:"numPulls"`
	NumClosedPulls int `db:"repo_numClosedPulls"       json:"numClosedPulls"`
	NumOpenPulls   int `db:"repo_numOpenPulls"         json:"numOpenPulls"`

	// git urls
	URL string `db:"-" json:"url"`
}

// RepoFilter stores repo query parameters.
type RepoFilter struct {
	Page  int           `json:"page"`
	Size  int           `json:"size"`
	Query string        `json:"query"`
	Sort  enum.RepoAttr `json:"sort"`
	Order enum.Order    `json:"direction"`
}
