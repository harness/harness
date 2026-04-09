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

package protection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type (
	MergeVerifier interface {
		MergeVerify(ctx context.Context, in MergeVerifyInput) (MergeVerifyOutput, []types.RuleViolations, error)
		RequiredChecks(ctx context.Context, in RequiredChecksInput) (RequiredChecksOutput, error)
	}

	MergeVerifyInput struct {
		ResolveUserGroupIDs func(ctx context.Context, userGroupIDs []int64) ([]int64, error)
		MapUserGroupIDs     func(ctx context.Context, userGroupIDs []int64) (map[int64][]*types.Principal, error)
		Actor               *types.Principal
		AllowBypass         bool
		IsRepoOwner         bool
		TargetRepo          *types.RepositoryCore
		SourceRepo          *types.RepositoryCore
		PullReq             *types.PullReq
		Reviewers           []*types.PullReqReviewer
		Method              enum.MergeMethod // the method can be empty for dry run or dry run rules
		CheckResults        []types.CheckResult
		CodeOwners          *codeowners.Evaluation
	}

	MergeVerifyOutput struct {
		AllowedMethods                      []enum.MergeMethod
		DeleteSourceBranch                  bool
		MinimumRequiredApprovalsCount       int
		MinimumRequiredApprovalsCountLatest int
		RequiresCodeOwnersApproval          bool
		RequiresCodeOwnersApprovalLatest    bool
		RequiresCommentResolution           bool
		RequiresNoChangeRequests            bool
		RequiresBypassMessage               bool
		DefaultReviewerApprovals            []*types.DefaultReviewerApprovalsResponse
	}

	RequiredChecksInput struct {
		ResolveUserGroupID func(ctx context.Context, userGroupIDs []int64) ([]int64, error)
		Actor              *types.Principal
		IsRepoOwner        bool
		Repo               *types.RepositoryCore
		PullReq            *types.PullReq
	}

	RequiredChecksOutput struct {
		RequiredIdentifiers   map[string]struct{}
		BypassableIdentifiers map[string]struct{}
	}

	MergeQueueSetup struct {
		RequiredChecks          []string `json:"required_checks"`
		GroupSize               int      `json:"group_size"`
		ChecksConcurrency       int      `json:"checks_concurrency"`
		MaxCheckDurationSeconds int      `json:"max_check_duration_seconds"`
	}

	MergeQueueBranchUpdateInput struct {
		Repo         *types.RepositoryCore
		TargetBranch string
	}

	MergeQueueBranchUpdateVerifier interface {
		MergeQueueBranchUpdateVerify(in MergeQueueBranchUpdateInput) ([]types.RuleViolations, error)
	}

	MergeQueueSetupInput struct {
		Repo         *types.RepositoryCore
		TargetBranch string
	}

	MergeQueueSetupGetter interface {
		GetMergeQueueSetup(in MergeQueueSetupInput) (MergeQueueSetup, error)
	}

	CreatePullReqVerifier interface {
		CreatePullReqVerify(
			ctx context.Context,
			in CreatePullReqVerifyInput,
		) (CreatePullReqVerifyOutput, []types.RuleViolations, error)
	}

	CreatePullReqVerifyInput struct {
		ResolveUserGroupID func(ctx context.Context, userGroupIDs []int64) ([]int64, error)
		Actor              *types.Principal
		AllowBypass        bool
		IsRepoOwner        bool
		DefaultBranch      string
		TargetBranch       string
		RepoID             int64
		RepoIdentifier     string
	}

	CreatePullReqVerifyOutput struct {
		RequestCodeOwners       bool
		DefaultReviewerIDs      []int64
		DefaultGroupReviewerIDs []int64
	}
)

// Ensures that the DefPullReq type implements MergeVerifier and CreatePullReqVerifier interface.
var (
	_ MergeVerifier         = (*DefPullReq)(nil)
	_ CreatePullReqVerifier = (*DefPullReq)(nil)
)

const (
	codePullReqApprovalReqMinCount                      = "pullreq.approvals.require_minimum_count"
	codePullReqApprovalReqMinCountLatest                = "pullreq.approvals.require_minimum_count:latest_commit"
	codePullReqApprovalReqDefaultReviewerMinCount       = "pullreq.approvals.require_default_reviewer_minimum_count"
	codePullReqApprovalReqDefaultReviewerMinCountLatest = "pullreq.approvals.require_default_reviewer_minimum_count:latest_commit" //nolint:lll

	codePullReqApprovalReqLatestCommit          = "pullreq.approvals.require_latest_commit"
	codePullReqApprovalReqChangeRequested       = "pullreq.approvals.require_change_requested"
	codePullReqApprovalReqChangeRequestedOldSHA = "pullreq.approvals.require_change_requested_old_SHA"

	codePullReqApprovalReqCodeOwnersNoApproval       = "pullreq.approvals.require_code_owners:no_approval"
	codePullReqApprovalReqCodeOwnersChangeRequested  = "pullreq.approvals.require_code_owners:change_requested"
	codePullReqApprovalReqCodeOwnersNoLatestApproval = "pullreq.approvals.require_code_owners:no_latest_approval"

	codePullReqMergeStrategiesAllowed = "pullreq.merge.strategies_allowed"
	codePullReqMergeDeleteBranch      = "pullreq.merge.delete_branch"
	codePullReqMergeBlock             = "pullreq.merge.blocked"

	codeMergeQueueBranchUpdateVerify = "pullreq.merge_queue.branch_change_block"

	codePullReqCommentsReqResolveAll      = "pullreq.comments.require_resolve_all"
	codePullReqStatusChecksReqIdentifiers = "pullreq.status_checks.required_identifiers"
)

//nolint:gocognit,gocyclo,cyclop // well aware of this
func (v *DefPullReq) MergeVerify(
	ctx context.Context,
	in MergeVerifyInput,
) (MergeVerifyOutput, []types.RuleViolations, error) {
	var out MergeVerifyOutput
	var violations types.RuleViolations

	// set static merge verify output that comes from the PR definition
	out.DeleteSourceBranch = v.Merge.DeleteBranch
	out.RequiresCommentResolution = v.Comments.RequireResolveAll
	out.RequiresNoChangeRequests = v.Approvals.RequireNoChangeRequest

	// output that depends on approval of latest commit
	if v.Approvals.RequireLatestCommit {
		out.RequiresCodeOwnersApprovalLatest = v.Approvals.RequireCodeOwners
		out.MinimumRequiredApprovalsCountLatest = v.Approvals.RequireMinimumCount
	} else {
		out.RequiresCodeOwnersApproval = v.Approvals.RequireCodeOwners
		out.MinimumRequiredApprovalsCount = v.Approvals.RequireMinimumCount
	}

	// pullreq.approvals

	reviewerMap := make(map[int64]*types.PullReqReviewer)
	approvedBy := make(map[int64]struct{})
	for _, reviewer := range in.Reviewers {
		reviewerMap[reviewer.Reviewer.ID] = reviewer
		switch reviewer.ReviewDecision {
		case enum.PullReqReviewDecisionApproved:
			if v.Approvals.RequireLatestCommit && reviewer.SHA != in.PullReq.SourceSHA {
				continue
			}
			approvedBy[reviewer.Reviewer.ID] = struct{}{}
		case enum.PullReqReviewDecisionChangeReq:
			if v.Approvals.RequireNoChangeRequest {
				if reviewer.SHA == in.PullReq.SourceSHA {
					violations.Addf(
						codePullReqApprovalReqChangeRequested,
						"Reviewer %s requested changes",
						reviewer.Reviewer.DisplayName,
					)
				} else {
					violations.Addf(
						codePullReqApprovalReqChangeRequestedOldSHA,
						"Reviewer %s requested changes for an older commit",
						reviewer.Reviewer.DisplayName,
					)
				}
			}
		case enum.PullReqReviewDecisionPending,
			enum.PullReqReviewDecisionReviewed:
		}
	}

	if len(approvedBy) < v.Approvals.RequireMinimumCount {
		if v.Approvals.RequireLatestCommit {
			violations.Addf(codePullReqApprovalReqMinCountLatest,
				"Insufficient number of approvals of the latest commit. Have %d but need at least %d.",
				len(approvedBy), v.Approvals.RequireMinimumCount)
		} else {
			violations.Addf(codePullReqApprovalReqMinCount,
				"Insufficient number of approvals. Have %d but need at least %d.",
				len(approvedBy), v.Approvals.RequireMinimumCount)
		}
	}

	defaultReviewerIDs := make([]int64, 0, len(v.Reviewers.DefaultReviewerIDs))
	uniqueDefaultReviewerIDs := make(map[int64]struct{})
	for _, id := range v.Reviewers.DefaultReviewerIDs {
		if id != in.PullReq.Author.ID {
			defaultReviewerIDs = append(defaultReviewerIDs, id)
			uniqueDefaultReviewerIDs[id] = struct{}{}
		}
	}

	if in.MapUserGroupIDs != nil {
		userGroupsMap, err := in.MapUserGroupIDs(ctx, v.Reviewers.DefaultUserGroupReviewerIDs)
		if err != nil {
			return MergeVerifyOutput{}, []types.RuleViolations{},
				fmt.Errorf("failed to map principals to user group ids: %w", err)
		}

		for _, principals := range userGroupsMap {
			for _, principal := range principals {
				uniqueDefaultReviewerIDs[principal.ID] = struct{}{}
			}
		}
	}

	effectiveDefaultReviewerIDs := maps.Keys(uniqueDefaultReviewerIDs)
	var evaluations []*types.ReviewerEvaluation
	for id := range uniqueDefaultReviewerIDs {
		if reviewer, ok := reviewerMap[id]; ok {
			evaluations = append(evaluations, &types.ReviewerEvaluation{
				Reviewer: reviewer.Reviewer,
				SHA:      reviewer.SHA,
				Decision: reviewer.ReviewDecision,
			})
		}
	}

	// if author is default reviewer and required minimum == number of default reviewers, reduce minimum by one.
	effectiveMinimumRequiredDefaultReviewerCount := v.Approvals.RequireMinimumDefaultReviewerCount
	if len(effectiveDefaultReviewerIDs) < len(v.Reviewers.DefaultReviewerIDs) &&
		len(v.Reviewers.DefaultReviewerIDs) == v.Approvals.RequireMinimumDefaultReviewerCount {
		effectiveMinimumRequiredDefaultReviewerCount--
	}

	//nolint:nestif
	if effectiveMinimumRequiredDefaultReviewerCount > 0 {
		var defaultReviewerApprovalCount int
		for _, id := range effectiveDefaultReviewerIDs {
			if _, ok := approvedBy[id]; ok {
				defaultReviewerApprovalCount++
			}
		}
		if defaultReviewerApprovalCount < effectiveMinimumRequiredDefaultReviewerCount {
			if v.Approvals.RequireLatestCommit {
				violations.Addf(codePullReqApprovalReqDefaultReviewerMinCountLatest,
					"Insufficient number of default reviewer approvals of the latest commit. Have %d but need at least %d.",
					defaultReviewerApprovalCount, effectiveMinimumRequiredDefaultReviewerCount)
			} else {
				violations.Addf(codePullReqApprovalReqDefaultReviewerMinCount,
					"Insufficient number of default reviewer approvals. Have %d but need at least %d.",
					defaultReviewerApprovalCount, effectiveMinimumRequiredDefaultReviewerCount)
			}
		}

		out.DefaultReviewerApprovals = []*types.DefaultReviewerApprovalsResponse{{
			PrincipalIDs: defaultReviewerIDs,
			UserGroupIDs: v.Reviewers.DefaultUserGroupReviewerIDs,
			CurrentCount: defaultReviewerApprovalCount,
			Evaluations:  evaluations,
		}}
		if v.Approvals.RequireLatestCommit {
			out.DefaultReviewerApprovals[0].MinimumRequiredCountLatest = effectiveMinimumRequiredDefaultReviewerCount
		} else {
			out.DefaultReviewerApprovals[0].MinimumRequiredCount = effectiveMinimumRequiredDefaultReviewerCount
		}
	}

	if v.Approvals.RequireCodeOwners {
		for _, entry := range in.CodeOwners.EvaluationEntries {
			reviewDecision, approvers := getCodeOwnerApprovalStatus(entry)

			if reviewDecision == enum.PullReqReviewDecisionPending {
				violations.Addf(codePullReqApprovalReqCodeOwnersNoApproval,
					"Code owners approval pending for %q", entry.Pattern)
				continue
			}

			if reviewDecision == enum.PullReqReviewDecisionChangeReq {
				violations.Addf(codePullReqApprovalReqCodeOwnersChangeRequested,
					"Code owners requested changes for %q", entry.Pattern)
				continue
			}

			// pull req approved. check other settings
			if !v.Approvals.RequireLatestCommit {
				continue
			}
			latestSHAApproved := slices.ContainsFunc(approvers, func(ev codeowners.UserEvaluation) bool {
				return ev.ReviewSHA == in.PullReq.SourceSHA
			})
			if !latestSHAApproved {
				violations.Addf(codePullReqApprovalReqCodeOwnersNoLatestApproval,
					"Code owners approval pending on latest commit for %q", entry.Pattern)
			}
		}
	}

	// pullreq.comments

	if v.Comments.RequireResolveAll && in.PullReq.UnresolvedCount > 0 {
		violations.Addf(codePullReqCommentsReqResolveAll,
			"All comments must be resolved. There are %d unresolved comments.",
			in.PullReq.UnresolvedCount)
	}

	// pullreq.status_checks

	var violatingStatusCheckIdentifiers []string
	for _, requiredIdentifier := range v.StatusChecks.RequireIdentifiers {
		var succeeded bool
		for i := range in.CheckResults {
			if in.CheckResults[i].Identifier == requiredIdentifier {
				succeeded = in.CheckResults[i].Status.IsSuccess()
				break
			}
		}

		if !succeeded {
			violatingStatusCheckIdentifiers = append(violatingStatusCheckIdentifiers, requiredIdentifier)
		}
	}

	if len(violatingStatusCheckIdentifiers) > 0 {
		violations.Addf(
			codePullReqStatusChecksReqIdentifiers,
			"The following status checks are required to be completed successfully: %s",
			strings.Join(violatingStatusCheckIdentifiers, ", "),
		)
	}

	// pullreq.merge

	out.AllowedMethods = enum.MergeMethods

	// Note: Empty allowed strategies list means all are allowed
	if len(v.Merge.StrategiesAllowed) > 0 {
		// if the Method isn't provided return allowed strategies
		out.AllowedMethods = v.Merge.StrategiesAllowed

		if in.Method != "" {
			// if the Method is provided report violations if any
			if !slices.Contains(v.Merge.StrategiesAllowed, in.Method) {
				violations.Addf(codePullReqMergeStrategiesAllowed,
					"The requested merge strategy %q is not allowed. Allowed strategies are %v.",
					in.Method, v.Merge.StrategiesAllowed)
			}
		}
	}

	if v.Merge.Block {
		violations.Addf(
			codePullReqMergeBlock,
			"The merge for the branch %s is not allowed.", in.PullReq.TargetBranch)
	}

	if len(violations.Violations) > 0 {
		return out, []types.RuleViolations{violations}, nil
	}

	return out, nil, nil
}

func (v *DefPullReq) RequiredChecks(
	_ context.Context,
	_ RequiredChecksInput,
) (RequiredChecksOutput, error) {
	m := make(map[string]struct{}, len(v.StatusChecks.RequireIdentifiers))
	for _, id := range v.StatusChecks.RequireIdentifiers {
		m[id] = struct{}{}
	}
	return RequiredChecksOutput{
		RequiredIdentifiers: m,
	}, nil
}

func (v *DefPullReq) CreatePullReqVerify(
	context.Context,
	CreatePullReqVerifyInput,
) (CreatePullReqVerifyOutput, []types.RuleViolations, error) {
	var out CreatePullReqVerifyOutput

	out.RequestCodeOwners = v.Reviewers.RequestCodeOwners
	out.DefaultReviewerIDs = v.Reviewers.DefaultReviewerIDs
	out.DefaultGroupReviewerIDs = v.Reviewers.DefaultUserGroupReviewerIDs

	return out, nil, nil
}

type DefApprovals struct {
	RequireCodeOwners                  bool `json:"require_code_owners,omitempty"`
	RequireMinimumCount                int  `json:"require_minimum_count,omitempty"`
	RequireLatestCommit                bool `json:"require_latest_commit,omitempty"`
	RequireNoChangeRequest             bool `json:"require_no_change_request,omitempty"`
	RequireMinimumDefaultReviewerCount int  `json:"require_minimum_default_reviewer_count,omitempty"`
}

func (v *DefApprovals) Sanitize() error {
	if v.RequireMinimumCount < 0 {
		return errors.InvalidArgument("Require minimum count must be zero or a positive integer.")
	}

	if v.RequireLatestCommit && !v.RequireCodeOwners &&
		v.RequireMinimumCount == 0 && v.RequireMinimumDefaultReviewerCount == 0 {
		return errors.InvalidArgument("Require latest commit can only be used with require code owners, " +
			"require minimum count or require default reviewer minimum count.")
	}

	return nil
}

type DefComments struct {
	RequireResolveAll bool `json:"require_resolve_all,omitempty"`
}

func (DefComments) Sanitize() error {
	return nil
}

type DefStatusChecks struct {
	RequireIdentifiers []string `json:"require_identifiers,omitempty"`
}

// TODO [CODE-1363]: remove after identifier migration.
func (c DefStatusChecks) MarshalJSON() ([]byte, error) {
	// alias allows us to embed the original object while avoiding an infinite loop of marshaling.
	type alias DefStatusChecks
	return json.Marshal(&struct {
		alias
		RequireUIDs []string `json:"require_uids"`
	}{
		alias:       (alias)(c),
		RequireUIDs: c.RequireIdentifiers,
	})
}

// TODO [CODE-1363]: remove if we don't have any require_uids left in our DB.
func (c *DefStatusChecks) UnmarshalJSON(data []byte) error {
	// alias allows us to extract the original object while avoiding an infinite loop of unmarshaling.
	type alias DefStatusChecks
	res := struct {
		alias
		RequireUIDs []string `json:"require_uids"`
	}{}

	err := json.Unmarshal(data, &res)
	if err != nil {
		return fmt.Errorf("failed to unmarshal to alias type with required uids: %w", err)
	}

	*c = DefStatusChecks(res.alias)
	if len(c.RequireIdentifiers) == 0 {
		c.RequireIdentifiers = res.RequireUIDs
	}

	return nil
}

func (c *DefStatusChecks) Sanitize() error {
	if err := validateIdentifierSlice(c.RequireIdentifiers); err != nil {
		return fmt.Errorf("required identifiers error: %w", err)
	}

	return nil
}

type DefMerge struct {
	StrategiesAllowed    []enum.MergeMethod `json:"strategies_allowed,omitempty"`
	DeleteBranch         bool               `json:"delete_branch,omitempty"`
	Block                bool               `json:"block,omitempty"`
	RequireBypassMessage bool               `json:"require_bypass_message,omitempty"`
}

func (v *DefMerge) Sanitize() error {
	m := make(map[enum.MergeMethod]struct{}, 0)
	for _, strategy := range v.StrategiesAllowed {
		if _, ok := strategy.Sanitize(); !ok {
			return errors.InvalidArgumentf("Unrecognized merge strategy: %q.", strategy)
		}

		if _, ok := m[strategy]; ok {
			return errors.InvalidArgumentf("Duplicate entry in merge strategy list: %q.", strategy)
		}

		m[strategy] = struct{}{}
	}

	slices.Sort(v.StrategiesAllowed)

	return nil
}

type DefReviewers struct {
	RequestCodeOwners           bool    `json:"request_code_owners,omitempty"`
	DefaultReviewerIDs          []int64 `json:"default_reviewer_ids,omitempty"`
	DefaultUserGroupReviewerIDs []int64 `json:"default_user_group_reviewer_ids,omitempty"`
}

func (v *DefReviewers) Sanitize() error {
	if err := validateIDSlice(v.DefaultReviewerIDs); err != nil {
		return fmt.Errorf("default reviewer IDs error: %w", err)
	}

	if err := validateIDSlice(v.DefaultUserGroupReviewerIDs); err != nil {
		return fmt.Errorf("default user group reviewer IDs error: %w", err)
	}

	return nil
}

const MaxGroupSize = 10

type DefMergeQueue struct {
	StatusChecks            DefStatusChecks `json:"status_checks"`
	GroupSize               int             `json:"group_size"`
	ChecksConcurrency       int             `json:"checks_concurrency"`
	MaxCheckDurationSeconds int             `json:"max_check_duration_seconds"`
}

func (v *DefMergeQueue) Sanitize() error {
	if v == nil {
		return nil
	}

	if len(v.StatusChecks.RequireIdentifiers) == 0 {
		return errors.InvalidArgument(
			"To define a merge queue at least one required check must be specified.")
	}

	if v.GroupSize <= 0 || v.GroupSize >= MaxGroupSize {
		return errors.InvalidArgumentf(
			"Group size must be greater than 0 and less than %d.", MaxGroupSize)
	}

	if v.ChecksConcurrency <= 0 || v.ChecksConcurrency >= MaxGroupSize {
		return errors.InvalidArgumentf(
			"Checks concurrency must be greater than 0 and less than %d.", MaxGroupSize)
	}

	if v.MaxCheckDurationSeconds <= 0 {
		return errors.InvalidArgument("Max check duration seconds must be greater than 0.")
	}

	if err := v.StatusChecks.Sanitize(); err != nil {
		return fmt.Errorf("status checks: %w", err)
	}

	return nil
}

func (v *DefMergeQueue) MergeQueueBranchUpdateVerify(in MergeQueueBranchUpdateInput) ([]types.RuleViolations, error) {
	if v == nil || len(v.StatusChecks.RequireIdentifiers) == 0 {
		return nil, nil
	}

	var violations types.RuleViolations
	violations.Addf(codeMergeQueueBranchUpdateVerify,
		"Direct modification of branch %q is not allowed: the branch has a merge queue configured.",
		in.TargetBranch)

	return []types.RuleViolations{violations}, nil
}

func (v *DefMergeQueue) GetMergeQueueSetup(_ MergeQueueSetupInput) (MergeQueueSetup, error) {
	if v == nil {
		return MergeQueueSetup{}, nil
	}
	return MergeQueueSetup{
		RequiredChecks:          v.StatusChecks.RequireIdentifiers,
		GroupSize:               v.GroupSize,
		ChecksConcurrency:       v.ChecksConcurrency,
		MaxCheckDurationSeconds: v.MaxCheckDurationSeconds,
	}, nil
}

type DefPullReq struct {
	Approvals    DefApprovals    `json:"approvals"`
	Comments     DefComments     `json:"comments"`
	StatusChecks DefStatusChecks `json:"status_checks"`
	Merge        DefMerge        `json:"merge"`
	Reviewers    DefReviewers    `json:"reviewers"`
	MergeQueue   *DefMergeQueue  `json:"merge_queue,omitempty"`
}

func (v *DefPullReq) Sanitize() error {
	if err := v.Approvals.Sanitize(); err != nil {
		return fmt.Errorf("approvals: %w", err)
	}

	if err := v.Comments.Sanitize(); err != nil {
		return fmt.Errorf("comments: %w", err)
	}

	if err := v.StatusChecks.Sanitize(); err != nil {
		return fmt.Errorf("status checks: %w", err)
	}

	if err := v.Merge.Sanitize(); err != nil {
		return fmt.Errorf("merge: %w", err)
	}

	if err := v.Reviewers.Sanitize(); err != nil {
		return fmt.Errorf("reviewers: %w", err)
	}

	if err := v.MergeQueue.Sanitize(); err != nil {
		return fmt.Errorf("merge queue: %w", err)
	}

	return nil
}

func (v MergeQueueSetup) IsActive() bool {
	return len(v.RequiredChecks) > 0
}

func getCodeOwnerApprovalStatus(
	entry codeowners.EvaluationEntry,
) (enum.PullReqReviewDecision, []codeowners.UserEvaluation) {
	approvers := make([]codeowners.UserEvaluation, 0)

	// users
	for _, o := range entry.UserEvaluations {
		if o.ReviewDecision == enum.PullReqReviewDecisionChangeReq {
			return enum.PullReqReviewDecisionChangeReq, nil
		}
		if o.ReviewDecision == enum.PullReqReviewDecisionApproved {
			approvers = append(approvers, o)
		}
	}

	// usergroups
	for _, u := range entry.UserGroupEvaluations {
		for _, o := range u.Evaluations {
			if o.ReviewDecision == enum.PullReqReviewDecisionChangeReq {
				return enum.PullReqReviewDecisionChangeReq, nil
			}
			if o.ReviewDecision == enum.PullReqReviewDecisionApproved {
				approvers = append(approvers, o)
			}
		}
	}

	if len(approvers) > 0 {
		return enum.PullReqReviewDecisionApproved, approvers
	}
	return enum.PullReqReviewDecisionPending, nil
}
