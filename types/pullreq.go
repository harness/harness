// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import (
	"github.com/harness/gitness/types/enum"
)

// PullReq represents a pull request.
type PullReq struct {
	// TODO: int64 ID doesn't match DB
	ID        int64 `db:"pullreq_id"         json:"id"`
	CreatedBy int64 `db:"pullreq_created_by" json:"createdBy"`
	Created   int64 `db:"pullreq_created"    json:"created"`
	Updated   int64 `db:"pullreq_updated"    json:"updated"`
	Number    int64 `db:"pullreq_number"    json:"number"`

	State enum.PullReqState `db:"pullreq_state" json:"state"`

	Title       string `db:"pullreq_title"        json:"title"`
	Description string `db:"pullreq_description"  json:"description"`

	SourceRepoID int64  `db:"pullreq_source_repo_id"   json:"sourceRepoID"`
	SourceBranch string `db:"pullreq_source_branch"    json:"sourceBranch"`
	TargetRepoID int64  `db:"pullreq_target_repo_id"   json:"targetRepoID"`
	TargetBranch string `db:"pullreq_target_branch"    json:"targetBranch"`

	MergedBy      *int64  `db:"pullreq_merged_by"        json:"mergedBy"`
	Merged        *int64  `db:"pullreq_merged"           json:"merged"`
	MergeStrategy *string `db:"pullreq_merge_strategy"   json:"merge_strategy"`
}

// PullReqFilter stores pull request query parameters.
type PullReqFilter struct {
	Page      int                 `json:"page"`
	Size      int                 `json:"size"`
	Query     string              `json:"query"`
	CreatedBy int64               `json:"createdBy"`
	States    []enum.PullReqState `json:"state"`
	Sort      enum.PullReqSort    `json:"sort"`
	Order     enum.Order          `json:"direction"`
}
