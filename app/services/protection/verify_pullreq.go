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
	"errors"
	"fmt"

	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"golang.org/x/exp/slices"
)

type (
	MergeVerifier interface {
		MergeVerify(ctx context.Context, in MergeVerifyInput) (MergeVerifyOutput, []types.RuleViolations, error)
	}

	MergeVerifyInput struct {
		Actor        *types.Principal
		AllowBypass  bool
		IsRepoOwner  bool
		TargetRepo   *types.Repository
		SourceRepo   *types.Repository
		PullReq      *types.PullReq
		Reviewers    []*types.PullReqReviewer
		Method       enum.MergeMethod
		CheckResults []types.CheckResult
		CodeOwners   *codeowners.Evaluation
	}

	MergeVerifyOutput struct {
		DeleteSourceBranch bool
		AllowedMethods     []enum.MergeMethod
	}
)

// ensures that the DefPullReq type implements Sanitizer and MergeVerifier interface.
var (
	_ Sanitizer     = (*DefPullReq)(nil)
	_ MergeVerifier = (*DefPullReq)(nil)
)

const (
	codePullReqApprovalReqMinCount                   = "pullreq.approvals.require_minimum_count"
	codePullReqApprovalReqLatestCommit               = "pullreq.approvals.require_latest_commit"
	codePullReqApprovalReqCodeOwnersNoApproval       = "pullreq.approvals.require_code_owners:no_approval"
	codePullReqApprovalReqCodeOwnersChangeRequested  = "pullreq.approvals.require_code_owners:change_requested"
	codePullReqApprovalReqCodeOwnersNoLatestApproval = "pullreq.approvals.require_code_owners:no_latest_approval"
	codePullReqCommentsReqResolveAll                 = "pullreq.comments.require_resolve_all"
	codePullReqStatusChecksReqUIDs                   = "pullreq.status_checks.required_uids"
	codePullReqMergeStrategiesAllowed                = "pullreq.merge.strategies_allowed"
	codePullReqMergeDeleteBranch                     = "pullreq.merge.delete_branch"
)

//nolint:gocognit // well aware of this
func (v *DefPullReq) MergeVerify(
	_ context.Context,
	in MergeVerifyInput,
) (MergeVerifyOutput, []types.RuleViolations, error) {
	var out MergeVerifyOutput
	var violations types.RuleViolations

	out.DeleteSourceBranch = v.Merge.DeleteBranch

	// pullreq.approvals

	approvedBy := make([]types.PrincipalInfo, 0, len(in.Reviewers))
	for _, reviewer := range in.Reviewers {
		if reviewer.ReviewDecision != enum.PullReqReviewDecisionApproved {
			continue
		}
		if v.Approvals.RequireLatestCommit && reviewer.SHA != in.PullReq.SourceSHA {
			continue
		}

		approvedBy = append(approvedBy, reviewer.Reviewer)
	}

	if len(approvedBy) < v.Approvals.RequireMinimumCount {
		violations.Addf(codePullReqApprovalReqMinCount,
			"Insufficient number of approvals. Have %d but need at least %d.",
			len(approvedBy), v.Approvals.RequireMinimumCount)
	}

	if v.Approvals.RequireCodeOwners {
		for _, entry := range in.CodeOwners.EvaluationEntries {
			reviewDecision, approvers := getCodeOwnerApprovalStatus(entry.OwnerEvaluations)

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
			latestSHAApproved := slices.ContainsFunc(approvers, func(ev codeowners.OwnerEvaluation) bool {
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

	for _, requiredUID := range v.StatusChecks.RequireUIDs {
		var succeeded bool

		for i := range in.CheckResults {
			if in.CheckResults[i].UID == requiredUID {
				succeeded = in.CheckResults[i].Status == enum.CheckStatusSuccess
				break
			}
		}

		if !succeeded {
			violations.Add(codePullReqStatusChecksReqUIDs,
				"At least one required status check hasn't completed successfully.")
		}
	}

	// pullreq.merge

	if in.Method == "" {
		out.AllowedMethods = enum.MergeMethods
	}

	// Note: Empty allowed strategies list means all are allowed
	if len(v.Merge.StrategiesAllowed) > 0 {
		if in.Method != "" {
			// if the Method is provided report violations if any
			if !slices.Contains(v.Merge.StrategiesAllowed, in.Method) {
				violations.Addf(codePullReqMergeStrategiesAllowed,
					"The requested merge strategy %q is not allowed. Allowed strategies are %v.",
					in.Method, v.Merge.StrategiesAllowed)
			}
		} else {
			// if the Method isn't provided return allowed strategies
			out.AllowedMethods = v.Merge.StrategiesAllowed
		}
	}

	if len(violations.Violations) > 0 {
		return out, []types.RuleViolations{violations}, nil
	}

	return out, nil, nil
}

type DefApprovals struct {
	RequireCodeOwners   bool `json:"require_code_owners,omitempty"`
	RequireMinimumCount int  `json:"require_minimum_count,omitempty"`
	RequireLatestCommit bool `json:"require_latest_commit,omitempty"`
}

func (v *DefApprovals) Sanitize() error {
	if v.RequireMinimumCount < 0 {
		return errors.New("minimum count must be zero or a positive integer")
	}

	if v.RequireLatestCommit && v.RequireMinimumCount == 0 && !v.RequireCodeOwners {
		return errors.New("require latest commit can only be used with require code owners or require minimum count")
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
	RequireUIDs []string `json:"require_uids,omitempty"`
}

func (v *DefStatusChecks) Sanitize() error {
	if err := validateUIDSlice(v.RequireUIDs); err != nil {
		return fmt.Errorf("required UIDs error: %w", err)
	}

	return nil
}

type DefMerge struct {
	StrategiesAllowed []enum.MergeMethod `json:"strategies_allowed,omitempty"`
	DeleteBranch      bool               `json:"delete_branch,omitempty"`
}

func (v *DefMerge) Sanitize() error {
	m := make(map[enum.MergeMethod]struct{}, 0)
	for _, strategy := range v.StrategiesAllowed {
		if _, ok := strategy.Sanitize(); !ok {
			return fmt.Errorf("unrecognized merge strategy: %s", strategy)
		}

		if _, ok := m[strategy]; ok {
			return fmt.Errorf("duplicate entry in merge strategy list: %s", strategy)
		}

		m[strategy] = struct{}{}
	}

	slices.Sort(v.StrategiesAllowed)

	return nil
}

type DefPush struct {
	Block bool `json:"block,omitempty"`
}

func (v *DefPush) Sanitize() error {
	return nil
}

type DefPullReq struct {
	Approvals    DefApprovals    `json:"approvals"`
	Comments     DefComments     `json:"comments"`
	StatusChecks DefStatusChecks `json:"status_checks"`
	Merge        DefMerge        `json:"merge"`
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

	return nil
}

func getCodeOwnerApprovalStatus(
	ownerStatus []codeowners.OwnerEvaluation,
) (enum.PullReqReviewDecision, []codeowners.OwnerEvaluation) {
	approvers := make([]codeowners.OwnerEvaluation, 0)
	for _, o := range ownerStatus {
		if o.ReviewDecision == enum.PullReqReviewDecisionChangeReq {
			return enum.PullReqReviewDecisionChangeReq, nil
		}
		if o.ReviewDecision == enum.PullReqReviewDecisionApproved {
			approvers = append(approvers, o)
		}
	}
	if len(approvers) > 0 {
		return enum.PullReqReviewDecisionApproved, approvers
	}
	return enum.PullReqReviewDecisionPending, nil
}
