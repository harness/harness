// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import (
	"github.com/harness/gitness/types/enum"
)

// Repository represents a code repository.
type Repository struct {
	// Core properties
	ID          int64  `db:"repo_id"              json:"id"`
	Name        string `db:"repo_name"            json:"name"`
	SpaceID     int64  `db:"repo_spaceId"         json:"spaceId"`
	Path        string `db:"repo_path"            json:"path"`
	DisplayName string `db:"repo_displayName"     json:"displayName"`
	Description string `db:"repo_description"     json:"description"`
	IsPublic    bool   `db:"repo_isPublic"        json:"isPublic"`
	CreatedBy   int64  `db:"repo_createdBy"       json:"createdBy"`
	Created     int64  `db:"repo_created"         json:"created"`
	Updated     int64  `db:"repo_updated"         json:"updated"`

	// Forking (omit isFork ... ForkID <= 0 is giving the same information)
	ForkID int64 `db:"repo_forkId"             json:"forkId"`

	// TODO: Check if we want to keep those values here
	NumForks       int `db:"repo_numForks"             json:"numForks"`
	NumPulls       int `db:"repo_numPulls"             json:"numPulls"`
	NumClosedPulls int `db:"repo_numClosedPulls"       json:"numClosedPulls"`
	NumOpenPulls   int `db:"repo_numOpenPulls"         json:"numOpenPulls"`
}

// RepoFilter stores repo query parameters.
type RepoFilter struct {
	Page  int           `json:"page"`
	Size  int           `json:"size"`
	Sort  enum.RepoAttr `json:"sort"`
	Order enum.Order    `json:"direction"`
}
