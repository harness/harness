// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import (
	"github.com/harness/gitness/types/enum"
)

// PullReq represents a pull request.
type PullReq struct {
	ID        int64 `db:"pullreq_id"         json:"id"`
	CreatedBy int64 `db:"pullreq_created_by" json:"-"`
	Created   int64 `db:"pullreq_created"    json:"created"`
	Updated   int64 `db:"pullreq_updated"    json:"updated"`
	Number    int64 `db:"pullreq_number"    json:"number"`

	State enum.PullReqState `db:"pullreq_state" json:"state"`

	Title       string `db:"pullreq_title"        json:"title"`
	Description string `db:"pullreq_description"  json:"description"`

	SourceRepoID int64  `db:"pullreq_source_repo_id"   json:"source_repo_id"`
	SourceBranch string `db:"pullreq_source_branch"    json:"source_branch"`
	TargetRepoID int64  `db:"pullreq_target_repo_id"   json:"target_repo_id"`
	TargetBranch string `db:"pullreq_target_branch"    json:"target_branch"`

	MergedBy      *int64  `db:"pullreq_merged_by"        json:"-"`
	Merged        *int64  `db:"pullreq_merged"           json:"merged"`
	MergeStrategy *string `db:"pullreq_merge_strategy"   json:"merge_strategy"`
}

// PullReqFilter stores pull request query parameters.
type PullReqFilter struct {
	Page      int                 `json:"page"`
	Size      int                 `json:"size"`
	Query     string              `json:"query"`
	CreatedBy int64               `json:"created_by"`
	States    []enum.PullReqState `json:"state"`
	Sort      enum.PullReqSort    `json:"sort"`
	Order     enum.Order          `json:"direction"`
}

// PullReqInfo is used to fetch pull request data from the database.
// The object should be later re-packed into a different struct to return it as an API response.
type PullReqInfo struct {
	PullReq
	AuthorID    int64   `db:"author_id"`
	AuthorName  string  `db:"author_name"`
	AuthorEmail string  `db:"author_email"`
	MergerID    *int64  `db:"merger_id"`
	MergerName  *string `db:"merger_name"`
	MergerEmail *string `db:"merger_email"`
}
