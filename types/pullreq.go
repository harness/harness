// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import (
	"github.com/harness/gitness/types/enum"
)

// PullReq represents a pull request.
type PullReq struct {
	ID      int64 `json:"id"`
	Version int64 `json:"version"`
	Number  int64 `json:"number"`

	CreatedBy int64 `json:"-"` // not returned, because the author info is in the Author field
	Created   int64 `json:"created"`
	Updated   int64 `json:"updated"`
	Edited    int64 `json:"edited"`

	State enum.PullReqState `json:"state"`

	Title       string `json:"title"`
	Description string `json:"description"`

	SourceRepoID int64  `json:"source_repo_id"`
	SourceBranch string `json:"source_branch"`
	TargetRepoID int64  `json:"target_repo_id"`
	TargetBranch string `json:"target_branch"`

	PullReqActivitySeq int64 `json:"-"` // not returned, because it's a server internal field

	MergedBy      *int64  `json:"-"` // not returned, because the merger info is in the Merger field
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

// PullReqActivity represents a pull request activity.
type PullReqActivity struct {
	ID      int64 `json:"id"`
	Version int64 `json:"version"`

	CreatedBy int64 `json:"-"` // not returned, because the author info is in the Author field
	Created   int64 `json:"created"`
	Updated   int64 `json:"updated"`
	Edited    int64 `json:"edited"`
	Deleted   int64 `json:"deleted"`

	RepoID    int64 `json:"repo_id"`
	PullReqID int64 `json:"pullreq_id"`

	Seq    int64 `json:"seq"`
	SubSeq int64 `json:"subseq"`

	Type int64 `json:"type"`
	Kind int64 `json:"kind"`

	Text    string                 `json:"title"`
	Payload map[string]interface{} `json:"payload"`

	ResolvedBy *int64 `json:"-"` // not returned, because the resolver info is in the Resolver field
	Resolved   *int64 `json:"resolved"`

	Author   PrincipalInfo  `json:"author"`
	Resolver *PrincipalInfo `json:"resolver"`
}
