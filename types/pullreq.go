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

	ActivitySeq int64 `json:"-"` // not returned, because it's a server's internal field

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

	CreatedBy int64  `json:"-"` // not returned, because the author info is in the Author field
	Created   int64  `json:"created"`
	Updated   int64  `json:"updated"`
	Edited    int64  `json:"edited"`
	Deleted   *int64 `json:"deleted"`

	ParentID  *int64 `json:"parent_id"`
	RepoID    int64  `json:"repo_id"`
	PullReqID int64  `json:"pullreq_id"`

	Order    int64 `json:"order"`
	SubOrder int64 `json:"sub_order"`
	ReplySeq int64 `json:"-"` // not returned, because it's a server's internal field

	Type enum.PullReqActivityType `json:"type"`
	Kind enum.PullReqActivityKind `json:"kind"`

	Text     string                 `json:"text"`
	Payload  map[string]interface{} `json:"payload"`
	Metadata map[string]interface{} `json:"metadata"`

	ResolvedBy *int64 `json:"-"` // not returned, because the resolver info is in the Resolver field
	Resolved   *int64 `json:"resolved"`

	Author   PrincipalInfo  `json:"author"`
	Resolver *PrincipalInfo `json:"resolver"`
}

func (a *PullReqActivity) IsReplyable() bool {
	return (a.Type == enum.PullReqActivityTypeComment || a.Type == enum.PullReqActivityTypeCodeComment) &&
		a.SubOrder == 0
}

func (a *PullReqActivity) IsReply() bool {
	return a.SubOrder > 0
}

// PullReqActivityFilter stores pull request activity query parameters.
type PullReqActivityFilter struct {
	Since int64 `json:"since"`
	Until int64 `json:"until"`
	Limit int   `json:"limit"`

	Types []enum.PullReqActivityType `json:"type"`
	Kinds []enum.PullReqActivityKind `json:"kind"`
}
