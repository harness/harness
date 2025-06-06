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

	"github.com/gotidy/ptr"
)

// PullReq represents a pull request.
type PullReq struct {
	ID      int64 `json:"-"` // not returned, it's an internal field
	Version int64 `json:"-"` // not returned, it's an internal field
	Number  int64 `json:"number"`

	CreatedBy int64  `json:"-"` // not returned, because the author info is in the Author field
	Created   int64  `json:"created"`
	Updated   int64  `json:"updated"`
	Edited    int64  `json:"edited"` // TODO: Remove. Field Edited is equal to Updated
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

	MergedBy                *int64            `json:"-"` // not returned, because the merger info is in the Merger field
	Merged                  *int64            `json:"merged"`
	MergeMethod             *enum.MergeMethod `json:"merge_method"`
	MergeViolationsBypassed *bool             `json:"merge_violations_bypassed"`

	MergeTargetSHA *string `json:"merge_target_sha"`
	MergeBaseSHA   string  `json:"merge_base_sha"`
	MergeSHA       *string `json:"-"` // TODO: either remove or ensure it's being set (merge dry-run)

	MergeCheckStatus  enum.MergeCheckStatus `json:"merge_check_status"`
	MergeConflicts    []string              `json:"merge_conflicts,omitempty"`
	RebaseCheckStatus enum.MergeCheckStatus `json:"rebase_check_status"`
	RebaseConflicts   []string              `json:"rebase_conflicts,omitempty"`

	Author PrincipalInfo  `json:"author"`
	Merger *PrincipalInfo `json:"merger"`
	Stats  PullReqStats   `json:"stats"`

	Labels       []*LabelPullReqAssignmentInfo `json:"labels,omitempty"`
	CheckSummary *CheckCountSummary            `json:"check_summary,omitempty"`
	Rules        []RuleInfo                    `json:"rules,omitempty"`
}

func (pr *PullReq) UpdateMergeOutcome(method enum.MergeMethod, conflictFiles []string) {
	switch method {
	case enum.MergeMethodMerge, enum.MergeMethodSquash:
		if len(conflictFiles) > 0 {
			pr.MergeCheckStatus = enum.MergeCheckStatusConflict
			pr.MergeConflicts = conflictFiles
		} else {
			pr.MergeCheckStatus = enum.MergeCheckStatusMergeable
			pr.MergeConflicts = nil
		}
	case enum.MergeMethodRebase:
		if len(conflictFiles) > 0 {
			pr.RebaseCheckStatus = enum.MergeCheckStatusConflict
			pr.RebaseConflicts = conflictFiles
		} else {
			pr.RebaseCheckStatus = enum.MergeCheckStatusMergeable
			pr.RebaseConflicts = nil
		}
	case enum.MergeMethodFastForward:
		// fast-forward merge can't have conflicts
	}
}

func (pr *PullReq) MarkAsMergeUnchecked() {
	pr.MergeCheckStatus = enum.MergeCheckStatusUnchecked
	pr.MergeConflicts = nil
	pr.RebaseCheckStatus = enum.MergeCheckStatusUnchecked
	pr.RebaseConflicts = nil
}

func (pr *PullReq) MarkAsMergeable() {
	pr.MergeCheckStatus = enum.MergeCheckStatusMergeable
	pr.MergeConflicts = nil
}

func (pr *PullReq) MarkAsRebaseable() {
	pr.RebaseCheckStatus = enum.MergeCheckStatusMergeable
	pr.RebaseConflicts = nil
}

func (pr *PullReq) MarkAsMerged() {
	pr.MergeCheckStatus = enum.MergeCheckStatusMergeable
	pr.MergeConflicts = nil
	pr.RebaseCheckStatus = enum.MergeCheckStatusMergeable
	pr.RebaseConflicts = nil
}

// DiffStats holds summary of changes in git:
// total number of commits, number modified files and number of line changes.
type DiffStats struct {
	Commits      *int64 `json:"commits,omitempty"`
	FilesChanged *int64 `json:"files_changed,omitempty"`
	Additions    *int64 `json:"additions,omitempty"`
	Deletions    *int64 `json:"deletions,omitempty"`
}

func NewDiffStats(commitCount, fileCount, additions, deletions int) DiffStats {
	return DiffStats{
		Commits:      ptr.Int64(int64(commitCount)),
		FilesChanged: ptr.Int64(int64(fileCount)),
		Additions:    ptr.Int64(int64(additions)),
		Deletions:    ptr.Int64(int64(deletions)),
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
	Page               int                          `json:"page"`
	Size               int                          `json:"size"`
	Query              string                       `json:"query"`
	CreatedBy          []int64                      `json:"created_by"`
	SourceRepoID       int64                        `json:"-"` // caller should use source_repo_ref
	SourceRepoRef      string                       `json:"source_repo_ref"`
	SourceBranch       string                       `json:"source_branch"`
	TargetRepoID       int64                        `json:"-"`
	TargetBranch       string                       `json:"target_branch"`
	States             []enum.PullReqState          `json:"state"`
	Sort               enum.PullReqSort             `json:"sort"`
	Order              enum.Order                   `json:"order"`
	LabelID            []int64                      `json:"label_id"`
	ValueID            []int64                      `json:"value_id"`
	CommenterID        int64                        `json:"commenter_id"`
	ReviewerID         int64                        `json:"reviewer_id"`
	ReviewDecisions    []enum.PullReqReviewDecision `json:"review_decisions"`
	MentionedID        int64                        `json:"mentioned_id"`
	ExcludeDescription bool                         `json:"exclude_description"`
	CreatedFilter
	UpdatedFilter
	EditedFilter
	PullReqMetadataOptions

	// internal use only
	SpaceIDs        []int64
	RepoIDBlacklist []int64
}

type PullReqMetadataOptions struct {
	IncludeGitStats bool `json:"include_git_stats"`
	IncludeChecks   bool `json:"include_checks"`
	IncludeRules    bool `json:"include_rules"`
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

type UserGroupReviewer struct {
	PullReqID   int64 `json:"-"`
	UserGroupID int64 `json:"-"`

	CreatedBy int64 `json:"-"`
	Created   int64 `json:"created"`
	Updated   int64 `json:"updated"`

	RepoID int64 `json:"-"`

	UserGroup UserGroupInfo `json:"user_group"`
	AddedBy   PrincipalInfo `json:"added_by"`

	// invdividual decisions by group users
	UserDecisions []UserGroupReviewerDecision `json:"user_decisions,omitempty"`
	// derived user group decision: change_req > approved > reviewed > pending
	Decision enum.PullReqReviewDecision `json:"decision,omitempty"`
}

type UserGroupReviewerDecision struct {
	Decision enum.PullReqReviewDecision `json:"decision"`
	SHA      string                     `json:"sha,omitempty"`
	Reviewer PrincipalInfo              `json:"reviewer"`
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

type DefaultReviewerApprovalsResponse struct {
	MinimumRequiredCount       int              `json:"minimum_required_count"`
	MinimumRequiredCountLatest int              `json:"minimum_required_count_latest"`
	CurrentCount               int              `json:"current_count"`
	PrincipalIDs               []int64          `json:"-"`
	PrincipalInfos             []*PrincipalInfo `json:"principals"`
}

type MergeResponse struct {
	SHA            string           `json:"sha,omitempty"`
	BranchDeleted  bool             `json:"branch_deleted,omitempty"`
	RuleViolations []RuleViolations `json:"rule_violations,omitempty"`

	// values only returned on dryrun
	DryRunRules    bool               `json:"dry_run_rules,omitempty"`
	DryRun         bool               `json:"dry_run,omitempty"`
	Mergeable      bool               `json:"mergeable,omitempty"`
	ConflictFiles  []string           `json:"conflict_files,omitempty"`
	AllowedMethods []enum.MergeMethod `json:"allowed_methods,omitempty"`

	MinimumRequiredApprovalsCount       int `json:"minimum_required_approvals_count,omitempty"`
	MinimumRequiredApprovalsCountLatest int `json:"minimum_required_approvals_count_latest,omitempty"`

	DefaultReviewerApprovals []*DefaultReviewerApprovalsResponse `json:"default_reviewer_aprovals,omitempty"`

	RequiresCodeOwnersApproval       bool `json:"requires_code_owners_approval,omitempty"`
	RequiresCodeOwnersApprovalLatest bool `json:"requires_code_owners_approval_latest,omitempty"`
	RequiresCommentResolution        bool `json:"requires_comment_resolution,omitempty"`
	RequiresNoChangeRequests         bool `json:"requires_no_change_requests,omitempty"`
}

type MergeViolations struct {
	Message        string           `json:"message,omitempty"`
	ConflictFiles  []string         `json:"conflict_files,omitempty"`
	RuleViolations []RuleViolations `json:"rule_violations,omitempty"`
}

type PullReqRepo struct {
	PullRequest *PullReq        `json:"pull_request"`
	Repository  *RepositoryCore `json:"repository"`
}

type RevertResponse struct {
	Branch string `json:"branch"`
	Commit Commit `json:"commit"`
}
