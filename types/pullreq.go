// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import (
	"github.com/harness/gitness/types/enum"
)

// PullReq represents a pull request.
type PullReq struct {
	ID        int64 `json:"id"`
	CreatedBy int64 `json:"-"`
	Created   int64 `json:"created"`
	Updated   int64 `json:"updated"`
	Number    int64 `json:"number"`

	State enum.PullReqState `json:"state"`

	Title       string `json:"title"`
	Description string `json:"description"`

	SourceRepoID int64  `json:"source_repo_id"`
	SourceBranch string `json:"source_branch"`
	TargetRepoID int64  `json:"target_repo_id"`
	TargetBranch string `json:"target_branch"`

	MergedBy      *int64  `json:"-"`
	Merged        *int64  `json:"merged"`
	MergeStrategy *string `json:"merge_strategy"`

	Author PrincipalInfo  `json:"author"`
	Merger *PrincipalInfo `json:"merger"`
}

// PullReqFilter stores pull request query parameters.
type PullReqFilter struct {
	Page          int                 `json:"page"`
	Size          int                 `json:"size"`
	Query         string              `json:"query"`
	CreatedBy     int64               `json:"created_by"`
	SourceRepoID  int64               `json:"-"` // caller should use source_repo_ref
	SourceRepoRef string              `json:"source_repo_ref"`
	SourceBranch  string              `json:"source_branch"`
	TargetBranch  string              `json:"target_branch"`
	States        []enum.PullReqState `json:"state"`
	Sort          enum.PullReqSort    `json:"sort"`
	Order         enum.Order          `json:"direction"`
}
