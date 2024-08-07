// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"github.com/harness/gitness/types/enum"
)

// PullReq represents a pull request.
type PullReq struct {
	ID      int64 `json:"-"` // not returned, it's an internal field
	Version int64 `json:"-"` // not returned, it's an internal field
	Number  int64 `json:"number"`

	CreatedBy int64  `json:"-"` // not returned, because the author info is in the Author field
	Created   int64  `json:"created"`
	Updated   int64  `json:"-"` // not returned, it's updated by the server internally. Clients should use EditedAt.
	Edited    int64  `json:"edited"`
	Closed    *int64 `json:"closed,omitempty"`

	State   enum.PullReqState `json:"state"`
	IsDraft bool              `json:"is_draft"`

	CommentCount    int `json:"-"` // returned as "conversations" in the Stats
	UnresolvedCount int `json:"-"` // returned as "unresolved_count" in the Stats

	Title       string `json:"title"`
	Description string `json:"description"`

	SourceRepoID int64  `json:"source_repo_id"`
	SourceBranch string `json:"source_branch"`
	SourceSHA    string `json:"source_sha"`
	TargetRepoID int64  `json:"target_repo_id"`
	TargetBranch string `json:"target_branch"`

	ActivitySeq int64 `json:"-"` // not returned, because it's a server's internal field

	MergedBy    *int64            `json:"-"` // not returned, because the merger info is in the Merger field
	Merged      *int64            `json:"merged"`
	MergeMethod *enum.MergeMethod `json:"merge_method"`

	MergeCheckStatus enum.MergeCheckStatus `json:"merge_check_status"`
	MergeTargetSHA   *string               `json:"merge_target_sha"`
	MergeBaseSHA     string                `json:"merge_base_sha"`
	MergeSHA         *string               `json:"-"` // TODO: either remove or ensure it's being set (merge dry-run)
	MergeConflicts   []string              `json:"merge_conflicts,omitempty"`

	Author PrincipalInfo  `json:"author"`
	Merger *PrincipalInfo `json:"merger"`
	Stats  PullReqStats   `json:"stats"`

	Labels []*LabelPullReqAssignmentInfo `json:"labels,omitempty"`
}

// DiffStats shows total number of commits and modified files.
type DiffStats struct {
	Commits      *int64 `json:"commits,omitempty"`
	FilesChanged *int64 `json:"files_changed,omitempty"`
	Additions    *int64 `json:"additions"`
	Deletions    *int64 `json:"deletions"`
}

func NewDiffStats(commitCount, fileCount, additions, deletions int) DiffStats {
	cc := int64(commitCount)
	fc := int64(fileCount)
	add := int64(additions)
	del := int64(deletions)
	return DiffStats{
		Commits:      &cc,
		FilesChanged: &fc,
		Additions:    &add,
		Deletions:    &del,
	}
}

// PullReqStats shows Diff statistics and number of conversations.
type PullReqStats struct {
	DiffStats
	Conversations   int `json:"conversations,omitempty"`
	UnresolvedCount int `json:"unresolved_count,omitempty"`
}

// PullReqFilter stores pull request query parameters.
type PullReqFilter struct {
	Page          int                 `json:"page"`
	Size          int                 `json:"size"`
	Query         string              `json:"query"`
	CreatedBy     []int64             `json:"created_by"`
	SourceRepoID  int64               `json:"-"` // caller should use source_repo_ref
	SourceRepoRef string              `json:"source_repo_ref"`
	SourceBranch  string              `json:"source_branch"`
	TargetRepoID  int64               `json:"-"`
	TargetBranch  string              `json:"target_branch"`
	States        []enum.PullReqState `json:"state"`
	Sort          enum.PullReqSort    `json:"sort"`
	Order         enum.Order          `json:"order"`
	LabelID       []int64             `json:"label_id"`
	ValueID       []int64             `json:"value_id"`
	CreatedFilter
}

// PullReqReview holds pull request review.
type PullReqReview struct {
	ID int64 `json:"id"`

	CreatedBy int64 `json:"created_by"`
	Created   int64 `json:"created"`
	Updated   int64 `json:"updated"`

	PullReqID int64 `json:"pullreq_id"`

	Decision enum.PullReqReviewDecision `json:"decision"`
	SHA      string                     `json:"sha"`
}

// PullReqReviewer holds pull request reviewer.
type PullReqReviewer struct {
	PullReqID   int64 `json:"-"`
	PrincipalID int64 `json:"-"`

	CreatedBy int64 `json:"-"`
	Created   int64 `json:"created"`
	Updated   int64 `json:"updated"`

	RepoID         int64                    `json:"-"`
	Type           enum.PullReqReviewerType `json:"type"`
	LatestReviewID *int64                   `json:"latest_review_id"`

	ReviewDecision enum.PullReqReviewDecision `json:"review_decision"`
	SHA            string                     `json:"sha"`

	Reviewer PrincipalInfo `json:"reviewer"`
	AddedBy  PrincipalInfo `json:"added_by"`
}

// PullReqFileView represents a file reviewed entry for a given pr and principal.
// NOTE: keep api lightweight and don't return unnecessary extra data.
type PullReqFileView struct {
	PullReqID   int64 `json:"-"`
	PrincipalID int64 `json:"-"`

	Path     string `json:"path"`
	SHA      string `json:"sha"`
	Obsolete bool   `json:"obsolete"`

	Created int64 `json:"-"`
	Updated int64 `json:"-"`
}

type MergeResponse struct {
	SHA            string           `json:"sha,omitempty"`
	BranchDeleted  bool             `json:"branch_deleted,omitempty"`
	RuleViolations []RuleViolations `json:"rule_violations,omitempty"`

	// values only returned on dryrun
	DryRun                              bool               `json:"dry_run,omitempty"`
	ConflictFiles                       []string           `json:"conflict_files,omitempty"`
	AllowedMethods                      []enum.MergeMethod `json:"allowed_methods,omitempty"`
	MinimumRequiredApprovalsCount       int                `json:"minimum_required_approvals_count,omitempty"`
	MinimumRequiredApprovalsCountLatest int                `json:"minimum_required_approvals_count_latest,omitempty"`
	RequiresCodeOwnersApproval          bool               `json:"requires_code_owners_approval,omitempty"`
	RequiresCodeOwnersApprovalLatest    bool               `json:"requires_code_owners_approval_latest,omitempty"`
	RequiresCommentResolution           bool               `json:"requires_comment_resolution,omitempty"`
	RequiresNoChangeRequests            bool               `json:"requires_no_change_requests,omitempty"`
}

type MergeViolations struct {
	ConflictFiles  []string         `json:"conflict_files,omitempty"`
	RuleViolations []RuleViolations `json:"rule_violations,omitempty"`
}
